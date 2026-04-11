---
title: Herramientas Disponibles
description: Las 10 herramientas especializadas para operaciones de base de datos
---

MCP Go MySQL proporciona **10 herramientas especializadas** para interactuar con tu base de datos.

## Herramientas de Lectura

### 1. query - Ejecutar Consultas SELECT

**Propósito:** Realizar consultas de lectura (SELECT) en la base de datos.

**Uso:** "Muestra los 10 usuarios más recientes"

**Seguridad:** Validación automática contra SQL injection. Solo consultas SELECT.

```sql
SELECT * FROM users ORDER BY created_at DESC LIMIT 10
```

### 2. tables - Listar Tablas

**Propósito:** Obtener lista de todas las tablas con metadata.

**Uso:** "¿Qué tablas hay en la base de datos?"

**Información:** Nombre, motor de almacenamiento, número de filas, tamaño.

### 3. describe - Describir Estructura

**Propósito:** Ver la estructura detallada de una tabla o vista.

**Uso:** "Describe la tabla users"

**Información:** Columnas, tipos de datos, claves, índices, restricciones.

### 4. views - Listar Vistas

**Propósito:** Mostrar todas las vistas de la base de datos.

**Uso:** "Lista las vistas disponibles"

**Información:** Nombre de vista y definición SQL.

### 5. indexes - Ver Índices

**Propósito:** Mostrar índices de una tabla específica.

**Uso:** "¿Qué índices tiene la tabla orders?"

**Información:** Nombre del índice, columnas, tipo, unicidad.

### 6. count - Contar Filas

**Propósito:** Contar registros con condiciones opcionales.

**Uso:** "Cuenta usuarios activos"

```sql
SELECT COUNT(*) FROM users WHERE active = 1
```

### 7. sample - Obtener Muestra

**Propósito:** Obtener filas de ejemplo (máximo 100).

**Uso:** "Dame 5 ejemplos de productos"

**Límite:** Máximo 100 filas por seguridad.

## Herramientas de Escritura

### 8. execute - Ejecutar INSERT/UPDATE/DELETE

**Propósito:** Ejecutar operaciones de escritura con confirmación.

**Uso:** "Actualiza el estado del pedido 123 a 'enviado'"

**Protección:**

- Operaciones pequeñas (≤100 filas): Se ejecutan directamente
- Operaciones grandes (>100 filas): Requieren clave de confirmación
- DELETE/UPDATE sin WHERE: Bloqueadas automáticamente

:::caution
Requiere confirmación para operaciones masivas que afecten más de 100 filas.
:::

## Herramientas de Análisis

### 9. explain - Analizar Plan de Ejecución

**Propósito:** Analizar cómo MySQL ejecutará una consulta.

**Uso:** "Explica esta consulta: SELECT * FROM orders WHERE user_id = 123"

**Información:** Uso de índices, tipo de join, filas examinadas, costo.

```sql
EXPLAIN SELECT * FROM orders WHERE user_id = 123
```

### 10. database_info - Información del Servidor

**Propósito:** Obtener información de conexión y servidor.

**Uso:** "¿Qué versión de MySQL estoy usando?"

**Información:**

- Versión de MySQL/MariaDB
- Base de datos actual
- Host y puerto
- Usuario conectado
- Charset y collation

## Ejemplos de Uso con Claude

| El usuario dice | Claude usa |
|-----------------|-----------|
| "¿Cuántos pedidos tenemos hoy?" | `count` con condición de fecha |
| "Muéstrame la estructura de la tabla products" | `describe` |
| "Actualiza el email del usuario ID 42 a nuevo@email.com" | `execute` (operación pequeña, sin confirmación) |
| "Esta consulta es lenta, ¿por qué?" | `explain` para analizar el plan |

## Operaciones Bloqueadas

Por seguridad, estas operaciones están **siempre bloqueadas**:

| Operación | Estado |
|-----------|--------|
| `DROP DATABASE` / `DROP SCHEMA` | Bloqueada |
| `TRUNCATE TABLE` | Bloqueada |
| `DELETE FROM table` (sin WHERE) | Bloqueada |
| `UPDATE table SET` (sin WHERE) | Bloqueada |
| `INTO OUTFILE` / `DUMPFILE` | Bloqueada |
| `LOAD_FILE` / `LOAD DATA` | Bloqueada |
