package mysql

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

// Client cliente MySQL simplificado
type Client struct {
	defaultDSN string
}

// NewClient crea una nueva instancia del cliente MySQL
func NewClient() *Client {
	dsn := buildDSNFromEnv()
	return &Client{defaultDSN: dsn}
}

// buildDSNFromEnv construye DSN desde variables de entorno
func buildDSNFromEnv() string {
	host := getEnv("MYSQL_HOST", "localhost")
	port := getEnv("MYSQL_PORT", "3306")
	user := getEnv("MYSQL_USER", "root")
	password := getEnv("MYSQL_PASSWORD", "")
	database := getEnv("MYSQL_DATABASE", "")

	var dsn string
	if password != "" {
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&allowNativePasswords=true",
			user, password, host, port, database)
	} else {
		dsn = fmt.Sprintf("%s@tcp(%s:%s)/%s?parseTime=true&allowNativePasswords=true",
			user, host, port, database)
	}

	return dsn
}

// getEnv obtiene variable de entorno con valor por defecto
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getDB obtiene conexión a la base de datos
func (c *Client) getDB() (*sql.DB, error) {
	db, err := sql.Open("mysql", c.defaultDSN)
	if err != nil {
		return nil, err
	}
	// Configurar pool de conexiones (valores conservadores por defecto)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(0) // sin límite; ajustar por entorno si se desea
	return db, nil
}

// ExecuteQuerySimple ejecuta consulta SELECT simple
func (c *Client) ExecuteQuerySimple(query string) (string, error) {
	db, err := c.getDB()
	if err != nil {
		return "", fmt.Errorf("conexión DB: %w", err)
	}
	defer db.Close()

	// Validar que sea SELECT
	queryLower := strings.ToLower(strings.TrimSpace(query))
	if !strings.HasPrefix(queryLower, "select") {
		return "", fmt.Errorf("solo consultas SELECT permitidas")
	}

	rows, err := db.Query(query)
	if err != nil {
		return "", fmt.Errorf("error consulta: %w", err)
	}
	defer rows.Close()

	// Obtener columnas
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

	jsonData, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("error JSON: %w", err)
	}

	return string(jsonData), nil
}

// ListTablesSimple lista todas las tablas
func (c *Client) ListTablesSimple() (string, error) {
	db, err := c.getDB()
	if err != nil {
		return "", fmt.Errorf("conexión DB: %w", err)
	}
	defer db.Close()

	rows, err := db.Query("SHOW TABLES")
	if err != nil {
		return "", fmt.Errorf("error SHOW TABLES: %w", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			return "", fmt.Errorf("error scan tabla: %w", err)
		}
		tables = append(tables, table)
	}

	if len(tables) == 0 {
		return "No hay tablas", nil
	}

	return strings.Join(tables, "\n"), nil
}

// ListViewsSimple lista todas las vistas
func (c *Client) ListViewsSimple() (string, error) {
	db, err := c.getDB()
	if err != nil {
		return "", fmt.Errorf("conexión DB: %w", err)
	}
	defer db.Close()

	query := `
		SELECT TABLE_NAME 
		FROM INFORMATION_SCHEMA.VIEWS 
		WHERE TABLE_SCHEMA = DATABASE()
		ORDER BY TABLE_NAME`

	rows, err := db.Query(query)
	if err != nil {
		return "", fmt.Errorf("error listando vistas: %w", err)
	}
	defer rows.Close()

	var views []string
	for rows.Next() {
		var view string
		if err := rows.Scan(&view); err != nil {
			return "", fmt.Errorf("error scan vista: %w", err)
		}
		views = append(views, view)
	}

	if len(views) == 0 {
		return "No hay vistas", nil
	}

	return strings.Join(views, "\n"), nil
}

// DescribeSimple describe tabla o vista
func (c *Client) DescribeSimple(name string) (string, error) {
	db, err := c.getDB()
	if err != nil {
		return "", fmt.Errorf("conexión DB: %w", err)
	}
	defer db.Close()

	query := fmt.Sprintf("DESCRIBE `%s`", name)
	rows, err := db.Query(query)
	if err != nil {
		return "", fmt.Errorf("error DESCRIBE: %w", err)
	}
	defer rows.Close()

	type ColumnInfo struct {
		Field   string  `json:"field"`
		Type    string  `json:"type"`
		Null    string  `json:"null"`
		Key     string  `json:"key"`
		Default *string `json:"default"`
		Extra   string  `json:"extra"`
	}

	var columns []ColumnInfo
	for rows.Next() {
		var col ColumnInfo
		var defaultVal sql.NullString

		err := rows.Scan(&col.Field, &col.Type, &col.Null, &col.Key, &defaultVal, &col.Extra)
		if err != nil {
			return "", fmt.Errorf("error scan columna: %w", err)
		}

		if defaultVal.Valid {
			col.Default = &defaultVal.String
		}

		columns = append(columns, col)
	}

	jsonData, err := json.Marshal(columns)
	if err != nil {
		return "", fmt.Errorf("error JSON: %w", err)
	}

	return string(jsonData), nil
}

// ViewDefinitionSimple obtiene definición de vista
func (c *Client) ViewDefinitionSimple(name string) (string, error) {
	db, err := c.getDB()
	if err != nil {
		return "", fmt.Errorf("conexión DB: %w", err)
	}
	defer db.Close()

	query := `
		SELECT VIEW_DEFINITION
		FROM INFORMATION_SCHEMA.VIEWS 
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?`

	var definition string
	err = db.QueryRow(query, name).Scan(&definition)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("vista '%s' no encontrada", name)
	}
	if err != nil {
		return "", fmt.Errorf("error obteniendo definición: %w", err)
	}

	return definition, nil
}

// CreateViewSimple crea una vista
func (c *Client) CreateViewSimple(name, query string, replace bool) (string, error) {
	db, err := c.getDB()
	if err != nil {
		return "", fmt.Errorf("conexión DB: %w", err)
	}
	defer db.Close()

	// Validar que el query sea SELECT
	queryLower := strings.ToLower(strings.TrimSpace(query))
	if !strings.HasPrefix(queryLower, "select") {
		return "", fmt.Errorf("solo consultas SELECT permitidas para vistas")
	}

	var sqlCmd string
	if replace {
		sqlCmd = fmt.Sprintf("CREATE OR REPLACE VIEW `%s` AS %s", name, query)
	} else {
		sqlCmd = fmt.Sprintf("CREATE VIEW `%s` AS %s", name, query)
	}

	_, err = db.Exec(sqlCmd)
	if err != nil {
		return "", fmt.Errorf("error creando vista: %w", err)
	}

	return fmt.Sprintf("Vista '%s' creada exitosamente", name), nil
}
