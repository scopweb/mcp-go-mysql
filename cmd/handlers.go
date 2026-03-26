package main

import (
	"log"
	mysql "mcp-gp-mysql/internal"
)

func handleMessage(client *mysql.Client, msg *MCPMessage) *MCPMessage {
	log.Printf("Handling method: %s", msg.Method)

	switch msg.Method {
	case "initialize":
		log.Println("-> initialize")

		// Extract client's protocol version for auto-detection (Claude Desktop compatibility)
		clientVersion := "2024-11-05" // Safe fallback
		if params, ok := msg.Params.(map[string]interface{}); ok {
			if v, ok := params["protocolVersion"].(string); ok && v != "" {
				clientVersion = v
			}
		}
		log.Printf("Client protocol version: %s (echoing back)", clientVersion)

		return &MCPMessage{
			JSONRpc: JSONRPCVer,
			ID:      msg.ID,
			Result: map[string]interface{}{
				"protocolVersion": clientVersion, // Echo client's version for compatibility
				"capabilities": map[string]interface{}{
					"tools": map[string]interface{}{
						"listChanged": false,
					},
				},
				"serverInfo": map[string]interface{}{
					"name":    ServerName,
					"version": Version,
				},
				"instructions": "MySQL database MCP server with security hardening. Available tools:\n" +
					"- query: Execute read-only SELECT, WITH (CTE), and SHOW queries\n" +
					"- execute: Run INSERT/UPDATE/DELETE statements (requires confirm_key for operations affecting >100 rows)\n" +
					"- tables: List all tables with metadata (type, engine, row count)\n" +
					"- describe: Show table structure (columns, types, keys, constraints)\n" +
					"- views: List all database views\n" +
					"- indexes: Show indexes and cardinality for a table\n" +
					"- explain: Get EXPLAIN execution plan (SELECT queries only)\n" +
					"- count: Count rows in a table with optional WHERE condition\n" +
					"- sample: Get sample rows from a table (default 10, max 100)\n" +
					"- database_info: Get server version, user, hostname, port, and database name\n\n" +
					"Workflow: Use 'tables' and 'describe' to explore the schema before writing queries. " +
					"Use 'query' for all read operations. Use 'explain' to optimize slow queries. " +
					"Use 'execute' only for data modifications. " +
					"SQL injection patterns and dangerous operations (DROP DATABASE, TRUNCATE, etc.) are blocked.",
			},
		}

	case "ping":
		log.Println("-> ping")
		return &MCPMessage{
			JSONRpc: JSONRPCVer,
			ID:      msg.ID,
			Result:  map[string]interface{}{},
		}

	case "tools/list":
		log.Println("-> tools/list")
		return &MCPMessage{
			JSONRpc: JSONRPCVer,
			ID:      msg.ID,
			Result: map[string]interface{}{
				"tools": getToolsList(),
			},
		}
		
	case "tools/call":
		log.Println("-> tools/call")
		return handleToolCall(client, msg)
		
	case "notifications/initialized":
		log.Println("-> notifications/initialized (ignored)")
		return nil // No response for notifications

	default:
		log.Printf("Unknown method: %s", msg.Method)
		return &MCPMessage{
			JSONRpc: JSONRPCVer,
			ID:      msg.ID,
			Error: &MCPError{
				Code:    -32601,
				Message: "Method not found",
			},
		}
	}
}

func handleToolCall(client *mysql.Client, msg *MCPMessage) *MCPMessage {
	params, ok := msg.Params.(map[string]interface{})
	if !ok {
		log.Printf("Invalid params: %+v", msg.Params)
		return &MCPMessage{
			JSONRpc: JSONRPCVer,
			ID:      msg.ID,
			Error: &MCPError{
				Code:    -32602,
				Message: "Invalid params",
			},
		}
	}
	
	toolName, ok := params["name"].(string)
	if !ok {
		log.Printf("Missing tool name")
		return &MCPMessage{
			JSONRpc: JSONRPCVer,
			ID:      msg.ID,
			Error: &MCPError{
				Code:    -32602,
				Message: "Missing tool name",
			},
		}
	}
	
	arguments, ok := params["arguments"].(map[string]interface{})
	if !ok {
		arguments = make(map[string]interface{}) // Empty map if not provided
	}
	log.Printf("Executing tool: %s with args: %+v", toolName, arguments)
	
	result, err := callClientMethod(client, toolName, arguments)

	if err != nil {
		log.Printf("Error in %s: %v", toolName, err)
		// MCP spec: Tool execution errors should use isError: true in result,
		// NOT JSON-RPC protocol errors. Protocol errors are for transport/parsing issues.
		sanitizedErr := client.SanitizeError(err)
		return &MCPMessage{
			JSONRpc: JSONRPCVer,
			ID:      msg.ID,
			Result: ToolResponse{
				Content: []ContentItem{
					{
						Type: "text",
						Text: sanitizedErr.Message,
					},
				},
				IsError: true,
			},
		}
	}

	log.Printf("Tool %s executed successfully", toolName)
	return &MCPMessage{
		JSONRpc: JSONRPCVer,
		ID:      msg.ID,
		Result: ToolResponse{
			Content: []ContentItem{
				{
					Type: "text",
					Text: result,
				},
			},
		},
	}
}