# Zero-Config Echo API to MCP

Adapt any existing Echo API into MCP tools in seconds, enabling AI assistants to interact with your API through [Model Context Protocol](https://modelcontextprotocol.io/introduction).

Inspired by [gin-mcp](https://github.com/ckanthony/gin-mcp) but for the [Echo framework](https://echo.labstack.com/).

## Key Features

- **Zero Configuration**: Works out-of-the-box with any existing Echo API
- **Swagger Integration**: Automatic schema generation from Swagger/OpenAPI annotations
- **Smart Filtering**: Include/exclude endpoints with wildcard patterns
- **MCP Compatible**: Works with Cursor, Claude Desktop, VS Code and more.

## Installation

```bash
go get github.com/BrunoKrugel/echo-mcp
```

## Quick Start

### Basic Usage (30 seconds)

```go
package main

import (
    "net/http"
    server "github.com/BrunoKrugel/echo-mcp"
    "github.com/labstack/echo/v4"
)

func main() {
    e := echo.New()

    // Your existing API routes
    e.GET("/ping", func(c echo.Context) error {
        return c.JSON(http.StatusOK, map[string]string{"message": "pong"})
    })


    // Add MCP support (this is all you need!)
    mcp := server.New(e, &server.Config{
        BaseURL: "http://localhost:8080",
    })
    mcp.Mount("/mcp")

    e.Start(":8080")
}
```

Now the API is accessible at `http://localhost:8080/mcp`

## Advanced Usage

### Automatic Swagger Schemas

Let swagger annotations drive your MCP schemas for type-safe, well-documented API tools:

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
    mcp := server.New(e, &server.Config{
        BaseURL:              "http://localhost:8080",
        EnableSwaggerSchemas: true,
    })
    mcp.Mount("/mcp")

    e.Start(":8080")
}
```

### Endpoint Filtering

Control which endpoints become MCP tools:

```go
mcp := server.New(e, &server.Config{
    BaseURL: "http://localhost:8080",
})

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

### 🔧 Manual Schema Registration

For maximum control, register schemas manually:

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

### Cursor IDE
1. Settings → MCP → Add Server
2. Server URL: `http://localhost:8080/mcp`
3. AI can now call your API!

### Claude Desktop
```json
{
  "mcpServers": {
    "echo-api": {
      "command": "npx",
      "args": ["-y", "@anthropic-ai/mcp-client", "http://localhost:8080/mcp"]
    }
  }
}
```

### Continue.dev
```json
{
  "mcpServers": [
    {
      "name": "echo-api",
      "url": "http://localhost:8080/mcp"
    }
  ]
}
```

## Acknowledgments

- [Model Context Protocol](https://modelcontextprotocol.io/) - Universal protocol for AI-tool interaction
- [Echo Framework](https://echo.labstack.com/) - High performance Go web framework
- [Swaggo](https://github.com/swaggo/swag) - Swagger documentation generator
- [Echo Swagger](https://github.com/swaggo/echo-swagger) - Swagger UI middleware for Echo
