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
		return &MCPMessage{
			JSONRpc: "2.0",
			ID:      msg.ID,
			Result: map[string]interface{}{
				"protocolVersion": "2024-11-05",
				"capabilities": map[string]interface{}{
					"tools": map[string]interface{}{},
				},
				"serverInfo": map[string]interface{}{
					"name":    "mysql-mcp-advanced",
					"version": "1.3.0",
				},
			},
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
		return &MCPMessage{
			JSONRpc: "2.0",
			ID:      msg.ID,
			Error: &MCPError{
				Code:    -32603,
				Message: err.Error(),
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