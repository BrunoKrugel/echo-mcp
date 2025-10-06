// Package server provides zero-configuration conversion of Echo web APIs to Model Context Protocol (MCP) tools.
//
// This package automatically exposes your Echo routes as MCP tools that can be called by
// AI assistants and other MCP clients. It supports multiple schema generation methods
// including Swagger/OpenAPI documentation, manual schema registration, and automatic
// type inference.
//
// Basic usage:
//
//	e := echo.New()
//	e.GET("/users/:id", getUserHandler)
//	e.POST("/users", createUserHandler)
//
//	// Convert to MCP server
//	mcp := server.New(e, &server.Config{
//		BaseURL: "http://localhost:8080",
//	})
//
//	// Mount MCP endpoint
//	mcp.Mount("/mcp")
//
// For advanced usage with Swagger schemas:
//
//	mcp := server.New(e, &server.Config{
//		BaseURL:              "http://localhost:8080",
//		EnableSwaggerSchemas: true,
//	})
package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/bytedance/sonic"
	"github.com/labstack/echo/v4"

	"github.com/BrunoKrugel/echo-mcp/pkg/convert"
	"github.com/BrunoKrugel/echo-mcp/pkg/swagger"
	"github.com/BrunoKrugel/echo-mcp/pkg/transport"
	"github.com/BrunoKrugel/echo-mcp/pkg/types"
)

// EchoMCP represents an MCP server that exposes Echo routes as MCP tools.
// It handles the conversion of HTTP endpoints to MCP tool definitions and
// manages the execution of tool calls by forwarding them to the original Echo handlers.
type EchoMCP struct {
	transport         transport.Transport
	echo              *echo.Echo
	operations        map[string]types.Operation
	config            *Config
	registeredSchemas map[string]types.RegisteredSchemaInfo
	executeToolFunc   func(operationID string, parameters map[string]any) (any, error)
	name              string
	version           string
	description       string
	baseURL           string
	tools             []types.Tool
	includeEndpoints  []string
	excludeEndpoints  []string
	schemasMu         sync.RWMutex
}

// Config holds configuration options for the EchoMCP server.
type Config struct {
	// Name is the MCP server name. If empty and EnableSwaggerSchemas is true,
	// it will be automatically extracted from Swagger @title annotation.
	Name string

	// Version is the MCP server version. If empty and EnableSwaggerSchemas is true,
	// it will be automatically extracted from Swagger @version annotation.
	Version string

	// Description is the MCP server description. If empty and EnableSwaggerSchemas is true,
	// it will be automatically extracted from Swagger @description annotation.
	Description string

	// BaseURL is the base URL for HTTP requests made by MCP tools.
	// Required for tool execution. Example: "http://localhost:8080"
	BaseURL string

	// IncludeOperations lists specific endpoints to expose as MCP tools.
	// If specified, only these endpoints will be available as tools.
	// Supports exact paths like "/users/:id" and wildcard patterns like "/admin/*".
	// Takes precedence over ExcludeOperations.
	IncludeOperations []string

	// ExcludeOperations lists endpoints to exclude from MCP tools.
	// Supports exact paths like "/users/:id" and wildcard patterns like "/admin/*".
	// Ignored if IncludeOperations is specified.
	ExcludeOperations []string

	// IncludeTags lists Swagger tags to include when filtering endpoints.
	// Only endpoints with these tags will be exposed as MCP tools.
	// Takes precedence over ExcludeTags.
	IncludeTags []string

	// ExcludeTags lists Swagger tags to exclude when filtering endpoints.
	// Endpoints with these tags will not be exposed as MCP tools.
	// Ignored if IncludeTags is specified.
	ExcludeTags []string

	// EnableSwaggerSchemas enables automatic schema generation from Swagger/OpenAPI documentation.
	// When true, the server will use Swagger annotations to generate type-safe MCP schemas.
	// Also auto-populates Name, Description, and Version from Swagger info if not provided.
	EnableSwaggerSchemas bool

	// DescribeAllResponses determines whether to include all response schemas in tool descriptions.
	// When true, MCP tool descriptions will include information about all possible response types.
	DescribeAllResponses bool

	// DescribeFullResponseSchema determines whether to include complete response schemas.
	// When true, full response object structures are included in tool descriptions.
	DescribeFullResponseSchema bool
}

// NewWithConfig creates a new EchoMCP instance with the provided configuration.
// The config parameter can be nil, in which case default values are used.
//
// If EnableSwaggerSchemas is true and Name, Description, or Version are empty,
// they will be automatically populated from Swagger annotations if available.
//
// Example:
//
//	config := &server.Config{
//		BaseURL:              "http://localhost:8080",
//		EnableSwaggerSchemas: true,
//		ExcludeOperations:    []string{"/health", "/metrics"},
//	}
//	mcp := server.NewWithConfig(e, config)
func NewWithConfig(e *echo.Echo, config *Config) *EchoMCP {
	if config == nil {
		config = &Config{}
	}

	// Auto-populate name, description, and version from Swagger if available and not provided
	name := config.Name
	description := config.Description
	version := config.Version

	if config.EnableSwaggerSchemas && (name == "" || description == "" || version == "") {
		if spec, err := swagger.GetSwaggerSpec(); err == nil && spec.Info != nil {
			if name == "" && spec.Info.Title != "" {
				name = spec.Info.Title
			}
			if description == "" && spec.Info.Description != "" {
				description = spec.Info.Description
			}
			if version == "" && spec.Info.Version != "" {
				version = spec.Info.Version
			}
		}
	}

	echoMCP := &EchoMCP{
		echo:              e,
		name:              name,
		version:           version,
		description:       description,
		baseURL:           config.BaseURL,
		config:            config,
		registeredSchemas: make(map[string]types.RegisteredSchemaInfo),
		tools:             []types.Tool{},
		operations:        make(map[string]types.Operation),
	}

	// Set default execute function (in the future )
	echoMCP.executeToolFunc = echoMCP.defaultExecuteTool

	return echoMCP
}

// New creates a new EchoMCP instance with default configuration.
// EnableSwaggerSchemas is enabled by default. Name, Description, and Version
// are automatically populated from Swagger annotations if available.
//
// This is equivalent to calling NewWithConfig with EnableSwaggerSchemas: true.
//
// Example:
//
//	e := echo.New()
//	e.GET("/users/:id", getUserHandler)
//	mcp := server.New(e)
//	mcp.Mount("/mcp")
func New(e *echo.Echo) *EchoMCP {
	config := &Config{
		EnableSwaggerSchemas: true,
	}

	// Auto-populate name, description, and version from Swagger if available and not provided
	name := config.Name
	description := config.Description
	version := config.Version

	if config.EnableSwaggerSchemas && (name == "" || description == "" || version == "") {
		if spec, err := swagger.GetSwaggerSpec(); err == nil && spec.Info != nil {
			if name == "" && spec.Info.Title != "" {
				name = spec.Info.Title
			}
			if description == "" && spec.Info.Description != "" {
				description = spec.Info.Description
			}
			if version == "" && spec.Info.Version != "" {
				version = spec.Info.Version
			}
		}
	}

	echoMCP := &EchoMCP{
		echo:              e,
		name:              name,
		version:           version,
		description:       description,
		baseURL:           config.BaseURL,
		config:            config,
		registeredSchemas: make(map[string]types.RegisteredSchemaInfo),
		tools:             []types.Tool{},
		operations:        make(map[string]types.Operation),
	}

	// Set default execute function (in the future we should handle SSE)
	echoMCP.executeToolFunc = echoMCP.defaultExecuteTool

	return echoMCP
}

// RegisterSchema registers Go types for query parameters and request body for a specific route.
// This provides type-safe schema generation for routes that aren't covered by Swagger annotations.
//
// Parameters:
//   - method: HTTP method (e.g., "GET", "POST")
//   - path: Route path as defined in Echo (e.g., "/users/:id")
//   - querySchema: Go struct representing query parameters (can be nil)
//   - bodySchema: Go struct representing request body (can be nil)
//
// Example:
//
//	type UserQuery struct {
//		Page  int  `form:"page" jsonschema:"minimum=1"`
//		Limit int  `form:"limit" jsonschema:"maximum=100"`
//	}
//
//	type CreateUserRequest struct {
//		Name  string `json:"name" jsonschema:"required"`
//		Email string `json:"email" jsonschema:"required"`
//	}
//
//	mcp.RegisterSchema("GET", "/users", UserQuery{}, nil)
//	mcp.RegisterSchema("POST", "/users", nil, CreateUserRequest{})
func (e *EchoMCP) RegisterSchema(method, path string, querySchema, bodySchema any) {
	e.schemasMu.Lock()
	defer e.schemasMu.Unlock()

	key := fmt.Sprintf("%s %s", method, path)
	e.registeredSchemas[key] = types.RegisteredSchemaInfo{
		QuerySchema: querySchema,
		BodySchema:  bodySchema,
	}
}

// RegisterEndpoints sets the specific endpoints to include in MCP tools.
// Only endpoints matching these paths will be registered as MCP tools.
// If set, this takes precedence over ExcludeEndpoints.
//
// Supports exact paths and wildcard patterns:
//   - "/users/:id" - exact path match
//   - "/admin/*" - prefix match (all paths starting with /admin/)
//
// Example:
//
//	// Only expose user and order endpoints
//	mcp.RegisterEndpoints([]string{
//		"/api/v1/users/:id",
//		"/api/v1/users",
//		"/api/v1/orders/*",
//	})
func (e *EchoMCP) RegisterEndpoints(endpoints []string) {
	e.includeEndpoints = endpoints
}

// ExcludeEndpoints sets endpoints to exclude from MCP tools.
// Endpoints matching these paths will not be registered as MCP tools.
// This is ignored if RegisterEndpoints is set.
//
// Supports exact paths and wildcard patterns:
//   - "/health" - exact path match
//   - "/admin/*" - prefix match (all paths starting with /admin/)
//
// Example:
//
//	// Exclude internal and debug endpoints
//	mcp.ExcludeEndpoints([]string{
//		"/health",
//		"/metrics",
//		"/debug/*",
//		"/swagger/*",
//	})
func (e *EchoMCP) ExcludeEndpoints(endpoints []string) {
	e.excludeEndpoints = endpoints
}

// Mount mounts the MCP server at the specified path and registers it with the Echo instance.
// This creates the HTTP endpoint that MCP clients will connect to.
//
// The mounted endpoint accepts POST requests with MCP protocol messages and returns
// appropriate responses for initialize, tools/list, and tools/call requests.
//
// Example:
//
//	if err := mcp.Mount("/mcp"); err != nil {
//		log.Fatal("Failed to mount MCP server:", err)
//	}
//
// After mounting, the MCP server will be available at the specified path.
// MCP clients can connect to this endpoint to discover and execute tools.
func (e *EchoMCP) Mount(path string) error {
	// Create HTTP transport first
	e.transport = transport.NewHTTPTransport(path)

	if err := e.setupServer(); err != nil {
		return fmt.Errorf("failed to setup server: %w", err)
	}

	// Register handlers
	e.transport.RegisterHandler("initialize", e.handleInitialize)
	e.transport.RegisterHandler("tools/list", e.handleToolsList)
	e.transport.RegisterHandler("tools/call", e.handleToolCall)

	// Handle HTTP messages (Streamable HTTP transport)
	e.echo.POST(path, e.transport.HandleMessage)
	return nil
}

// setupServer initializes tools and operations from registered routes
func (e *EchoMCP) setupServer() error {
	e.schemasMu.RLock()
	registeredSchemas := make(map[string]types.RegisteredSchemaInfo)
	maps.Copy(registeredSchemas, e.registeredSchemas)
	e.schemasMu.RUnlock()

	// Get routes from Echo
	routes := e.echo.Routes()

	// Filter routes
	filteredRoutes := e.filterRoutes(routes)

	// Convert routes to tools
	tools, operations := convert.ConvertRoutesToTools(filteredRoutes, registeredSchemas, e.config.EnableSwaggerSchemas)

	e.tools = tools
	e.operations = operations

	return nil
}

// filterRoutes filters routes based on configuration
func (e *EchoMCP) filterRoutes(routes []*echo.Route) []*echo.Route {
	var filtered []*echo.Route

	for _, route := range routes {
		// Skip MCP endpoints (only if transport is initialized)
		if e.transport != nil && strings.HasPrefix(route.Path, e.transport.MountPath()) {
			continue
		}

		// Apply endpoint filtering
		if !e.shouldIncludeRoute(route) {
			continue
		}

		filtered = append(filtered, route)
	}

	return filtered
}

// shouldIncludeRoute determines if a route should be included based on include/exclude filters
func (e *EchoMCP) shouldIncludeRoute(route *echo.Route) bool {
	routePath := route.Path

	// If includeEndpoints is set, only include routes that match
	if len(e.includeEndpoints) > 0 {
		for _, included := range e.includeEndpoints {
			if e.matchesEndpoint(routePath, included) {
				return true
			}
		}
		return false
	}

	// If excludeEndpoints is set, exclude routes that match
	if len(e.excludeEndpoints) > 0 {
		for _, excluded := range e.excludeEndpoints {
			if e.matchesEndpoint(routePath, excluded) {
				return false
			}
		}
	}

	// Include by default if no specific filtering rules apply
	return true
}

// matchesEndpoint checks if a route path matches an endpoint pattern
func (e *EchoMCP) matchesEndpoint(routePath, pattern string) bool {
	// Exact match
	if routePath == pattern {
		return true
	}

	// Prefix match (for patterns ending with *)
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(routePath, prefix)
	}

	// Wildcard match for path parameters
	// Convert Echo path params (:param) to match pattern
	if strings.Contains(routePath, ":") {
		// Simple pattern matching - replace :param with actual values
		// This is a basic implementation, could be enhanced
		routePattern := strings.ReplaceAll(routePath, ":id", "*")
		routePattern = strings.ReplaceAll(routePattern, ":param", "*")

		if pattern == routePattern {
			return true
		}
	}

	return false
}

// handleInitialize handles MCP initialize requests
func (e *EchoMCP) handleInitialize(params any) (any, error) {
	version := e.version
	if version == "" {
		version = "1.0.0" // Fallback default
	}

	return InitializeResponse{
		ProtocolVersion: "2024-11-05",
		Capabilities: &Capabilities{
			Tools: map[string]any{},
		},
		ServerInfo: &ServerInfo{
			Name:    e.name,
			Version: version,
		},
	}, nil
}

// handleToolsList handles tools/list requests
func (e *EchoMCP) handleToolsList(params any) (any, error) {
	if err := e.setupServer(); err != nil {
		return nil, fmt.Errorf("failed to setup server: %w", err)
	}

	return ToolsListResponse{
		Tools: e.tools,
	}, nil
}

// handleToolCall handles tools/call requests
func (e *EchoMCP) handleToolCall(params any) (any, error) {
	paramMap, ok := params.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid parameters")
	}

	toolName, ok := paramMap["name"].(string)
	if !ok {
		return nil, fmt.Errorf("missing tool name")
	}

	arguments, ok := paramMap["arguments"].(map[string]any)
	if !ok {
		arguments = make(map[string]any)
	}

	result, err := e.executeToolFunc(toolName, arguments)
	if err != nil {
		return nil, err
	}

	return ToolCallResponse{
		Content: []Content{
			{
				Type: "text",
				Text: fmt.Sprintf("%v", result),
			},
		},
	}, nil
}

// defaultExecuteTool executes a tool by making an HTTP request to the corresponding endpoint
func (e *EchoMCP) defaultExecuteTool(operationID string, parameters map[string]any) (any, error) {
	operation, exists := e.operations[operationID]
	if !exists {
		return nil, fmt.Errorf("tool '%s' not found in operations map", operationID)
	}

	// Build the request URL
	requestURL := e.buildRequestURL(operation, parameters)

	// Create HTTP request with appropriate body format
	var body io.Reader
	var contentType string

	if isBodyMethod(operation.Method) {
		// Check if this operation uses form data
		if len(operation.FormDataParams) > 0 {
			// Handle form data
			formData := url.Values{}
			for key, value := range parameters {
				if isFormDataParameter(operation, key) {
					formData.Add(key, fmt.Sprintf("%v", value))
				}
			}

			if len(formData) > 0 {
				body = strings.NewReader(formData.Encode())
				contentType = "application/x-www-form-urlencoded"
			}
		} else {
			// Handle JSON body (exclude path, header, query, and form data parameters)
			bodyData := make(map[string]any)
			for key, value := range parameters {
				if !isPathParameter(operation.Path, key) &&
					!isHeaderParameter(operation, key) &&
					!isQueryParameter(operation, key) &&
					!isFormDataParameter(operation, key) {
					bodyData[key] = value
				}
			}

			if len(bodyData) > 0 {
				jsonBody, err := json.Marshal(bodyData)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal request body: %w", err)
				}
				body = bytes.NewReader(jsonBody)
				contentType = "application/json"
			}
		}
	}

	req, err := http.NewRequest(operation.Method, requestURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set appropriate Content-Type
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	// Add header parameters
	for key, value := range parameters {
		if isHeaderParameter(operation, key) {
			req.Header.Set(key, fmt.Sprintf("%v", value))
		}
	}

	// Execute request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			// Log the error but don't fail the operation
			fmt.Printf("Warning: failed to close response body: %v\n", closeErr)
		}
	}()

	// Read response
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Try to parse as JSON, fall back to string
	var result any
	if jsonErr := sonic.Unmarshal(responseBody, &result); jsonErr != nil {
		result = string(responseBody)
	}

	return result, nil
}

// buildRequestURL builds the complete request URL with path and query parameters
func (e *EchoMCP) buildRequestURL(operation types.Operation, parameters map[string]any) string {
	baseURL := e.baseURL
	if baseURL == "" {
		baseURL = "http://localhost:8080" // Default
	}

	// Replace path parameters
	finalPath := operation.Path
	for key, value := range parameters {
		placeholder := ":" + key
		if strings.Contains(finalPath, placeholder) {
			finalPath = strings.ReplaceAll(finalPath, placeholder, fmt.Sprintf("%v", value))
		}
	}

	// Build query parameters (only include explicit query parameters)
	queryParams := url.Values{}
	for key, value := range parameters {
		if isQueryParameter(operation, key) {
			queryParams.Add(key, fmt.Sprintf("%v", value))
		}
	}

	requestURL := baseURL + finalPath
	if len(queryParams) > 0 {
		requestURL += "?" + queryParams.Encode()
	}

	return requestURL
}

// Helper functions
func isBodyMethod(method string) bool {
	method = strings.ToUpper(method)
	return method == "POST" || method == "PUT" || method == "PATCH"
}

func isPathParameter(path, paramName string) bool {
	return strings.Contains(path, ":"+paramName)
}

func isHeaderParameter(operation types.Operation, paramName string) bool {
	return slices.Contains(operation.HeaderParams, paramName)
}

func isQueryParameter(operation types.Operation, paramName string) bool {
	return slices.Contains(operation.QueryParams, paramName)
}

func isFormDataParameter(operation types.Operation, paramName string) bool {
	return slices.Contains(operation.FormDataParams, paramName)
}

// GetServerInfo returns the server information (useful for testing)
func (e *EchoMCP) GetServerInfo() (name, version, description string) {
	return e.name, e.version, e.description
}
