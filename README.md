# Zero-Config Echo API to MCP

[![Build Status](https://github.com/BrunoKrugel/echo-mcp/actions/workflows/run-test.yaml/badge.svg?branch=main)](https://github.com/features/actions)
[![Codecov branch](https://img.shields.io/codecov/c/github/BrunoKrugel/echo-mcp/main.svg)](https://codecov.io/gh/BrunoKrugel/echo-mcp)
[![Go Report Card](https://goreportcard.com/badge/github.com/BrunoKrugel/echo-mcp)](https://goreportcard.com/report/github.com/BrunoKrugel/echo-mcp)
[![Release](https://img.shields.io/github/release/BrunoKrugel/echo-mcp.svg?style=flat-square)](https://github.com/BrunoKrugel/echo-mcp/releases)

Adapt any existing Echo API into MCP tools, enabling AI agents to interact with your API through [Model Context Protocol](https://modelcontextprotocol.io/introduction).

Inspired by [gin-mcp](https://github.com/ckanthony/gin-mcp) but for the [Echo framework](https://echo.labstack.com/).

## Key Features

- **Zero Configuration**: Works with any existing Echo API
- **Swagger Integration**: Automatic schema generation from Swagger/OpenAPI annotations
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

Echo-MCP supports three schema generation approaches, with automatic fallback:

| Method | Use Case | Priority |
|--------|----------|----------|
| **Swagger** | Production APIs with OpenAPI docs | First |
| **Manual** | Fine-grained control, complex validation | Second |
| **Automatic** | Quick prototyping, simple endpoints | Fallback |

```go
mcp := server.New(e, &server.Config{
    EnableSwaggerSchemas: true, // Try swagger first
})

// Manual schemas override swagger for specific routes
mcp.RegisterSchema("POST", "/users", nil, CreateUserRequest{})

// Remaining routes use automatic inference
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
npx -y @anthropic-ai/mcp-inspector http://localhost:8080/mcp
```

## Acknowledgments

- [Swaggo](https://github.com/swaggo/swag) - Swagger documentation generator
- [Echo Framework](https://echo.labstack.com/) - High performance Go web framework
- [Echo Swagger](https://github.com/swaggo/echo-swagger) - Swagger UI middleware for Echo
- [Model Context Protocol](https://modelcontextprotocol.io/) - Universal protocol for AI-tool interaction
