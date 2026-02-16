package main

// MCPMessage estructura MCP 2.0 compliant
type MCPMessage struct {
	JSONRpc string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Method  string      `json:"method,omitempty"`
	Params  interface{} `json:"params,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *MCPError   `json:"error,omitempty"`
}

// MCPError estructura de error MCP
type MCPError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// ToolResponse respuesta de herramienta (MCP spec compliant)
type ToolResponse struct {
	Content []ContentItem `json:"content"`
	IsError bool          `json:"isError,omitempty"` // MCP spec: tool execution errors use isError, not protocol errors
}

type ContentItem struct {
	Type string `json:"type"`
	Text string `json:"text"`
}