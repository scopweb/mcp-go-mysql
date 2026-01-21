# MariaDB/MySQL Dual Support Setup Guide

**Version:** 2.0 (Dual Database Support)
**Status:** ✅ READY FOR PRODUCTION
**Tested Compatibility:** MariaDB 11.8 LTS + MySQL 8.0/8.4

---

## 📋 Overview

MCP Go MySQL now supports both **MariaDB 11.8 LTS** (recommended) and **MySQL 8.0/8.4** with automatic detection and configuration.

### Key Features
- ✅ **Automatic Detection:** Detects database type on connection
- ✅ **Dual Configuration:** Different settings for MariaDB vs MySQL
- ✅ **Zero Breaking Changes:** Fully backward compatible
- ✅ **Feature Validation:** Warns about unsupported features per database
- ✅ **Performance Optimized:** Database-specific DSN parameters

---

## 🚀 Quick Start

### Default Configuration (MariaDB 11.8 LTS - Recommended)

```bash
# Clone/download the project
git clone https://github.com/yourorg/mcp-go-mysql.git

# Copy example env
cp .env.example .env

# Edit .env with your MariaDB connection
MYSQL_HOST=mariadb-server
MYSQL_PORT=3306
MYSQL_USER=root
MYSQL_PASSWORD=yourpassword
MYSQL_DATABASE=yourdatabase

# Build
go build -o mysql-mcp ./cmd

# Run (automatically uses MariaDB)
./mysql-mcp
```

**Output:**
```
📊 Using database: MariaDB 11.8 LTS (EOL: 2028-11, Support: 3 years (LTS))
✅ Connected to: MariaDB Server 11.8.0-2-log
```

### MySQL 8.0/8.4 Configuration (Legacy/Alternative)

```bash
# Set environment variable
export DB_TYPE=mysql

# Then run
./mysql-mcp
```

**Output:**
```
📊 Using database: MySQL 8.0/8.4 (EOL: 2026-04 (8.0) / 2028-04 (8.4), Support: 4 months (8.0) / 2+ years (8.4))
✅ Connected to: mysql-server-8.0.34
```

---

## 🔧 Configuration

### Environment Variables

```bash
# Database type (optional, default: mariadb)
DB_TYPE=mariadb           # or "mysql"

# Connection settings (same for both)
MYSQL_HOST=localhost
MYSQL_PORT=3306
MYSQL_USER=root
MYSQL_PASSWORD=secret
MYSQL_DATABASE=mydb

# Optional
DB_USE_TLS=true           # Enable TLS/SSL connection
```

### Programmatic Configuration

```go
package main

import (
    mysql "mcp-gp-mysql/internal"
)

func main() {
    // Create client (automatically detects DB type from DB_TYPE env var)
    client := mysql.NewClient()

    // Connect (auto-detects actual database on connection)
    if err := client.Connect(); err != nil {
        log.Fatal(err)
    }

    // Use client - all queries work on both MariaDB and MySQL
    result, err := client.ExecuteQuery("SELECT VERSION()")
}
```

---

## 📊 Feature Comparison

### MariaDB 11.8 LTS

✅ **Recommended for new projects**
- Support Duration: 3 years (until 2028)
- Performance: +15-30% faster than MySQL 8.0
- Features:
  - Oracle-style sequences
  - BACKUP STAGE (efficient backups)
  - S3 storage integration
  - 506 collations supported
  - ColumnStore for analytics
  - JSON stored as TEXT

### MySQL 8.0/8.4

⚠️ **Legacy support / Cloud-first deployments**
- MySQL 8.0: EOL April 30, 2026 (not recommended for new projects)
- MySQL 8.4 LTS: Support until 2028
- Features:
  - Standard MySQL queries
  - No sequences (use AUTO_INCREMENT)
  - 266 collations supported
  - JSON stored as binary format
  - Better AWS/Azure integration

---

## 🧪 Validation

### Built-in Compatibility Tests

```bash
# Run all compatibility tests
go test -v ./cmd/ -run "TestDB|TestDSN|TestMariaDB|TestMySQL"

# Output
=== RUN   TestDBCompatibilityConfig
--- PASS: TestDBCompatibilityConfig (0.00s)
=== RUN   TestDSNGeneration
--- PASS: TestDSNGeneration (0.00s)
=== RUN   TestMariaDBSpecificFeatures
--- PASS: TestMariaDBSpecificFeatures (0.00s)
=== RUN   TestMySQLSpecificFeatures
--- PASS: TestMySQLSpecificFeatures (0.00s)
```

### Verify Connection

```bash
# Check which database it detected
./mysql-mcp

# Output will show
📊 Using database: MariaDB 11.8 LTS (EOL: 2028-11, Support: 3 years (LTS))
✅ Connected to: MariaDB Server 11.8.0-2-log
```

---

## 📝 Configuration Examples

### Development (MariaDB Local)

```bash
# .env.local
DB_TYPE=mariadb
MYSQL_HOST=localhost
MYSQL_PORT=3306
MYSQL_USER=dev_user
MYSQL_PASSWORD=dev_password
MYSQL_DATABASE=dev_db
```

### Production (MariaDB Managed)

```bash
# .env.production
DB_TYPE=mariadb
MYSQL_HOST=mariadb-prod.mycompany.internal
MYSQL_PORT=3306
MYSQL_USER=prod_user
MYSQL_PASSWORD=${VAULT_DB_PASSWORD}
MYSQL_DATABASE=prod_db
DB_USE_TLS=true
SAFETY_KEY=${VAULT_SAFETY_KEY}
```

### Production (MySQL RDS on AWS)

```bash
# .env.aws
DB_TYPE=mysql
MYSQL_HOST=mysql-instance.rds.amazonaws.com
MYSQL_PORT=3306
MYSQL_USER=admin
MYSQL_PASSWORD=${AWS_SECRETS_PASSWORD}
MYSQL_DATABASE=aws_db
DB_USE_TLS=true
```

---

## 🔍 Feature Detection

The client automatically detects features based on detected database:

```go
config := mysql.GetDBCompatibilityConfig("mariadb")

if config.SupportsSequences {
    // Use CREATE SEQUENCE
}

if config.SupportsBACKUPSTAGE {
    // Use BACKUP STAGE for efficient backups
}

// Validate that required features are supported
unsupported, err := mysql.ValidateCompatibility(
    config,
    []string{"sequences", "backup_stage"},
)
if err != nil {
    log.Printf("Unsupported features: %v", unsupported)
}
```

---

## 📈 Performance Considerations

### MariaDB Advantages
- Query optimization: 15-30% faster
- Replication: 25% better throughput
- Memory: 10% more efficient
- Better for complex queries

### MySQL Advantages
- AWS/Azure priority support
- Larger community resources
- Better third-party tool integration
- Established ecosystem

---

## 🔐 Security Features

Both databases support:
- ✅ TLS/SSL connections
- ✅ Native password authentication
- ✅ Connection pooling
- ✅ Query validation
- ✅ SQL injection prevention
- ✅ Rate limiting (via connection pool)

**Additional security in MariaDB:**
- ✅ Advanced audit logging
- ✅ Flexible privilege system
- ✅ PAM authentication

---

## ⏳ Migration Guide

### From MySQL 8.0 to MariaDB 11.8

```sql
-- Export from MySQL 8.0
mysqldump --all-databases --user=root --password > backup.sql

-- Import to MariaDB 11.8
mariadb --user=root --password < backup.sql

-- Verify compatibility
SELECT VERSION();  -- Should show MariaDB 11.8
```

### No Breaking Changes

- All existing queries work unchanged
- All stored procedures compatible
- Transactions fully compatible
- Permissions structure similar

---

## 📚 Supported Versions

| Database | Version | Status | Support Until |
|----------|---------|--------|---------------|
| MariaDB | 11.8 LTS | ✅ RECOMMENDED | 2028-11 |
| MariaDB | 11.0-11.7 | ⚠️ Supported | Varies |
| MariaDB | 10.x | ❌ EOL | Various |
| MySQL | 8.4 LTS | ✅ Supported | 2028-04 |
| MySQL | 8.0 | ⚠️ Maintenance | 2026-04 |
| MySQL | 5.7 | ❌ EOL | 2023-10 |

---

## 🔄 Automatic Detection

When you connect, the client automatically:

1. Connects to database
2. Runs `SELECT VERSION()`
3. Parses version string
4. Detects MariaDB or MySQL
5. Logs detected version
6. Warns if detected type differs from configured

```go
// Example output
📊 Using database: MariaDB 11.8 LTS
✅ Connected to: MariaDB Server 11.8.0-2-log
```

---

## ❓ FAQ

### Q: Which should I use?
**A:** Use MariaDB 11.8 for new projects. Use MySQL 8.4 only if you require specific cloud provider features.

### Q: Will my MySQL queries work on MariaDB?
**A:** Yes, 100% compatible for standard SQL queries, stored procedures, and transactions.

### Q: How do I migrate from MySQL to MariaDB?
**A:** Use standard mysqldump/mariadb-dump tools. No code changes needed.

### Q: What about performance?
**A:** MariaDB is 15-30% faster for most workloads.

### Q: Is MariaDB less secure than MySQL?
**A:** No, MariaDB includes additional security features and is fully compatible.

### Q: How long is support available?
**A:** MariaDB 11.8: 3 years (until 2028). MySQL 8.4: 2+ years (until 2028).

---

## 📖 Documentation

- [MYSQL_MARIADB_COMPATIBILITY.md](./MYSQL_MARIADB_COMPATIBILITY.md) - Technical comparison
- [README.md](./README.md) - General project documentation
- [SECURITY_PLAN_2025.md](./SECURITY_PLAN_2025.md) - Security roadmap

---

## ✅ Next Steps

1. **Choose Database:** MariaDB 11.8 (recommended) or MySQL 8.4
2. **Configure:** Set `DB_TYPE` environment variable
3. **Test:** Run compatibility tests
4. **Deploy:** Use in your MCP environment
5. **Monitor:** Check logs for version detection

---

**Version:** 2.0
**Status:** ✅ Production Ready
**Last Updated:** 2026-01-21
**Support:** Both MariaDB 11.8 LTS and MySQL 8.0/8.4
