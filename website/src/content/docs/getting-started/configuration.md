---
title: Configuration
description: Step-by-step guide to configure MCP Go MySQL in Claude Desktop
---

This guide covers the complete configuration of MCP Go MySQL for Claude Desktop on all platforms.

## Prerequisites

- **Claude Desktop** installed and running
- **MySQL 8.0+** or **MariaDB 10.x/11.x** installed and accessible
- Database access credentials (username and password)
- The `mysql-mcp` executable (see [Download](#download-the-executable) section)

## Download the Executable

### Option 1: Download Pre-built Binary

Download the latest release for your platform from GitHub:

```bash
# Visit the releases page
https://github.com/scopweb/mcp-go-mysql/releases
```

| Platform | File |
|----------|------|
| Windows | `mysql-mcp-windows-amd64.exe` |
| macOS (Intel) | `mysql-mcp-darwin-amd64` |
| macOS (Apple Silicon) | `mysql-mcp-darwin-arm64` |
| Linux | `mysql-mcp-linux-amd64` |

### Option 2: Build from Source

```bash
# Clone the repository
git clone https://github.com/scopweb/mcp-go-mysql.git
cd mcp-go-mysql

# Build the executable
go mod tidy
go build -o mysql-mcp ./cmd

# On Windows, the output will be mysql-mcp.exe
```

## Step 1: Prepare MySQL/MariaDB User

:::caution
Never use the `root` user in production environments.
:::

Create a dedicated user with appropriate permissions:

```sql
-- Create user for MCP (works on both MySQL and MariaDB)
CREATE USER 'mcp_user'@'%' IDENTIFIED BY 'your_secure_password';

-- Grant read-only permissions (recommended for production)
GRANT SELECT ON your_database.* TO 'mcp_user'@'%';

-- Grant write permissions if needed
GRANT INSERT, UPDATE, DELETE ON your_database.* TO 'mcp_user'@'%';

-- Grant DDL permissions only if absolutely necessary
GRANT CREATE, DROP, ALTER ON your_database.* TO 'mcp_user'@'%';

-- Apply changes
FLUSH PRIVILEGES;
```

## Step 2: Locate Configuration File

Claude Desktop uses a JSON file to configure MCP servers:

| Operating System | Configuration File Path |
|------------------|------------------------|
| **Windows** | `%APPDATA%\Claude\claude_desktop_config.json` |
| **macOS** | `~/Library/Application Support/Claude/claude_desktop_config.json` |
| **Linux** | `~/.config/Claude/claude_desktop_config.json` |

:::tip
On Windows, press `Win+R`, type `%APPDATA%\Claude`, and press Enter to open the folder directly.
:::

## Step 3: Configure Claude Desktop

### Windows Configuration

```json
{
  "mcpServers": {
    "mysql": {
      "command": "C:\\Users\\YourUser\\mcp-go-mysql\\mysql-mcp.exe",
      "args": [],
      "env": {
        "MYSQL_HOST": "localhost",
        "MYSQL_PORT": "3306",
        "MYSQL_USER": "mcp_user",
        "MYSQL_PASSWORD": "your_secure_password",
        "MYSQL_DATABASE": "your_database",
        "LOG_PATH": "C:\\Users\\YourUser\\mcp-go-mysql\\mysql-mcp.log"
      }
    }
  }
}
```

### macOS Configuration

```json
{
  "mcpServers": {
    "mysql": {
      "command": "/Users/youruser/mcp-go-mysql/mysql-mcp",
      "args": [],
      "env": {
        "MYSQL_HOST": "localhost",
        "MYSQL_PORT": "3306",
        "MYSQL_USER": "mcp_user",
        "MYSQL_PASSWORD": "your_secure_password",
        "MYSQL_DATABASE": "your_database",
        "LOG_PATH": "/Users/youruser/mcp-go-mysql/mysql-mcp.log"
      }
    }
  }
}
```

### Linux Configuration

```json
{
  "mcpServers": {
    "mysql": {
      "command": "/home/youruser/mcp-go-mysql/mysql-mcp",
      "args": [],
      "env": {
        "MYSQL_HOST": "localhost",
        "MYSQL_PORT": "3306",
        "MYSQL_USER": "mcp_user",
        "MYSQL_PASSWORD": "your_secure_password",
        "MYSQL_DATABASE": "your_database",
        "LOG_PATH": "/home/youruser/mcp-go-mysql/mysql-mcp.log"
      }
    }
  }
}
```

## Complete Configuration Reference

Here's a full example with **all available options**:

```json
{
  "mcpServers": {
    "mysql": {
      "command": "/path/to/mysql-mcp",
      "args": [],
      "env": {
        "MYSQL_HOST": "localhost",
        "MYSQL_PORT": "3306",
        "MYSQL_USER": "mcp_user",
        "MYSQL_PASSWORD": "your_secure_password",
        "MYSQL_DATABASE": "your_database",
        "LOG_PATH": "/path/to/mysql-mcp.log",
        "ALLOWED_TABLES": "users,orders,products,categories",
        "ALLOW_DDL": "false",
        "SAFETY_KEY": "PRODUCTION_CONFIRMED_2025"
      }
    }
  }
}
```

## Environment Variables Reference

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `MYSQL_HOST` | **Yes** | `localhost` | MySQL/MariaDB server hostname or IP |
| `MYSQL_PORT` | No | `3306` | Server port number |
| `MYSQL_USER` | **Yes** | - | Database username |
| `MYSQL_PASSWORD` | **Yes** | - | Database password |
| `MYSQL_DATABASE` | **Yes** | - | Default database to connect to |
| `LOG_PATH` | No | `mysql-mcp.log` | Path for audit log file |
| `ALLOWED_TABLES` | No | *(all tables)* | Comma-separated whitelist of allowed tables |
| `ALLOW_DDL` | No | `false` | Enable CREATE, DROP, ALTER operations |
| `SAFETY_KEY` | No | `PRODUCTION_CONFIRMED_2025` | Confirmation key for bulk operations |

## Advanced Security Configuration

### Restrict Access to Specific Tables

Limit operations to only certain tables:

```json
"env": {
  "ALLOWED_TABLES": "users,orders,products,categories"
}
```

When configured, any attempt to access tables not in the list will be blocked.

### Disable DDL Operations

Block all CREATE, DROP, and ALTER statements:

```json
"env": {
  "ALLOW_DDL": "false"
}
```

:::caution
Even with `ALLOW_DDL=true`, dangerous operations like `DROP DATABASE` are **always blocked**.
:::

### Custom Confirmation Key

Change the confirmation key required for bulk operations (>100 rows):

```json
"env": {
  "SAFETY_KEY": "MY_CUSTOM_KEY_2026"
}
```

## Multiple Database Configuration

You can configure multiple MySQL/MariaDB servers in Claude Desktop:

```json
{
  "mcpServers": {
    "mysql-production": {
      "command": "/path/to/mysql-mcp",
      "args": [],
      "env": {
        "MYSQL_HOST": "prod-db.example.com",
        "MYSQL_PORT": "3306",
        "MYSQL_USER": "readonly_user",
        "MYSQL_PASSWORD": "prod_password",
        "MYSQL_DATABASE": "production",
        "ALLOWED_TABLES": "users,orders"
      }
    },
    "mysql-development": {
      "command": "/path/to/mysql-mcp",
      "args": [],
      "env": {
        "MYSQL_HOST": "localhost",
        "MYSQL_PORT": "3306",
        "MYSQL_USER": "dev_user",
        "MYSQL_PASSWORD": "dev_password",
        "MYSQL_DATABASE": "development",
        "ALLOW_DDL": "true"
      }
    },
    "mariadb-analytics": {
      "command": "/path/to/mysql-mcp",
      "args": [],
      "env": {
        "MYSQL_HOST": "analytics.example.com",
        "MYSQL_PORT": "3306",
        "MYSQL_USER": "analytics_user",
        "MYSQL_PASSWORD": "analytics_password",
        "MYSQL_DATABASE": "analytics"
      }
    }
  }
}
```

:::note
Each server will appear as a separate tool set in Claude. You can ask Claude to use a specific one, e.g., "Using mysql-production, show me all orders from today".
:::

## Docker Configuration

If MySQL/MariaDB is running in Docker:

```json
"env": {
  "MYSQL_HOST": "localhost",
  "MYSQL_PORT": "3307",
  "MYSQL_USER": "mcp_user",
  "MYSQL_PASSWORD": "password",
  "MYSQL_DATABASE": "mydb"
}
```

:::note
Use the **mapped port** (e.g., `3307`) if Docker exposes the container on a different port than the default `3306`.
:::

## Remote Database Connection

To connect to a remote MySQL/MariaDB server:

```json
"env": {
  "MYSQL_HOST": "db.example.com",
  "MYSQL_PORT": "3306",
  "MYSQL_USER": "remote_user",
  "MYSQL_PASSWORD": "secure_password",
  "MYSQL_DATABASE": "production_db",
  "ALLOWED_TABLES": "users,orders"
}
```

:::tip
For production databases, use **read-only permissions** (SELECT only) to prevent accidental data modification.
:::

## Step 4: Verify Configuration

1. **Save the configuration file** and close it
2. **Restart Claude Desktop** completely (quit and reopen)
3. **Open a new conversation** with Claude
4. **Test the connection** by asking:
   - "What MySQL tools do you have available?"
   - "List all tables in my database"
   - "What version of MySQL am I using?"

### Expected Response

Claude should list the 10 available tools:
- `query`, `execute`, `tables`, `describe`, `views`, `indexes`, `explain`, `count`, `sample`, `database_info`

## Troubleshooting

### Connection Issues

| Error | Solution |
|-------|----------|
| "Connection refused" | Check MySQL/MariaDB is running: `mysql -u mcp_user -p` |
| "Access denied" | Verify username/password and user permissions |
| "Unknown database" | Confirm database exists and user has access |
| "Host not allowed" | Add user permission for the connecting host |

### Verification Commands

```bash
# Test MySQL connection directly
mysql -h localhost -u mcp_user -p your_database

# Check if MySQL is listening
netstat -an | findstr 3306   # Windows
netstat -an | grep 3306      # macOS/Linux

# View MCP server logs
type mysql-mcp.log           # Windows
cat mysql-mcp.log            # macOS/Linux
```

### Log Analysis

If issues persist, check the log file at the path specified in `LOG_PATH`:

```bash
# View recent log entries
tail -f mysql-mcp.log

# Search for errors
grep -i error mysql-mcp.log
```

:::tip
The log file contains detailed information about queries executed, security validations, and any errors encountered.
:::
