# Claude Desktop Integration Guide

This guide explains how to configure and use MCP Go MySQL with Claude Desktop.

## Prerequisites

1. **Claude Desktop** installed on your system
2. **Go 1.21+** for building the server
3. **MySQL Server** running and accessible
4. A MySQL user with appropriate permissions

## Quick Start

### 1. Build the Server

```bash
# Clone the repository
git clone https://github.com/scopweb/mcp-go-mysql.git
cd mcp-go-mysql

# Download dependencies
go mod tidy

# Build the executable
go build -o mysql-mcp ./cmd

# On Windows, this creates mysql-mcp.exe
```

### 2. Locate Configuration File

Find your Claude Desktop configuration file:

| Platform | Configuration File Path |
|----------|------------------------|
| **Windows** | `%APPDATA%\Claude\claude_desktop_config.json` |
| **macOS** | `~/Library/Application Support/Claude/claude_desktop_config.json` |
| **Linux** | `~/.config/Claude/claude_desktop_config.json` |

### 3. Add MCP Server Configuration

Edit the configuration file and add the MySQL server:

#### Windows Configuration

```json
{
  "mcpServers": {
    "mysql": {
      "command": "C:\\path\\to\\mcp-go-mysql\\mysql-mcp.exe",
      "args": [],
      "env": {
        "MYSQL_HOST": "localhost",
        "MYSQL_PORT": "3306",
        "MYSQL_USER": "your_username",
        "MYSQL_PASSWORD": "your_password",
        "MYSQL_DATABASE": "your_database",
        "LOG_PATH": "C:\\path\\to\\mysql-mcp.log"
      }
    }
  }
}
```

#### macOS Configuration

```json
{
  "mcpServers": {
    "mysql": {
      "command": "/Users/youruser/mcp-go-mysql/mysql-mcp",
      "args": [],
      "env": {
        "MYSQL_HOST": "localhost",
        "MYSQL_PORT": "3306",
        "MYSQL_USER": "your_username",
        "MYSQL_PASSWORD": "your_password",
        "MYSQL_DATABASE": "your_database",
        "LOG_PATH": "/Users/youruser/mcp-go-mysql/mysql-mcp.log"
      }
    }
  }
}
```

#### Linux Configuration

```json
{
  "mcpServers": {
    "mysql": {
      "command": "/home/youruser/mcp-go-mysql/mysql-mcp",
      "args": [],
      "env": {
        "MYSQL_HOST": "localhost",
        "MYSQL_PORT": "3306",
        "MYSQL_USER": "your_username",
        "MYSQL_PASSWORD": "your_password",
        "MYSQL_DATABASE": "your_database",
        "LOG_PATH": "/home/youruser/mcp-go-mysql/mysql-mcp.log"
      }
    }
  }
}
```

### 4. Restart Claude Desktop

After saving the configuration:
1. Completely close Claude Desktop
2. Reopen Claude Desktop
3. The MySQL tools should now be available

## Verification

### Check Available Tools

Ask Claude: "What MySQL tools do you have available?"

You should see the following 10 tools:
- `query` - Execute SELECT queries
- `execute` - Execute INSERT/UPDATE/DELETE
- `tables` - List all tables
- `describe` - Show table structure
- `views` - List all views
- `indexes` - Show table indexes
- `explain` - Query execution plan
- `count` - Count rows
- `sample` - Get sample rows
- `database_info` - Connection info

### Test the Connection

Ask Claude: "List all tables in my database"

If configured correctly, Claude will use the `tables` tool and show your database tables.

## Advanced Configurations

### Multiple Databases

Configure multiple MySQL servers:

```json
{
  "mcpServers": {
    "mysql-dev": {
      "command": "/path/to/mysql-mcp",
      "env": {
        "MYSQL_HOST": "localhost",
        "MYSQL_DATABASE": "development",
        "MYSQL_USER": "dev_user",
        "MYSQL_PASSWORD": "dev_password"
      }
    },
    "mysql-prod": {
      "command": "/path/to/mysql-mcp",
      "env": {
        "MYSQL_HOST": "prod-server.example.com",
        "MYSQL_DATABASE": "production",
        "MYSQL_USER": "readonly_user",
        "MYSQL_PASSWORD": "secure_password",
        "ALLOWED_TABLES": "users,orders,products"
      }
    }
  }
}
```

### Read-Only Configuration

For production environments with read-only access:

```json
{
  "mcpServers": {
    "mysql-readonly": {
      "command": "/path/to/mysql-mcp",
      "env": {
        "MYSQL_HOST": "prod-server.example.com",
        "MYSQL_PORT": "3306",
        "MYSQL_USER": "readonly_user",
        "MYSQL_PASSWORD": "secure_password",
        "MYSQL_DATABASE": "production",
        "ALLOWED_TABLES": "users,orders,products,categories",
        "ALLOW_DDL": "false"
      }
    }
  }
}
```

### Remote MySQL Server

For connecting to remote servers:

```json
{
  "mcpServers": {
    "mysql-remote": {
      "command": "/path/to/mysql-mcp",
      "env": {
        "MYSQL_HOST": "db.example.com",
        "MYSQL_PORT": "3306",
        "MYSQL_USER": "remote_user",
        "MYSQL_PASSWORD": "remote_password",
        "MYSQL_DATABASE": "mydb"
      }
    }
  }
}
```

### Docker MySQL

For connecting to MySQL in Docker:

```json
{
  "mcpServers": {
    "mysql-docker": {
      "command": "/path/to/mysql-mcp",
      "env": {
        "MYSQL_HOST": "host.docker.internal",
        "MYSQL_PORT": "3306",
        "MYSQL_USER": "docker_user",
        "MYSQL_PASSWORD": "docker_password",
        "MYSQL_DATABASE": "docker_db"
      }
    }
  }
}
```

## Security Best Practices

### 1. Create a Dedicated MySQL User

```sql
-- Create user with limited permissions
CREATE USER 'mcp_user'@'%' IDENTIFIED BY 'strong_password';

-- Grant only necessary permissions
GRANT SELECT, INSERT, UPDATE, DELETE ON your_database.* TO 'mcp_user'@'%';

-- For read-only access
GRANT SELECT ON your_database.* TO 'mcp_readonly'@'%';

-- Apply changes
FLUSH PRIVILEGES;
```

### 2. Use Table Whitelists

Restrict access to specific tables:

```json
{
  "env": {
    "ALLOWED_TABLES": "users,orders,products,categories"
  }
}
```

### 3. Disable DDL Operations

For production environments:

```json
{
  "env": {
    "ALLOW_DDL": "false"
  }
}
```

### 4. Customize Safety Key

Change the default confirmation key:

```json
{
  "env": {
    "SAFETY_KEY": "YOUR_CUSTOM_KEY_2025"
  }
}
```

## Troubleshooting

### Connection Refused

**Error:** "Connection refused"

**Solutions:**
1. Verify MySQL server is running: `systemctl status mysql`
2. Check if the port is correct (default: 3306)
3. Verify MySQL is accepting connections: `mysql -h localhost -u user -p`

### Access Denied

**Error:** "Access denied for user"

**Solutions:**
1. Verify username and password are correct
2. Check user has permissions on the database
3. Verify user can connect from the MCP server's host

### Unknown Database

**Error:** "Unknown database"

**Solutions:**
1. Verify the database name is spelled correctly
2. Check the database exists: `SHOW DATABASES;`
3. Verify user has access to the database

### Tools Not Appearing

**Problem:** MySQL tools don't show up in Claude

**Solutions:**
1. Verify the executable path is correct
2. Check the configuration file syntax (valid JSON)
3. Restart Claude Desktop completely
4. Check the log file for errors

### Security Validation Failed

**Error:** "Security validation failed"

**Cause:** Query contains patterns that look like SQL injection

**Solutions:**
1. Review your query for SQL injection patterns
2. Use parameterized queries when possible
3. Check the specific pattern that was blocked

## Usage Examples

### Query Data

Ask Claude:
```
Query the users table to show all users created in the last 7 days
```

### Describe Table Structure

Ask Claude:
```
Describe the structure of the orders table
```

### Count Records

Ask Claude:
```
How many orders are there with status 'pending'?
```

### Analyze Query Performance

Ask Claude:
```
Explain the execution plan for: SELECT * FROM orders WHERE customer_id = 123
```

### Get Sample Data

Ask Claude:
```
Show me 5 sample rows from the products table
```

## Environment Variables Reference

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `MYSQL_HOST` | Yes | localhost | MySQL server hostname |
| `MYSQL_PORT` | No | 3306 | MySQL server port |
| `MYSQL_USER` | Yes | - | MySQL username |
| `MYSQL_PASSWORD` | Yes | - | MySQL password |
| `MYSQL_DATABASE` | Yes | - | Default database |
| `LOG_PATH` | No | mysql-mcp.log | Log file path |
| `ALLOWED_TABLES` | No | (all) | Comma-separated whitelist |
| `ALLOW_DDL` | No | false | Enable DDL operations |
| `SAFETY_KEY` | No | PRODUCTION_CONFIRMED_2025 | Confirmation key |
| `MAX_SAFE_ROWS` | No | 100 | Row threshold for confirmation |

## Log Analysis

Monitor the log file for debugging:

```bash
# Follow logs in real-time
tail -f mysql-mcp.log

# Search for errors
grep -i error mysql-mcp.log

# View recent activity
tail -100 mysql-mcp.log
```

## Support

If you encounter issues:
1. Check the [Troubleshooting](#troubleshooting) section
2. Review the log file for detailed error messages
3. Open an issue at: https://github.com/scopweb/mcp-go-mysql/issues
