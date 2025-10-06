package transport

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/BrunoKrugel/echo-mcp/pkg/types"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	log "github.com/sirupsen/logrus"
)

type HTTPTransport struct {
	handlers  map[string]MessageHandler
	sessions  map[string]*Session
	mountPath string
	mu        sync.RWMutex
}

type Session struct {
	ID      string
	Created int64
}

// NewHTTPTransport creates a new HTTP transport
func NewHTTPTransport(mountPath string) *HTTPTransport {
	return &HTTPTransport{
		mountPath: mountPath,
		handlers:  make(map[string]MessageHandler),
		sessions:  make(map[string]*Session),
	}
}

// RegisterHandler registers a message handler
func (h *HTTPTransport) RegisterHandler(method string, handler MessageHandler) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.handlers[method] = handler
}

// MountPath returns the mount path
func (h *HTTPTransport) MountPath() string {
	return h.mountPath
}

// HandleConnection handles incoming MCP connections (not used in HTTP transport)
func (h *HTTPTransport) HandleConnection(c echo.Context) error {
	return echo.NewHTTPError(http.StatusMethodNotAllowed, "GET method not supported for HTTP transport")
}

// HandleMessage processes incoming MCP messages via POST
func (h *HTTPTransport) HandleMessage(c echo.Context) error {

	sessionID := c.Request().Header.Get("Mcp-Session-Id")

	var msg types.MCPMessage
	if err := c.Bind(&msg); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid message format")
	}

	if msg.Method == "initialize" {
		return h.handleInitialize(c, &msg)
	}

	if sessionID != "" && !h.isValidSession(sessionID) {
		return echo.NewHTTPError(http.StatusNotFound, "Session not found")
	}

	response := h.processMessage(&msg)

	return c.JSON(http.StatusOK, response)
}

// handleInitialize specifically handles initialize requests
func (h *HTTPTransport) handleInitialize(c echo.Context, msg *types.MCPMessage) error {
	response := h.processMessage(msg)

	sessionID := h.createSession()
	c.Response().Header().Set("Mcp-Session-Id", sessionID)

	return c.JSON(http.StatusOK, response)
}

// processMessage handles an incoming MCP message and returns a response
func (h *HTTPTransport) processMessage(msg *types.MCPMessage) *types.MCPMessage {
	h.mu.RLock()
	handler, exists := h.handlers[msg.Method]
	h.mu.RUnlock()

	response := &types.MCPMessage{
		Jsonrpc: "2.0",
		ID:      msg.ID,
	}

	if !exists {
		response.Error = &types.MCPError{
			Code:    -32601,
			Message: fmt.Sprintf("Method '%s' not found", msg.Method),
		}
		return response
	}

	result, err := handler(msg.Params)
	if err != nil {
		response.Error = &types.MCPError{
			Code:    -32603,
			Message: err.Error(),
		}
	} else {
		response.Result = result
	}

	return response
}

// createSession creates a new session
func (h *HTTPTransport) createSession() string {
	h.mu.Lock()
	defer h.mu.Unlock()

	sessionID := uuid.New().String()
	h.sessions[sessionID] = &Session{
		ID:      sessionID,
		Created: time.Now().Unix(),
	}

	return sessionID
}

// isValidSession checks if a session ID is valid
func (h *HTTPTransport) isValidSession(sessionID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, exists := h.sessions[sessionID]
	return exists
}

// NotifyToolsChanged sends a tools changed notification (not applicable for HTTP transport)
func (h *HTTPTransport) NotifyToolsChanged() {
	log.Debug("[HTTP] NotifyToolsChanged called (no-op for HTTP transport)")
}
