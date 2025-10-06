package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/BrunoKrugel/echo-mcp/pkg/convert"
	"github.com/BrunoKrugel/echo-mcp/pkg/transport"
	"github.com/BrunoKrugel/echo-mcp/pkg/types"
)

// EchoMCP represents the MCP server configuration for an Echo application
type EchoMCP struct {
	transport         transport.Transport
	echo              *echo.Echo
	operations        map[string]types.Operation
	config            *Config
	registeredSchemas map[string]types.RegisteredSchemaInfo
	executeToolFunc   func(operationID string, parameters map[string]any) (any, error)
	name              string
	description       string
	baseURL           string
	tools             []types.Tool
	includeEndpoints  []string
	excludeEndpoints  []string
	schemasMu         sync.RWMutex
}

// Config represents the configuration options for EchoMCP
type Config struct {
	Name                       string
	Description                string
	BaseURL                    string
	IncludeOperations          []string
	ExcludeOperations          []string
	IncludeTags                []string
	ExcludeTags                []string
	EnableSwaggerSchemas       bool
	DescribeAllResponses       bool
	DescribeFullResponseSchema bool
}

// New creates a new EchoMCP instance
func New(e *echo.Echo, config *Config) *EchoMCP {
	if config == nil {
		config = &Config{}
	}

	echoMCP := &EchoMCP{
		echo:              e,
		name:              config.Name,
		description:       config.Description,
		baseURL:           config.BaseURL,
		config:            config,
		registeredSchemas: make(map[string]types.RegisteredSchemaInfo),
		tools:             []types.Tool{},
		operations:        make(map[string]types.Operation),
	}

	// Set default execute function
	echoMCP.executeToolFunc = echoMCP.defaultExecuteTool

	return echoMCP
}

// RegisterSchema registers Go types for query parameters and request body for a specific route
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
func (e *EchoMCP) RegisterEndpoints(endpoints []string) {
	e.includeEndpoints = endpoints
}

// ExcludeEndpoints sets endpoints to exclude from MCP tools.
// Endpoints matching these paths will not be registered as MCP tools.
// This is ignored if RegisterEndpoints is set.
func (e *EchoMCP) ExcludeEndpoints(endpoints []string) {
	e.excludeEndpoints = endpoints
}

// Mount mounts the MCP server at the specified path
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

	// Filter routes if needed (implement filtering logic similar to gin-mcp)
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
	return InitializeResponse{
		ProtocolVersion: "2024-11-05",
		Capabilities: &Capabilities{
			Tools: map[string]any{},
		},
		ServerInfo: &ServerInfo{
			Name:    e.name,
			Version: "1.0.0",
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
	requestURL := e.buildRequestURL(operation.Path, parameters)

	// Create HTTP request
	var body io.Reader
	if isBodyMethod(operation.Method) {
		// Extract body parameters
		bodyData := make(map[string]any)
		for key, value := range parameters {
			if !isPathParameter(operation.Path, key) {
				bodyData[key] = value
			}
		}

		if len(bodyData) > 0 {
			jsonBody, err := json.Marshal(bodyData)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal request body: %w", err)
			}
			body = bytes.NewReader(jsonBody)
		}
	}

	req, err := http.NewRequest(operation.Method, requestURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
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
	if err := json.Unmarshal(responseBody, &result); err != nil {
		result = string(responseBody)
	}

	return result, nil
}

// buildRequestURL builds the complete request URL with path and query parameters
func (e *EchoMCP) buildRequestURL(path string, parameters map[string]any) string {
	baseURL := e.baseURL
	if baseURL == "" {
		baseURL = "http://localhost:8080" // Default
	}

	// Replace path parameters
	finalPath := path
	for key, value := range parameters {
		placeholder := ":" + key
		if strings.Contains(finalPath, placeholder) {
			finalPath = strings.ReplaceAll(finalPath, placeholder, fmt.Sprintf("%v", value))
		}
	}

	// Build query parameters
	queryParams := url.Values{}
	for key, value := range parameters {
		if !isPathParameter(path, key) && !isBodyMethod(e.getMethodForPath(path)) {
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

func (e *EchoMCP) getMethodForPath(path string) string {
	for _, operation := range e.operations {
		if operation.Path == path {
			return operation.Method
		}
	}
	return "GET" // Default
}

// BaseURLResolver interface for dynamic base URL resolution
type BaseURLResolver interface {
	ResolveBaseURL() (string, error)
}

// QuicknodeResolver resolves base URLs for Quicknode environments
type QuicknodeResolver struct {
	fallbackURL string
}
