# Security tests — mcp-go-mysql

This package holds the security-relevant tests for the MCP server: the
verb-classifier behaviour and the dependency-integrity checks referenced in
`CLAUDE.md`. The tests exercise the real classifier in `internal` — they do not
re-implement "dangerous pattern" lists (which the project deliberately avoids;
see `docs/SECURITY.md`).

## Files

### `classifier_test.go`
Tests `internal.ValidateQuery`, `internal.ValidateTableAccess`, and
`internal.StripComments` directly (no database connection required):

- `TestValidateQueryClassifier` — the verb whitelist: read/write verbs pass;
  privilege/filesystem verbs (`GRANT`, `SET`, `FLUSH`, `LOAD`, …), unknown
  verbs, stacked statements, and `INTO OUTFILE`/`DUMPFILE` are rejected; DDL is
  rejected unless `ALLOW_DDL=true`.
- `TestOutfileExecutableCommentBypass` — regression: `INTO OUTFILE`/`DUMPFILE`
  cannot be smuggled past the classifier inside a MySQL conditional-execution
  comment (`/*! … */`) or via comment/whitespace obfuscation.
- `TestExecutableCommentLeadingVerb` — regression: a verb hidden in a `/*! … */`
  comment (the SQL MySQL actually runs) is classified, not erased.
- `TestStackedDetectionEscapes` — regression: backslash escaping in the
  stacked-statement detector (`'\\'` closes the string; `'\''`-style escapes
  keep it open).
- `TestDDLGate` — `ALLOW_DDL=true` lets DDL through, but forbidden verbs stay
  blocked.
- `TestTableWhitelist` — `ALLOWED_TABLES` governs `ValidateTableAccess`
  (enforced by `describe`, `count`, `sample`, `indexes`).
- `TestStripComments` — the comment scrubber.

### `dependencies_test.go`
- `TestNoUnexpectedDependencies` — pins the module surface to the MySQL driver
  (plus its transitive `edwards25519`).
- `TestNoReplaceDirectives` — flags `replace`/`exclude` directives in `go.mod`.
- `TestGoSumWellFormed` — checks `go.sum` entries are well formed.

## Running

```bash
go test -v ./cmd/security/...
go test -race ./...
govulncheck ./...   # requires network access to vuln.go.dev
```
