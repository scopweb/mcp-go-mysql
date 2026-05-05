---
title: Herramientas Disponibles
description: Referencia de las 10 herramientas de base de datos que ofrece MCP Go MySQL
---

MCP Go MySQL expone 10 herramientas. Las de lectura cubren todo lo necesario para inspeccionar un esquema y traer datos; la única herramienta `execute` cubre las escrituras; `explain` y `database_info` cubren análisis y metadatos.

## Herramientas de lectura

### query — Ejecutar SELECT/WITH/SHOW

**Propósito:** ejecutar cualquier sentencia de solo lectura.

**Verbos aceptados:** `SELECT`, `WITH` (CTEs), `SHOW`, `DESCRIBE`, `EXPLAIN`, `USE`.

**Uso:** "Muéstrame los 10 usuarios más recientes."

```sql
SELECT * FROM users ORDER BY created_at DESC LIMIT 10
```

**Los conteos filtrados también van aquí:**

```sql
SELECT COUNT(*) FROM users WHERE active = 1
```

La herramienta `count` solo gestiona conteos sin filtro; cualquier cosa con `WHERE` debe ir por `query` para que pase por el clasificador de verbos y el detector de sentencias apiladas.

### tables — Listar tablas

**Propósito:** todas las tablas del esquema actual con metadatos.

**Devuelve:** nombre, tipo, motor de almacenamiento, número aproximado de filas, comentario.

**Uso:** "¿Qué tablas hay en la base de datos?"

### describe — Describir estructura

**Propósito:** mostrar las columnas de una tabla o vista.

**Devuelve:** nombre de columna, tipo, nullability, clave, valor por defecto, extra, comentario.

**Uso:** "Describe la tabla users."

Si `ALLOWED_TABLES` está configurado, esta herramienta rechazará tablas fuera de la lista blanca.

### views — Listar vistas

**Propósito:** todas las vistas del esquema actual con su definición.

**Uso:** "Lista las vistas disponibles."

### indexes — Mostrar índices

**Propósito:** todos los índices de una tabla dada.

**Devuelve:** nombre del índice, columna, secuencia, unicidad, cardinalidad.

**Uso:** "¿Qué índices tiene la tabla orders?"

Internamente usa una sentencia preparada, así que el nombre de la tabla no puede colar SQL.

### count — Contar filas

**Propósito:** conteo sin filtro de una tabla.

**Uso:** "¿Cuántas filas tiene la tabla users?"

```sql
SELECT COUNT(*) FROM users
```

Para conteos con filtro, usa `query` con `SELECT COUNT(*) FROM tabla WHERE ...`. Es deliberado: así el `WHERE` que escribe el llamante pasa por la misma validación que cualquier otro SELECT.

### sample — Filas de muestra

**Propósito:** primeras N filas de una tabla (por defecto 10, máximo 100).

**Uso:** "Dame 5 productos de ejemplo."

## Herramienta de escritura

### execute — Ejecutar INSERT/UPDATE/DELETE/REPLACE

**Propósito:** una única herramienta para todas las modificaciones de datos.

**Uso:** "Actualiza el estado del pedido 123 a 'enviado'."

```sql
UPDATE orders SET status = 'shipped' WHERE order_id = 123
```

**Umbral por filas afectadas:**

- Operaciones que afectan a **≤ `MAX_SAFE_ROWS`** filas (por defecto 100): se ejecutan directamente.
- Operaciones que afectan a **más** filas: se revierten salvo que pases `confirm_key` igual a `SAFETY_KEY`.

Esto cubre el caso típico de "uy, se me olvidó el `WHERE`". El clasificador en sí **no** rechaza `UPDATE`/`DELETE` sin `WHERE` — esa decisión se toma sobre el número real de filas, no sobre la sintaxis.

:::caution[El MCP confirma después de contar]
Un `DELETE FROM tabla_grande` se envía a la base de datos; las filas se localizan (no se confirma); si el conteo supera `MAX_SAFE_ROWS`, la operación falla. No hay "dry run" — quien cuenta es la base de datos. Asegúrate de que el usuario use un motor con rollback (InnoDB).
:::

## Herramientas de análisis

### explain — Plan de ejecución

**Propósito:** ver cómo MySQL/MariaDB ejecutará un SELECT.

**Uso:** "¿Por qué es lenta esta consulta?"

```sql
EXPLAIN SELECT * FROM orders WHERE user_id = 123
```

**Devuelve:** tipo de join, claves posibles, clave usada, filas examinadas, info adicional.

`explain` solo acepta sentencias SELECT.

### database_info — Metadatos del servidor

**Propósito:** información de conexión y servidor.

**Uso:** "¿A qué versión de MySQL estoy conectado?"

**Devuelve:** versión del servidor, comentario de versión, base de datos actual, usuario actual, hostname, puerto.

## Ejemplos de uso con Claude

| El usuario dice | Claude usa |
|-----------------|-----------|
| "¿Cuántos pedidos tenemos hoy?" | `query` con `SELECT COUNT(*) ... WHERE date = CURDATE()` |
| "Muéstrame la estructura de la tabla products" | `describe` |
| "Actualiza el email del usuario 42 a nuevo@email.com" | `execute` (una fila, sin confirmación) |
| "Pon todos los productos en oferta al 10% de descuento" | `execute` — si toca >100 filas, pide `confirm_key` |
| "Esta consulta es lenta, ¿por qué?" | `explain` |
| "¿A qué base de datos estoy conectado?" | `database_info` |

## Qué se rechaza

El clasificador de verbos se ejecuta sobre cada sentencia antes de llegar al driver. Ver la [página de seguridad](/es/security/overview/) para la categorización completa. Resumen rápido:

| Categoría | Ejemplos | Por qué |
|-----------|----------|---------|
| **Verbos prohibidos** | `GRANT`, `REVOKE`, `SET`, `FLUSH`, `RESET`, `KILL`, `SHUTDOWN`, `LOAD`, `HANDLER`, `INSTALL`, `LOCK` | Gestión de privilegios, acceso al sistema de archivos, control del servidor. Siempre rechazados. |
| **Cláusulas de filesystem** | `... INTO OUTFILE '/tmp/x'`, `... INTO DUMPFILE '/tmp/x'` | Escritura al sistema de archivos. Siempre rechazadas. |
| **Sentencias apiladas** | `SELECT 1; DROP DATABASE foo` | Múltiples sentencias en una llamada. Rechazadas. |
| **DDL** | `CREATE`, `DROP`, `ALTER`, `TRUNCATE`, `RENAME` | Rechazadas salvo que `ALLOW_DDL=true`. |
| **Verbo desconocido** | `FOOBAR users` | Solo lista blanca. Rechazado. |

Lo que **no** está en esta lista (deliberadamente): `SELECT SLEEP(1)`, `SELECT BENCHMARK(...)`, `SELECT EXTRACTVALUE(...)`. Son funciones SQL legítimas y el clasificador ya no las trata como caso especial.
