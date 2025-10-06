// Package swagger provides utilities for parsing Swagger/OpenAPI documentation
// and converting it to MCP-compatible schemas. It handles $ref resolution,
// schema conversion, and extraction of operation metadata from Swagger specs.
package swagger

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/swaggo/swag"
)

type SwaggerSpec struct {
	Paths       map[string]SwaggerPath    `json:"paths"`
	Definitions map[string]*SwaggerSchema `json:"definitions"`
	Info        *SwaggerInfo              `json:"info"`
	Swagger     string                    `json:"swagger"`
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
	Ref                  string                    `json:"$ref,omitempty"`
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
	if err := sonic.Unmarshal([]byte(swaggerJSON), &spec); err != nil {
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
		if param.In == "path" || param.In == "query" || param.In == "header" || param.In == "formData" {
			propSchema := map[string]any{
				"type": param.Type,
			}

			if param.Description != "" {
				propSchema["description"] = param.Description
			} else if param.In == "header" {
				propSchema["description"] = fmt.Sprintf("Header parameter: %s", param.Name)
			} else if param.In == "formData" {
				propSchema["description"] = fmt.Sprintf("Form data parameter: %s", param.Name)
			}

			properties[param.Name] = propSchema

			if param.Required {
				required = append(required, param.Name)
			}
		} else if param.In == "body" && param.Schema != nil {
			// Skip body parameters for GET requests (they're likely response schemas mistakenly marked as body)
			if strings.ToUpper(method) == "GET" {
				continue
			}

			// Handle request body as a nested object under "body" property
			bodySchema := spec.convertSwaggerSchemaToMCP(param.Schema)
			properties["body"] = bodySchema

			if param.Required {
				required = append(required, "body")
			}
		}
	}

	if len(required) > 0 {
		schema["required"] = required
	}

	return schema, nil
}

// convertSwaggerSchemaToMCP converts swagger schema to MCP-compatible schema
func (spec *SwaggerSpec) convertSwaggerSchemaToMCP(schema *SwaggerSchema) any {
	if schema == nil {
		return map[string]any{"type": "object"}
	}

	// Handle $ref resolution
	if schema.Ref != "" {
		// Extract definition name from $ref (e.g., "#/definitions/main.User" -> "main.User")
		refParts := strings.Split(schema.Ref, "/")
		if len(refParts) >= 3 && refParts[0] == "#" && refParts[1] == "definitions" {
			defName := refParts[2]
			if refSchema, exists := spec.Definitions[defName]; exists {
				// Recursively convert the referenced schema
				return spec.convertSwaggerSchemaToMCP(refSchema)
			}
		}
		// If $ref cannot be resolved, return a basic object
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
			properties[key] = spec.convertSwaggerSchemaToMCP(prop)
		}
		result["properties"] = properties
	}

	if schema.AdditionalProperties != nil {
		result["additionalProperties"] = spec.convertSwaggerSchemaToMCP(schema.AdditionalProperties)
	}

	if len(schema.Required) > 0 {
		result["required"] = schema.Required
	}

	return result
}
