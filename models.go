package server

import "github.com/BrunoKrugel/echo-mcp/pkg/types"

// InitializeResponse represents the response for MCP initialize requests
type InitializeResponse struct {
	Capabilities    *Capabilities `json:"capabilities"`
	ServerInfo      *ServerInfo   `json:"serverInfo"`
	ProtocolVersion string        `json:"protocolVersion"`
}

// Capabilities represents the capabilities of the MCP server
type Capabilities struct {
	Tools map[string]any `json:"tools"`
}

// ServerInfo represents the server information
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ToolsListResponse represents the response for tools/list requests
type ToolsListResponse struct {
	Tools []types.Tool `json:"tools"`
}

// ToolCallRequest represents the request for tools/call
type ToolCallRequest struct {
	Arguments map[string]any `json:"arguments"`
	Name      string         `json:"name"`
}

// ToolCallResponse represents the response for tools/call requests
type ToolCallResponse struct {
	Content []Content `json:"content"`
}

// Content represents the content structure in tool call responses
type Content struct {
	Type string `json:"type"`
	Text string `json:"text"`
}
