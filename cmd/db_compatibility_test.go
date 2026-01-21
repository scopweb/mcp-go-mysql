package main

import (
	"os"
	"testing"

	mysql "mcp-gp-mysql/internal"
)

// TestDBCompatibilityConfig tests database compatibility configuration
func TestDBCompatibilityConfig(t *testing.T) {
	tests := []struct {
		name               string
		dbType             string
		expectedType       mysql.DatabaseType
		expectedDisplay    string
		expectedSequences  bool
		expectedCollations int
	}{
		{
			name:              "Default (MariaDB)",
			dbType:            "",
			expectedType:      mysql.DBTypeMariaDB,
			expectedDisplay:   "MariaDB 11.8 LTS",
			expectedSequences: true,
			expectedCollations: 506,
		},
		{
			name:              "Explicit MariaDB",
			dbType:            "mariadb",
			expectedType:      mysql.DBTypeMariaDB,
			expectedDisplay:   "MariaDB 11.8 LTS",
			expectedSequences: true,
			expectedCollations: 506,
		},
		{
			name:              "MySQL",
			dbType:            "mysql",
			expectedType:      mysql.DBTypeMySQL,
			expectedDisplay:   "MySQL 8.0/8.4",
			expectedSequences: false,
			expectedCollations: 266,
		},
		{
			name:              "Uppercase MariaDB",
			dbType:            "MARIADB",
			expectedType:      mysql.DBTypeMariaDB,
			expectedDisplay:   "MariaDB 11.8 LTS",
			expectedSequences: true,
			expectedCollations: 506,
		},
		{
			name:              "Unknown (default to MariaDB)",
			dbType:            "unknown",
			expectedType:      mysql.DBTypeMariaDB,
			expectedDisplay:   "MariaDB 11.8 LTS",
			expectedSequences: true,
			expectedCollations: 506,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := mysql.GetDBCompatibilityConfig(tt.dbType)

			if config.Type != tt.expectedType {
				t.Errorf("Expected type %s, got %s", tt.expectedType, config.Type)
			}

			if config.DisplayName != tt.expectedDisplay {
				t.Errorf("Expected display %s, got %s", tt.expectedDisplay, config.DisplayName)
			}

			if config.SupportsSequences != tt.expectedSequences {
				t.Errorf("Expected sequences %v, got %v", tt.expectedSequences, config.SupportsSequences)
			}

			if config.CollationSupport != tt.expectedCollations {
				t.Errorf("Expected collations %d, got %d", tt.expectedCollations, config.CollationSupport)
			}
		})
	}
}

// TestGetDBTypeFromEnv tests environment variable parsing
func TestGetDBTypeFromEnv(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected mysql.DatabaseType
	}{
		{
			name:     "Empty env (default)",
			envValue: "",
			expected: mysql.DBTypeMariaDB,
		},
		{
			name:     "MariaDB",
			envValue: "mariadb",
			expected: mysql.DBTypeMariaDB,
		},
		{
			name:     "MySQL",
			envValue: "mysql",
			expected: mysql.DBTypeMySQL,
		},
		{
			name:     "Uppercase MySQL",
			envValue: "MYSQL",
			expected: mysql.DBTypeMySQL,
		},
		{
			name:     "Whitespace padded",
			envValue: "  mariadb  ",
			expected: mysql.DBTypeMariaDB,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("DB_TYPE", tt.envValue)
			defer os.Unsetenv("DB_TYPE")

			dbType := mysql.GetDBTypeFromEnv()
			if dbType != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, dbType)
			}
		})
	}
}

// TestDSNGeneration tests DSN generation for different databases
func TestDSNGeneration(t *testing.T) {
	tests := []struct {
		name           string
		dbType         mysql.DatabaseType
		user           string
		password       string
		host           string
		port           string
		database       string
		shouldContain  string
		shouldNotContain string
	}{
		{
			name:          "MariaDB DSN",
			dbType:        mysql.DBTypeMariaDB,
			user:          "root",
			password:      "secret",
			host:          "localhost",
			port:          "3306",
			database:      "testdb",
			shouldContain: "root:secret@tcp(localhost:3306)/testdb",
		},
		{
			name:          "MySQL DSN",
			dbType:        mysql.DBTypeMySQL,
			user:          "root",
			password:      "secret",
			host:          "localhost",
			port:          "3306",
			database:      "testdb",
			shouldContain: "root:secret@tcp(localhost:3306)/testdb",
		},
		{
			name:          "DSN includes charset",
			dbType:        mysql.DBTypeMariaDB,
			user:          "root",
			password:      "secret",
			host:          "localhost",
			port:          "3306",
			database:      "testdb",
			shouldContain: "charset=utf8mb4",
		},
		{
			name:          "DSN includes parseTime",
			dbType:        mysql.DBTypeMariaDB,
			user:          "root",
			password:      "secret",
			host:          "localhost",
			port:          "3306",
			database:      "testdb",
			shouldContain: "parseTime=true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dsn := mysql.GetDSNByType(tt.dbType, tt.user, tt.password, tt.host, tt.port, tt.database)

			if tt.shouldContain != "" && !contains(dsn, tt.shouldContain) {
				t.Errorf("DSN should contain '%s', but got: %s", tt.shouldContain, dsn)
			}

			if tt.shouldNotContain != "" && contains(dsn, tt.shouldNotContain) {
				t.Errorf("DSN should not contain '%s', but got: %s", tt.shouldNotContain, dsn)
			}
		})
	}
}

// TestMariaDBSpecificFeatures tests MariaDB-specific features
func TestMariaDBSpecificFeatures(t *testing.T) {
	config := mysql.GetDBCompatibilityConfig("mariadb")

	tests := []struct {
		name     string
		feature  bool
		expected bool
	}{
		{"Sequences", config.SupportsSequences, true},
		{"PL/SQL", config.SupportsPLSQL, true},
		{"BACKUP STAGE", config.SupportsBACKUPSTAGE, true},
		{"S3 Storage", config.SupportsS3Storage, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.feature != tt.expected {
				t.Errorf("%s: expected %v, got %v", tt.name, tt.expected, tt.feature)
			}
		})
	}
}

// TestMySQLSpecificFeatures tests MySQL-specific limitations
func TestMySQLSpecificFeatures(t *testing.T) {
	config := mysql.GetDBCompatibilityConfig("mysql")

	tests := []struct {
		name     string
		feature  bool
		expected bool
	}{
		{"Sequences", config.SupportsSequences, false},
		{"PL/SQL", config.SupportsPLSQL, false},
		{"BACKUP STAGE", config.SupportsBACKUPSTAGE, false},
		{"S3 Storage", config.SupportsS3Storage, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.feature != tt.expected {
				t.Errorf("%s: expected %v, got %v", tt.name, tt.expected, tt.feature)
			}
		})
	}
}

// TestCompatibilityValidation tests feature compatibility validation
func TestCompatibilityValidation(t *testing.T) {
	mariaConfig := mysql.GetDBCompatibilityConfig("mariadb")
	mysqlConfig := mysql.GetDBCompatibilityConfig("mysql")

	tests := []struct {
		name       string
		config     *mysql.DBCompatibilityConfig
		features   []string
		shouldPass bool
	}{
		{
			name:       "MariaDB with sequences",
			config:     mariaConfig,
			features:   []string{"sequences"},
			shouldPass: true,
		},
		{
			name:       "MySQL without sequences",
			config:     mysqlConfig,
			features:   []string{"sequences"},
			shouldPass: false,
		},
		{
			name:       "MariaDB with backup stage",
			config:     mariaConfig,
			features:   []string{"backup_stage"},
			shouldPass: true,
		},
		{
			name:       "MySQL without backup stage",
			config:     mysqlConfig,
			features:   []string{"backup_stage"},
			shouldPass: false,
		},
		{
			name:       "Both with JSON",
			config:     mariaConfig,
			features:   []string{"json_text"},
			shouldPass: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := mysql.ValidateCompatibility(tt.config, tt.features)

			if tt.shouldPass && err != nil {
				t.Errorf("Expected validation to pass, but got error: %v", err)
			}

			if !tt.shouldPass && err == nil {
				t.Errorf("Expected validation to fail, but it passed")
			}
		})
	}
}

// TestJSONStorageMode tests JSON storage differences
func TestJSONStorageMode(t *testing.T) {
	mariaConfig := mysql.GetDBCompatibilityConfig("mariadb")
	mysqlConfig := mysql.GetDBCompatibilityConfig("mysql")

	if mariaConfig.JSONStorageMode != "text" {
		t.Errorf("MariaDB should use text JSON storage, got %s", mariaConfig.JSONStorageMode)
	}

	if mysqlConfig.JSONStorageMode != "binary" {
		t.Errorf("MySQL should use binary JSON storage, got %s", mysqlConfig.JSONStorageMode)
	}
}

// TestCollationSupport tests collation differences
func TestCollationSupport(t *testing.T) {
	mariaConfig := mysql.GetDBCompatibilityConfig("mariadb")
	mysqlConfig := mysql.GetDBCompatibilityConfig("mysql")

	if mariaConfig.CollationSupport <= mysqlConfig.CollationSupport {
		t.Errorf("MariaDB should support more collations than MySQL: %d vs %d",
			mariaConfig.CollationSupport, mysqlConfig.CollationSupport)
	}
}

// Helper function
func contains(str, substr string) bool {
	return len(str) > 0 && len(substr) > 0 && indexOf(str, substr) >= 0
}

func indexOf(str, substr string) int {
	for i := 0; i < len(str); i++ {
		if i+len(substr) > len(str) {
			return -1
		}
		match := true
		for j := 0; j < len(substr); j++ {
			if str[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}
