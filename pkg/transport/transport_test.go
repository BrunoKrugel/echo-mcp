package transport

import (
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

// MockTransport implements the Transport interface for testing
type MockTransport struct {
	handlers    map[string]MessageHandler
	mountPath   string
	toolsNotify bool
}

func NewMockTransport(path string) *MockTransport {
	return &MockTransport{
		handlers:  make(map[string]MessageHandler),
		mountPath: path,
	}
}

func (m *MockTransport) RegisterHandler(method string, handler MessageHandler) {
	m.handlers[method] = handler
}

func (m *MockTransport) HandleConnection(c echo.Context) error {
	// Mock implementation
	return nil
}

func (m *MockTransport) HandleMessage(c echo.Context) error {
	// Mock implementation
	return nil
}

func (m *MockTransport) NotifyToolsChanged() {
	m.toolsNotify = true
}

func (m *MockTransport) MountPath() string {
	return m.mountPath
}

// Helper method for testing
func (m *MockTransport) GetHandler(method string) MessageHandler {
	return m.handlers[method]
}

func (m *MockTransport) GetToolsNotified() bool {
	return m.toolsNotify
}

func TestMessageHandler(t *testing.T) {
	t.Run("Should define correct function signature", func(t *testing.T) {
		// Test that MessageHandler can accept any params and return any result with error
		var handler MessageHandler = func(_ any) (any, error) {
			return "test result", nil
		}

		result, err := handler("test params")

		assert.NoError(t, err)
		assert.Equal(t, "test result", result)
	})

	t.Run("Should handle error returns", func(t *testing.T) {
		var handler MessageHandler = func(_ any) (any, error) {
			return nil, assert.AnError
		}

		result, err := handler(nil)

		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("Should handle various parameter types", func(t *testing.T) {
		var handler MessageHandler = func(params any) (any, error) {
			switch v := params.(type) {
			case string:
				return "string: " + v, nil
			case map[string]any:
				return v, nil
			case nil:
				return "nil params", nil
			default:
				return "unknown type", nil
			}
		}

		// Test string params
		result, err := handler("test")
		assert.NoError(t, err)
		assert.Equal(t, "string: test", result)

		// Test map params
		params := map[string]any{"key": "value"}
		result, err = handler(params)
		assert.NoError(t, err)
		assert.Equal(t, params, result)

		// Test nil params
		result, err = handler(nil)
		assert.NoError(t, err)
		assert.Equal(t, "nil params", result)
	})
}

func TestTransportInterface(t *testing.T) {
	t.Run("Should implement all Transport methods", func(t *testing.T) {
		transport := NewMockTransport("/test")

		// Verify it implements Transport interface
		var _ Transport = transport

		// Test RegisterHandler
		handler := func(params any) (any, error) {
			return "test", nil
		}
		transport.RegisterHandler("test/method", handler)

		registeredHandler := transport.GetHandler("test/method")
		assert.NotNil(t, registeredHandler)

		result, err := registeredHandler(nil)
		assert.NoError(t, err)
		assert.Equal(t, "test", result)
	})

	t.Run("Should track mount path", func(t *testing.T) {
		transport := NewMockTransport("/custom/path")

		assert.Equal(t, "/custom/path", transport.MountPath())
	})

	t.Run("Should handle tools notification", func(t *testing.T) {
		transport := NewMockTransport("/test")

		assert.False(t, transport.GetToolsNotified())

		transport.NotifyToolsChanged()

		assert.True(t, transport.GetToolsNotified())
	})

	t.Run("Should register multiple handlers", func(t *testing.T) {
		transport := NewMockTransport("/test")

		handler1 := func(params any) (any, error) { return "handler1", nil }
		handler2 := func(params any) (any, error) { return "handler2", nil }

		transport.RegisterHandler("method1", handler1)
		transport.RegisterHandler("method2", handler2)

		result1, err := transport.GetHandler("method1")(nil)
		assert.NoError(t, err)
		assert.Equal(t, "handler1", result1)

		result2, err := transport.GetHandler("method2")(nil)
		assert.NoError(t, err)
		assert.Equal(t, "handler2", result2)
	})

	t.Run("Should overwrite handler when registered twice", func(t *testing.T) {
		transport := NewMockTransport("/test")

		handler1 := func(params any) (any, error) { return "first", nil }
		handler2 := func(params any) (any, error) { return "second", nil }

		transport.RegisterHandler("same/method", handler1)
		transport.RegisterHandler("same/method", handler2)

		result, err := transport.GetHandler("same/method")(nil)
		assert.NoError(t, err)
		assert.Equal(t, "second", result)
	})
}

func TestTransportMethods(t *testing.T) {
	t.Run("Should handle connection gracefully", func(t *testing.T) {
		transport := NewMockTransport("/test")

		// Mock echo context would be needed for real test
		// For now, just verify the method exists and can be called
		err := transport.HandleConnection(nil)
		assert.NoError(t, err)
	})

	t.Run("Should handle message gracefully", func(t *testing.T) {
		transport := NewMockTransport("/test")

		// Mock echo context would be needed for real test
		// For now, just verify the method exists and can be called
		err := transport.HandleMessage(nil)
		assert.NoError(t, err)
	})
}

func TestTransportUseCases(t *testing.T) {
	t.Run("Should support MCP protocol methods", func(t *testing.T) {
		transport := NewMockTransport("/mcp")

		// Register typical MCP handlers
		transport.RegisterHandler("initialize", func(params any) (any, error) {
			return map[string]any{"capabilities": map[string]any{}}, nil
		})

		transport.RegisterHandler("tools/list", func(params any) (any, error) {
			return map[string]any{"tools": []any{}}, nil
		})

		transport.RegisterHandler("tools/call", func(params any) (any, error) {
			return map[string]any{"content": []any{}}, nil
		})

		// Test initialize
		result, err := transport.GetHandler("initialize")(nil)
		assert.NoError(t, err)
		assert.Contains(t, result.(map[string]any), "capabilities")

		// Test tools/list
		result, err = transport.GetHandler("tools/list")(nil)
		assert.NoError(t, err)
		assert.Contains(t, result.(map[string]any), "tools")

		// Test tools/call
		result, err = transport.GetHandler("tools/call")(nil)
		assert.NoError(t, err)
		assert.Contains(t, result.(map[string]any), "content")
	})

	t.Run("Should handle handler not found case", func(t *testing.T) {
		transport := NewMockTransport("/test")

		handler := transport.GetHandler("nonexistent/method")

		assert.Nil(t, handler)
	})

	t.Run("Should support custom mount paths", func(t *testing.T) {
		testCases := []string{
			"/mcp",
			"/api/mcp",
			"/custom/path",
			"/v1/mcp",
		}

		for _, path := range testCases {
			transport := NewMockTransport(path)
			assert.Equal(t, path, transport.MountPath())
		}
	})
}
