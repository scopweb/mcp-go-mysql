---
title: Herramientas Disponibles
description: Las 10 herramientas especializadas para operaciones de base de datos
---

MCP Go MySQL proporciona **10 herramientas especializadas** para interactuar con tu base de datos.

## Herramientas de Lectura

### 1. query - Ejecutar Consultas SELECT

**Proposito:** Realizar consultas de lectura (SELECT) en la base de datos.

**Uso:** "Muestra los 10 usuarios mas recientes"

**Seguridad:** Validacion automatica contra SQL injection. Solo consultas SELECT.

```sql
SELECT * FROM users ORDER BY created_at DESC LIMIT 10
```

### 2. tables - Listar Tablas

**Proposito:** Obtener lista de todas las tablas con metadata.

**Uso:** "Que tablas hay en la base de datos?"

**Informacion:** Nombre, motor de almacenamiento, numero de filas, tamano.

### 3. describe - Describir Estructura

**Proposito:** Ver la estructura detallada de una tabla o vista.

**Uso:** "Describe la tabla users"

**Informacion:** Columnas, tipos de datos, claves, indices, restricciones.

### 4. views - Listar Vistas

**Proposito:** Mostrar todas las vistas de la base de datos.

**Uso:** "Lista las vistas disponibles"

**Informacion:** Nombre de vista y definicion SQL.

### 5. indexes - Ver Indices

**Proposito:** Mostrar indices de una tabla especifica.

**Uso:** "Que indices tiene la tabla orders?"

**Informacion:** Nombre del indice, columnas, tipo, unicidad.

### 6. count - Contar Filas

**Proposito:** Contar registros con condiciones opcionales.

**Uso:** "Cuenta usuarios activos"

```sql
SELECT COUNT(*) FROM users WHERE active = 1
```

### 7. sample - Obtener Muestra

**Proposito:** Obtener filas de ejemplo (maximo 100).

**Uso:** "Dame 5 ejemplos de productos"

**Limite:** Maximo 100 filas por seguridad.

## Herramientas de Escritura

### 8. execute - Ejecutar INSERT/UPDATE/DELETE

**Proposito:** Ejecutar operaciones de escritura con confirmacion.

**Uso:** "Actualiza el estado del pedido 123 a 'enviado'"

**Proteccion:**

- Operaciones pequenas (≤100 filas): Se ejecutan directamente
- Operaciones grandes (>100 filas): Requieren clave de confirmacion
- DELETE/UPDATE sin WHERE: Bloqueadas automaticamente

:::caution
Requiere confirmacion para operaciones masivas que afecten mas de 100 filas.
:::

## Herramientas de Analisis

### 9. explain - Analizar Plan de Ejecucion

**Proposito:** Analizar como MySQL ejecutara una consulta.

**Uso:** "Explica esta consulta: SELECT * FROM orders WHERE user_id = 123"

**Informacion:** Uso de indices, tipo de join, filas examinadas, costo.

```sql
EXPLAIN SELECT * FROM orders WHERE user_id = 123
```

### 10. database_info - Informacion del Servidor

**Proposito:** Obtener informacion de conexion y servidor.

**Uso:** "Que version de MySQL estoy usando?"

**Informacion:**

- Version de MySQL/MariaDB
- Base de datos actual
- Host y puerto
- Usuario conectado
- Charset y collation

## Ejemplos de Uso con Claude

| El usuario dice | Claude usa |
|-----------------|-----------|
| "Cuantos pedidos tenemos hoy?" | `count` con condicion de fecha |
| "Muestrame la estructura de la tabla products" | `describe` |
| "Actualiza el email del usuario ID 42 a nuevo@email.com" | `execute` (operacion pequena, sin confirmacion) |
| "Esta consulta es lenta, por que?" | `explain` para analizar el plan |

## Operaciones Bloqueadas

Por seguridad, estas operaciones estan **siempre bloqueadas**:

| Operacion | Estado |
|-----------|--------|
| `DROP DATABASE` / `DROP SCHEMA` | Bloqueada |
| `TRUNCATE TABLE` | Bloqueada |
| `DELETE FROM table` (sin WHERE) | Bloqueada |
| `UPDATE table SET` (sin WHERE) | Bloqueada |
| `INTO OUTFILE` / `DUMPFILE` | Bloqueada |
| `LOAD_FILE` / `LOAD DATA` | Bloqueada |
