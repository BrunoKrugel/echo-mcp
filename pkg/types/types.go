// Package types defines the core data structures used throughout the echo-mcp library.
// This includes MCP protocol types, tool definitions, schema representations,
// and utility functions for JSON schema generation from Go types.
package types

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

type MCPMessage struct {
	Params  any             `json:"params,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *MCPError       `json:"error,omitempty"`
	Jsonrpc string          `json:"jsonrpc"`
	Method  string          `json:"method,omitempty"`
	ID      json.RawMessage `json:"id,omitempty"`
}

type MCPError struct {
	Data    any    `json:"data,omitempty"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}

type Tool struct {
	InputSchema any    `json:"inputSchema"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type Operation struct {
	Parameters     map[string]any
	Method         string
	Path           string
	Description    string
	HeaderParams   []string
	QueryParams    []string
	FormDataParams []string
}

type RegisteredSchemaInfo struct {
	QuerySchema any
	BodySchema  any
}

// GetSchema generates a JSON schema from a Go type using reflection and struct tags
func GetSchema(input any) map[string]any {
	if input == nil {
		return map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		}
	}

	val := reflect.ValueOf(input)
	typ := reflect.TypeOf(input)

	if typ.Kind() == reflect.Pointer {
		if val.IsNil() {
			return map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			}
		}
		typ = typ.Elem()
	}

	if typ.Kind() != reflect.Struct {
		fmt.Printf("Warning: Cannot generate schema for non-struct type: %s\n", typ.Kind())
		return map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		}
	}

	properties := make(map[string]any)
	var required []string

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if !field.IsExported() {
			continue
		}

		jsonTag := field.Tag.Get("json")
		if jsonTag == "-" {
			continue
		}

		fieldName := field.Name
		if jsonTag != "" {
			parts := strings.Split(jsonTag, ",")
			if parts[0] != "" {
				fieldName = parts[0]
			}
		}

		fieldSchema := reflectType(field.Type)

		if schemaTag := field.Tag.Get("jsonschema"); schemaTag != "" {
			applySchemaTag(fieldSchema, schemaTag)
		}

		formTag := field.Tag.Get("form")
		if strings.Contains(jsonTag, "required") || strings.Contains(formTag, "required") || strings.Contains(field.Tag.Get("jsonschema"), "required") {
			required = append(required, fieldName)
		}

		properties[fieldName] = fieldSchema
	}

	schema := map[string]any{
		"type":       "object",
		"properties": properties,
	}

	if len(required) > 0 {
		schema["required"] = required
	}

	return schema
}

// getUnderlyingType returns the underlying type, following pointers
func getUnderlyingType(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t
}

// reflectType converts a Go type to JSON schema type
func reflectType(t reflect.Type) map[string]any {
	underlyingType := getUnderlyingType(t)

	switch underlyingType.Kind() {
	case reflect.String:
		return map[string]any{"type": "string"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return map[string]any{"type": "integer"}
	case reflect.Float32, reflect.Float64:
		return map[string]any{"type": "number"}
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.Slice, reflect.Array:
		return map[string]any{
			"type":  "array",
			"items": reflectType(underlyingType.Elem()),
		}
	case reflect.Map:
		return map[string]any{
			"type":                 "object",
			"additionalProperties": reflectType(underlyingType.Elem()),
		}
	case reflect.Struct:
		// For nested structs, generate nested schema
		return GetSchema(reflect.New(underlyingType).Interface())
	default:
		return map[string]any{"type": "string"} // fallback
	}
}

// applySchemaTag applies jsonschema tag attributes to field schema
func applySchemaTag(fieldSchema map[string]any, tag string) {
	parts := strings.SplitSeq(tag, ",")
	for part := range parts {
		part = strings.TrimSpace(part)
		if after, ok := strings.CutPrefix(part, "description="); ok {
			fieldSchema["description"] = after
		} else if after0, ok0 := strings.CutPrefix(part, "minimum="); ok0 {
			if minimum, err := strconv.ParseFloat(after0, 64); err == nil {
				fieldSchema["minimum"] = minimum
			}
		} else if after1, ok1 := strings.CutPrefix(part, "maximum="); ok1 {
			if maximum, err := strconv.ParseFloat(after1, 64); err == nil {
				fieldSchema["maximum"] = maximum
			}
		}
	}
}
