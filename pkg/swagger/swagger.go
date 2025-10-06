package swagger

import (
	"encoding/json"
	"fmt"
	"maps"
	"regexp"
	"strings"

	"github.com/swaggo/swag"
)

type SwaggerSpec struct {
	Paths   map[string]SwaggerPath `json:"paths"`
	Info    *SwaggerInfo           `json:"info"`
	Swagger string                 `json:"swagger"`
}

type SwaggerInfo struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Version     string `json:"version"`
}

type SwaggerPath map[string]SwaggerOperation

type SwaggerOperation struct {
	Responses   map[string]SwaggerResponse `json:"responses"`
	Summary     string                     `json:"summary"`
	Description string                     `json:"description"`
	Tags        []string                   `json:"tags"`
	Parameters  []SwaggerParameter         `json:"parameters"`
}

type SwaggerParameter struct {
	Schema      *SwaggerSchema `json:"schema,omitempty"`
	Name        string         `json:"name"`
	In          string         `json:"in"`
	Type        string         `json:"type"`
	Description string         `json:"description"`
	Required    bool           `json:"required"`
}

type SwaggerResponse struct {
	Schema      *SwaggerSchema `json:"schema,omitempty"`
	Description string         `json:"description"`
}

type SwaggerSchema struct {
	Properties           map[string]*SwaggerSchema `json:"properties,omitempty"`
	AdditionalProperties *SwaggerSchema            `json:"additionalProperties,omitempty"`
	Minimum              *float64                  `json:"minimum,omitempty"`
	Maximum              *float64                  `json:"maximum,omitempty"`
	Type                 string                    `json:"type,omitempty"`
	Description          string                    `json:"description,omitempty"`
	Format               string                    `json:"format,omitempty"`
	Required             []string                  `json:"required,omitempty"`
}

// GetSwaggerSpec retrieves the swagger specification from swaggo
func GetSwaggerSpec() (*SwaggerSpec, error) {

	info := swag.GetSwagger("swagger")
	if info == nil {
		return nil, fmt.Errorf("swagger documentation not found - make sure to import docs package and generate swagger")
	}

	swaggerJSON := info.ReadDoc()
	if swaggerJSON == "" {
		return nil, fmt.Errorf("swagger documentation is empty")
	}

	var spec SwaggerSpec
	if err := json.Unmarshal([]byte(swaggerJSON), &spec); err != nil {
		return nil, fmt.Errorf("failed to parse swagger JSON: %w", err)
	}

	return &spec, nil
}

// echoPathToSwaggerPath converts Echo path syntax (:id) to Swagger path syntax ({id})
func echoPathToSwaggerPath(echoPath string) string {
	re := regexp.MustCompile(`:(\w+)`)
	return re.ReplaceAllString(echoPath, "{$1}")
}

// GetOperationSchema returns the MCP schema for a specific operation
func (spec *SwaggerSpec) GetOperationSchema(method, path string) (map[string]any, error) {
	// Normalize method
	method = strings.ToLower(method)

	// Convert Echo path to Swagger path format
	swaggerPath := echoPathToSwaggerPath(path)

	pathSpec, exists := spec.Paths[swaggerPath]
	if !exists {
		return nil, fmt.Errorf("path %s not found in swagger spec", swaggerPath)
	}

	operation, exists := pathSpec[method]
	if !exists {
		return nil, fmt.Errorf("method %s not found for path %s in swagger spec", method, swaggerPath)
	}

	// Build MCP schema from swagger operation
	schema := map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}

	properties := schema["properties"].(map[string]any)
	var required []string

	// Process parameters
	for _, param := range operation.Parameters {
		if param.In == "path" || param.In == "query" || param.In == "header" {
			propSchema := map[string]any{
				"type": param.Type,
			}

			if param.Description != "" {
				propSchema["description"] = param.Description
			} else if param.In == "header" {
				propSchema["description"] = fmt.Sprintf("Header parameter: %s", param.Name)
			}

			properties[param.Name] = propSchema

			if param.Required {
				required = append(required, param.Name)
			}
		} else if param.In == "body" && param.Schema != nil {
			// Handle request body
			bodyProps := convertSwaggerSchemaToMCP(param.Schema)
			if bodySchema, ok := bodyProps.(map[string]any); ok {
				if props, ok := bodySchema["properties"].(map[string]any); ok {
					maps.Copy(properties, props)
				}
				if reqFields, ok := bodySchema["required"].([]string); ok {
					required = append(required, reqFields...)
				}
			}
		}
	}

	if len(required) > 0 {
		schema["required"] = required
	}

	return schema, nil
}

// convertSwaggerSchemaToMCP converts swagger schema to MCP-compatible schema
func convertSwaggerSchemaToMCP(schema *SwaggerSchema) any {
	if schema == nil {
		return map[string]any{"type": "object"}
	}

	result := map[string]any{}

	if schema.Type != "" {
		result["type"] = schema.Type
	}

	if schema.Description != "" {
		result["description"] = schema.Description
	}

	if schema.Format != "" {
		result["format"] = schema.Format
	}

	if schema.Minimum != nil {
		result["minimum"] = *schema.Minimum
	}

	if schema.Maximum != nil {
		result["maximum"] = *schema.Maximum
	}

	if schema.Properties != nil {
		properties := map[string]any{}
		for key, prop := range schema.Properties {
			properties[key] = convertSwaggerSchemaToMCP(prop)
		}
		result["properties"] = properties
	}

	if schema.AdditionalProperties != nil {
		result["additionalProperties"] = convertSwaggerSchemaToMCP(schema.AdditionalProperties)
	}

	if len(schema.Required) > 0 {
		result["required"] = schema.Required
	}

	return result
}
