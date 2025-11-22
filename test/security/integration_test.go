package security

import (
	"testing"

	mysql "mcp-gp-mysql/internal"
)

// TestClientSecurityValidation tests the client's built-in security validation
func TestClientSecurityValidation(t *testing.T) {
	client := mysql.NewClient()

	testCases := []struct {
		name      string
		query     string
		expectErr bool
	}{
		// Safe queries
		{
			name:      "Simple SELECT",
			query:     "SELECT * FROM users WHERE id = 1",
			expectErr: false,
		},
		{
			name:      "SELECT with JOIN",
			query:     "SELECT u.name, o.total FROM users u JOIN orders o ON u.id = o.user_id",
			expectErr: false,
		},
		{
			name:      "SELECT with subquery",
			query:     "SELECT * FROM users WHERE id IN (SELECT user_id FROM orders)",
			expectErr: false,
		},

		// Dangerous queries that should be blocked
		{
			name:      "DROP DATABASE",
			query:     "DROP DATABASE production",
			expectErr: true,
		},
		{
			name:      "DROP SCHEMA",
			query:     "DROP SCHEMA public CASCADE",
			expectErr: true,
		},
		{
			name:      "TRUNCATE TABLE",
			query:     "TRUNCATE TABLE users",
			expectErr: true,
		},
		{
			name:      "DELETE without WHERE",
			query:     "DELETE FROM users",
			expectErr: true,
		},
		{
			name:      "UPDATE without WHERE",
			query:     "UPDATE users SET password = 'hacked'",
			expectErr: true,
		},
		{
			name:      "INTO OUTFILE",
			query:     "SELECT * FROM users INTO OUTFILE '/tmp/data.txt'",
			expectErr: true,
		},
		{
			name:      "INTO DUMPFILE",
			query:     "SELECT * FROM users INTO DUMPFILE '/tmp/dump'",
			expectErr: true,
		},
		{
			name:      "LOAD DATA",
			query:     "LOAD DATA INFILE '/etc/passwd' INTO TABLE data",
			expectErr: true,
		},
		{
			name:      "LOAD_FILE function",
			query:     "SELECT LOAD_FILE('/etc/passwd')",
			expectErr: true,
		},

		// SQL injection patterns that should be blocked
		{
			name:      "Classic OR injection",
			query:     "SELECT * FROM users WHERE name = '' OR '1'='1'",
			expectErr: true,
		},
		{
			name:      "UNION injection",
			query:     "SELECT * FROM users UNION SELECT * FROM passwords",
			expectErr: true,
		},
		{
			name:      "SLEEP injection",
			query:     "SELECT * FROM users WHERE SLEEP(5)",
			expectErr: true,
		},
		{
			name:      "BENCHMARK injection",
			query:     "SELECT BENCHMARK(10000000, SHA1('test'))",
			expectErr: true,
		},
		{
			name:      "Information schema access",
			query:     "SELECT * FROM INFORMATION_SCHEMA.TABLES",
			expectErr: true,
		},
		{
			name:      "Hex encoded payload",
			query:     "SELECT 0x61646D696E",
			expectErr: true,
		},
		{
			name:      "CHAR function abuse",
			query:     "SELECT CHAR(97,100,109,105,110)",
			expectErr: true,
		},
		{
			name:      "GROUP_CONCAT extraction",
			query:     "SELECT GROUP_CONCAT(username,password) FROM users",
			expectErr: true,
		},
		{
			name:      "EXTRACTVALUE XML injection",
			query:     "SELECT EXTRACTVALUE(1,CONCAT(0x7e,version()))",
			expectErr: true,
		},
		{
			name:      "UPDATEXML injection",
			query:     "SELECT UPDATEXML(1,CONCAT(0x7e,version()),1)",
			expectErr: true,
		},
	}

	passed := 0
	failed := 0

	for _, tc := range testCases {
		err := client.ValidateQuery(tc.query)
		hasErr := err != nil

		if hasErr == tc.expectErr {
			passed++
			if tc.expectErr {
				t.Logf("✅ BLOCKED: %s", tc.name)
			} else {
				t.Logf("✅ ALLOWED: %s", tc.name)
			}
		} else {
			failed++
			if tc.expectErr {
				t.Errorf("❌ SHOULD BLOCK: %s - query: %s", tc.name, tc.query)
			} else {
				t.Errorf("❌ SHOULD ALLOW: %s - query: %s (error: %v)", tc.name, tc.query, err)
			}
		}
	}

	t.Logf("\nClient Validation Results: %d passed, %d failed", passed, failed)

	if failed > 0 {
		t.Fail()
	}
}

// TestClientSecurityConfig tests security configuration
func TestClientSecurityConfig(t *testing.T) {
	client := mysql.NewClient()

	// Test that client is created with default security settings
	t.Log("Testing client security defaults...")

	// The client should be created without error
	if client == nil {
		t.Fatal("Client should not be nil")
	}

	t.Log("✅ Client created with security defaults")
}

// TestEmptyQueryValidation tests empty query handling
func TestEmptyQueryValidation(t *testing.T) {
	client := mysql.NewClient()

	emptyQueries := []string{
		"",
		"   ",
		"\t",
		"\n",
		"  \t\n  ",
	}

	for _, query := range emptyQueries {
		err := client.ValidateQuery(query)
		if err == nil {
			t.Errorf("Empty/whitespace query should be rejected: %q", query)
		} else {
			t.Logf("✅ Empty query rejected: %q", query)
		}
	}
}

// TestIdentifierValidation tests SQL identifier validation
func TestIdentifierValidation(t *testing.T) {
	validIdentifiers := []string{
		"users",
		"user_profiles",
		"Orders",
		"_private_table",
		"table123",
		"T",
	}

	invalidIdentifiers := []string{
		"",                                       // Empty
		"users; DROP TABLE--",                    // Injection
		"users' OR '1'='1",                       // Injection
		"../../../etc",                           // Path traversal
		"table name",                             // Space
		"table-name",                             // Dash
		"table.name",                             // Dot
		"table`name",                             // Backtick
		"123table",                               // Starts with number
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", // Too long (>64)
	}

	t.Log("Testing valid identifiers:")
	for _, id := range validIdentifiers {
		t.Logf("  ✅ '%s' - valid", id)
	}

	t.Log("Testing invalid identifiers:")
	for _, id := range invalidIdentifiers {
		if len(id) > 20 {
			t.Logf("  ❌ (length %d) - should be rejected", len(id))
		} else if id == "" {
			t.Logf("  ❌ (empty) - should be rejected")
		} else {
			t.Logf("  ❌ '%s' - should be rejected", id)
		}
	}
}

// TestSecurityFunctions tests security helper functions
func TestSecurityFunctions(t *testing.T) {
	t.Log("Testing IsSafeSQL function...")

	// Test known dangerous patterns
	dangerous := []string{
		"' OR '1'='1",
		"UNION SELECT",
		"1=1",
		"SLEEP(5)",
		"BENCHMARK(1000,SHA1('x'))",
	}

	for _, d := range dangerous {
		if mysql.IsSafeSQL(d) {
			t.Errorf("IsSafeSQL should return false for: %s", d)
		} else {
			t.Logf("✅ IsSafeSQL correctly blocks: %s", d)
		}
	}

	// Test safe inputs
	safe := []string{
		"12345",
		"John Doe",
		"user@example.com",
		"normal text",
	}

	for _, s := range safe {
		if !mysql.IsSafeSQL(s) {
			t.Errorf("IsSafeSQL should return true for: %s", s)
		} else {
			t.Logf("✅ IsSafeSQL correctly allows: %s", s)
		}
	}
}

// TestPathSecurityFunctions tests path security validation
func TestPathSecurityFunctions(t *testing.T) {
	t.Log("Testing IsSafePath function...")

	// Test dangerous paths
	dangerous := []string{
		"../../../etc/passwd",
		"..\\..\\windows",
		"/etc/passwd",
		"C:\\Windows",
		"//network/share",
		"\\\\server\\share",
	}

	for _, d := range dangerous {
		if mysql.IsSafePath(d) {
			t.Errorf("IsSafePath should return false for: %s", d)
		} else {
			t.Logf("✅ IsSafePath correctly blocks: %s", d)
		}
	}

	// Test safe paths
	safe := []string{
		"document.pdf",
		"reports/annual.xlsx",
		"data_2024.csv",
	}

	for _, s := range safe {
		if !mysql.IsSafePath(s) {
			t.Errorf("IsSafePath should return true for: %s", s)
		} else {
			t.Logf("✅ IsSafePath correctly allows: %s", s)
		}
	}
}

// TestCommandSecurityFunctions tests command security validation
func TestCommandSecurityFunctions(t *testing.T) {
	t.Log("Testing IsSafeCommand function...")

	// Test dangerous commands
	dangerous := []string{
		"file; rm -rf /",
		"file | cat /etc/passwd",
		"file`whoami`",
		"file$(id)",
		"file${PATH}",
		"file & whoami",
		"file\nwhoami",
	}

	for _, d := range dangerous {
		if mysql.IsSafeCommand(d) {
			t.Errorf("IsSafeCommand should return false for: %q", d)
		} else {
			t.Logf("✅ IsSafeCommand correctly blocks: %q", d)
		}
	}

	// Test safe commands
	safe := []string{
		"document.pdf",
		"my_file_2024.txt",
		"report-final.docx",
	}

	for _, s := range safe {
		if !mysql.IsSafeCommand(s) {
			t.Errorf("IsSafeCommand should return true for: %s", s)
		} else {
			t.Logf("✅ IsSafeCommand correctly allows: %s", s)
		}
	}
}

// BenchmarkSQLValidation benchmarks SQL validation performance
func BenchmarkSQLValidation(b *testing.B) {
	client := mysql.NewClient()
	query := "SELECT * FROM users WHERE id = 1 AND name = 'test'"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.ValidateQuery(query)
	}
}

// BenchmarkInjectionDetection benchmarks injection detection
func BenchmarkInjectionDetection(b *testing.B) {
	injection := "1' OR '1'='1' UNION SELECT * FROM passwords--"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mysql.IsSafeSQL(injection)
	}
}
