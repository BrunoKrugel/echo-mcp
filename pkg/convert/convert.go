package convert

import (
	"fmt"
	"maps"
	"net/http"
	"regexp"
	"strings"

	"github.com/BrunoKrugel/echo-mcp/pkg/swagger"
	"github.com/BrunoKrugel/echo-mcp/pkg/types"
	"github.com/labstack/echo/v4"
)

// ConvertRoutesToTools converts Echo routes into a list of MCP Tools and an operation map.
func ConvertRoutesToTools(routes []*echo.Route, registeredSchemas map[string]types.RegisteredSchemaInfo, enableSwagger bool) ([]types.Tool, map[string]types.Operation) {
	tools := make([]types.Tool, 0)
	operations := make(map[string]types.Operation)

	// Get swagger spec if enabled
	var swaggerSpec *swagger.SwaggerSpec
	if enableSwagger {
		if spec, err := swagger.GetSwaggerSpec(); err == nil {
			swaggerSpec = spec
		}
	}

	for _, route := range routes {
		if route.Method == "" || route.Path == "" {
			continue
		}

		operationID := generateOperationID(route.Method, route.Path)

		tool := generateTool(route, operationID, registeredSchemas, swaggerSpec)
		tools = append(tools, tool)

		// Extract header and query parameters from swagger if available
		var headerParams []string
		var queryParams []string
		if swaggerSpec != nil {
			headerParams = extractHeaderParameters(route, swaggerSpec)
			queryParams = extractQueryParameters(route, swaggerSpec)
		}

		operations[operationID] = types.Operation{
			Method:       route.Method,
			Path:         route.Path,
			HeaderParams: headerParams,
			QueryParams:  queryParams,
		}
	}

	return tools, operations
}

// generateOperationID creates a unique operation ID for a route
func generateOperationID(method, path string) string {
	// Convert path parameters to a consistent format
	// /users/:id -> /users/{id}
	normalizedPath := strings.ReplaceAll(path, ":", "")
	normalizedPath = strings.ReplaceAll(normalizedPath, "/", "_")
	normalizedPath = strings.Trim(normalizedPath, "_")

	if normalizedPath == "" {
		normalizedPath = "root"
	}

	return fmt.Sprintf("%s_%s", method, normalizedPath)
}

// generateTool converts an Echo route to an MCP Tool
func generateTool(route *echo.Route, operationID string, registeredSchemas map[string]types.RegisteredSchemaInfo, swaggerSpec *swagger.SwaggerSpec) types.Tool {
	schemaKey := fmt.Sprintf("%s %s", route.Method, route.Path)
	registeredSchema, hasRegisteredSchema := registeredSchemas[schemaKey]

	inputSchema := generateInputSchema(route, registeredSchema, hasRegisteredSchema, swaggerSpec)

	description := fmt.Sprintf("Execute %s request to %s", route.Method, route.Path)

	// Try to get description from swagger first, then fallback to handler description
	if swaggerSpec != nil {
		if swaggerDesc := getSwaggerDescription(route, swaggerSpec); swaggerDesc != "" {
			description = swaggerDesc
		}
	} else if handlerDesc := getHandlerDescription(route); handlerDesc != "" {
		description = handlerDesc
	}

	return types.Tool{
		Name:        operationID,
		Description: description,
		InputSchema: inputSchema,
	}
}

// generateInputSchema creates the input schema for a tool based on the route
func generateInputSchema(route *echo.Route, registeredSchema types.RegisteredSchemaInfo, hasRegisteredSchema bool, swaggerSpec *swagger.SwaggerSpec) map[string]any {
	schema := map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}

	properties := schema["properties"].(map[string]any)
	var required []string

	// Extract path parameters
	pathParams := extractPathParameters(route.Path)
	for _, param := range pathParams {
		properties[param] = map[string]any{
			"type":        "string",
			"description": fmt.Sprintf("Path parameter: %s", param),
		}
		required = append(required, param)
	}

	// Try swagger schema first, then registered schema, then fallback
	swaggerUsed := false
	if swaggerSpec != nil {
		if swaggerSchema, err := swaggerSpec.GetOperationSchema(route.Method, route.Path); err == nil {
			// Use swagger schema
			if props, ok := swaggerSchema["properties"].(map[string]any); ok {
				maps.Copy(properties, props)
			}
			if reqFields, ok := swaggerSchema["required"].([]string); ok {
				required = append(required, reqFields...)
			}
			swaggerUsed = true
		}
	}

	// Fallback to registered schemas if swagger not used
	if !swaggerUsed {
		// Add query parameters from registered schema if available
		if hasRegisteredSchema && registeredSchema.QuerySchema != nil {
			querySchema := types.GetSchema(registeredSchema.QuerySchema)
			if queryProps, ok := querySchema["properties"].(map[string]any); ok {
				maps.Copy(properties, queryProps)
			}
			if queryRequired, ok := querySchema["required"].([]string); ok {
				required = append(required, queryRequired...)
			}
		}

		// Add request body schema for methods that typically have bodies
		if isBodyMethod(route.Method) {
			if hasRegisteredSchema && registeredSchema.BodySchema != nil {
				bodySchema := types.GetSchema(registeredSchema.BodySchema)
				if bodyProps, ok := bodySchema["properties"].(map[string]any); ok {
					maps.Copy(properties, bodyProps)
				}
				if bodyRequired, ok := bodySchema["required"].([]string); ok {
					required = append(required, bodyRequired...)
				}
			} else {
				// Generic body parameter
				properties["body"] = map[string]any{
					"type":        "object",
					"description": "Request body",
				}
			}
		}
	}

	if len(required) > 0 {
		schema["required"] = required
	}

	return schema
}

// isBodyMethod returns true if the HTTP method typically has a request body
func isBodyMethod(method string) bool {
	method = strings.ToUpper(method)
	return method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch
}

// extractPathParameters extracts parameter names from an Echo route path
func extractPathParameters(path string) []string {
	var params []string
	re := regexp.MustCompile(`:(\w+)`)
	matches := re.FindAllStringSubmatch(path, -1)

	for _, match := range matches {
		if len(match) > 1 {
			params = append(params, match[1])
		}
	}

	return params
}

// getHandlerDescription attempts to extract description from handler function comments
func getHandlerDescription(route *echo.Route) string {
	if route.Name != "" {
		return fmt.Sprintf("Handler: %s", route.Name)
	}
	return fmt.Sprintf("Execute %s %s", route.Method, route.Path)
}

// echoPathToSwaggerPath converts Echo path syntax (:id) to Swagger path syntax ({id})
func echoPathToSwaggerPath(echoPath string) string {
	re := regexp.MustCompile(`:(\w+)`)
	return re.ReplaceAllString(echoPath, "{$1}")
}

// getSwaggerDescription gets the description from swagger specification
func getSwaggerDescription(route *echo.Route, swaggerSpec *swagger.SwaggerSpec) string {
	if swaggerSpec == nil {
		return ""
	}

	swaggerPath := echoPathToSwaggerPath(route.Path)

	if pathSpec, exists := swaggerSpec.Paths[swaggerPath]; exists {
		method := strings.ToLower(route.Method)
		if operation, exists := pathSpec[method]; exists {
			if operation.Summary != "" {
				return operation.Summary
			}
			if operation.Description != "" {
				return operation.Description
			}
		}
	}

	return ""
}

// extractHeaderParameters extracts header parameter names from swagger specification
func extractHeaderParameters(route *echo.Route, swaggerSpec *swagger.SwaggerSpec) []string {
	var headerParams []string

	if swaggerSpec == nil {
		return headerParams
	}

	swaggerPath := echoPathToSwaggerPath(route.Path)

	if pathSpec, exists := swaggerSpec.Paths[swaggerPath]; exists {
		method := strings.ToLower(route.Method)
		if operation, exists := pathSpec[method]; exists {
			for _, param := range operation.Parameters {
				if param.In == "header" {
					headerParams = append(headerParams, param.Name)
				}
			}
		}
	}

	return headerParams
}

// extractQueryParameters extracts query parameter names from swagger specification
func extractQueryParameters(route *echo.Route, swaggerSpec *swagger.SwaggerSpec) []string {
	var queryParams []string

	if swaggerSpec == nil {
		return queryParams
	}

	swaggerPath := echoPathToSwaggerPath(route.Path)

	if pathSpec, exists := swaggerSpec.Paths[swaggerPath]; exists {
		method := strings.ToLower(route.Method)
		if operation, exists := pathSpec[method]; exists {
			for _, param := range operation.Parameters {
				if param.In == "query" {
					queryParams = append(queryParams, param.Name)
				}
			}
		}
	}

	return queryParams
}
