package transport

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/BrunoKrugel/echo-mcp/pkg/types"
)

func TestNewHTTPTransport(t *testing.T) {
	t.Run("Should create new HTTP transport", func(t *testing.T) {
		transport := NewHTTPTransport("/mcp")

		assert.Equal(t, "/mcp", transport.mountPath)
		assert.NotNil(t, transport.handlers)
		assert.NotNil(t, transport.sessions)
		assert.Len(t, transport.handlers, 0)
		assert.Len(t, transport.sessions, 0)
	})

	t.Run("Should handle different mount paths", func(t *testing.T) {
		testPaths := []string{"/api/mcp", "/v1/mcp", "/custom/path"}

		for _, path := range testPaths {
			transport := NewHTTPTransport(path)
			assert.Equal(t, path, transport.mountPath)
		}
	})
}

func TestHTTPTransport_RegisterHandler(t *testing.T) {
	t.Run("Should register message handler", func(t *testing.T) {
		transport := NewHTTPTransport("/mcp")

		handler := func(params any) (any, error) {
			return "test result", nil
		}

		transport.RegisterHandler("test/method", handler)

		transport.mu.RLock()
		registeredHandler := transport.handlers["test/method"]
		transport.mu.RUnlock()

		assert.NotNil(t, registeredHandler)

		result, err := registeredHandler(nil)
		assert.NoError(t, err)
		assert.Equal(t, "test result", result)
	})

	t.Run("Should handle concurrent handler registration", func(t *testing.T) {
		transport := NewHTTPTransport("/mcp")

		// Register handlers concurrently
		go transport.RegisterHandler("method1", func(params any) (any, error) { return "1", nil })
		go transport.RegisterHandler("method2", func(params any) (any, error) { return "2", nil })
		go transport.RegisterHandler("method3", func(params any) (any, error) { return "3", nil })

		// Give goroutines time to complete
		time.Sleep(10 * time.Millisecond)

		transport.mu.RLock()
		assert.Len(t, transport.handlers, 3)
		transport.mu.RUnlock()
	})

	t.Run("Should overwrite existing handlers", func(t *testing.T) {
		transport := NewHTTPTransport("/mcp")

		handler1 := func(params any) (any, error) { return "first", nil }
		handler2 := func(params any) (any, error) { return "second", nil }

		transport.RegisterHandler("same/method", handler1)
		transport.RegisterHandler("same/method", handler2)

		transport.mu.RLock()
		handler := transport.handlers["same/method"]
		transport.mu.RUnlock()

		result, err := handler(nil)
		assert.NoError(t, err)
		assert.Equal(t, "second", result)
	})
}

func TestHTTPTransport_MountPath(t *testing.T) {
	t.Run("Should return correct mount path", func(t *testing.T) {
		transport := NewHTTPTransport("/api/v1/mcp")

		assert.Equal(t, "/api/v1/mcp", transport.MountPath())
	})
}

func TestHTTPTransport_HandleConnection(t *testing.T) {
	t.Run("Should return method not allowed error", func(t *testing.T) {
		transport := NewHTTPTransport("/mcp")

		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := transport.HandleConnection(c)

		assert.Error(t, err)
		httpErr, ok := err.(*echo.HTTPError)
		assert.True(t, ok)
		assert.Equal(t, http.StatusMethodNotAllowed, httpErr.Code)
		assert.Contains(t, httpErr.Message.(string), "GET method not supported")
	})
}

func TestHTTPTransport_HandleMessage(t *testing.T) {
	t.Run("Should handle valid MCP initialize message", func(t *testing.T) {
		transport := NewHTTPTransport("/mcp")

		// Register initialize handler
		transport.RegisterHandler("initialize", func(params any) (any, error) {
			return map[string]any{
				"protocolVersion": "2024-11-05",
				"capabilities":    map[string]any{},
				"serverInfo":      map[string]any{"name": "test", "version": "1.0.0"},
			}, nil
		})

		message := types.MCPMessage{
			Jsonrpc: "2.0",
			ID:      json.RawMessage(`"test-id"`),
			Method:  "initialize",
			Params:  map[string]any{},
		}

		msgBytes, err := json.Marshal(message)
		require.NoError(t, err)

		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewBuffer(msgBytes))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err = transport.HandleMessage(c)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var response map[string]any
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "2.0", response["jsonrpc"])
		assert.Equal(t, "test-id", response["id"])
		assert.Contains(t, response, "result")
	})

	t.Run("Should handle tools/list message", func(t *testing.T) {
		transport := NewHTTPTransport("/mcp")

		transport.RegisterHandler("tools/list", func(params any) (any, error) {
			return map[string]any{
				"tools": []map[string]any{
					{"name": "test_tool", "description": "A test tool"},
				},
			}, nil
		})

		message := types.MCPMessage{
			Jsonrpc: "2.0",
			ID:      json.RawMessage(`"list-id"`),
			Method:  "tools/list",
			Params:  map[string]any{},
		}

		msgBytes, err := json.Marshal(message)
		require.NoError(t, err)

		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewBuffer(msgBytes))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err = transport.HandleMessage(c)

		assert.NoError(t, err)

		var response map[string]any
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		assert.NoError(t, err)

		result := response["result"].(map[string]any)
		tools := result["tools"].([]any)
		assert.Len(t, tools, 1)
	})

	t.Run("Should handle tools/call message", func(t *testing.T) {
		transport := NewHTTPTransport("/mcp")

		transport.RegisterHandler("tools/call", func(params any) (any, error) {
			return map[string]any{
				"content": []map[string]any{
					{"type": "text", "text": "Tool executed successfully"},
				},
			}, nil
		})

		message := types.MCPMessage{
			Jsonrpc: "2.0",
			ID:      json.RawMessage(`"call-id"`),
			Method:  "tools/call",
			Params: map[string]any{
				"name":      "test_tool",
				"arguments": map[string]any{"param": "value"},
			},
		}

		msgBytes, err := json.Marshal(message)
		require.NoError(t, err)

		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewBuffer(msgBytes))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err = transport.HandleMessage(c)

		assert.NoError(t, err)

		var response map[string]any
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		assert.NoError(t, err)

		result := response["result"].(map[string]any)
		content := result["content"].([]any)
		assert.Len(t, content, 1)
	})

	t.Run("Should handle invalid JSON", func(t *testing.T) {
		transport := NewHTTPTransport("/mcp")

		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader("invalid json"))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := transport.HandleMessage(c)

		assert.Error(t, err)
	})

	t.Run("Should handle missing handler", func(t *testing.T) {
		transport := NewHTTPTransport("/mcp")

		message := types.MCPMessage{
			Jsonrpc: "2.0",
			ID:      json.RawMessage(`"test-id"`),
			Method:  "nonexistent/method",
			Params:  map[string]any{},
		}

		msgBytes, err := json.Marshal(message)
		require.NoError(t, err)

		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewBuffer(msgBytes))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err = transport.HandleMessage(c)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var response map[string]any
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response, "error")

		errorObj := response["error"].(map[string]any)
		assert.Contains(t, errorObj["message"], "Method 'nonexistent/method' not found")
	})

	t.Run("Should handle handler error", func(t *testing.T) {
		transport := NewHTTPTransport("/mcp")

		transport.RegisterHandler("error/method", func(params any) (any, error) {
			return nil, assert.AnError
		})

		message := types.MCPMessage{
			Jsonrpc: "2.0",
			ID:      json.RawMessage(`"error-id"`),
			Method:  "error/method",
			Params:  map[string]any{},
		}

		msgBytes, err := json.Marshal(message)
		require.NoError(t, err)

		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewBuffer(msgBytes))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err = transport.HandleMessage(c)

		assert.NoError(t, err)

		var response map[string]any
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response, "error")
	})

	t.Run("Should handle notification messages", func(t *testing.T) {
		transport := NewHTTPTransport("/mcp")

		// Handler should still be called for notifications
		handlerCalled := false
		transport.RegisterHandler("notifications/test", func(params any) (any, error) {
			handlerCalled = true
			return nil, nil
		})

		// Notification message has no ID
		message := map[string]any{
			"jsonrpc": "2.0",
			"method":  "notifications/test",
			"params":  map[string]any{},
		}

		msgBytes, err := json.Marshal(message)
		require.NoError(t, err)

		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewBuffer(msgBytes))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err = transport.HandleMessage(c)

		assert.NoError(t, err)
		assert.True(t, handlerCalled)

		// Notifications should still return 200 OK with empty response
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestHTTPTransport_NotifyToolsChanged(t *testing.T) {
	t.Run("Should handle tools changed notification", func(t *testing.T) {
		transport := NewHTTPTransport("/mcp")

		// This is a no-op in HTTP transport, just verify it doesn't panic
		transport.NotifyToolsChanged()

		// Should not crash or cause any issues
		assert.True(t, true)
	})
}

func TestSession(t *testing.T) {
	t.Run("Should create session with ID and timestamp", func(t *testing.T) {
		session := &Session{
			ID:      "test-session-id",
			Created: time.Now().Unix(),
		}

		assert.Equal(t, "test-session-id", session.ID)
		assert.Greater(t, session.Created, int64(0))
	})
}

func TestHTTPTransport_Integration(t *testing.T) {
	t.Run("Should handle complete MCP workflow", func(t *testing.T) {
		transport := NewHTTPTransport("/mcp")

		// Register all MCP handlers
		transport.RegisterHandler("initialize", func(params any) (any, error) {
			return map[string]any{
				"protocolVersion": "2024-11-05",
				"capabilities":    map[string]any{"tools": map[string]any{}},
				"serverInfo":      map[string]any{"name": "test", "version": "1.0.0"},
			}, nil
		})

		transport.RegisterHandler("tools/list", func(params any) (any, error) {
			return map[string]any{
				"tools": []map[string]any{
					{
						"name":        "test_tool",
						"description": "A test tool",
						"inputSchema": map[string]any{"type": "object"},
					},
				},
			}, nil
		})

		transport.RegisterHandler("tools/call", func(params any) (any, error) {
			paramMap := params.(map[string]any)
			toolName := paramMap["name"].(string)
			return map[string]any{
				"content": []map[string]any{
					{"type": "text", "text": "Executed " + toolName + " successfully"},
				},
			}, nil
		})

		e := echo.New()

		// 1. Initialize
		initMsg := types.MCPMessage{
			Jsonrpc: "2.0", ID: json.RawMessage(`"1"`), Method: "initialize", Params: map[string]any{},
		}
		msgBytes, _ := json.Marshal(initMsg)
		req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewBuffer(msgBytes))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := transport.HandleMessage(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		// 2. List tools
		listMsg := types.MCPMessage{
			Jsonrpc: "2.0", ID: json.RawMessage(`"2"`), Method: "tools/list", Params: map[string]any{},
		}
		msgBytes, _ = json.Marshal(listMsg)
		req = httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewBuffer(msgBytes))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec = httptest.NewRecorder()
		c = e.NewContext(req, rec)

		err = transport.HandleMessage(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		// 3. Call tool
		callMsg := types.MCPMessage{
			Jsonrpc: "2.0",
			ID:      json.RawMessage(`"3"`),
			Method:  "tools/call",
			Params: map[string]any{
				"name":      "test_tool",
				"arguments": map[string]any{"param": "value"},
			},
		}
		msgBytes, _ = json.Marshal(callMsg)
		req = httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewBuffer(msgBytes))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec = httptest.NewRecorder()
		c = e.NewContext(req, rec)

		err = transport.HandleMessage(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		var response map[string]any
		err = json.Unmarshal(rec.Body.Bytes(), &response)
		assert.NoError(t, err)

		result := response["result"].(map[string]any)
		content := result["content"].([]any)[0].(map[string]any)
		assert.Contains(t, content["text"], "Executed test_tool successfully")
	})
}
