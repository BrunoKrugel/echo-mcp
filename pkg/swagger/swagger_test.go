package swagger

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetOperationSchemaBodyParameters(t *testing.T) {
	t.Run("Should handle body parameters with $ref resolution", func(t *testing.T) {
		spec := &SwaggerSpec{
			Definitions: map[string]*SwaggerSchema{
				"main.User": {
					Type: "object",
					Properties: map[string]*SwaggerSchema{
						"id":   {Type: "string"},
						"name": {Type: "string"},
					},
					Required: []string{"id"},
				},
			},
			Paths: map[string]SwaggerPath{
				"/users": {
					"post": SwaggerOperation{
						Parameters: []SwaggerParameter{
							{
								Name:     "Request",
								In:       "body",
								Required: true,
								Schema: &SwaggerSchema{
									Ref: "#/definitions/main.User",
								},
							},
						},
					},
				},
			},
		}

		schema, err := spec.GetOperationSchema("POST", "/users")
		assert.NoError(t, err)

		properties, ok := schema["properties"].(map[string]any)
		assert.True(t, ok)

		// Should have body parameter
		bodyProp, exists := properties["body"]
		assert.True(t, exists)

		bodySchema, ok := bodyProp.(map[string]any)
		assert.True(t, ok)
		assert.Equal(t, "object", bodySchema["type"])

		bodyProps, ok := bodySchema["properties"].(map[string]any)
		assert.True(t, ok)
		assert.Contains(t, bodyProps, "id")
		assert.Contains(t, bodyProps, "name")
	})
}

func TestGetOperationSchemaSkipsBodyForGET(t *testing.T) {
	t.Run("Should skip body parameters for GET requests", func(t *testing.T) {
		spec := &SwaggerSpec{
			Definitions: map[string]*SwaggerSchema{
				"main.PingPong": {
					Type: "object",
					Properties: map[string]*SwaggerSchema{
						"message": {Type: "string"},
					},
				},
			},
			Paths: map[string]SwaggerPath{
				"/ping": {
					"get": SwaggerOperation{
						Parameters: []SwaggerParameter{
							{
								Name:     "Request",
								In:       "body",
								Required: true,
								Schema: &SwaggerSchema{
									Ref: "#/definitions/main.PingPong",
								},
							},
						},
					},
				},
			},
		}

		schema, err := spec.GetOperationSchema("GET", "/ping")
		assert.NoError(t, err)

		properties, ok := schema["properties"].(map[string]any)
		assert.True(t, ok)

		// Should not have body parameter for GET
		_, bodyExists := properties["body"]
		assert.False(t, bodyExists)

		// Should have empty properties
		assert.Len(t, properties, 0)
	})
}

func TestConvertSwaggerSchemaToMCP(t *testing.T) {
	spec := &SwaggerSpec{
		Definitions: map[string]*SwaggerSchema{
			"User": {
				Type: "object",
				Properties: map[string]*SwaggerSchema{
					"name": {Type: "string"},
					"age":  {Type: "integer", Minimum: &[]float64{0}[0]},
				},
				Required: []string{"name"},
			},
		},
	}

	t.Run("Should convert simple schema", func(t *testing.T) {
		schema := &SwaggerSchema{
			Type:        "object",
			Description: "A user object",
			Properties: map[string]*SwaggerSchema{
				"name": {Type: "string"},
				"age":  {Type: "integer"},
			},
			Required: []string{"name"},
		}

		result := spec.convertSwaggerSchemaToMCP(schema)

		resultMap, ok := result.(map[string]any)
		assert.True(t, ok)
		assert.Equal(t, "object", resultMap["type"])
		assert.Equal(t, "A user object", resultMap["description"])
		assert.Contains(t, resultMap, "properties")
		assert.Contains(t, resultMap, "required")
	})

	t.Run("Should resolve $ref references", func(t *testing.T) {
		schema := &SwaggerSchema{
			Ref: "#/definitions/User",
		}

		result := spec.convertSwaggerSchemaToMCP(schema)

		resultMap, ok := result.(map[string]any)
		assert.True(t, ok)
		assert.Equal(t, "object", resultMap["type"])

		properties, ok := resultMap["properties"].(map[string]any)
		assert.True(t, ok)
		assert.Contains(t, properties, "name")
		assert.Contains(t, properties, "age")

		ageSchema := properties["age"].(map[string]any)
		assert.Equal(t, float64(0), ageSchema["minimum"])
	})

	t.Run("Should handle invalid $ref", func(t *testing.T) {
		schema := &SwaggerSchema{
			Ref: "#/definitions/NonExistent",
		}

		result := spec.convertSwaggerSchemaToMCP(schema)

		resultMap, ok := result.(map[string]any)
		assert.True(t, ok)
		assert.Equal(t, "object", resultMap["type"])
		// Should return basic object when $ref can't be resolved
	})

	t.Run("Should handle nil schema", func(t *testing.T) {
		result := spec.convertSwaggerSchemaToMCP(nil)

		resultMap, ok := result.(map[string]any)
		assert.True(t, ok)
		assert.Equal(t, "object", resultMap["type"])
	})

	t.Run("Should handle nested properties", func(t *testing.T) {
		schema := &SwaggerSchema{
			Type: "object",
			Properties: map[string]*SwaggerSchema{
				"user": {
					Type: "object",
					Properties: map[string]*SwaggerSchema{
						"name": {Type: "string"},
					},
				},
			},
		}

		result := spec.convertSwaggerSchemaToMCP(schema)

		resultMap, ok := result.(map[string]any)
		assert.True(t, ok)

		properties, ok := resultMap["properties"].(map[string]any)
		assert.True(t, ok)

		userSchema := properties["user"].(map[string]any)
		userProps := userSchema["properties"].(map[string]any)
		assert.Contains(t, userProps, "name")
	})

	t.Run("Should handle additional properties", func(t *testing.T) {
		schema := &SwaggerSchema{
			Type: "object",
			AdditionalProperties: &SwaggerSchema{
				Type: "string",
			},
		}

		result := spec.convertSwaggerSchemaToMCP(schema)

		resultMap, ok := result.(map[string]any)
		assert.True(t, ok)
		assert.Contains(t, resultMap, "additionalProperties")
	})
}

func TestEchoPathToSwaggerPath(t *testing.T) {
	t.Run("Should convert Echo path parameters to Swagger format", func(t *testing.T) {
		testCases := []struct {
			input    string
			expected string
		}{
			{"/users/:id", "/users/{id}"},
			{"/users/:userID/orders/:orderID", "/users/{userID}/orders/{orderID}"},
			{"/simple", "/simple"},
			{"", ""},
			{"/users/:id/profile", "/users/{id}/profile"},
		}

		for _, tc := range testCases {
			result := echoPathToSwaggerPath(tc.input)
			assert.Equal(t, tc.expected, result)
		}
	})
}

func TestGetSwaggerSpec(t *testing.T) {
	t.Run("Should handle missing swagger documentation", func(t *testing.T) {
		// This test will likely fail in test environment since swagger isn't initialized
		// But it should return an appropriate error, not panic
		_, err := GetSwaggerSpec()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "swagger documentation not found")
	})
}
