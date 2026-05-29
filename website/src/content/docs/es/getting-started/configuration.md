---
title: Configuración
description: Guía paso a paso para configurar MCP Go MySQL en Claude Desktop, Grok Builder y otros clientes MCP.
---

Esta guía cubre cómo configurar MCP Go MySQL para clientes MCP populares como Claude Desktop y Grok Builder.

## Requisitos Previos

- **Claude Desktop** instalado y ejecutándose
- **MySQL 8.0+** o **MariaDB 10.x/11.x** instalado y accesible
- Credenciales de acceso a la base de datos (usuario y contraseña)
- El ejecutable `mysql-mcp` (ver sección [Descargar](#descargar-el-ejecutable))

## Descargar el Ejecutable

### Opción 1: Descargar Binario Pre-compilado

Descarga la última versión para tu plataforma desde GitHub:

```bash
# Visita la página de releases
https://github.com/scopweb/mcp-go-mysql/releases
```

| Plataforma | Archivo |
|------------|---------|
| Windows | `mysql-mcp-windows-amd64.exe` |
| macOS (Intel) | `mysql-mcp-darwin-amd64` |
| macOS (Apple Silicon) | `mysql-mcp-darwin-arm64` |
| Linux | `mysql-mcp-linux-amd64` |

### Opción 2: Compilar desde Código Fuente

```bash
# Clonar el repositorio
git clone https://github.com/scopweb/mcp-go-mysql.git
cd mcp-go-mysql

# Compilar el ejecutable
go mod tidy
go build -o mysql-mcp ./cmd

# En Windows, la salida será mysql-mcp.exe
```

## Paso 1: Preparar Usuario MySQL/MariaDB

:::caution
Nunca uses el usuario `root` en entornos de producción.
:::

Crea un usuario dedicado con los permisos apropiados:

```sql
-- Crear usuario para MCP (funciona en MySQL y MariaDB)
CREATE USER 'mcp_user'@'%' IDENTIFIED BY 'tu_password_seguro';

-- Otorgar permisos de solo lectura (recomendado para producción)
GRANT SELECT ON tu_base_datos.* TO 'mcp_user'@'%';

-- Otorgar permisos de escritura si es necesario
GRANT INSERT, UPDATE, DELETE ON tu_base_datos.* TO 'mcp_user'@'%';

-- Otorgar permisos DDL solo si es absolutamente necesario
GRANT CREATE, DROP, ALTER ON tu_base_datos.* TO 'mcp_user'@'%';

-- Aplicar cambios
FLUSH PRIVILEGES;
```

## Paso 2: Ubicar Archivo de Configuración

Claude Desktop usa un archivo JSON para configurar servidores MCP:

| Sistema Operativo | Ruta del Archivo de Configuración |
|-------------------|----------------------------------|
| **Windows** | `%APPDATA%\Claude\claude_desktop_config.json` |
| **macOS** | `~/Library/Application Support/Claude/claude_desktop_config.json` |
| **Linux** | `~/.config/Claude/claude_desktop_config.json` |

:::tip
En Windows, presiona `Win+R`, escribe `%APPDATA%\Claude`, y presiona Enter para abrir la carpeta directamente.
:::

## Paso 3: Configurar Claude Desktop

### Configuración Windows

```json
{
  "mcpServers": {
    "mysql": {
      "command": "C:\\Users\\TuUsuario\\mcp-go-mysql\\mysql-mcp.exe",
      "args": [],
      "env": {
        "MYSQL_HOST": "localhost",
        "MYSQL_PORT": "3306",
        "MYSQL_USER": "mcp_user",
        "MYSQL_PASSWORD": "tu_password_seguro",
        "MYSQL_DATABASE": "tu_base_datos",
        "LOG_PATH": "C:\\Users\\TuUsuario\\mcp-go-mysql\\mysql-mcp.log"
      }
    }
  }
}
```

### Configuración macOS

```json
{
  "mcpServers": {
    "mysql": {
      "command": "/Users/tuusuario/mcp-go-mysql/mysql-mcp",
      "args": [],
      "env": {
        "MYSQL_HOST": "localhost",
        "MYSQL_PORT": "3306",
        "MYSQL_USER": "mcp_user",
        "MYSQL_PASSWORD": "tu_password_seguro",
        "MYSQL_DATABASE": "tu_base_datos",
        "LOG_PATH": "/Users/tuusuario/mcp-go-mysql/mysql-mcp.log"
      }
    }
  }
}
```

### Configuración Linux

```json
{
  "mcpServers": {
    "mysql": {
      "command": "/home/tuusuario/mcp-go-mysql/mysql-mcp",
      "args": [],
      "env": {
        "MYSQL_HOST": "localhost",
        "MYSQL_PORT": "3306",
        "MYSQL_USER": "mcp_user",
        "MYSQL_PASSWORD": "tu_password_seguro",
        "MYSQL_DATABASE": "tu_base_datos",
        "LOG_PATH": "/home/tuusuario/mcp-go-mysql/mysql-mcp.log"
      }
    }
  }
}
```

## Configuración para Grok Builder / Grok TUI

Grok (a través del Grok Build TUI) soporta servidores MCP vía stdio.

La configuración en Grok se realiza normalmente a través de su interfaz de ajustes o archivos de configuración (consulta `~/.grok/` o la ayuda del TUI para el método actual).

### Configuración Básica

Necesitas proporcionar:

- La ruta completa al ejecutable `mysql-mcp`.
- Las variables de entorno obligatorias (`MYSQL_HOST`, `MYSQL_USER`, `MYSQL_PASSWORD`, `MYSQL_DATABASE` como mínimo).
- Opcionalmente `SAFETY_KEY`, `MAX_SAFE_ROWS`, `LOG_PATH`, etc.

Ejemplo de configuración (adáptalo al formato actual de Grok):

```json
{
  "mcpServers": {
    "mysql": {
      "command": "/ruta/a/mysql-mcp",
      "args": [],
      "env": {
        "MYSQL_HOST": "localhost",
        "MYSQL_PORT": "3306",
        "MYSQL_USER": "mcp_user",
        "MYSQL_PASSWORD": "tu_password_seguro",
        "MYSQL_DATABASE": "tu_base_de_datos",
        "SAFETY_KEY": "tu-clave-personal",
        "MAX_SAFE_ROWS": "100"
      }
    }
  }
}
```

**Consejos para Grok:**
- Usa siempre rutas absolutas al binario.
- Configura un `SAFETY_KEY` personalizado (nunca uses el valor por defecto en producción).
- El nombre del servidor (`"mysql"`) puedes cambiarlo al que prefieras.

Para la forma más actualizada de añadir servidores MCP en Grok Builder, consulta la documentación oficial de Grok o la ayuda del TUI.

## Referencia Completa de Configuración

Aquí tienes un ejemplo completo con **todas las opciones disponibles**:

```json
{
  "mcpServers": {
    "mysql": {
      "command": "/ruta/a/mysql-mcp",
      "args": [],
      "env": {
        "MYSQL_HOST": "localhost",
        "MYSQL_PORT": "3306",
        "MYSQL_USER": "mcp_user",
        "MYSQL_PASSWORD": "tu_password_seguro",
        "MYSQL_DATABASE": "tu_base_datos",
        "LOG_PATH": "/ruta/a/mysql-mcp.log",
        "ALLOWED_TABLES": "users,orders,products,categories",
        "ALLOW_DDL": "false",
        "SAFETY_KEY": "PRODUCTION_CONFIRMED_2025"
      }
    }
  }
}
```

## Referencia de Variables de Entorno

| Variable | Obligatoria | Valor por Defecto | Descripción |
|----------|-------------|-------------------|-------------|
| `MYSQL_HOST` | **Sí** | `localhost` | Hostname o IP del servidor MySQL/MariaDB |
| `MYSQL_PORT` | No | `3306` | Número de puerto del servidor |
| `MYSQL_USER` | **Sí** | - | Nombre de usuario de la base de datos |
| `MYSQL_PASSWORD` | **Sí** | - | Contraseña de la base de datos |
| `MYSQL_DATABASE` | **Sí** | - | Base de datos por defecto a conectar |
| `LOG_PATH` | No | `mysql-mcp.log` | Ruta para el archivo de logs |
| `ALLOWED_TABLES` | No | *(todas)* | Lista de tablas permitidas separadas por coma |
| `ALLOW_DDL` | No | `false` | Habilitar operaciones CREATE, DROP, ALTER |
| `SAFETY_KEY` | No | `PRODUCTION_CONFIRMED_2025` | Clave de confirmación para operaciones masivas |

## Configuración Avanzada de Seguridad

### Restringir Acceso a Tablas Específicas

Limita las operaciones solo a ciertas tablas:

```json
"env": {
  "ALLOWED_TABLES": "users,orders,products,categories"
}
```

Cuando se configura, cualquier intento de acceder a tablas fuera de la lista será bloqueado.

### Deshabilitar Operaciones DDL

Bloquea todas las sentencias CREATE, DROP y ALTER:

```json
"env": {
  "ALLOW_DDL": "false"
}
```

:::caution
Con `ALLOW_DDL=false` (valor por defecto), el clasificador de verbos rechaza todos los verbos DDL. Para permitir DDL, ponlo a `true`. Aun con `ALLOW_DDL=true`, la gestión de privilegios (`GRANT`, `REVOKE`, `CREATE USER`, `DROP USER`, `SET PASSWORD`, `FLUSH PRIVILEGES`) y el acceso al sistema de archivos (`LOAD DATA`, `INTO OUTFILE`) siguen **siempre bloqueados** por la lista de verbos prohibidos.
:::

### Clave de Confirmación Personalizada

Cambia la clave de confirmación requerida para operaciones masivas (>100 filas):

```json
"env": {
  "SAFETY_KEY": "MI_CLAVE_PERSONALIZADA_2026"
}
```

## Configuración de Múltiples Bases de Datos

Puedes configurar múltiples servidores MySQL/MariaDB en Claude Desktop:

```json
{
  "mcpServers": {
    "mysql-produccion": {
      "command": "/ruta/a/mysql-mcp",
      "args": [],
      "env": {
        "MYSQL_HOST": "prod-db.ejemplo.com",
        "MYSQL_PORT": "3306",
        "MYSQL_USER": "usuario_readonly",
        "MYSQL_PASSWORD": "password_prod",
        "MYSQL_DATABASE": "produccion",
        "ALLOWED_TABLES": "users,orders"
      }
    },
    "mysql-desarrollo": {
      "command": "/ruta/a/mysql-mcp",
      "args": [],
      "env": {
        "MYSQL_HOST": "localhost",
        "MYSQL_PORT": "3306",
        "MYSQL_USER": "usuario_dev",
        "MYSQL_PASSWORD": "password_dev",
        "MYSQL_DATABASE": "desarrollo",
        "ALLOW_DDL": "true"
      }
    },
    "mariadb-analytics": {
      "command": "/ruta/a/mysql-mcp",
      "args": [],
      "env": {
        "MYSQL_HOST": "analytics.ejemplo.com",
        "MYSQL_PORT": "3306",
        "MYSQL_USER": "usuario_analytics",
        "MYSQL_PASSWORD": "password_analytics",
        "MYSQL_DATABASE": "analytics"
      }
    }
  }
}
```

:::note
Cada servidor aparecerá como un conjunto de herramientas separado en Claude. Puedes pedir a Claude que use uno específico, ej: "Usando mysql-produccion, muéstrame todos los pedidos de hoy".
:::

## Configuración con Docker

Si MySQL/MariaDB está ejecutándose en Docker:

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
Usa el **puerto mapeado** (ej. `3307`) si Docker expone el contenedor en un puerto diferente al `3306` por defecto.
:::

## Conexión a Base de Datos Remota

Para conectar a un servidor MySQL/MariaDB remoto:

```json
"env": {
  "MYSQL_HOST": "db.ejemplo.com",
  "MYSQL_PORT": "3306",
  "MYSQL_USER": "usuario_remoto",
  "MYSQL_PASSWORD": "password_seguro",
  "MYSQL_DATABASE": "base_produccion",
  "ALLOWED_TABLES": "users,orders"
}
```

:::tip
Para bases de datos de producción, usa **permisos de solo lectura** (solo SELECT) para prevenir modificaciones accidentales.
:::

## Paso 4: Verificar Configuración

1. **Guarda el archivo de configuración** y ciérralo
2. **Reinicia Claude Desktop** completamente (cerrar y volver a abrir)
3. **Abre una nueva conversación** con Claude
4. **Prueba la conexión** preguntando:
   - "¿Qué herramientas MySQL tienes disponibles?"
   - "Lista todas las tablas de mi base de datos"
   - "¿Qué versión de MySQL estoy usando?"

### Respuesta Esperada

Claude debería listar las 10 herramientas disponibles:
- `query`, `execute`, `tables`, `describe`, `views`, `indexes`, `explain`, `count`, `sample`, `database_info`

## Solución de Problemas

### Problemas de Conexión

| Error | Solución |
|-------|----------|
| "Connection refused" | Verifica que MySQL/MariaDB esté ejecutándose: `mysql -u mcp_user -p` |
| "Access denied" | Verifica usuario/contraseña y permisos del usuario |
| "Unknown database" | Confirma que la base de datos existe y el usuario tiene acceso |
| "Host not allowed" | Añade permiso de usuario para el host que se conecta |

### Comandos de Verificación

```bash
# Probar conexión MySQL directamente
mysql -h localhost -u mcp_user -p tu_base_datos

# Verificar si MySQL está escuchando
netstat -an | findstr 3306   # Windows
netstat -an | grep 3306      # macOS/Linux

# Ver logs del servidor MCP
type mysql-mcp.log           # Windows
cat mysql-mcp.log            # macOS/Linux
```

### Análisis de Logs

Si los problemas persisten, revisa el archivo de log en la ruta especificada en `LOG_PATH`:

```bash
# Ver últimas entradas del log
tail -f mysql-mcp.log

# Buscar errores
grep -i error mysql-mcp.log
```

:::tip
El archivo de log contiene información detallada sobre las queries ejecutadas, validaciones de seguridad y cualquier error encontrado.
:::
