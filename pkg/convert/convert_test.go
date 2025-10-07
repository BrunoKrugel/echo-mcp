package convert

import (
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"

	"github.com/BrunoKrugel/echo-mcp/pkg/swagger"
	"github.com/BrunoKrugel/echo-mcp/pkg/types"
)

func TestConvertRoutesToTools(t *testing.T) {
	t.Run("Should convert simple routes to tools", func(t *testing.T) {
		routes := []*echo.Route{
			{Path: "/users", Method: "GET"},
			{Path: "/users/:id", Method: "GET"},
			{Path: "/users", Method: "POST"},
		}

		tools, operations := ConvertRoutesToTools(routes, nil, false)

		assert.Len(t, tools, 3)
		assert.Len(t, operations, 3)

		// Check tool names follow expected pattern
		expectedNames := []string{"GET_users", "GET_users_id", "POST_users"}
		actualNames := make([]string, len(tools))
		for i, tool := range tools {
			actualNames[i] = tool.Name
		}

		for _, expected := range expectedNames {
			assert.Contains(t, actualNames, expected)
		}
	})

	t.Run("Should handle empty routes", func(t *testing.T) {
		routes := []*echo.Route{}

		tools, operations := ConvertRoutesToTools(routes, nil, false)

		assert.Len(t, tools, 0)
		assert.Len(t, operations, 0)
	})

	t.Run("Should use registered schemas when available", func(t *testing.T) {
		routes := []*echo.Route{
			{Path: "/users", Method: "GET"},
		}

		type QuerySchema struct {
			Page int `json:"page" jsonschema:"minimum=1"`
		}

		registeredSchemas := map[string]types.RegisteredSchemaInfo{
			"GET /users": {
				QuerySchema: QuerySchema{},
				BodySchema:  nil,
			},
		}

		tools, operations := ConvertRoutesToTools(routes, registeredSchemas, false)

		assert.Len(t, tools, 1)
		assert.Len(t, operations, 1)

		tool := tools[0]
		assert.Equal(t, "GET_users", tool.Name)
		assert.NotNil(t, tool.InputSchema)

		// Verify that registered schema was used (properties should exist)
		schema, ok := tool.InputSchema.(map[string]any)
		assert.True(t, ok)

		properties, ok := schema["properties"].(map[string]any)
		assert.True(t, ok)

		// The schema should contain the page property from the registered QuerySchema
		assert.Contains(t, properties, "page")
	})

	t.Run("Should enable swagger schemas when requested", func(t *testing.T) {
		routes := []*echo.Route{
			{Path: "/test", Method: "GET"},
		}

		// This will attempt to use swagger but likely fail in test environment
		// The important thing is that it doesn't crash
		tools, operations := ConvertRoutesToTools(routes, nil, true)

		assert.Len(t, tools, 1)
		assert.Len(t, operations, 1)
	})
}

func TestGenerateTool(t *testing.T) {
	t.Run("Should generate tool with basic info", func(t *testing.T) {
		route := &echo.Route{
			Path:   "/users/:id",
			Method: "GET",
		}

		tool := generateTool(route, "GET_users_id", nil, nil)

		assert.Equal(t, "GET_users_id", tool.Name)
		assert.Contains(t, tool.Description, "GET")
		assert.Contains(t, tool.Description, "/users/:id")
		assert.NotNil(t, tool.InputSchema)
	})

	t.Run("Should use swagger description when available", func(t *testing.T) {
		route := &echo.Route{
			Path:   "/users",
			Method: "GET",
		}

		// Create mock swagger spec
		swaggerSpec := &swagger.SwaggerSpec{
			Paths: map[string]swagger.SwaggerPath{
				"/users": swagger.SwaggerPath{
					"get": swagger.SwaggerOperation{
						Summary:     "Get all users",
						Description: "Retrieves a list of all users",
					},
				},
			},
		}

		tool := generateTool(route, "GET_users", nil, swaggerSpec)

		assert.Equal(t, "Get all users", tool.Description)
	})
}

func TestGenerateInputSchema(t *testing.T) {
	t.Run("Should generate schema with path parameters", func(t *testing.T) {
		route := &echo.Route{
			Path:   "/users/:id/orders/:orderID",
			Method: "GET",
		}

		schema := generateInputSchema(route, types.RegisteredSchemaInfo{}, false, nil)

		assert.Equal(t, "object", schema["type"])

		properties, ok := schema["properties"].(map[string]any)
		assert.True(t, ok)
		assert.Contains(t, properties, "id")
		assert.Contains(t, properties, "orderID")

		required, ok := schema["required"].([]string)
		assert.True(t, ok)
		assert.Contains(t, required, "id")
		assert.Contains(t, required, "orderID")
	})

	t.Run("Should add body parameter for POST requests", func(t *testing.T) {
		route := &echo.Route{
			Path:   "/users",
			Method: "POST",
		}

		schema := generateInputSchema(route, types.RegisteredSchemaInfo{}, false, nil)

		properties, ok := schema["properties"].(map[string]any)
		assert.True(t, ok)
		assert.Contains(t, properties, "body")

		bodySchema := properties["body"].(map[string]any)
		assert.Equal(t, "object", bodySchema["type"])
	})

	t.Run("Should not add body parameter for GET requests", func(t *testing.T) {
		route := &echo.Route{
			Path:   "/users",
			Method: "GET",
		}

		schema := generateInputSchema(route, types.RegisteredSchemaInfo{}, false, nil)

		properties, ok := schema["properties"].(map[string]any)
		assert.True(t, ok)
		assert.NotContains(t, properties, "body")
	})

	t.Run("Should use registered body schema", func(t *testing.T) {
		route := &echo.Route{
			Path:   "/users",
			Method: "POST",
		}

		type BodySchema struct {
			Name  string `json:"name" jsonschema:"required"`
			Email string `json:"email" jsonschema:"required"`
		}

		registeredSchema := types.RegisteredSchemaInfo{
			BodySchema: BodySchema{},
		}

		schema := generateInputSchema(route, registeredSchema, true, nil)

		properties, ok := schema["properties"].(map[string]any)
		assert.True(t, ok)
		assert.Contains(t, properties, "name")
		assert.Contains(t, properties, "email")

		required, ok := schema["required"].([]string)
		assert.True(t, ok)
		assert.Contains(t, required, "name")
		assert.Contains(t, required, "email")
	})

	t.Run("Should use registered query schema", func(t *testing.T) {
		route := &echo.Route{
			Path:   "/users",
			Method: "GET",
		}

		type QuerySchema struct {
			Sort  string `form:"sort"`
			Page  int    `form:"page" jsonschema:"minimum=1"`
			Limit int    `form:"limit" jsonschema:"maximum=100"`
		}

		registeredSchema := types.RegisteredSchemaInfo{
			QuerySchema: QuerySchema{},
		}

		schema := generateInputSchema(route, registeredSchema, true, nil)

		properties, ok := schema["properties"].(map[string]any)
		assert.True(t, ok)
		assert.Contains(t, properties, "Page")
		assert.Contains(t, properties, "Limit")
		assert.Contains(t, properties, "Sort")

		pageSchema := properties["Page"].(map[string]any)
		assert.Equal(t, float64(1), pageSchema["minimum"])

		limitSchema := properties["Limit"].(map[string]any)
		assert.Equal(t, float64(100), limitSchema["maximum"])
	})
}

func TestIsBodyMethod(t *testing.T) {
	t.Run("Should return true for POST", func(t *testing.T) {
		assert.True(t, isBodyMethod("POST"))
		assert.True(t, isBodyMethod("post"))
	})

	t.Run("Should return true for PUT", func(t *testing.T) {
		assert.True(t, isBodyMethod("PUT"))
		assert.True(t, isBodyMethod("put"))
	})

	t.Run("Should return true for PATCH", func(t *testing.T) {
		assert.True(t, isBodyMethod("PATCH"))
		assert.True(t, isBodyMethod("patch"))
	})

	t.Run("Should return false for GET", func(t *testing.T) {
		assert.False(t, isBodyMethod("GET"))
		assert.False(t, isBodyMethod("get"))
	})

	t.Run("Should return false for DELETE", func(t *testing.T) {
		assert.False(t, isBodyMethod("DELETE"))
		assert.False(t, isBodyMethod("delete"))
	})

	t.Run("Should return false for HEAD", func(t *testing.T) {
		assert.False(t, isBodyMethod("HEAD"))
		assert.False(t, isBodyMethod("head"))
	})
}

func TestExtractPathParameters(t *testing.T) {
	t.Run("Should extract single parameter", func(t *testing.T) {
		params := extractPathParameters("/users/:id")

		assert.Len(t, params, 1)
		assert.Contains(t, params, "id")
	})

	t.Run("Should extract multiple parameters", func(t *testing.T) {
		params := extractPathParameters("/users/:userID/orders/:orderID")

		assert.Len(t, params, 2)
		assert.Contains(t, params, "userID")
		assert.Contains(t, params, "orderID")
	})

	t.Run("Should handle path without parameters", func(t *testing.T) {
		params := extractPathParameters("/users")

		assert.Len(t, params, 0)
	})

	t.Run("Should handle complex path patterns", func(t *testing.T) {
		params := extractPathParameters("/api/v1/users/:id/profile/:profileID/settings")

		assert.Len(t, params, 2)
		assert.Contains(t, params, "id")
		assert.Contains(t, params, "profileID")
	})

	t.Run("Should handle empty path", func(t *testing.T) {
		params := extractPathParameters("")

		assert.Len(t, params, 0)
	})
}

func TestEchoPathToSwaggerPath(t *testing.T) {
	t.Run("Should convert single parameter", func(t *testing.T) {
		swaggerPath := echoPathToSwaggerPath("/users/:id")

		assert.Equal(t, "/users/{id}", swaggerPath)
	})

	t.Run("Should convert multiple parameters", func(t *testing.T) {
		swaggerPath := echoPathToSwaggerPath("/users/:userID/orders/:orderID")

		assert.Equal(t, "/users/{userID}/orders/{orderID}", swaggerPath)
	})

	t.Run("Should handle path without parameters", func(t *testing.T) {
		swaggerPath := echoPathToSwaggerPath("/users")

		assert.Equal(t, "/users", swaggerPath)
	})

	t.Run("Should handle empty path", func(t *testing.T) {
		swaggerPath := echoPathToSwaggerPath("")

		assert.Equal(t, "", swaggerPath)
	})
}

func TestGetSwaggerDescription(t *testing.T) {
	t.Run("Should return empty string when swagger is nil", func(t *testing.T) {
		route := &echo.Route{Path: "/test", Method: "GET"}

		description := getSwaggerDescription(route, nil)

		assert.Equal(t, "", description)
	})

	t.Run("Should return summary from swagger spec", func(t *testing.T) {
		route := &echo.Route{Path: "/users", Method: "GET"}

		swaggerSpec := &swagger.SwaggerSpec{
			Paths: map[string]swagger.SwaggerPath{
				"/users": swagger.SwaggerPath{
					"get": swagger.SwaggerOperation{
						Summary: "Get all users",
					},
				},
			},
		}

		description := getSwaggerDescription(route, swaggerSpec)

		assert.Equal(t, "Get all users", description)
	})

	t.Run("Should return description when summary is empty", func(t *testing.T) {
		route := &echo.Route{Path: "/users", Method: "GET"}

		swaggerSpec := &swagger.SwaggerSpec{
			Paths: map[string]swagger.SwaggerPath{
				"/users": swagger.SwaggerPath{
					"get": swagger.SwaggerOperation{
						Description: "Retrieve all users from database",
					},
				},
			},
		}

		description := getSwaggerDescription(route, swaggerSpec)

		assert.Equal(t, "Retrieve all users from database", description)
	})

	t.Run("Should prioritize summary over description", func(t *testing.T) {
		route := &echo.Route{Path: "/users", Method: "GET"}

		swaggerSpec := &swagger.SwaggerSpec{
			Paths: map[string]swagger.SwaggerPath{
				"/users": swagger.SwaggerPath{
					"get": swagger.SwaggerOperation{
						Summary:     "Get all users",
						Description: "Retrieve all users from database",
					},
				},
			},
		}

		description := getSwaggerDescription(route, swaggerSpec)

		assert.Equal(t, "Get all users", description)
	})

	t.Run("Should return empty for non-existent path", func(t *testing.T) {
		route := &echo.Route{Path: "/nonexistent", Method: "GET"}

		swaggerSpec := &swagger.SwaggerSpec{
			Paths: map[string]swagger.SwaggerPath{
				"/users": swagger.SwaggerPath{
					"get": swagger.SwaggerOperation{
						Summary: "Get all users",
					},
				},
			},
		}

		description := getSwaggerDescription(route, swaggerSpec)

		assert.Equal(t, "", description)
	})
}

func TestExtractHeaderParameters(t *testing.T) {
	t.Run("Should extract header parameters from swagger spec", func(t *testing.T) {
		route := &echo.Route{Path: "/users", Method: "GET"}

		swaggerSpec := &swagger.SwaggerSpec{
			Paths: map[string]swagger.SwaggerPath{
				"/users": swagger.SwaggerPath{
					"get": swagger.SwaggerOperation{
						Parameters: []swagger.SwaggerParameter{
							{Name: "Authorization", In: "header"},
							{Name: "Content-Type", In: "header"},
							{Name: "id", In: "path"},
						},
					},
				},
			},
		}

		headers := extractHeaderParameters(route, swaggerSpec)

		assert.Len(t, headers, 2)
		assert.Contains(t, headers, "Authorization")
		assert.Contains(t, headers, "Content-Type")
		assert.NotContains(t, headers, "id")
	})

	t.Run("Should return empty slice when no headers", func(t *testing.T) {
		route := &echo.Route{Path: "/users", Method: "GET"}

		headers := extractHeaderParameters(route, nil)

		assert.Len(t, headers, 0)
	})
}

func TestExtractQueryParameters(t *testing.T) {
	t.Run("Should extract query parameters from swagger spec", func(t *testing.T) {
		route := &echo.Route{Path: "/users", Method: "GET"}

		swaggerSpec := &swagger.SwaggerSpec{
			Paths: map[string]swagger.SwaggerPath{
				"/users": swagger.SwaggerPath{
					"get": swagger.SwaggerOperation{
						Parameters: []swagger.SwaggerParameter{
							{Name: "page", In: "query"},
							{Name: "limit", In: "query"},
							{Name: "Authorization", In: "header"},
						},
					},
				},
			},
		}

		queryParams := extractQueryParameters(route, swaggerSpec)

		assert.Len(t, queryParams, 2)
		assert.Contains(t, queryParams, "page")
		assert.Contains(t, queryParams, "limit")
		assert.NotContains(t, queryParams, "Authorization")
	})

	t.Run("Should return empty slice when no query params", func(t *testing.T) {
		route := &echo.Route{Path: "/users", Method: "GET"}

		queryParams := extractQueryParameters(route, nil)

		assert.Len(t, queryParams, 0)
	})
}

func TestExtractFormDataParameters(t *testing.T) {
	t.Run("Should extract form data parameters from swagger spec", func(t *testing.T) {
		route := &echo.Route{Path: "/upload", Method: "POST"}

		swaggerSpec := &swagger.SwaggerSpec{
			Paths: map[string]swagger.SwaggerPath{
				"/upload": swagger.SwaggerPath{
					"post": swagger.SwaggerOperation{
						Parameters: []swagger.SwaggerParameter{
							{Name: "file", In: "formData"},
							{Name: "description", In: "formData"},
							{Name: "Authorization", In: "header"},
						},
					},
				},
			},
		}

		formDataParams := extractFormDataParameters(route, swaggerSpec)

		assert.Len(t, formDataParams, 2)
		assert.Contains(t, formDataParams, "file")
		assert.Contains(t, formDataParams, "description")
		assert.NotContains(t, formDataParams, "Authorization")
	})

	t.Run("Should return empty slice when no form data params", func(t *testing.T) {
		route := &echo.Route{Path: "/users", Method: "GET"}

		formDataParams := extractFormDataParameters(route, nil)

		assert.Len(t, formDataParams, 0)
	})
}
