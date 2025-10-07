package types

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetSchema(t *testing.T) {
	t.Run("Should generate schema for simple struct", func(t *testing.T) {
		type TestStruct struct {
			Name string `json:"name"`
			Age  int    `json:"age"`
		}

		schema := GetSchema(TestStruct{})

		assert.Equal(t, "object", schema["type"])

		properties, ok := schema["properties"].(map[string]any)
		assert.True(t, ok)
		assert.Contains(t, properties, "name")
		assert.Contains(t, properties, "age")

		nameSchema := properties["name"].(map[string]any)
		assert.Equal(t, "string", nameSchema["type"])

		ageSchema := properties["age"].(map[string]any)
		assert.Equal(t, "integer", ageSchema["type"])
	})

	t.Run("Should handle required fields with jsonschema tag", func(t *testing.T) {
		type TestStruct struct {
			Name  string `json:"name" jsonschema:"required"`
			Email string `json:"email" jsonschema:"required,description=User email"`
			Age   int    `json:"age,omitempty"`
		}

		schema := GetSchema(TestStruct{})

		required, ok := schema["required"].([]string)
		assert.True(t, ok)
		assert.Contains(t, required, "name")
		assert.Contains(t, required, "email")
		assert.NotContains(t, required, "age")
	})

	t.Run("Should handle form tags", func(t *testing.T) {
		type TestStruct struct {
			Page  int `form:"page,required"`
			Limit int `form:"limit"`
		}

		schema := GetSchema(TestStruct{})

		required, ok := schema["required"].([]string)
		assert.True(t, ok)
		assert.Contains(t, required, "Page")
		assert.NotContains(t, required, "Limit")
	})

	t.Run("Should handle nested structs", func(t *testing.T) {
		type Address struct {
			Street string `json:"street"`
			City   string `json:"city"`
		}

		type User struct {
			Name    string  `json:"name"`
			Address Address `json:"address"`
		}

		schema := GetSchema(User{})

		properties, ok := schema["properties"].(map[string]any)
		assert.True(t, ok)

		addressSchema, ok := properties["address"].(map[string]any)
		assert.True(t, ok)
		assert.Equal(t, "object", addressSchema["type"])

		addressProps, ok := addressSchema["properties"].(map[string]any)
		assert.True(t, ok)
		assert.Contains(t, addressProps, "street")
		assert.Contains(t, addressProps, "city")
	})

	t.Run("Should handle pointer to struct", func(t *testing.T) {
		type TestStruct struct {
			Name string `json:"name"`
		}

		schema := GetSchema(&TestStruct{})

		assert.Equal(t, "object", schema["type"])
		properties, ok := schema["properties"].(map[string]any)
		assert.True(t, ok)
		assert.Contains(t, properties, "name")
	})

	t.Run("Should handle nil pointer", func(t *testing.T) {
		var nilStruct *struct{}
		schema := GetSchema(nilStruct)

		assert.Equal(t, "object", schema["type"])
		properties, ok := schema["properties"].(map[string]any)
		assert.True(t, ok)
		assert.Len(t, properties, 0)
	})

	t.Run("Should handle nil input", func(t *testing.T) {
		schema := GetSchema(nil)

		assert.Equal(t, "object", schema["type"])
		properties, ok := schema["properties"].(map[string]any)
		assert.True(t, ok)
		assert.Len(t, properties, 0)
	})

	t.Run("Should handle non-struct types", func(t *testing.T) {
		schema := GetSchema("string")

		assert.Equal(t, "object", schema["type"])
		properties, ok := schema["properties"].(map[string]any)
		assert.True(t, ok)
		assert.Len(t, properties, 0)
	})

	t.Run("Should handle arrays and slices", func(t *testing.T) {
		type TestStruct struct {
			Names []string `json:"names"`
			Ages  [3]int   `json:"ages"`
		}

		schema := GetSchema(TestStruct{})

		properties, ok := schema["properties"].(map[string]any)
		assert.True(t, ok)

		namesSchema := properties["names"].(map[string]any)
		assert.Equal(t, "array", namesSchema["type"])
		assert.Equal(t, "string", namesSchema["items"].(map[string]any)["type"])

		agesSchema := properties["ages"].(map[string]any)
		assert.Equal(t, "array", agesSchema["type"])
		assert.Equal(t, "integer", agesSchema["items"].(map[string]any)["type"])
	})

	t.Run("Should handle maps", func(t *testing.T) {
		type TestStruct struct {
			Metadata map[string]string `json:"metadata"`
			Counts   map[string]int    `json:"counts"`
		}

		schema := GetSchema(TestStruct{})

		properties, ok := schema["properties"].(map[string]any)
		assert.True(t, ok)

		metadataSchema := properties["metadata"].(map[string]any)
		assert.Equal(t, "object", metadataSchema["type"])
		assert.Equal(t, "string", metadataSchema["additionalProperties"].(map[string]any)["type"])

		countsSchema := properties["counts"].(map[string]any)
		assert.Equal(t, "object", countsSchema["type"])
		assert.Equal(t, "integer", countsSchema["additionalProperties"].(map[string]any)["type"])
	})
}

func TestApplySchemaTag(t *testing.T) {
	t.Run("Should apply minimum constraint", func(t *testing.T) {
		schema := map[string]any{"type": "integer"}
		applySchemaTag(schema, "minimum=0")

		assert.Equal(t, float64(0), schema["minimum"])
	})

	t.Run("Should apply maximum constraint", func(t *testing.T) {
		schema := map[string]any{"type": "integer"}
		applySchemaTag(schema, "maximum=100")

		assert.Equal(t, float64(100), schema["maximum"])
	})

	t.Run("Should apply description", func(t *testing.T) {
		schema := map[string]any{"type": "string"}
		applySchemaTag(schema, "description=User full name")

		assert.Equal(t, "User full name", schema["description"])
	})

	t.Run("Should apply multiple constraints", func(t *testing.T) {
		schema := map[string]any{"type": "integer"}
		applySchemaTag(schema, "minimum=1,maximum=10,description=Page number")

		assert.Equal(t, float64(1), schema["minimum"])
		assert.Equal(t, float64(10), schema["maximum"])
		assert.Equal(t, "Page number", schema["description"])
	})

	t.Run("Should handle invalid minimum/maximum values", func(t *testing.T) {
		schema := map[string]any{"type": "integer"}
		applySchemaTag(schema, "minimum=invalid,maximum=also-invalid")

		// Should not add invalid values
		assert.NotContains(t, schema, "minimum")
		assert.NotContains(t, schema, "maximum")
	})

	t.Run("Should handle empty tag", func(t *testing.T) {
		schema := map[string]any{"type": "string"}
		originalLen := len(schema)
		applySchemaTag(schema, "")

		assert.Len(t, schema, originalLen)
	})
}

func TestReflectType(t *testing.T) {
	t.Run("Should reflect string type", func(t *testing.T) {
		schema := reflectType(reflect.TypeOf(""))
		assert.Equal(t, "string", schema["type"])
	})

	t.Run("Should reflect int types", func(t *testing.T) {
		intTypes := []any{int(0), int8(0), int16(0), int32(0), int64(0)}

		for _, intType := range intTypes {
			schema := reflectType(reflect.TypeOf(intType))
			assert.Equal(t, "integer", schema["type"])
		}
	})

	t.Run("Should reflect uint types", func(t *testing.T) {
		uintTypes := []any{uint(0), uint8(0), uint16(0), uint32(0), uint64(0)}

		for _, uintType := range uintTypes {
			schema := reflectType(reflect.TypeOf(uintType))
			assert.Equal(t, "integer", schema["type"])
		}
	})

	t.Run("Should reflect float types", func(t *testing.T) {
		floatTypes := []any{float32(0), float64(0)}

		for _, floatType := range floatTypes {
			schema := reflectType(reflect.TypeOf(floatType))
			assert.Equal(t, "number", schema["type"])
		}
	})

	t.Run("Should reflect bool type", func(t *testing.T) {
		schema := reflectType(reflect.TypeOf(true))
		assert.Equal(t, "boolean", schema["type"])
	})

	t.Run("Should reflect slice type", func(t *testing.T) {
		schema := reflectType(reflect.TypeOf([]string{}))

		assert.Equal(t, "array", schema["type"])
		items, ok := schema["items"].(map[string]any)
		assert.True(t, ok)
		assert.Equal(t, "string", items["type"])
	})

	t.Run("Should reflect array type", func(t *testing.T) {
		schema := reflectType(reflect.TypeOf([3]int{}))

		assert.Equal(t, "array", schema["type"])
		items, ok := schema["items"].(map[string]any)
		assert.True(t, ok)
		assert.Equal(t, "integer", items["type"])
	})

	t.Run("Should reflect map type", func(t *testing.T) {
		schema := reflectType(reflect.TypeOf(map[string]int{}))

		assert.Equal(t, "object", schema["type"])
		additionalProps, ok := schema["additionalProperties"].(map[string]any)
		assert.True(t, ok)
		assert.Equal(t, "integer", additionalProps["type"])
	})

	t.Run("Should reflect struct type", func(t *testing.T) {
		type TestStruct struct {
			Name string `json:"name"`
		}

		schema := reflectType(reflect.TypeOf(TestStruct{}))

		assert.Equal(t, "object", schema["type"])
		properties, ok := schema["properties"].(map[string]any)
		assert.True(t, ok)
		assert.Contains(t, properties, "name")
	})

	t.Run("Should handle pointer types", func(t *testing.T) {
		schema := reflectType(reflect.TypeOf((*string)(nil)))

		assert.Equal(t, "string", schema["type"])
	})

	t.Run("Should fallback to string for unknown types", func(t *testing.T) {
		// Using channel as an uncommon type
		schema := reflectType(reflect.TypeOf(make(chan int)))

		assert.Equal(t, "string", schema["type"])
	})
}

func TestTool(t *testing.T) {
	t.Run("Should create tool with proper structure", func(t *testing.T) {
		tool := Tool{
			Name:        "test_tool",
			Description: "A test tool",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"param1": map[string]any{"type": "string"},
				},
			},
		}

		assert.Equal(t, "test_tool", tool.Name)
		assert.Equal(t, "A test tool", tool.Description)
		assert.NotNil(t, tool.InputSchema)
	})
}

func TestOperation(t *testing.T) {
	t.Run("Should create operation with all fields", func(t *testing.T) {
		operation := Operation{
			Path:           "/users/:id",
			Method:         "GET",
			QueryParams:    []string{"page", "limit"},
			HeaderParams:   []string{"Authorization"},
			FormDataParams: []string{"file"},
		}

		assert.Equal(t, "/users/:id", operation.Path)
		assert.Equal(t, "GET", operation.Method)
		assert.Contains(t, operation.QueryParams, "page")
		assert.Contains(t, operation.HeaderParams, "Authorization")
		assert.Contains(t, operation.FormDataParams, "file")
	})
}

func TestRegisteredSchemaInfo(t *testing.T) {
	t.Run("Should hold query and body schemas", func(t *testing.T) {
		type QuerySchema struct {
			Page int `form:"page"`
		}
		type BodySchema struct {
			Name string `json:"name"`
		}

		info := RegisteredSchemaInfo{
			QuerySchema: QuerySchema{},
			BodySchema:  BodySchema{},
		}

		assert.NotNil(t, info.QuerySchema)
		assert.NotNil(t, info.BodySchema)
	})
}

func TestMCPMessage(t *testing.T) {
	t.Run("Should create MCP message with all fields", func(t *testing.T) {
		params := map[string]any{"key": "value"}
		message := MCPMessage{
			Jsonrpc: "2.0",
			ID:      json.RawMessage(`"test-id"`),
			Method:  "tools/list",
			Params:  params,
		}

		assert.Equal(t, "2.0", message.Jsonrpc)
		assert.Equal(t, json.RawMessage(`"test-id"`), message.ID)
		assert.Equal(t, "tools/list", message.Method)
		assert.Equal(t, params, message.Params)
	})
}
