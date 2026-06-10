package security

import (
	"os"
	"strings"
	"testing"
)

// allowedModules is the dependency surface this project is supposed to have.
// CLAUDE.md: "No external dependencies beyond github.com/go-sql-driver/mysql."
// edwards25519 is its transitive dependency.
var allowedModules = map[string]bool{
	"github.com/go-sql-driver/mysql": true,
	"filippo.io/edwards25519":        true,
}

// TestNoUnexpectedDependencies pins the module surface. A new require line must
// be a deliberate, reviewed decision — adding it here forces that review.
func TestNoUnexpectedDependencies(t *testing.T) {
	content, err := os.ReadFile("../../go.mod")
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}

	for _, raw := range strings.Split(string(content), "\n") {
		line := strings.TrimSpace(raw)
		// Dependency lines start with a domain-qualified module path.
		if !(strings.HasPrefix(line, "github.com/") ||
			strings.HasPrefix(line, "filippo.io/") ||
			strings.HasPrefix(line, "golang.org/")) {
			continue
		}
		name := strings.Fields(line)[0]
		if !allowedModules[name] {
			t.Errorf("unexpected dependency %q in go.mod; add it to allowedModules only after review", name)
		}
	}
}

// TestNoReplaceDirectives flags directives that change where module code comes
// from. They have legitimate uses, but should never appear here unnoticed.
func TestNoReplaceDirectives(t *testing.T) {
	content, err := os.ReadFile("../../go.mod")
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}
	for _, directive := range []string{"replace ", "exclude "} {
		if strings.Contains(string(content), directive) {
			t.Errorf("go.mod contains a %q directive — review it", strings.TrimSpace(directive))
		}
	}
}

// TestGoSumWellFormed verifies every go.sum entry has the expected three fields
// (module, version, hash), catching truncation or tampering.
func TestGoSumWellFormed(t *testing.T) {
	content, err := os.ReadFile("../../go.sum")
	if err != nil {
		t.Fatalf("read go.sum: %v", err)
	}
	for _, raw := range strings.Split(string(content), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		if len(strings.Fields(line)) != 3 {
			t.Errorf("malformed go.sum line: %q", line)
		}
	}
}
