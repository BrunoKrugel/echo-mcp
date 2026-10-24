package transport

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v4"
)

// MessageHandler defines the function signature for handling MCP messages
type MessageHandler func(params any) (any, error)

// ContextMessageHandler handles an MCP message with the context of the HTTP
// request that carried it. Use RequestFromContext to reach that request.
type ContextMessageHandler func(ctx context.Context, params any) (any, error)

type requestContextKey struct{}

// ContextWithRequest returns a context that carries the incoming MCP HTTP request.
func ContextWithRequest(ctx context.Context, r *http.Request) context.Context {
	return context.WithValue(ctx, requestContextKey{}, r)
}

// RequestFromContext returns the incoming MCP HTTP request, if ctx carries one.
func RequestFromContext(ctx context.Context) (*http.Request, bool) {
	r, ok := ctx.Value(requestContextKey{}).(*http.Request)
	return r, ok && r != nil
}

// Transport defines the interface for MCP transport mechanisms
type Transport interface {
	// RegisterHandler registers a message handler for a specific method
	RegisterHandler(method string, handler MessageHandler)

	// HandleConnection handles incoming MCP connections
	HandleConnection(c echo.Context) error

	// HandleMessage processes incoming MCP messages
	HandleMessage(c echo.Context) error

	// NotifyToolsChanged sends a notification that tools have changed
	NotifyToolsChanged()

	// MountPath returns the path where this transport is mounted
	MountPath() string
}
