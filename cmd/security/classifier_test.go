package security

import (
	"testing"

	mysql "mcp-gp-mysql/internal"
)

// newClassifierClient builds a Client whose statement classifier is configured
// purely from the environment. NewClient does not open a database connection,
// so ValidateQuery / ValidateTableAccess can be exercised in isolation.
func newClassifierClient(t *testing.T, env map[string]string) *mysql.Client {
	t.Helper()
	for k, v := range env {
		t.Setenv(k, v)
	}
	return mysql.NewClient()
}

// TestValidateQueryClassifier covers the verb-classifier whitelist: read/write
// verbs pass, privilege/filesystem verbs and unknown verbs are rejected, and
// DDL is rejected while ALLOW_DDL is unset.
func TestValidateQueryClassifier(t *testing.T) {
	c := newClassifierClient(t, nil) // defaults: DDL blocked

	cases := []struct {
		name    string
		query   string
		wantErr bool
	}{
		{"select", "SELECT * FROM users", false},
		{"with cte", "WITH t AS (SELECT 1) SELECT * FROM t", false},
		{"show", "SHOW TABLES", false},
		{"insert", "INSERT INTO t(a) VALUES (1)", false},
		{"update", "UPDATE t SET a = 1 WHERE id = 2", false},
		{"delete", "DELETE FROM t WHERE id = 2", false},
		{"grant blocked", "GRANT ALL ON *.* TO 'x'@'%'", true},
		{"revoke blocked", "REVOKE ALL ON *.* FROM 'x'@'%'", true},
		{"set blocked", "SET GLOBAL general_log = 1", true},
		{"flush blocked", "FLUSH PRIVILEGES", true},
		{"load data blocked", "LOAD DATA INFILE '/etc/passwd' INTO TABLE t", true},
		{"ddl blocked by default", "DROP TABLE t", true},
		{"unknown verb blocked", "FOOBAR something", true},
		{"empty blocked", "   ", true},
		{"stacked blocked", "SELECT 1; DROP TABLE t", true},
		{"into outfile blocked", "SELECT * FROM users INTO OUTFILE '/tmp/x'", true},
		{"into dumpfile blocked", "SELECT * FROM users INTO DUMPFILE '/tmp/x'", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := c.ValidateQuery(tc.query)
			if tc.wantErr && err == nil {
				t.Errorf("expected rejection for %q, got nil", tc.query)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("expected %q to pass, got: %v", tc.query, err)
			}
		})
	}
}

// TestOutfileExecutableCommentBypass is a regression test: INTO OUTFILE /
// DUMPFILE must not be smuggled past the classifier through a MySQL
// conditional-execution comment (/*! ... */), which the server executes but
// StripComments used to delete, nor through comment/whitespace obfuscation.
func TestOutfileExecutableCommentBypass(t *testing.T) {
	c := newClassifierClient(t, nil)

	bypasses := []string{
		`SELECT * FROM users /*!50000 INTO OUTFILE '/tmp/p' */`,
		`SELECT * FROM users /*! INTO OUTFILE '/tmp/p' */`,
		`SELECT * FROM users INTO/**/OUTFILE '/tmp/p'`,
		`SELECT * FROM users INTO  OUTFILE '/tmp/p'`,
		`SELECT * FROM users /*!50000 INTO DUMPFILE '/tmp/p' */`,
	}
	for _, q := range bypasses {
		if err := c.ValidateQuery(q); err == nil {
			t.Errorf("expected INTO OUTFILE/DUMPFILE to be rejected: %q", q)
		}
	}
}

// TestExecutableCommentLeadingVerb is a regression test: a leading verb hidden
// in an executable comment is what MySQL actually runs, so it must be
// classified accordingly (DROP -> DDL, blocked by default) rather than being
// erased to an empty statement.
func TestExecutableCommentLeadingVerb(t *testing.T) {
	c := newClassifierClient(t, nil)
	if err := c.ValidateQuery(`/*!50000 DROP TABLE t */`); err == nil {
		t.Error("expected DROP inside an executable comment to be blocked")
	}
}

// TestStackedDetectionEscapes is a regression test for backslash handling in
// the stacked-statement detector. A literal backslash (\\) closes the string,
// so the following ';' starts a real second statement and must be rejected; an
// escaped quote (\') keeps the string open, so its ';' is data, not a separator.
func TestStackedDetectionEscapes(t *testing.T) {
	c := newClassifierClient(t, nil)

	if err := c.ValidateQuery(`SELECT '\\'; DROP TABLE t`); err == nil {
		t.Error("expected stacked statement after escaped backslash to be rejected")
	}
	if err := c.ValidateQuery(`SELECT '\'; still inside one string'`); err != nil {
		t.Errorf("single statement with ';' inside a string literal should pass, got: %v", err)
	}
}

// TestDDLGate verifies DDL flows with ALLOW_DDL, and that forbidden verbs stay
// blocked even when DDL is enabled.
func TestDDLGate(t *testing.T) {
	c := newClassifierClient(t, map[string]string{"ALLOW_DDL": "true"})

	if err := c.ValidateQuery("CREATE TABLE t (a INT)"); err != nil {
		t.Errorf("CREATE should be allowed with ALLOW_DDL=true, got: %v", err)
	}
	if err := c.ValidateQuery("GRANT ALL ON *.* TO 'x'@'%'"); err == nil {
		t.Error("GRANT must stay blocked even with ALLOW_DDL=true")
	}
}

// TestTableWhitelist verifies ALLOWED_TABLES governs ValidateTableAccess, the
// check the identifier-based tools (describe/count/sample/indexes) now apply.
func TestTableWhitelist(t *testing.T) {
	c := newClassifierClient(t, map[string]string{"ALLOWED_TABLES": "users, orders"})

	if err := c.ValidateTableAccess("users"); err != nil {
		t.Errorf("whitelisted table should be allowed, got: %v", err)
	}
	if err := c.ValidateTableAccess("ORDERS"); err != nil {
		t.Errorf("whitelist should be case-insensitive, got: %v", err)
	}
	if err := c.ValidateTableAccess("secrets"); err == nil {
		t.Error("table outside the whitelist must be rejected")
	}

	// With no whitelist configured, all tables are allowed.
	open := newClassifierClient(t, map[string]string{"ALLOWED_TABLES": ""})
	if err := open.ValidateTableAccess("anything"); err != nil {
		t.Errorf("empty whitelist should allow any table, got: %v", err)
	}
}

// TestStripComments verifies the comment scrubber used by the classifier.
func TestStripComments(t *testing.T) {
	cases := map[string]string{
		"SELECT 1 -- trailing":     "SELECT 1",
		"SELECT 1 # trailing":      "SELECT 1",
		"SELECT /* inline */ 1":    "SELECT 1",
		"SELECT\t1\r\n":            "SELECT 1",
		"SELECT 1 /* a */ /* b */": "SELECT 1",
	}
	for in, want := range cases {
		if got := mysql.StripComments(in); got != want {
			t.Errorf("StripComments(%q) = %q, want %q", in, got, want)
		}
	}
}
