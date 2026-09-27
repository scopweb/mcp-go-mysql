package main

// MCPMessage is one JSON-RPC 2.0 message on the MCP stdio channel.
type MCPMessage struct {
	JSONRpc string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Method  string      `json:"method,omitempty"`
	Params  interface{} `json:"params,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *MCPError   `json:"error,omitempty"`
}

// MCPError is a JSON-RPC protocol error. Tool failures do not use this type;
// they are returned as a ToolResponse with IsError set.
type MCPError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// ToolResponse is the MCP tools/call result. IsError marks a tool failure
// without turning it into a protocol error.
type ToolResponse struct {
	Content []ContentItem `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

// ContentItem is one text block inside a tool result.
type ContentItem struct {
	Type string `json:"type"`
	Text string `json:"text"`
}
