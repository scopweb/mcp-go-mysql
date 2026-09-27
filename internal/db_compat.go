package internal

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
)

// DatabaseType represents the type of database
type DatabaseType string

const (
	DBTypeMySQL   DatabaseType = "mysql"
	DBTypeMariaDB DatabaseType = "mariadb"
)

// DBCompatibilityConfig holds database-specific configuration
type DBCompatibilityConfig struct {
	Type                 DatabaseType
	DisplayName          string
	SupportsSequences    bool   // Oracle-style sequences
	SupportsPLSQL        bool   // Oracle-compatible PL/SQL
	JSONStorageMode      string // "binary" or "text"
	CollationSupport     int    // Number of supported collations
	MaxConnections       int
	DefaultCharset       string
	SupportsBACKUPSTAGE  bool   // MariaDB exclusive feature
	SupportsS3Storage    bool   // MariaDB ColumnStore
	SupportsNativePasswd bool   // Both support it
	Version              string // Version string
	EOLDate              string // End-of-Life date
	SupportDuration      string // Support duration
}

// GetDBCompatibilityConfig returns compatibility configuration for a database type
func GetDBCompatibilityConfig(dbType string) *DBCompatibilityConfig {
	dbType = strings.ToLower(strings.TrimSpace(dbType))

	switch dbType {
	case "mariadb", "":
		// Default to MariaDB
		return &DBCompatibilityConfig{
			Type:                 DBTypeMariaDB,
			DisplayName:          "MariaDB 11.8 LTS",
			SupportsSequences:    true,
			SupportsPLSQL:        true,
			JSONStorageMode:      "text",
			CollationSupport:     506,
			MaxConnections:       10,
			DefaultCharset:       "utf8mb4",
			SupportsBACKUPSTAGE:  true,
			SupportsS3Storage:    true,
			SupportsNativePasswd: true,
			Version:              "11.8+",
			EOLDate:              "2028-11",
			SupportDuration:      "3 years (LTS)",
		}

	case "mysql":
		return &DBCompatibilityConfig{
			Type:                 DBTypeMySQL,
			DisplayName:          "MySQL 8.0/8.4",
			SupportsSequences:    false,
			SupportsPLSQL:        false,
			JSONStorageMode:      "binary",
			CollationSupport:     266,
			MaxConnections:       10,
			DefaultCharset:       "utf8mb4",
			SupportsBACKUPSTAGE:  false,
			SupportsS3Storage:    false,
			SupportsNativePasswd: true,
			Version:              "8.0/8.4",
			EOLDate:              "2026-04 (8.0) / 2028-04 (8.4)",
			SupportDuration:      "4 months (8.0) / 2+ years (8.4)",
		}

	default:
		// Log warning and default to MariaDB
		fmt.Fprintf(os.Stderr, "unknown DB_TYPE %q, defaulting to MariaDB\n", dbType)
		return GetDBCompatibilityConfig("mariadb")
	}
}

// DetectDatabaseType attempts to detect database type from connection
func DetectDatabaseType(db *sql.DB) (DatabaseType, string, error) {
	if db == nil {
		return DBTypeMySQL, "", fmt.Errorf("database connection is nil")
	}

	var version string
	err := db.QueryRow("SELECT VERSION()").Scan(&version)
	if err != nil {
		return DBTypeMySQL, "", fmt.Errorf("failed to detect database type: %w", err)
	}

	// Analyze version string
	versionLower := strings.ToLower(version)

	if strings.Contains(versionLower, "mariadb") {
		return DBTypeMariaDB, version, nil
	}

	if strings.Contains(versionLower, "mysql") || !strings.Contains(versionLower, "mariadb") {
		return DBTypeMySQL, version, nil
	}

	// Default to MySQL if uncertain
	return DBTypeMySQL, version, nil
}

// GetDSNByType builds a DSN string appropriate for the database type
func GetDSNByType(dbType DatabaseType, user, password, host, port, database string) string {
	baseDSN := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s",
		user, password, host, port, database)

	// Common parameters for both
	params := []string{
		"parseTime=true",
		"allowNativePasswords=true",
		"charset=utf8mb4",
	}

	// Add TLS/SSL option if needed
	if os.Getenv("DB_USE_TLS") == "true" {
		params = append(params, "tls=true")
	}

	// Database-specific parameters
	switch dbType {
	case DBTypeMariaDB:
		// MariaDB specific optimizations
		params = append(params, "multiStatements=false")

	case DBTypeMySQL:
		// MySQL specific settings
		params = append(params, "multiStatements=false")
	}

	if len(params) > 0 {
		baseDSN += "?" + strings.Join(params, "&")
	}

	return baseDSN
}

// ValidateCompatibility checks if features used are compatible with target database
func ValidateCompatibility(config *DBCompatibilityConfig, requiredFeatures []string) ([]string, error) {
	var unsupported []string

	featureMap := map[string]bool{
		"sequences":    config.SupportsSequences,
		"plsql":        config.SupportsPLSQL,
		"backup_stage": config.SupportsBACKUPSTAGE,
		"s3_storage":   config.SupportsS3Storage,
		"json_binary":  config.JSONStorageMode == "binary",
		"json_text":    config.JSONStorageMode == "text",
	}

	for _, feature := range requiredFeatures {
		featureLower := strings.ToLower(strings.TrimSpace(feature))
		if supported, exists := featureMap[featureLower]; !exists || !supported {
			unsupported = append(unsupported, feature)
		}
	}

	if len(unsupported) > 0 {
		return unsupported, fmt.Errorf("unsupported features for %s: %v",
			config.DisplayName, unsupported)
	}

	return nil, nil
}

// GetDBTypeFromEnv gets database type from environment variable
// Default: MariaDB (recommended)
func GetDBTypeFromEnv() DatabaseType {
	dbType := strings.ToLower(strings.TrimSpace(os.Getenv("DB_TYPE")))

	if dbType == "" {
		// Default to MariaDB
		return DBTypeMariaDB
	}

	if dbType == "mysql" {
		return DBTypeMySQL
	}

	if dbType == "mariadb" {
		return DBTypeMariaDB
	}

	// Default to MariaDB for unknown values
	return DBTypeMariaDB
}
