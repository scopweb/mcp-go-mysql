// Package security tests the verb-based statement classifier in
// internal/client.go. The classifier is the secondary security layer of the
// MCP — the primary layer is the MySQL user's own grants.
//
// These tests do NOT need a live database connection: they exercise
// ValidateQuery() in isolation.
package security

import (
	"strings"
	"testing"

	mysql "mcp-gp-mysql/internal"
)

// TestVerbClassifier is the main test for ValidateQuery. It groups cases
// into "must allow" and "must reject" and reports each.
func TestVerbClassifier(t *testing.T) {
	client := mysql.NewClient()

	cases := []struct {
		name      string
		query     string
		expectErr bool
	}{
		// --- Must ALLOW ----------------------------------------------------

		{"simple SELECT", "SELECT * FROM users WHERE id = 1", false},
		{"SELECT with JOIN", "SELECT u.name FROM users u JOIN orders o ON u.id = o.user_id", false},
		{"SELECT with subquery", "SELECT * FROM users WHERE id IN (SELECT user_id FROM orders)", false},
		{"WITH (CTE)", "WITH t AS (SELECT 1) SELECT * FROM t", false},
		{"SHOW TABLES", "SHOW TABLES", false},
		{"DESCRIBE", "DESCRIBE users", false},
		{"EXPLAIN", "EXPLAIN SELECT * FROM users", false},
		{"USE db", "USE my_database", false},

		{"INSERT", "INSERT INTO users (name) VALUES ('a')", false},
		{"UPDATE with WHERE", "UPDATE users SET name = 'a' WHERE id = 1", false},
		// UPDATE/DELETE without WHERE are now allowed by the classifier and
		// gated by MaxSafeRows post-execution. The classifier no longer
		// produces false positives for this pattern.
		{"UPDATE without WHERE (gated by MaxSafeRows later)", "UPDATE users SET name = 'a'", false},
		{"DELETE without WHERE (gated by MaxSafeRows later)", "DELETE FROM users", false},
		{"REPLACE", "REPLACE INTO users (id, name) VALUES (1, 'a')", false},

		{"comment-prefixed SELECT", "-- this is a comment\nSELECT 1", false},
		{"block-comment-prefixed SELECT", "/* hello */ SELECT 1", false},
		{"trailing semicolon allowed", "SELECT 1;", false},

		// SLEEP / BENCHMARK are no longer special-cased. A SELECT remains a
		// SELECT regardless of the functions it calls.
		{"SELECT with SLEEP (legitimate debugging)", "SELECT SLEEP(1)", false},
		{"SELECT with BENCHMARK", "SELECT BENCHMARK(100, SHA1('x'))", false},

		// --- Must REJECT: forbidden verbs (privilege/filesystem/server) ----

		{"GRANT", "GRANT ALL ON *.* TO 'evil'@'%'", true},
		{"REVOKE", "REVOKE ALL ON *.* FROM 'foo'@'%'", true},
		{"SET PASSWORD", "SET PASSWORD FOR 'root'@'localhost' = PASSWORD('x')", true},
		{"SET GLOBAL", "SET GLOBAL general_log = 'ON'", true},
		{"FLUSH PRIVILEGES", "FLUSH PRIVILEGES", true},
		{"RESET MASTER", "RESET MASTER", true},
		{"KILL", "KILL 12345", true},
		{"SHUTDOWN", "SHUTDOWN", true},
		{"LOAD DATA INFILE", "LOAD DATA INFILE '/etc/passwd' INTO TABLE x", true},
		{"HANDLER", "HANDLER t OPEN", true},
		{"INSTALL PLUGIN", "INSTALL PLUGIN audit_log SONAME 'audit.so'", true},
		{"LOCK TABLES", "LOCK TABLES users WRITE", true},

		// --- Must REJECT: DDL (when ALLOW_DDL is unset) --------------------

		{"DROP DATABASE", "DROP DATABASE production", true},
		{"DROP SCHEMA", "DROP SCHEMA public CASCADE", true},
		{"DROP TABLE", "DROP TABLE users", true},
		{"CREATE TABLE", "CREATE TABLE foo (id INT)", true},
		{"CREATE USER", "CREATE USER 'evil'@'%' IDENTIFIED BY 'x'", true},
		{"DROP USER", "DROP USER 'foo'@'%'", true},
		{"ALTER TABLE", "ALTER TABLE users ADD COLUMN x INT", true},
		{"TRUNCATE TABLE", "TRUNCATE TABLE users", true},
		{"RENAME TABLE", "RENAME TABLE users TO old_users", true},

		// --- Must REJECT: filesystem clauses inside legal verbs ------------

		{"SELECT INTO OUTFILE", "SELECT * FROM users INTO OUTFILE '/tmp/x'", true},
		{"SELECT INTO DUMPFILE", "SELECT * FROM users INTO DUMPFILE '/tmp/x'", true},

		// --- Must REJECT: stacked statements -------------------------------

		{"stacked SELECT;DROP", "SELECT 1; DROP DATABASE foo", true},
		{"stacked SELECT;SELECT", "SELECT 1; SELECT 2", true},
		{"stacked with whitespace", "SELECT 1 ;\n DROP TABLE x", true},

		// --- Must REJECT: empty / unknown ----------------------------------

		{"empty", "", true},
		{"whitespace only", "   \n\t ", true},
		{"only comment", "-- nothing here\n", true},
		{"unknown verb", "FOOBAR users", true},
	}

	passed, failed := 0, 0
	for _, tc := range cases {
		err := client.ValidateQuery(tc.query)
		got := err != nil
		if got == tc.expectErr {
			passed++
			continue
		}
		failed++
		if tc.expectErr {
			t.Errorf("[%s] expected REJECT but got ALLOW — query: %q", tc.name, tc.query)
		} else {
			t.Errorf("[%s] expected ALLOW but got REJECT (%v) — query: %q", tc.name, err, tc.query)
		}
	}
	t.Logf("verb classifier: %d passed, %d failed", passed, failed)
}

// TestErrorMessagesAreInformative checks that rejection errors mention the
// reason category, so the LLM caller can correct itself instead of guessing.
func TestErrorMessagesAreInformative(t *testing.T) {
	client := mysql.NewClient()

	checks := []struct {
		query    string
		mustContain string
	}{
		{"GRANT ALL ON *.* TO x", "not allowed"},
		{"DROP DATABASE x", "DDL"},
		{"SELECT * FROM x INTO OUTFILE '/tmp/y'", "OUTFILE"},
		{"SELECT 1; SELECT 2", "multiple statements"},
		{"FOOBAR x", "unknown verb"},
	}
	for _, c := range checks {
		err := client.ValidateQuery(c.query)
		if err == nil {
			t.Errorf("query %q should have been rejected", c.query)
			continue
		}
		if !strings.Contains(err.Error(), c.mustContain) {
			t.Errorf("error message for %q should mention %q, got: %v", c.query, c.mustContain, err)
		}
	}
}

// BenchmarkValidateQuery measures the cost of the classifier on a typical
// SELECT. It should be cheap enough that we never need to cache results.
func BenchmarkValidateQuery(b *testing.B) {
	client := mysql.NewClient()
	query := "SELECT u.name, o.total FROM users u JOIN orders o ON u.id = o.user_id WHERE u.active = 1"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = client.ValidateQuery(query)
	}
}
