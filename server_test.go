package server

import (
	"net/http"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/BrunoKrugel/echo-mcp/pkg/types"
)

func TestNew(t *testing.T) {
	t.Run("Should create new EchoMCP instance with default config", func(t *testing.T) {
		e := echo.New()
		mcp := New(e)

		assert.NotNil(t, mcp)
		assert.Equal(t, e, mcp.echo)
		assert.True(t, mcp.config.EnableSwaggerSchemas)
		assert.NotNil(t, mcp.registeredSchemas)
		assert.NotNil(t, mcp.operations)
		assert.NotNil(t, mcp.executeToolFunc)
	})

	t.Run("Should set default version when none provided", func(t *testing.T) {
		e := echo.New()
		mcp := New(e)

		// Should have empty version initially (will be set to default in handleInitialize)
		assert.Equal(t, "", mcp.version)
	})
}

func TestNewWithConfig(t *testing.T) {
	t.Run("Should create EchoMCP instance with provided config", func(t *testing.T) {
		e := echo.New()
		config := &Config{
			Name:                 "Test API",
			Version:              "1.0.0",
			Description:          "Test description",
			BaseURL:              "http://localhost:8080",
			EnableSwaggerSchemas: false,
		}

		mcp := NewWithConfig(e, config)

		assert.NotNil(t, mcp)
		assert.Equal(t, "Test API", mcp.name)
		assert.Equal(t, "1.0.0", mcp.version)
		assert.Equal(t, "Test description", mcp.description)
		assert.Equal(t, "http://localhost:8080", mcp.baseURL)
		assert.False(t, mcp.config.EnableSwaggerSchemas)
	})

	t.Run("Should handle nil config", func(t *testing.T) {
		e := echo.New()
		mcp := NewWithConfig(e, nil)

		assert.NotNil(t, mcp)
		assert.NotNil(t, mcp.config)
	})

	t.Run("Should set include and exclude operations", func(t *testing.T) {
		e := echo.New()
		config := &Config{
			IncludeOperations: []string{"/users", "/orders"},
			ExcludeOperations: []string{"/health", "/metrics"},
		}

		mcp := NewWithConfig(e, config)

		assert.Equal(t, []string{"/users", "/orders"}, mcp.config.IncludeOperations)
		assert.Equal(t, []string{"/health", "/metrics"}, mcp.config.ExcludeOperations)
	})
}

func TestRegisterSchema(t *testing.T) {
	t.Run("Should register schema for route", func(t *testing.T) {
		e := echo.New()
		mcp := New(e)

		type TestQuery struct {
			Page int `form:"page"`
		}
		type TestBody struct {
			Name string `json:"name"`
		}

		mcp.RegisterSchema("GET", "/test", TestQuery{}, TestBody{})

		key := "GET /test"
		schema, exists := mcp.registeredSchemas[key]
		assert.True(t, exists)
		assert.NotNil(t, schema.QuerySchema)
		assert.NotNil(t, schema.BodySchema)
	})

	t.Run("Should handle multiple schema registrations", func(t *testing.T) {
		e := echo.New()
		mcp := New(e)

		mcp.RegisterSchema("GET", "/users", nil, nil)
		mcp.RegisterSchema("POST", "/users", nil, nil)

		assert.Len(t, mcp.registeredSchemas, 2)
		assert.Contains(t, mcp.registeredSchemas, "GET /users")
		assert.Contains(t, mcp.registeredSchemas, "POST /users")
	})
}

func TestRegisterEndpoints(t *testing.T) {
	t.Run("Should set include endpoints", func(t *testing.T) {
		e := echo.New()
		mcp := New(e)

		endpoints := []string{"/users/:id", "/orders"}
		mcp.RegisterEndpoints(endpoints)

		assert.Equal(t, endpoints, mcp.includeEndpoints)
	})

	t.Run("Should overwrite previous include endpoints", func(t *testing.T) {
		e := echo.New()
		mcp := New(e)

		mcp.RegisterEndpoints([]string{"/old"})
		mcp.RegisterEndpoints([]string{"/new"})

		assert.Equal(t, []string{"/new"}, mcp.includeEndpoints)
	})
}

func TestExcludeEndpoints(t *testing.T) {
	t.Run("Should set exclude endpoints", func(t *testing.T) {
		e := echo.New()
		mcp := New(e)

		endpoints := []string{"/health", "/metrics"}
		mcp.ExcludeEndpoints(endpoints)

		assert.Equal(t, endpoints, mcp.excludeEndpoints)
	})

	t.Run("Should overwrite previous exclude endpoints", func(t *testing.T) {
		e := echo.New()
		mcp := New(e)

		mcp.ExcludeEndpoints([]string{"/old"})
		mcp.ExcludeEndpoints([]string{"/new"})

		assert.Equal(t, []string{"/new"}, mcp.excludeEndpoints)
	})
}

func TestMount(t *testing.T) {
	t.Run("Should mount MCP server successfully", func(t *testing.T) {
		e := echo.New()
		mcp := New(e)

		err := mcp.Mount("/mcp")

		assert.NoError(t, err)
		assert.NotNil(t, mcp.transport)
	})

	t.Run("Should register routes after mounting", func(t *testing.T) {
		e := echo.New()
		e.GET("/test", func(c echo.Context) error {
			return c.JSON(http.StatusOK, map[string]string{"message": "test"})
		})

		mcp := New(e)
		err := mcp.Mount("/mcp")

		assert.NoError(t, err)
		assert.NotEmpty(t, mcp.tools)
		assert.NotEmpty(t, mcp.operations)
	})
}

func TestShouldIncludeRoute(t *testing.T) {
	t.Run("Should include all routes when no filters set", func(t *testing.T) {
		e := echo.New()
		mcp := New(e)

		route := &echo.Route{Path: "/users", Method: "GET"}
		result := mcp.shouldIncludeRoute(route)

		assert.True(t, result)
	})

	t.Run("Should only include routes in include list", func(t *testing.T) {
		e := echo.New()
		mcp := New(e)
		mcp.RegisterEndpoints([]string{"/users", "/orders"})

		userRoute := &echo.Route{Path: "/users", Method: "GET"}
		healthRoute := &echo.Route{Path: "/health", Method: "GET"}

		assert.True(t, mcp.shouldIncludeRoute(userRoute))
		assert.False(t, mcp.shouldIncludeRoute(healthRoute))
	})

	t.Run("Should exclude routes in exclude list", func(t *testing.T) {
		e := echo.New()
		mcp := New(e)
		mcp.ExcludeEndpoints([]string{"/health", "/metrics"})

		userRoute := &echo.Route{Path: "/users", Method: "GET"}
		healthRoute := &echo.Route{Path: "/health", Method: "GET"}

		assert.True(t, mcp.shouldIncludeRoute(userRoute))
		assert.False(t, mcp.shouldIncludeRoute(healthRoute))
	})

	t.Run("Should prioritize include over exclude", func(t *testing.T) {
		e := echo.New()
		mcp := New(e)
		mcp.RegisterEndpoints([]string{"/users"})
		mcp.ExcludeEndpoints([]string{"/users"}) // Should be ignored

		userRoute := &echo.Route{Path: "/users", Method: "GET"}

		assert.True(t, mcp.shouldIncludeRoute(userRoute))
	})
}

func TestMatchesEndpoint(t *testing.T) {
	e := echo.New()
	mcp := New(e)

	t.Run("Should match exact paths", func(t *testing.T) {
		assert.True(t, mcp.matchesEndpoint("/users", "/users"))
		assert.False(t, mcp.matchesEndpoint("/users", "/orders"))
	})

	t.Run("Should match wildcard patterns", func(t *testing.T) {
		assert.True(t, mcp.matchesEndpoint("/admin/users", "/admin/*"))
		assert.True(t, mcp.matchesEndpoint("/admin/orders", "/admin/*"))
		assert.False(t, mcp.matchesEndpoint("/users", "/admin/*"))
	})

	t.Run("Should handle path parameters", func(t *testing.T) {
		// This is a basic implementation, could be enhanced
		assert.False(t, mcp.matchesEndpoint("/users/:id", "/users/123"))
	})
}

func TestHandleInitialize(t *testing.T) {
	t.Run("Should return proper initialize response", func(t *testing.T) {
		e := echo.New()
		mcp := NewWithConfig(e, &Config{
			Name:    "Test API",
			Version: "1.0.0",
		})

		response, err := mcp.handleInitialize(nil)

		assert.NoError(t, err)
		assert.NotNil(t, response)

		initResp, ok := response.(InitializeResponse)
		assert.True(t, ok)
		assert.Equal(t, "2024-11-05", initResp.ProtocolVersion)
		assert.Equal(t, "Test API", initResp.ServerInfo.Name)
		assert.Equal(t, "1.0.0", initResp.ServerInfo.Version)
		assert.NotNil(t, initResp.Capabilities)
	})

	t.Run("Should use fallback version when empty", func(t *testing.T) {
		e := echo.New()
		mcp := New(e)

		response, err := mcp.handleInitialize(nil)

		assert.NoError(t, err)
		initResp, ok := response.(InitializeResponse)
		assert.True(t, ok)
		assert.Equal(t, "1.0.0", initResp.ServerInfo.Version)
	})
}

func TestHandleToolsList(t *testing.T) {
	t.Run("Should return tools list", func(t *testing.T) {
		e := echo.New()
		e.GET("/test", func(c echo.Context) error {
			return c.JSON(http.StatusOK, "test")
		})

		mcp := New(e)
		err := mcp.Mount("/mcp")
		require.NoError(t, err)

		response, err := mcp.handleToolsList(nil)

		assert.NoError(t, err)
		assert.NotNil(t, response)

		toolsResp, ok := response.(ToolsListResponse)
		assert.True(t, ok)
		assert.NotEmpty(t, toolsResp.Tools)
	})
}

func TestHandleToolCall(t *testing.T) {
	t.Run("Should handle valid tool call", func(t *testing.T) {
		e := echo.New()
		mcp := NewWithConfig(e, &Config{BaseURL: "http://localhost:8080"})

		// Mock execute function for testing
		mcp.executeToolFunc = func(operationID string, parameters map[string]any) (any, error) {
			return map[string]string{"result": "success"}, nil
		}

		params := map[string]any{
			"name":      "test_tool",
			"arguments": map[string]any{"param": "value"},
		}

		response, err := mcp.handleToolCall(params)

		assert.NoError(t, err)
		assert.NotNil(t, response)

		toolCallResp, ok := response.(ToolCallResponse)
		assert.True(t, ok)
		assert.NotEmpty(t, toolCallResp.Content)
		assert.Equal(t, "text", toolCallResp.Content[0].Type)
	})

	t.Run("Should handle missing tool name", func(t *testing.T) {
		e := echo.New()
		mcp := New(e)

		params := map[string]any{
			"arguments": map[string]any{"param": "value"},
		}

		response, err := mcp.handleToolCall(params)

		assert.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "missing tool name")
	})

	t.Run("Should handle invalid parameters", func(t *testing.T) {
		e := echo.New()
		mcp := New(e)

		response, err := mcp.handleToolCall("invalid")

		assert.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "invalid parameters")
	})
}

func TestBuildRequestURL(t *testing.T) {
	e := echo.New()
	mcp := NewWithConfig(e, &Config{BaseURL: "http://localhost:8080"})

	t.Run("Should build URL with path parameters", func(t *testing.T) {
		operation := types.Operation{
			Path:   "/users/:id",
			Method: "GET",
		}
		parameters := map[string]any{
			"id": "123",
		}

		url := mcp.buildRequestURL(operation, parameters)

		assert.Equal(t, "http://localhost:8080/users/123", url)
	})

	t.Run("Should build URL with query parameters", func(t *testing.T) {
		operation := types.Operation{
			Path:        "/users",
			Method:      "GET",
			QueryParams: []string{"page", "limit"},
		}
		parameters := map[string]any{
			"page":  "1",
			"limit": "10",
		}

		url := mcp.buildRequestURL(operation, parameters)

		assert.True(t, strings.HasPrefix(url, "http://localhost:8080/users?"))
		assert.Contains(t, url, "page=1")
		assert.Contains(t, url, "limit=10")
	})

	t.Run("Should use default base URL when not configured", func(t *testing.T) {
		mcpNoBase := New(e)
		operation := types.Operation{
			Path:   "/test",
			Method: "GET",
		}

		url := mcpNoBase.buildRequestURL(operation, map[string]any{})

		assert.Equal(t, "http://localhost:8080/test", url)
	})
}

func TestGetServerInfo(t *testing.T) {
	t.Run("Should return server info", func(t *testing.T) {
		e := echo.New()
		mcp := NewWithConfig(e, &Config{
			Name:        "Test API",
			Version:     "1.0.0",
			Description: "Test description",
		})

		name, version, description := mcp.GetServerInfo()

		assert.Equal(t, "Test API", name)
		assert.Equal(t, "1.0.0", version)
		assert.Equal(t, "Test description", description)
	})
}

func TestFilterRoutes(t *testing.T) {
	t.Run("Should filter out MCP routes", func(t *testing.T) {
		e := echo.New()
		e.GET("/test", func(c echo.Context) error { return nil })
		e.POST("/mcp", func(c echo.Context) error { return nil })

		mcp := New(e)
		err := mcp.Mount("/mcp")
		require.NoError(t, err)

		routes := e.Routes()
		filtered := mcp.filterRoutes(routes)

		// Should filter out the MCP route
		foundMCP := false
		foundTest := false
		for _, route := range filtered {
			if route.Path == "/mcp" {
				foundMCP = true
			}
			if route.Path == "/test" {
				foundTest = true
			}
		}

		assert.False(t, foundMCP)
		assert.True(t, foundTest)
	})

	t.Run("Should apply endpoint filtering", func(t *testing.T) {
		e := echo.New()
		e.GET("/users", func(c echo.Context) error { return nil })
		e.GET("/health", func(c echo.Context) error { return nil })

		mcp := New(e)
		mcp.RegisterEndpoints([]string{"/users"})

		routes := e.Routes()
		filtered := mcp.filterRoutes(routes)

		// Should only include /users
		paths := make([]string, 0, len(filtered))
		for _, route := range filtered {
			paths = append(paths, route.Path)
		}

		assert.Contains(t, paths, "/users")
		assert.NotContains(t, paths, "/health")
	})
}
