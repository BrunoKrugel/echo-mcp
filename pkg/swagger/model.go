package swagger

import "strings"

type OpenAPISpec struct {
	Paths      map[string]PathItem `yaml:"paths"`
	Components Components          `yaml:"components"`
	Info       Info                `yaml:"info"`
	OpenAPI    string              `yaml:"openapi"`
	Servers    []Server            `yaml:"servers"`
}

type Info struct {
	Title       string `yaml:"title"`
	Description string `yaml:"description"`
	Version     string `yaml:"version"`
}

type Server struct {
	URL         string `yaml:"url"`
	Description string `yaml:"description"`
}

type PathItem map[string]Operation

type Operation struct {
	RequestBody *RequestBody        `yaml:"requestBody,omitempty"`
	Responses   map[string]Response `yaml:"responses"`
	Description string              `yaml:"description"`
	Tags        []string            `yaml:"tags"`
	Parameters  []Parameter         `yaml:"parameters,omitempty"`
}

type Parameter struct {
	Schema   ParameterSchema `yaml:"schema"`
	In       string          `yaml:"in"`
	Name     string          `yaml:"name"`
	Required bool            `yaml:"required"`
}

type ParameterSchema struct {
	Type    string `yaml:"type"`
	Example string `yaml:"example,omitempty"`
}

type Response struct {
	Content     map[string]MediaType `yaml:"content,omitempty"`
	Description string               `yaml:"description"`
}

type MediaType struct {
	Schema Schema `yaml:"schema"`
}

type Schema struct {
	Properties map[string]SchemaProperty `yaml:"properties,omitempty"`
	Ref        string                    `yaml:"$ref,omitempty"`
	Type       string                    `yaml:"type,omitempty"`
}

type SchemaProperty struct {
	Type    string `yaml:"type,omitempty"`
	Example string `yaml:"example,omitempty"`
	Ref     string `yaml:"$ref,omitempty"`
}

type Components struct {
	Schemas map[string]Schema `yaml:"schemas"`
}

type RequestBody struct {
	Content map[string]MediaType `yaml:"content"`
}

func (o *OpenAPISpec) ToSwaggerSpec() *SwaggerSpec {
	spec := &SwaggerSpec{
		Swagger: "2.0",
		Info: &SwaggerInfo{
			Title:       o.Info.Title,
			Description: o.Info.Description,
			Version:     o.Info.Version,
		},
		Paths:       map[string]SwaggerPath{},
		Definitions: map[string]*SwaggerSchema{},
	}

	// Convert schemas
	for name, schema := range o.Components.Schemas {
		spec.Definitions[name] = convertSchema(schema)
	}

	// Convert paths
	for path, pathItem := range o.Paths {
		swaggerPath := SwaggerPath{}

		for method, op := range pathItem {
			swaggerPath[method] = convertOperation(&op)
		}

		spec.Paths[path] = swaggerPath
	}

	return spec
}

func convertOperation(op *Operation) SwaggerOperation {
	operation := SwaggerOperation{
		Summary:     op.Description,
		Description: op.Description,
		Tags:        op.Tags,
		Responses:   map[string]SwaggerResponse{},
	}

	// Parameters
	for _, p := range op.Parameters {
		operation.Parameters = append(operation.Parameters, SwaggerParameter{
			Name:     p.Name,
			In:       p.In,
			Type:     p.Schema.Type,
			Required: p.Required,
		})
	}

	// Request body → body parameter
	if op.RequestBody != nil {
		if mt, ok := op.RequestBody.Content["application/json"]; ok {
			operation.Parameters = append(operation.Parameters, SwaggerParameter{
				Name:     "body",
				In:       "body",
				Required: true,
				Schema:   convertSchema(mt.Schema),
			})
		}
	}

	// Responses
	for code, resp := range op.Responses {
		swaggerResp := SwaggerResponse{
			Description: resp.Description,
		}

		if mt, ok := resp.Content["application/json"]; ok {
			swaggerResp.Schema = convertSchema(mt.Schema)
		}

		operation.Responses[code] = swaggerResp
	}

	return operation
}

func convertSchema(s Schema) *SwaggerSchema {
	sw := &SwaggerSchema{
		Type: s.Type,
	}

	if s.Ref != "" {
		sw.Ref = convertRef(s.Ref)
		return sw
	}

	if len(s.Properties) > 0 {
		sw.Properties = map[string]*SwaggerSchema{}
		for name, prop := range s.Properties {
			sw.Properties[name] = convertSchemaProperty(prop)
		}
	}

	return sw
}

func convertSchemaProperty(prop SchemaProperty) *SwaggerSchema {
	sw := &SwaggerSchema{
		Type: prop.Type,
		Ref:  convertRef(prop.Ref),
	}
	return sw
}

func convertRef(ref string) string {
	// "#/components/schemas/User" → "#/definitions/User"
	return strings.Replace(ref, "#/components/schemas/", "#/definitions/", 1)
}
