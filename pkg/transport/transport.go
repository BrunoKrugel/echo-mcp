package transport

import "github.com/labstack/echo/v4"

// MessageHandler defines the function signature for handling MCP messages
type MessageHandler func(params any) (any, error)

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
