# echo-mcp for Echo v5

This is the **Echo v5 variant** of the [echo-mcp](https://github.com/BrunoKrugel/echo-mcp) library, living in the `v5/` directory of the main repository.

Wrap any existing Echo v5 API into MCP tools and enable AI agents to interact with your API through [Model Context Protocol](https://modelcontextprotocol.io/introduction).

For full documentation, examples, and the Echo v4 variant, see the [main repository README](https://github.com/BrunoKrugel/echo-mcp).

## Installation

```bash
go get github.com/BrunoKrugel/echo-mcp/v5
```

## Quick Start

```go
package main

import (
    "net/http"
    server "github.com/BrunoKrugel/echo-mcp/v5"
    "github.com/labstack/echo/v5"
)

func main() {
    e := echo.New()

    e.GET("/ping", func(c *echo.Context) error {
        return c.JSON(http.StatusOK, map[string]string{"message": "pong"})
    })

    mcp := server.New(e)
    mcp.Mount("/mcp")

    e.Start(":8080")
}
```

> Note: this module only works with Echo v5 (`github.com/labstack/echo/v5`). For Echo v4 applications use the root module `github.com/BrunoKrugel/echo-mcp`.
