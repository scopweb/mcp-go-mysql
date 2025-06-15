# Advanced MySQL MCP Server with Intelligent Security

Production-ready MySQL Model Context Protocol (MCP) server in Go with comprehensive database tools and intelligent security system. Features automatic protection for dangerous operations with confirmation keys and modular architecture.

## ⚠️ IMPORTANT SECURITY NOTICE

**🚨 ALWAYS BACKUP YOUR DATABASE BEFORE USING WRITE OPERATIONS**

This server provides powerful database tools that can modify your data. Please:
- **Create backups** before performing any write operations
- **Test operations** on development databases first
- **Use appropriate MySQL user permissions** - create a dedicated MySQL user with only the permissions you need
- **Review SQL statements** carefully before execution
- **Monitor operation logs** for security auditing

### 🔒 Recommended MySQL User Setup

Create a dedicated MySQL user with minimal required permissions:

```sql
-- Create dedicated user for MCP
CREATE USER 'mcp_user'@'%' IDENTIFIED BY 'secure_password';

-- Grant only necessary permissions (adjust as needed)
GRANT SELECT, INSERT, UPDATE, DELETE ON your_database.* TO 'mcp_user'@'%';
GRANT CREATE, DROP, ALTER ON your_database.* TO 'mcp_user'@'%';  -- Only if DDL needed
GRANT SHOW VIEW, CREATE VIEW, DROP VIEW ON your_database.* TO 'mcp_user'@'%';

-- Refresh privileges
FLUSH PRIVILEGES;
```

**Never use root or admin users in production!**

## 🚀 Features

### 19 Advanced Database Tools

**📋 Core Operations (3)**
- `query` - Execute SELECT queries (read-only)
- `tables` - List all tables  
- `describe` - Describe table/view structure

**👁️ Views Management (5)**
- `list_views` - List all views
- `view_definition` - Show complete view definition
- `create_view` - Create new views
- `drop_view` - Delete views  
- `view_dependencies` - Analyze view dependencies

**🔍 Performance Analysis (4)**
- `explain_query` - Analyze query execution plans
- `analyze_object` - Analyze database objects  
- `optimize_tables` - Optimize table performance
- `process_list` - Show active MySQL processes

**🛡️ Secure Write Operations (3)**
- `execute_write` - INSERT/UPDATE/DELETE with smart protection
- `execute_ddl` - CREATE/DROP/ALTER with mandatory confirmation
- `show_safety_info` - Display security configuration

**📊 Advanced Reports (3)**
- `create_report` - Generate custom reports (JSON/CSV/Summary)
- `view_report` - Comprehensive view analysis
- `database_report` - Complete database overview

### 🛡️ Intelligent Security System

**Automatic Risk Detection:**
- ✅ **Small operations** (≤100 rows) → Execute freely
- 🔑 **Large operations** (>100 rows) → Require confirmation key  
- 🔑 **DDL operations** (CREATE/DROP/ALTER) → Always require confirmation
- ❌ **Database drops** → Completely blocked

**Smart Protection:**
```sql
-- FREE: Specific updates
UPDATE users SET status='active' WHERE id=123

-- REQUIRES KEY: Mass updates  
UPDATE users SET status='inactive'  -- Affects all rows

-- REQUIRES KEY: Structure changes
CREATE TABLE backup_users AS SELECT * FROM users

-- BLOCKED: Database deletion
DROP DATABASE production  -- Always blocked
```

## 🔧 Installation

### 1. Clone and Build
```bash
git clone https://github.com/scopweb/mcp-go-mysql.git
cd mcp-go-mysql
go mod tidy
go build -o mysql-go-mcp.exe ./cmd
```

### 2. Configure Environment
Create `.env` file:
```env
MYSQL_HOST=localhost
MYSQL_PORT=3306
MYSQL_USER=mcp_user
MYSQL_PASSWORD=secure_password
MYSQL_DATABASE=your_database
LOG_PATH=mysql-mcp.log
```

### 3. Claude Desktop Configuration
```json
{
  "mcpServers": {
    "mysql-advanced": {
      "command": "C:\\path\\to\\mysql-go-mcp.exe",
      "args": [],
      "env": {
        "MYSQL_HOST": "localhost",
        "MYSQL_PORT": "3306", 
        "MYSQL_USER": "mcp_user",
        "MYSQL_PASSWORD": "secure_password",
        "MYSQL_DATABASE": "your_database"
      }
    }
  }
}
```

## 💡 Usage Examples

### 🔍 Safe Operations (No Confirmation Required)
```sql
-- Query data
SELECT * FROM products WHERE category='electronics' LIMIT 10

-- Small updates
UPDATE orders SET status='shipped' WHERE order_id=12345

-- Describe structures  
DESCRIBE customers
```

### 🔑 Protected Operations (Require Confirmation)
```sql
-- Mass updates (requires: confirm_key="PRODUCTION_CONFIRMED_2025")
UPDATE products SET discount=0.1 WHERE category='clearance'

-- DDL operations (always require confirmation)  
CREATE VIEW monthly_sales AS 
SELECT DATE_FORMAT(date,'%Y-%m') as month, SUM(total) 
FROM orders GROUP BY month

-- Table optimization
OPTIMIZE TABLE large_table
```

### 📊 Advanced Reporting
```sql
-- Generate CSV report
SELECT customer_name, SUM(order_total) as total_spent 
FROM customer_orders 
GROUP BY customer_name 
ORDER BY total_spent DESC
-- Format: CSV, Limit: 100

-- View analysis report  
ANALYZE VIEW complex_sales_view
-- Includes: schema, sample data, dependencies
```

## 🔐 Security Configuration

**Current Safety Key:** `PRODUCTION_CONFIRMED_2025`  
**Row Limit:** `100 rows` (configurable)

**To change security settings:**
Edit constants in `cmd/main.go`:
```go
const (
    SAFETY_KEY    = "YOUR_CUSTOM_KEY_2025"
    MAX_SAFE_ROWS = 50  // Adjust threshold
)
```

## 📁 Project Structure

```
cmd/
├── main.go           - Main server and configuration
├── types.go          - MCP message structures  
├── handlers.go       - Message handling logic
├── tools.go          - Tool definitions
├── client_methods.go - Database operations
└── security.go       - Security and validation

internal/
├── mysql.go          - Core MySQL client
├── views.go          - Views management
├── analysis.go       - Performance analysis
├── reports.go        - Report generation
└── query.go          - Query execution
```

## 🔄 Security Workflow

1. **Operation Analysis** - Server analyzes SQL commands
2. **Risk Assessment** - Determines if confirmation needed  
3. **Smart Protection** - Blocks/allows/requests confirmation
4. **Audit Logging** - All operations logged for security
5. **Graceful Errors** - Clear error messages with guidance

## 🚀 Production Best Practices

- **Test thoroughly** on development databases
- **Monitor logs** regularly for security events
- **Backup before** any structural changes
- **Use minimal** MySQL user permissions
- **Review operations** that require confirmation keys
- **Keep security keys** confidential and unique

## 📈 Performance Benefits

- **Modular architecture** - Fast compilation and maintenance
- **Efficient connections** - Optimized MySQL driver usage  
- **Smart caching** - Reduced redundant database calls
- **Comprehensive logging** - Detailed operation tracking
- **Memory efficient** - Minimal resource usage

---

**Built for production environments with security as the top priority. Always backup your data!** 🛡️