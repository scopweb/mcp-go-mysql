package main

import (
	"log"
	mysql "mcp-gp-mysql/internal"
)

func handleMessage(client *mysql.Client, msg *MCPMessage) *MCPMessage {
	log.Printf("Manejando método: %s", msg.Method)

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
			JSONRpc: "2.0",
			ID:      msg.ID,
			Result: map[string]interface{}{
				"protocolVersion": clientVersion, // Echo client's version for compatibility
				"capabilities": map[string]interface{}{
					"tools": map[string]interface{}{
						"listChanged": true,
					},
				},
				"serverInfo": map[string]interface{}{
					"name":    "mysql-mcp-advanced",
					"version": "2.0.2",
				},
			},
		}

	case "ping":
		log.Println("-> ping")
		return &MCPMessage{
			JSONRpc: "2.0",
			ID:      msg.ID,
			Result:  map[string]interface{}{},
		}

	case "tools/list":
		log.Println("-> tools/list")
		return &MCPMessage{
			JSONRpc: "2.0",
			ID:      msg.ID,
			Result: map[string]interface{}{
				"tools": getToolsList(),
			},
		}
		
	case "tools/call":
		log.Println("-> tools/call")
		return handleToolCall(client, msg)
		
	case "notifications/initialized":
		log.Println("-> notifications/initialized (ignorado)")
		return nil // No response for notifications
		
	default:
		log.Printf("Método desconocido: %s", msg.Method)
		return &MCPMessage{
			JSONRpc: "2.0",
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
		log.Printf("Parámetros inválidos: %+v", msg.Params)
		return &MCPMessage{
			JSONRpc: "2.0",
			ID:      msg.ID,
			Error: &MCPError{
				Code:    -32602,
				Message: "Invalid params",
			},
		}
	}
	
	toolName, ok := params["name"].(string)
	if !ok {
		log.Printf("Nombre de herramienta faltante")
		return &MCPMessage{
			JSONRpc: "2.0",
			ID:      msg.ID,
			Error: &MCPError{
				Code:    -32602,
				Message: "Missing tool name",
			},
		}
	}
	
	arguments, _ := params["arguments"].(map[string]interface{})
	log.Printf("Ejecutando: %s con args: %+v", toolName, arguments)
	
	result, err := callClientMethod(client, toolName, arguments)

	if err != nil {
		log.Printf("Error en %s: %v", toolName, err)
		// MCP spec: Tool execution errors should use isError: true in result,
		// NOT JSON-RPC protocol errors. Protocol errors are for transport/parsing issues.
		sanitizedErr := client.SanitizeError(err)
		return &MCPMessage{
			JSONRpc: "2.0",
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

	log.Printf("Herramienta %s ejecutada OK", toolName)
	return &MCPMessage{
		JSONRpc: "2.0",
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