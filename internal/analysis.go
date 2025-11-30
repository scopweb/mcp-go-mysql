package internal

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
)

// AnalysisArgs argumentos para análisis avanzados
type AnalysisArgs struct {
	Target string `json:"target"`
	Type   string `json:"type,omitempty"`
	DSN    string `json:"dsn,omitempty"`
}

// ExplainArgs argumentos para análisis de consultas
type ExplainArgs struct {
	Query string `json:"query"`
	Type  string `json:"type,omitempty"`
	DSN   string `json:"dsn,omitempty"`
}

// OptimizeArgs argumentos para optimización
type OptimizeArgs struct {
	Tables []string `json:"tables"`
	DSN    string   `json:"dsn,omitempty"`
}

// ExplainQuery analiza plan de ejecución de consultas
func (c *Client) ExplainQuery(args ExplainArgs) (string, error) {
	db, err := c.getDB()
	if err != nil {
		return "", fmt.Errorf("conexión DB: %w", err)
	}
	defer db.Close()

	// Validar que sea SELECT
	queryLower := strings.ToLower(strings.TrimSpace(args.Query))
	if !strings.HasPrefix(queryLower, "select") {
		return "", fmt.Errorf("solo consultas SELECT permitidas")
	}

	if args.Type == "" {
		args.Type = "simple"
	}

	var explainQuery string
	switch args.Type {
	case "extended":
		explainQuery = "EXPLAIN EXTENDED " + args.Query
	case "json":
		explainQuery = "EXPLAIN FORMAT=JSON " + args.Query
	default:
		explainQuery = "EXPLAIN " + args.Query
	}

	rows, err := db.Query(explainQuery)
	if err != nil {
		return "", fmt.Errorf("error EXPLAIN: %w", err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return "", fmt.Errorf("error columnas: %w", err)
	}

	var result []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(cols))
		pointers := make([]interface{}, len(cols))
		for i := range values {
			pointers[i] = &values[i]
		}

		if err := rows.Scan(pointers...); err != nil {
			return "", fmt.Errorf("error scan: %w", err)
		}

		row := make(map[string]interface{})
		for i, col := range cols {
			if values[i] != nil {
				switch v := values[i].(type) {
				case []byte:
					row[col] = string(v)
				default:
					row[col] = v
				}
			} else {
				row[col] = nil
			}
		}
		result = append(result, row)
	}

	// Agregar análisis de performance si es explain simple
	if args.Type == "simple" && len(result) > 0 {
		analysis := c.analyzeExplainResult(result)
		response := map[string]interface{}{
			"explain_result":       result,
			"performance_analysis": analysis,
		}

		jsonData, err := json.Marshal(response)
		if err != nil {
			return "", fmt.Errorf("error JSON: %w", err)
		}
		return string(jsonData), nil
	}

	jsonData, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("error JSON: %w", err)
	}

	return string(jsonData), nil
}

// AnalyzeObject analiza tablas, vistas o toda la base de datos
func (c *Client) AnalyzeObject(args AnalysisArgs) (string, error) {
	db, err := c.getDB()
	if err != nil {
		return "", fmt.Errorf("conexión DB: %w", err)
	}
	defer db.Close()

	if args.Type == "" {
		args.Type = "structure"
	}

	var result map[string]interface{}

	switch args.Type {
	case "performance":
		result = c.getBasicPerformance(db, args.Target)
	case "dependencies":
		result = c.getBasicDependencies(db, args.Target)
	case "usage":
		result = c.getBasicUsage(db, args.Target)
	default: // structure
		result = c.getBasicStructure(db, args.Target)
	}

	jsonData, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("error JSON: %w", err)
	}

	return string(jsonData), nil
}

// OptimizeTables optimiza tablas específicas
func (c *Client) OptimizeTables(args OptimizeArgs) (string, error) {
	db, err := c.getDB()
	if err != nil {
		return "", fmt.Errorf("conexión DB: %w", err)
	}
	defer db.Close()

	type OptimizeResult struct {
		Table       string `json:"table"`
		Operation   string `json:"operation"`
		MessageType string `json:"msg_type"`
		MessageText string `json:"msg_text"`
	}

	var results []OptimizeResult

	for _, table := range args.Tables {
		query := fmt.Sprintf("OPTIMIZE TABLE `%s`", table)
		rows, err := db.Query(query)
		if err != nil {
			results = append(results, OptimizeResult{
				Table:       table,
				Operation:   "optimize",
				MessageType: "error",
				MessageText: err.Error(),
			})
			continue
		}

		for rows.Next() {
			var result OptimizeResult
			rows.Scan(&result.Table, &result.Operation, &result.MessageType, &result.MessageText)
			results = append(results, result)
		}
		rows.Close()
	}

	jsonData, err := json.Marshal(results)
	if err != nil {
		return "", fmt.Errorf("error JSON: %w", err)
	}

	return string(jsonData), nil
}

// ShowProcessList muestra procesos activos de MySQL
func (c *Client) ShowProcessList(args DBArgs) (string, error) {
	db, err := c.getDB()
	if err != nil {
		return "", fmt.Errorf("conexión DB: %w", err)
	}
	defer db.Close()

	rows, err := db.Query("SHOW PROCESSLIST")
	if err != nil {
		return "", fmt.Errorf("error SHOW PROCESSLIST: %w", err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return "", fmt.Errorf("error columnas: %w", err)
	}

	var processes []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(cols))
		pointers := make([]interface{}, len(cols))
		for i := range values {
			pointers[i] = &values[i]
		}

		if err := rows.Scan(pointers...); err != nil {
			continue
		}

		row := make(map[string]interface{})
		for i, col := range cols {
			if values[i] != nil {
				switch v := values[i].(type) {
				case []byte:
					row[col] = string(v)
				default:
					row[col] = v
				}
			} else {
				row[col] = nil
			}
		}
		processes = append(processes, row)
	}

	jsonData, err := json.Marshal(processes)
	if err != nil {
		return "", fmt.Errorf("error JSON: %w", err)
	}

	return string(jsonData), nil
}

// Funciones auxiliares para análisis

func (c *Client) analyzeExplainResult(result []map[string]interface{}) map[string]interface{} {
	analysis := make(map[string]interface{})

	var warnings []string
	var suggestions []string
	totalRows := int64(0)

	for _, row := range result {
		// Analizar tipo de select
		var selectType string
		if v, ok := row["select_type"].(string); ok {
			selectType = v
		}
		if v, ok := row["SELECT_TYPE"].(string); ok && selectType == "" {
			selectType = v
		}
		if selectType != "" {
			if selectType == "DEPENDENT SUBQUERY" {
				warnings = append(warnings, "Contiene subconsultas dependientes que pueden ser lentas")
			}
		}

		// Analizar tipo de acceso
		var accessType string
		if v, ok := row["type"].(string); ok {
			accessType = v
		}
		if v, ok := row["TYPE"].(string); ok && accessType == "" {
			accessType = v
		}
		if accessType != "" {
			switch accessType {
			case "ALL":
				warnings = append(warnings, "Full table scan detectado")
				suggestions = append(suggestions, "Considerar agregar índices apropiados")
			case "index":
				suggestions = append(suggestions, "Usando índice, pero podría optimizarse")
			case "range":
				// Acceptable
			case "ref", "eq_ref", "const":
				// Good
			}
		}

		// Contar filas examinadas
		if rowsVal, ok := row["rows"]; ok {
			switch v := rowsVal.(type) {
			case int64:
				totalRows += v
			case int32:
				totalRows += int64(v)
			case float64:
				totalRows += int64(v)
			case string:
				// intentar parsear
				// ignoramos error silenciosamente
			}
		}

		// Verificar uso de índices
		if _, hasKey := row["key"]; hasKey {
			if row["key"] == nil || row["key"] == "" {
				warnings = append(warnings, "No se están usando índices")
			}
		}

		// Verificar Extra
		var extra string
		if v, ok := row["Extra"].(string); ok {
			extra = v
		}
		if v, ok := row["extra"].(string); ok && extra == "" {
			extra = v
		}
		if extra != "" {
			if strings.Contains(extra, "Using filesort") {
				warnings = append(warnings, "Requiere ordenamiento en disco (filesort)")
				suggestions = append(suggestions, "Considerar índice que cubra ORDER BY")
			}
			if strings.Contains(extra, "Using temporary") {
				warnings = append(warnings, "Requiere tabla temporal")
				suggestions = append(suggestions, "Optimizar GROUP BY o DISTINCT")
			}
		}
	}

	// Evaluar performance general
	var performance string
	if len(warnings) == 0 {
		performance = "Buena"
	} else if len(warnings) <= 2 {
		performance = "Aceptable"
	} else {
		performance = "Requiere optimización"
	}

	analysis["performance_rating"] = performance
	analysis["total_rows_examined"] = totalRows
	analysis["warnings"] = warnings
	analysis["suggestions"] = suggestions

	return analysis
}

func (c *Client) getBasicPerformance(db *sql.DB, target string) map[string]interface{} {
	result := map[string]interface{}{
		"analysis_type": "performance",
		"target":        target,
	}

	if target == "database" {
		var uptime sql.NullString
		db.QueryRow("SHOW STATUS LIKE 'Uptime'").Scan(&uptime, &uptime)
		if uptime.Valid {
			result["uptime"] = uptime.String
		}
	} else {
		query := `
			SELECT TABLE_ROWS, DATA_LENGTH, INDEX_LENGTH
			FROM INFORMATION_SCHEMA.TABLES 
			WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?`

		var tableRows, dataLength, indexLength sql.NullInt64
		db.QueryRow(query, target).Scan(&tableRows, &dataLength, &indexLength)

		result["estimated_rows"] = tableRows.Int64
		result["data_size_mb"] = float64(dataLength.Int64) / 1024 / 1024
		result["index_size_mb"] = float64(indexLength.Int64) / 1024 / 1024
	}

	return result
}

func (c *Client) getBasicDependencies(db *sql.DB, target string) map[string]interface{} {
	result := map[string]interface{}{
		"analysis_type": "dependencies",
		"target":        target,
	}

	query := `
		SELECT TABLE_NAME, REFERENCED_TABLE_NAME
		FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE 
		WHERE TABLE_SCHEMA = DATABASE() 
		AND REFERENCED_TABLE_NAME IS NOT NULL`

	if target != "database" {
		query += " AND (TABLE_NAME = ? OR REFERENCED_TABLE_NAME = ?)"
	}

	var rows *sql.Rows
	var err error

	if target != "database" {
		rows, err = db.Query(query, target, target)
	} else {
		rows, err = db.Query(query)
	}

	var dependencies []string
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var tableName, refTable string
			rows.Scan(&tableName, &refTable)
			dependencies = append(dependencies, fmt.Sprintf("%s -> %s", tableName, refTable))
		}
	}

	result["dependencies"] = dependencies
	result["count"] = len(dependencies)

	return result
}

func (c *Client) getBasicUsage(db *sql.DB, target string) map[string]interface{} {
	result := map[string]interface{}{
		"analysis_type": "usage",
		"target":        target,
	}

	if target == "database" {
		var dbSize sql.NullFloat64
		db.QueryRow(`
			SELECT ROUND(SUM(data_length + index_length) / 1024 / 1024, 2)
			FROM information_schema.tables 
			WHERE table_schema = DATABASE()`).Scan(&dbSize)
		result["total_size_mb"] = dbSize.Float64
	} else {
		query := `
			SELECT TABLE_ROWS, DATA_LENGTH
			FROM INFORMATION_SCHEMA.TABLES 
			WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?`

		var tableRows, dataLength sql.NullInt64
		db.QueryRow(query, target).Scan(&tableRows, &dataLength)
		result["rows"] = tableRows.Int64
		result["size_mb"] = float64(dataLength.Int64) / 1024 / 1024
	}

	return result
}

func (c *Client) getBasicStructure(db *sql.DB, target string) map[string]interface{} {
	result := map[string]interface{}{
		"analysis_type": "structure",
		"target":        target,
	}

	if target == "database" {
		var tableCount, viewCount int
		db.QueryRow("SELECT COUNT(*) FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_SCHEMA = DATABASE() AND TABLE_TYPE = 'BASE TABLE'").Scan(&tableCount)
		db.QueryRow("SELECT COUNT(*) FROM INFORMATION_SCHEMA.VIEWS WHERE TABLE_SCHEMA = DATABASE()").Scan(&viewCount)
		result["total_tables"] = tableCount
		result["total_views"] = viewCount
	} else {
		var columnCount int
		db.QueryRow("SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?", target).Scan(&columnCount)
		result["total_columns"] = columnCount
	}

	return result
}
