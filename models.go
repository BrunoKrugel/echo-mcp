package server

import "github.com/BrunoKrugel/echo-mcp/pkg/types"

type InitializeResponse struct {
	Capabilities    *Capabilities `json:"capabilities"`
	ServerInfo      *ServerInfo   `json:"serverInfo"`
	ProtocolVersion string        `json:"protocolVersion"`
}

type Capabilities struct {
	Tools map[string]any `json:"tools"`
}

type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type ToolsListResponse struct {
	Tools []types.Tool `json:"tools"`
}

type ToolCallRequest struct {
	Arguments map[string]any `json:"arguments"`
	Name      string         `json:"name"`
}

type ToolCallResponse struct {
	Content []Content `json:"content"`
}

type Content struct {
	Type string `json:"type"`
	Text string `json:"text"`
}
