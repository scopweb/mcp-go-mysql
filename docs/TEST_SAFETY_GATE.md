# Testing the Row-Count Safety Gate (Rollback Verification)

This guide lets you verify that the `MAX_SAFE_ROWS` + `confirm_key` mechanism **actually rolls back** large writes when the safety key is missing.

> **Important**: This test only works reliably with InnoDB (or other transactional engines). MyISAM and other non-transactional engines cannot roll back.

## 1. Create a Test Table

Run this SQL in your test database (adjust the name if you want):

```sql
-- Drop if it exists from previous tests
DROP TABLE IF EXISTS safety_gate_test;

CREATE TABLE safety_gate_test (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100),
    value INT,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB;

-- Insert enough rows so we can easily exceed a low threshold
INSERT INTO safety_gate_test (name, value) VALUES
    ('item-01', 100), ('item-02', 100), ('item-03', 100),
    ('item-04', 100), ('item-05', 100), ('item-06', 100),
    ('item-07', 100), ('item-08', 100), ('item-09', 100),
    ('item-10', 100), ('item-11', 100), ('item-12', 100),
    ('item-13', 100), ('item-14', 100), ('item-15', 100);
```

This gives us 15 rows. We will set `MAX_SAFE_ROWS=5` for the test.

## 2. Recommended Environment Variables for Testing

Create a temporary `.env.test` (or export them):

```bash
# Use a dedicated test database / user
MYSQL_HOST=127.0.0.1
MYSQL_PORT=3306
MYSQL_USER=mcp_test_user
MYSQL_PASSWORD=your_test_password
MYSQL_DATABASE=test_safety_gate

# Make the safety threshold easy to trigger
MAX_SAFE_ROWS=5

# Use a custom key (never use the default in real usage)
SAFETY_KEY=TEST_ROLLBACK_2026_INSECURE_BUT_OK_FOR_LOCAL_TEST

# Optional: lower log noise during test
LOG_PATH=mysql-mcp-test.log
```

**Windows PowerShell example:**

```powershell
$env:MYSQL_HOST = "127.0.0.1"
$env:MYSQL_PORT = "3306"
$env:MYSQL_USER = "mcp_test_user"
$env:MYSQL_PASSWORD = "your_test_password"
$env:MYSQL_DATABASE = "test_safety_gate"
$env:MAX_SAFE_ROWS = "5"
$env:SAFETY_KEY = "TEST_ROLLBACK_2026_INSECURE_BUT_OK_FOR_LOCAL_TEST"
```

## 3. Build the Binary

```bash
go build -o mysql-mcp-test.exe ./cmd
```

(or without `.exe` on Linux/macOS)

## 4. Verification Scenarios

You can test in two ways:

### A. Using Claude Desktop (recommended for real MCP usage)

1. Temporarily point your Claude Desktop config to the test binary + the test env vars above.
2. Restart Claude Desktop.
3. In a new chat, ask it to use the `mysql` tools (or whatever name you gave the server).

**Test 1 — Large update WITHOUT the key (must rollback)**

Ask Claude something like:

> "Using the execute tool, run this update on safety_gate_test: set value = 999 for all rows. Do not provide any confirm_key."

Expected result:
- Claude should receive an error containing: `operation affects 15 rows (>5). Provide safety key to confirm. Changes have been rolled back`
- If you query the table again (`SELECT COUNT(*) FROM safety_gate_test WHERE value = 999`), it should still be **0**.

**Test 2 — Same update WITH the correct key (must commit)**

Ask:

> "Now run the same update but pass confirm_key = TEST_ROLLBACK_2026_INSECURE_BUT_OK_FOR_LOCAL_TEST"

Expected result:
- Success message: `Query executed successfully. Rows affected: 15`
- `SELECT COUNT(*) FROM safety_gate_test WHERE value = 999` → should return **15**.

**Test 3 — Small update (below threshold) — no key needed**

```sql
UPDATE safety_gate_test SET value = 42 WHERE id = 1
```

This should always succeed without asking for the key.

### B. Direct test (using the binary + a simple query tool or mysql client)

You can also call the MCP tools manually via JSON-RPC if you want to script it, but the Claude flow above is the most realistic.

## 5. Cleanup After Test

```sql
DROP TABLE safety_gate_test;
```

And remove the test environment variables.

## 6. What Success Looks Like

- Without the key on a large operation → error + **zero rows changed**.
- With the correct key → changes applied.
- Small operations never require the key (current behavior preserved).

If you see the change applied even without the key, the old buggy behavior is still present (report it).

---

This test directly validates the fix done in the Unreleased version (explicit transaction + conditional rollback in `internal/client.go:Execute`).

Run this whenever you touch the write path or the safety configuration.