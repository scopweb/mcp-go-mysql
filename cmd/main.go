package main

import (
	"bufio"
	"encoding/json"
	"log"
	"os"
	"strconv"
	"strings"

	mysql "mcp-gp-mysql/internal"
)

// Valores por defecto; se pueden sobreescribir por variables de entorno
var (
	SAFETY_KEY    = getEnvDefault("SAFETY_KEY", "PRODUCTION_CONFIRMED_2025")
	MAX_SAFE_ROWS = getEnvIntDefault("MAX_SAFE_ROWS", 100)
)

func main() {
	// Cargar variables de entorno desde .env si no están configuradas
	loadEnvFile()

	// Configurar logging
	setupLogging()

	log.Println("=== Iniciando Servidor MCP MySQL v1.3 ===")

	// Mostrar configuración
	config := getConfiguration()
	log.Printf("Configuración: %+v", config)

	// Crear cliente MySQL
	client := mysql.NewClient()
	log.Println("Cliente MySQL creado")

	// Probar conexión
	if err := testConnection(client); err != nil {
		log.Printf("ADVERTENCIA: No se puede conectar a MySQL: %v", err)
		log.Println("Continuando... Las herramientas fallarán hasta que se configure correctamente")
	} else {
		log.Println("Conexión a MySQL exitosa")
	}

	log.Println("Iniciando procesamiento de mensajes...")

	// Procesamiento de mensajes MCP
	scanner := bufio.NewScanner(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)

	messageCount := 0

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Ignorar líneas vacías
		if line == "" {
			continue
		}

		messageCount++
		log.Printf("Mensaje #%d: %s", messageCount, line)

		var msg MCPMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			log.Printf("Error parsing JSON: %v", err)
			// Enviar error de parsing
			errorResponse := &MCPMessage{
				JSONRpc: "2.0",
				Error: &MCPError{
					Code:    -32700,
					Message: "Parse error",
					Data:    err.Error(),
				},
			}
			if encErr := encoder.Encode(errorResponse); encErr != nil {
				log.Printf("Error enviando respuesta de error: %v", encErr)
			}
			continue
		}

		// Asegurar versión JSON-RPC
		if msg.JSONRpc == "" {
			msg.JSONRpc = "2.0"
		}

		log.Printf("Método: %s, ID: %v", msg.Method, msg.ID)

		response := handleMessage(client, &msg)
		if response != nil {
			log.Printf("Enviando respuesta #%d", messageCount)
			if err := encoder.Encode(response); err != nil {
				log.Printf("Error enviando respuesta: %v", err)
			} else {
				log.Printf("Respuesta enviada OK")
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error del scanner: %v", err)
	}

	log.Println("=== Servidor terminado ===")
}

// loadEnvFile carga variables de entorno desde .env solo si no están configuradas
func loadEnvFile() {
	// Verificar si las variables críticas ya están configuradas
	if os.Getenv("MYSQL_HOST") != "" && os.Getenv("MYSQL_USER") != "" {
		log.Println("Variables de entorno ya configuradas, omitiendo .env")
		return
	}

	file, err := os.Open(".env")
	if err != nil {
		log.Printf("No se encontró .env: %v", err)
		return
	}
	defer file.Close()

	log.Println("Cargando configuración desde .env")
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			// Solo configurar si no existe
			if os.Getenv(key) == "" {
				os.Setenv(key, value)
				log.Printf("Cargada variable desde .env: %s", key)
			}
		}
	}
}

func setupLogging() {
	logPath := os.Getenv("LOG_PATH")
	if logPath == "" {
		logPath = "mysql-mcp.log"
	}

	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.SetOutput(os.Stderr)
		log.Printf("No se pudo crear archivo de log: %v", err)
		return
	}

	log.SetOutput(logFile)
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func getConfiguration() map[string]string {
	return map[string]string{
		"MYSQL_HOST":     os.Getenv("MYSQL_HOST"),
		"MYSQL_PORT":     os.Getenv("MYSQL_PORT"),
		"MYSQL_USER":     os.Getenv("MYSQL_USER"),
		"MYSQL_PASSWORD": "***", // No mostrar la contraseña en logs
		"MYSQL_DATABASE": os.Getenv("MYSQL_DATABASE"),
		"LOG_PATH":       os.Getenv("LOG_PATH"),
		"MAX_SAFE_ROWS":  os.Getenv("MAX_SAFE_ROWS"),
	}
}

func testConnection(client *mysql.Client) error {
	_, err := client.ListTablesSimple()
	return err
}

// Utilidades de entorno locales al paquete main
func getEnvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvIntDefault(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		// evitar dependencia de strconv en muchos sitios; conversión simple
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
