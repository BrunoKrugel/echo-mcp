# MCP Wrapper for Echo Framework

[![Build Status](https://github.com/BrunoKrugel/echo-mcp/actions/workflows/run-test.yaml/badge.svg?branch=main)](https://github.com/features/actions)
[![Codecov branch](https://img.shields.io/codecov/c/github/BrunoKrugel/echo-mcp/main.svg)](https://codecov.io/gh/BrunoKrugel/echo-mcp)
[![Go Report Card](https://goreportcard.com/badge/github.com/BrunoKrugel/echo-mcp)](https://goreportcard.com/report/github.com/BrunoKrugel/echo-mcp)
[![Release](https://img.shields.io/github/release/BrunoKrugel/echo-mcp.svg?style=flat-square)](https://github.com/BrunoKrugel/echo-mcp/releases)

Wrap any existing Echo API into MCP tools, enabling AI agents to interact with your API through [Model Context Protocol](https://modelcontextprotocol.io/introduction).

Inspired by [gin-mcp](https://github.com/ckanthony/gin-mcp) but for the [Echo framework](https://echo.labstack.com/).

## Key Features

- **Zero Configuration**: Works with any existing Echo API
- **Multiple Schema Sources**: Support for Swaggo, raw OpenAPI YAML/JSON, and manual schemas
- **Filtering**: Include/exclude endpoints with wildcard patterns
- **MCP Compatible**: Works with any agent that supports MCP.

## Installation

```bash
go get github.com/BrunoKrugel/echo-mcp
```

## Quick Start

```go
package main

import (
    "net/http"
    server "github.com/BrunoKrugel/echo-mcp"
    "github.com/labstack/echo/v4"
)

func main() {
    e := echo.New()

    // Existing API routes
    e.GET("/ping", func(c echo.Context) error {
        return c.JSON(http.StatusOK, map[string]string{"message": "pong"})
    })


    // Add MCP support
    mcp := server.New(e)
    mcp.Mount("/mcp")

    e.Start(":8080")
}
```

Now the API is accessible at `http://localhost:8080/mcp`

## Advanced Usage

### Automatic Swagger Schemas

If you already use Swaggo for Swagger documentation, enable automatic schema generation:

```go
// @Summary Get user by ID
// @Description Retrieve detailed user information
// @Tags users
// @Param id path int true "User ID" minimum(1)
// @Success 200 {object} User
// @Router /users/{id} [get]
func GetUser(c echo.Context) error {
    // Your handler code
}

func main() {
    e := echo.New()
    e.GET("/users/:id", GetUser)

    // Enable automatic swagger schema generation
    mcp := server.NewWithConfig(e, &server.Config{
        BaseURL:              "http://localhost:8080",
        EnableSwaggerSchemas: true,
    })
    mcp.Mount("/mcp")

    e.Start(":8080")
}
```

### Raw OpenAPI Schema Support

If you use other OpenAPI libraries like `swaggest/openapi-go`, you can pass a raw YAML or JSON schema string:

```go
import (
    "github.com/swaggest/openapi-go/openapi3"
    server "github.com/BrunoKrugel/echo-mcp"
)

func main() {
    e := echo.New()

    // ... define your routes ...

    // Generate OpenAPI schema
    reflector := openapi3.Reflector{}
    reflector.SpecEns().WithOpenapi("3.0.3")
    reflector.SpecEns().Info.WithTitle("My API").WithVersion("1.0.0")

    // Add operations to reflector
    // ...

    // Export to YAML (or JSON)
    schema, _ := reflector.Spec.MarshalYAML()

    // Pass raw schema to MCP server
    // ... you can also embed the schema from an exiting openapi.yaml file
    mcp := server.NewWithConfig(e, &server.Config{
        OpenAPISchema: string(schema),
    })
    mcp.Mount("/mcp")

    e.Start(":8080")
}
```

The `OpenAPISchema` field accepts both YAML and JSON formatted strings. When provided, it automatically populates:
- Server name from schema title
- Description from schema description
- Version from schema version
- Tool schemas from operation definitions

### Endpoint Filtering

Expose only the necessary endpoints to MCP tools:

```go
mcp := server.New(e)

// Include only specific endpoints
mcp.RegisterEndpoints([]string{
    "/api/v1/users/:id",
    "/api/v1/orders",
})

// Or exclude internal endpoints
mcp.ExcludeEndpoints([]string{
    "/health",      // Exclude health checks
})
```

### Manual Schema Registration (WIP)

For better control, register schemas manually:

```go
type CreateUserRequest struct {
    Name  string `json:"name" jsonschema:"required,description=User full name"`
    Email string `json:"email" jsonschema:"required,description=User email address"`
    Age   int    `json:"age,omitempty" jsonschema:"minimum=0,maximum=150"`
}

type UserQuery struct {
    Page   int    `form:"page,default=1" jsonschema:"minimum=1"`
    Limit  int    `form:"limit,default=10" jsonschema:"maximum=100"`
    Active bool   `form:"active" jsonschema:"description=Filter by active status"`
}

mcp := server.New(e, &server.Config{BaseURL: "http://localhost:8080"})

// Register schemas for specific routes
mcp.RegisterSchema("POST", "/users", nil, CreateUserRequest{})
mcp.RegisterSchema("GET", "/users", UserQuery{}, nil)
```

## Schema Generation Methods

Echo-MCP supports four schema generation approaches, with automatic fallback:

| Method | Use Case | Priority |
|--------|----------|----------|
| **Raw OpenAPI Schema** | Using OpenAPI libraries like swaggest/openapi-go | First (if OpenAPISchema is set) |
| **Swagger** | Production APIs with Swaggo annotations | First (if EnableSwaggerSchemas is set) |
| **Manual** | Fine-grained control, complex validation | Second |
| **Automatic** | Quick prototyping, simple endpoints | Fallback |

```go
// Option 1: Using raw OpenAPI schema (swaggest/openapi-go, etc.)
schema, _ := reflector.Spec.MarshalYAML()
mcp := server.New(e, &server.Config{
    OpenAPISchema: string(schema), // Use raw schema
})

// Option 2: Using Swaggo
mcp := server.New(e, &server.Config{
    EnableSwaggerSchemas: true, // Load from swaggo docs
})

// Option 3: Manual schemas for fine-grained control
mcp.RegisterSchema("POST", "/users", nil, CreateUserRequest{})

// Option 4: Automatic inference (fallback)
// No configuration needed - routes will use basic path/body inference
```

## MCP Client Integration

Once your server is running:

### Manual configuration

```json
{
  "mcpServers": {
    "echo-api": {
      "type": "http",
      "url": "http://localhost:8080/mcp",
      "timeout": 120
    },
  }
}
```

## Local Testing

For local testing, use MCP Inspector:

```bash
npx @modelcontextprotocol/inspector http://localhost:8080/mcp
```

## Acknowledgments

- [Swaggo](https://github.com/swaggo/swag) - Swagger documentation generator
- [swaggest/openapi-go](https://github.com/swaggest/openapi-go) - OpenAPI 3.0 toolkit for Go
- [Echo Framework](https://echo.labstack.com/) - High performance Go web framework
- [Echo Swagger](https://github.com/swaggo/echo-swagger) - Swagger UI middleware for Echo
- [Model Context Protocol](https://modelcontextprotocol.io/) - Universal protocol for AI-tool interaction
