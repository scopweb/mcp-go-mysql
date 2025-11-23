package main

import (
	"database/sql"
	"fmt"
	"log"
	mysql "mcp-gp-mysql/internal"
	"strings"
)

func executeWrite(client *mysql.Client, arguments map[string]interface{}) (string, error) {
	log.Printf("=== INICIO execute_write ===")

	sql, sqlOk := arguments["sql"].(string)
	if !sqlOk {
		return "", fmt.Errorf("parámetro 'sql' requerido")
	}

	// Verificar que sea comando de escritura
	sqlUpper := strings.ToUpper(strings.TrimSpace(stripSQLComments(sql)))
	writeCommands := []string{"INSERT", "UPDATE", "DELETE"}
	isWriteCommand := false
	for _, cmd := range writeCommands {
		if strings.HasPrefix(sqlUpper, cmd) {
			isWriteCommand = true
			break
		}
	}

	if !isWriteCommand {
		return "", fmt.Errorf("usa 'query' para SELECT o 'execute_ddl' para DDL")
	}

	// Estimar filas afectadas (simplificado)
	estimatedRows := estimateAffectedRowsSimple(sql)
	log.Printf("Filas estimadas a afectar: %d", estimatedRows)

	// Si afecta muchas filas, requerir confirmación
	if estimatedRows > MAX_SAFE_ROWS {
		confirmKey, _ := arguments["confirm_key"].(string)
		if confirmKey != SAFETY_KEY {
			return "", fmt.Errorf("operación afectará ~%d filas (>%d). Requiere confirm_key para seguridad",
				estimatedRows, MAX_SAFE_ROWS)
		}
		log.Printf("CONFIRMACIÓN: Operación masiva autorizada (%d filas)", estimatedRows)
	}

	log.Printf("Ejecutando: %s", sql)
	return client.ExecuteWrite(mysql.QueryArgs{SQL: sql})
}

func executeDDL(client *mysql.Client, arguments map[string]interface{}) (string, error) {
	log.Printf("=== INICIO execute_ddl (DDL SIEMPRE REQUIERE CONFIRMACIÓN) ===")

	sql, sqlOk := arguments["sql"].(string)
	confirmKey, keyOk := arguments["confirm_key"].(string)

	if !sqlOk || !keyOk {
		return "", fmt.Errorf("DDL requiere 'sql' y 'confirm_key'")
	}

	// Validar clave
	if confirmKey != SAFETY_KEY {
		return "", fmt.Errorf("clave de confirmación incorrecta - DDL bloqueado por seguridad")
	}

	// Verificar que sea comando DDL
	sqlUpper := strings.ToUpper(strings.TrimSpace(stripSQLComments(sql)))
	ddlCommands := []string{"CREATE", "DROP", "ALTER", "TRUNCATE"}
	isDDL := false
	for _, cmd := range ddlCommands {
		if strings.HasPrefix(sqlUpper, cmd) {
			isDDL = true
			break
		}
	}

	if !isDDL {
		return "", fmt.Errorf("usa 'execute_write' para INSERT/UPDATE/DELETE")
	}

	// Bloquear comandos extremos
	if strings.Contains(sqlUpper, "DROP DATABASE") || strings.Contains(sqlUpper, "DROP SCHEMA") {
		return "", fmt.Errorf("DROP DATABASE/SCHEMA bloqueado por seguridad")
	}

	log.Printf("CONFIRMACIÓN: Ejecutando DDL autorizado: %s", sql)
	return client.ExecuteWrite(mysql.QueryArgs{SQL: sql})
}

// estimateAffectedRowsSimple estima filas de forma simplificada sin conexión DB
func estimateAffectedRowsSimple(sqlQuery string) int {
	// Quitar comentarios y normalizar espacios
	sqlUpper := strings.ToUpper(strings.TrimSpace(stripSQLComments(sqlQuery)))

	// Separar por ; y analizar cada sentencia; si alguna es masiva, forzar confirmación
	statements := strings.Split(sqlUpper, ";")
	if len(statements) == 0 {
		statements = []string{sqlUpper}
	}
	maxEstimate := 1
	for _, st := range statements {
		st = strings.TrimSpace(st)
		if st == "" {
			continue
		}
		// Si no tiene WHERE y es UPDATE/DELETE, asumir masiva
		if (strings.HasPrefix(st, "UPDATE") || strings.HasPrefix(st, "DELETE")) && !strings.Contains(st, " WHERE ") {
			return MAX_SAFE_ROWS + 1
		}
		if strings.HasPrefix(st, "INSERT") {
			// INSERT ... VALUES (...),(...)
			valuesCount := strings.Count(st, "),(")
			estimate := 1 + valuesCount
			if estimate > maxEstimate {
				maxEstimate = estimate
			}
			continue
		}
		// Si tiene IN (...) con muchos elementos, incrementar estimación
		if idx := strings.Index(st, " IN ("); idx >= 0 {
			if j := strings.Index(st[idx+1:], ")"); j > 0 {
				items := strings.Count(st[idx:idx+j+1], ",") + 1
				if items > maxEstimate {
					maxEstimate = items
				}
			}
		}
		// Por defecto, estimación conservadora mínima
		if maxEstimate < 1 {
			maxEstimate = 1
		}
	}
	if maxEstimate <= 0 {
		maxEstimate = 1
	}
	return maxEstimate
}

// estimateAffectedRows estima cuántas filas se verán afectadas por una operación (no usado por ahora)
func estimateAffectedRows(db *sql.DB, sqlQuery string) int {
	sqlUpper := strings.ToUpper(strings.TrimSpace(stripSQLComments(sqlQuery)))

	// Para UPDATE y DELETE, intentar estimar con conteo
	if strings.HasPrefix(sqlUpper, "UPDATE") || strings.HasPrefix(sqlUpper, "DELETE") {
		var countQuery string

		if strings.HasPrefix(sqlUpper, "UPDATE") {
			// UPDATE tabla SET ... WHERE ... -> SELECT COUNT(*) FROM tabla WHERE ...
			parts := strings.Split(sqlQuery, "WHERE")
			if len(parts) > 1 {
				tablePart := strings.Split(parts[0], "SET")[0]
				tableName := strings.TrimSpace(strings.Replace(tablePart, "UPDATE", "", 1))
				countQuery = fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s", tableName, parts[1])
			} else {
				// UPDATE sin WHERE - contar toda la tabla
				tablePart := strings.Split(parts[0], "SET")[0]
				tableName := strings.TrimSpace(strings.Replace(tablePart, "UPDATE", "", 1))
				countQuery = fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)
			}
		} else if strings.HasPrefix(sqlUpper, "DELETE") {
			// DELETE FROM tabla WHERE ... -> SELECT COUNT(*) FROM tabla WHERE ...
			countQuery = strings.Replace(sqlQuery, "DELETE", "SELECT COUNT(*)", 1)
		}

		if countQuery != "" {
			var count int
			err := db.QueryRow(countQuery).Scan(&count)
			if err == nil {
				return count
			}
		}
	}

	// Si no podemos estimar, ser conservadores
	if strings.HasPrefix(sqlUpper, "INSERT") {
		return 1 // INSERT normalmente afecta pocas filas
	}

	return MAX_SAFE_ROWS + 1 // Forzar confirmación si no podemos estimar
}

// stripSQLComments elimina comentarios simples -- y /* */ para evitar falsos positivos
func stripSQLComments(s string) string {
	// Eliminar comentarios de línea
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		if idx := strings.Index(l, "--"); idx >= 0 {
			lines[i] = l[:idx]
		}
		if idx := strings.Index(l, "#"); idx >= 0 { // estilo MySQL
			lines[i] = l[:idx]
		}
	}
	s = strings.Join(lines, "\n")
	// Eliminar comentarios de bloque
	for {
		start := strings.Index(s, "/*")
		if start < 0 {
			break
		}
		end := strings.Index(s[start+2:], "*/")
		if end < 0 {
			break
		}
		s = s[:start] + s[start+2+end+2:]
	}
	// Normalizar espacios múltiples
	s = strings.ReplaceAll(s, "\t", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.Join(strings.Fields(s), " ")
	return s
}
