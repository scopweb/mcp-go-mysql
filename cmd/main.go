package main

import (
	"bufio"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	mysql "mcp-gp-mysql/internal"
)

func main() {
	// Load environment variables from .env if not already set
	loadEnvFile()

	// Setup logging and ensure file is closed on exit
	if logFile := setupLogging(); logFile != nil {
		defer logFile.Close()
	}

	log.Printf("=== Starting MCP MySQL Server %s ===", Version)

	// Show configuration
	config := getConfiguration()
	log.Printf("Configuration: %+v", config)

	// Create MySQL client
	client := mysql.NewClient()
	log.Println("MySQL client created")

	// Test connection
	if err := testConnection(client); err != nil {
		log.Printf("WARNING: Cannot connect to MySQL: %v", err)
		log.Println("Continuing... Tools will fail until properly configured")
	} else {
		log.Println("MySQL connection successful")
	}

	log.Println("Starting message processing...")

	// MCP message processing
	scanner := bufio.NewScanner(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)

	messageCount := 0

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Ignore empty lines
		if line == "" {
			continue
		}

		messageCount++
		log.Printf("Message #%d: %s", messageCount, line)

		var msg MCPMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			log.Printf("Error parsing JSON: %v", err)
			// Send parse error response
			errorResponse := &MCPMessage{
				JSONRpc: "2.0",
				Error: &MCPError{
					Code:    -32700,
					Message: "Parse error",
					Data:    err.Error(),
				},
			}
			if encErr := encoder.Encode(errorResponse); encErr != nil {
				log.Printf("Error sending error response: %v", encErr)
			}
			continue
		}

		// Ensure JSON-RPC version
		if msg.JSONRpc == "" {
			msg.JSONRpc = "2.0"
		}

		log.Printf("Method: %s, ID: %v", msg.Method, msg.ID)

		response := handleMessage(client, &msg)
		if response != nil {
			log.Printf("Sending response #%d", messageCount)
			if err := encoder.Encode(response); err != nil {
				log.Printf("Error sending response: %v", err)
			} else {
				log.Printf("Response sent OK")
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Scanner error: %v", err)
	}

	log.Println("=== Server terminated ===")
}

// loadEnvFile loads environment variables from .env only if not already set
func loadEnvFile() {
	// Check if critical variables are already set
	if os.Getenv("MYSQL_HOST") != "" && os.Getenv("MYSQL_USER") != "" {
		log.Println("Environment variables already set, skipping .env")
		return
	}

	file, err := os.Open(".env")
	if err != nil {
		log.Printf(".env file not found: %v", err)
		return
	}
	defer file.Close()

	log.Println("Loading configuration from .env")
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

			// Only set if not already defined
			if os.Getenv(key) == "" {
				os.Setenv(key, value)
				log.Printf("Loaded from .env: %s", key)
			}
		}
	}
}

func setupLogging() *os.File {
	logPath := os.Getenv("LOG_PATH")
	if logPath == "" {
		logPath = "mysql-mcp.log"
	}

	// Validate and sanitize path to prevent path traversal
	logPath = validateLogPath(logPath)

	// Restrictive permissions: 0600 = rw------- (owner only)
	// On Windows: 0600 is ignored, OS handles ACLs
	fileMode := os.FileMode(0600)

	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, fileMode)
	if err != nil {
		log.SetOutput(os.Stderr)
		log.Printf("Could not create log file: %v", err)
		return nil
	}

	log.SetOutput(logFile)
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Printf("Logging started: %s (permissions: %o)", logPath, fileMode)
	return logFile
}

// validateLogPath valida y sanitiza la ruta del archivo de log
// SECURITY FIX FASE 1: Prevenir path traversal
func validateLogPath(logPath string) string {
	// Obtener ruta absoluta
	absPath, err := filepath.Abs(logPath)
	if err != nil {
		// Si falla, usar ruta por defecto
		absPath = "mysql-mcp.log"
	}

	// Limpiar la ruta (remove .., etc)
	cleanPath := filepath.Clean(absPath)

	// Validar que no intente salir del directorio actual
	// Permitir solo rutas que comiencen con:
	// 1. Directorio actual
	// 2. Directorio temp del sistema
	// 3. Directorio de logs estándar
	currentDir, _ := os.Getwd()
	allowedDirs := []string{
		currentDir,
		os.TempDir(),
	}

	// En Unix/Linux, también permitir /var/log
	if runtime.GOOS != "windows" {
		allowedDirs = append(allowedDirs, "/var/log")
	}

	// Validar que la ruta esté dentro de directorios permitidos
	isAllowed := false
	for _, allowed := range allowedDirs {
		allowedAbs, err := filepath.Abs(allowed)
		if err == nil {
			allowedAbs = filepath.Clean(allowedAbs)
			// Verificar si cleanPath está dentro de allowedAbs o es el mismo
			if cleanPath == allowedAbs || strings.HasPrefix(cleanPath, allowedAbs+string(filepath.Separator)) {
				isAllowed = true
				break
			}
		}
	}

	if !isAllowed {
		log.Printf("⚠️ SECURITY: Log path fuera de directorios permitidos: %s. Usando default.", logPath)
		return "mysql-mcp.log"
	}

	return cleanPath
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
