---
title: Seguridad
description: Cómo MCP Go MySQL protege tu base de datos — y qué decide deliberadamente no hacer
---

El modelo de seguridad tiene **dos capas**, y solo dos. Versiones anteriores añadían más, pero la mayoría de esas comprobaciones duplicaban lo que la base de datos ya hace mejor, o protegían contra amenazas que no existen en este entorno. El modelo actual es pequeño, honesto y deliberado.

## Capa 1 — Privilegios de MySQL (primaria)

Esta es la frontera real. El servidor MCP se conecta con un usuario MySQL dedicado, y los privilegios de ese usuario deciden qué se puede hacer realmente. Un usuario sin el privilegio `FILE` no puede leer `/etc/passwd` por mucho que se reescriba el SQL. Un usuario sin `CREATE USER` no puede crear usuarios. Un usuario sin `GRANT OPTION` no puede dar privilegios a nadie.

Configura bien esta capa y casi todo lo demás es paranoia.

:::caution[Nunca uses root]
No conectes el MCP con `root` ni con ningún usuario que tenga `GRANT OPTION`, `FILE`, `CREATE USER` o `SUPER`. El clasificador (capa 2) es defensa en profundidad, no la frontera principal.
:::

### Privilegios recomendados

```sql
-- Usuario dedicado
CREATE USER 'mcp_user'@'%' IDENTIFIED BY 'password_seguro';

-- Solo lectura (más restrictivo)
GRANT SELECT, SHOW VIEW ON tu_base_datos.* TO 'mcp_user'@'%';

-- Lectura + escritura (más habitual)
GRANT SELECT, INSERT, UPDATE, DELETE, SHOW VIEW
  ON tu_base_datos.* TO 'mcp_user'@'%';

-- DDL (solo si realmente quieres que Claude modifique el esquema)
GRANT CREATE, ALTER, DROP, INDEX ON tu_base_datos.* TO 'mcp_user'@'%';

FLUSH PRIVILEGES;
```

## Capa 2 — Clasificador de verbos (defensa en profundidad)

Cada sentencia se analiza para extraer su verbo SQL inicial (después de quitar comentarios) y se compara contra una lista blanca. Cualquier cosa que no esté en la lista se rechaza antes de llegar al driver.

Esta es la capa que te protege cuando la capa 1 está mal configurada — por ejemplo, si alguien apunta el MCP a un usuario con demasiados privilegios por error.

### Categorías de verbos

| Categoría | Verbos | Comportamiento |
|-----------|--------|----------------|
| **Solo lectura** | `SELECT`, `WITH`, `SHOW`, `DESCRIBE`, `DESC`, `EXPLAIN`, `USE` | Permitidos. |
| **Escritura (DML)** | `INSERT`, `UPDATE`, `DELETE`, `REPLACE` | Permitidos. Sentencias que afectan a más de `MAX_SAFE_ROWS` filas requieren `confirm_key`. |
| **DDL** | `CREATE`, `DROP`, `ALTER`, `TRUNCATE`, `RENAME` | Rechazados salvo que `ALLOW_DDL=true`. |
| **Procedimientos** | `CALL`, `EXEC`, `EXECUTE` | Permitidos. |
| **Prohibidos** | `GRANT`, `REVOKE`, `SET`, `FLUSH`, `RESET`, `KILL`, `SHUTDOWN`, `LOAD`, `HANDLER`, `INSTALL`, `UNINSTALL`, `LOCK`, `UNLOCK` | **Siempre rechazados**, sin importar ningún flag. |
| **Desconocido** | cualquier otro | Rechazado. |

### Por qué lista blanca y no lista negra

Una lista negra debe enumerar cada forma peligrosa (y se le escapan las nuevas). Una lista blanca solo necesita enumerar los verbos aceptados; todo lo demás se bloquea por defecto. Mirar el **primer verbo** es inequívoco — no se puede confundir con la palabra `DROP` apareciendo dentro de una cadena o un nombre de columna.

### Comprobaciones extra sobre el verbo

Hay dos cláusulas que pueden colar comportamiento peligroso dentro de verbos legítimos, así que tienen comprobación explícita:

- **`INTO OUTFILE` / `INTO DUMPFILE`** — un `SELECT ... INTO OUTFILE` escribe al sistema de archivos. Estas cláusulas se rechazan donde aparezcan.
- **Sentencias apiladas** — una sola llamada al MCP debe contener una única sentencia. `SELECT 1; DROP DATABASE foo` se rechaza. El detector ignora los `;` dentro de literales de cadena o identificadores entre acentos graves.

### Umbral de filas afectadas

Un `UPDATE users SET x = 1` sin `WHERE` es *SQL válido*. El clasificador lo deja pasar. La sentencia se ejecuta dentro de una transacción explícita. Luego el MCP comprueba `RowsAffected()`. Si supera `MAX_SAFE_ROWS` (por defecto 100) y no se proporciona un `confirm_key` válido, la transacción se revierte antes del commit, por lo que los cambios nunca se hacen visibles.

Esto cubre el caso típico de "uy, se me olvidó el WHERE" sin intentar parsear el SQL — que es lo que intentaba la versión anterior con regex, y se equivocaba.

## Lo que el clasificador deliberadamente **no** hace

- **No** busca patrones tipo `SLEEP`, `BENCHMARK`, `EXTRACTVALUE`, etc. Son funciones SQL legítimas. La amenaza clásica de "time-based blind injection" asume que una aplicación está concatenando entrada de usuario en SQL — eso no es lo que ocurre aquí. El cliente del MCP es el LLM, y escribe sentencias completas directamente. Un `SELECT SLEEP(1)` para depurar es válido.
- **No** sanitiza los mensajes de error. El error del driver / la base de datos (`unknown column 'foo'`, `table 'x' doesn't exist`) es exactamente lo que el LLM necesita para corregir su consulta. Ocultarlo solo produce intentos siguientes peores.
- **No** aplica rate limiting. Un proceso stdio local con un humano usando un LLM no puede saturar una base de datos.

## Configuración

| Variable | Valor por defecto | Qué hace |
|----------|-------------------|----------|
| `SAFETY_KEY` | `PRODUCTION_CONFIRMED_2025` | Requerido para escrituras que afecten a más de `MAX_SAFE_ROWS` filas. Se registra una advertencia al arrancar si se deja en el valor por defecto. |
| `MAX_SAFE_ROWS` | `100` | Umbral por encima del cual se exige `confirm_key`. |
| `ALLOW_DDL` | `false` | Pon `true` para dejar pasar DDL por el clasificador. |
| `ALLOWED_TABLES` | vacío | Lista blanca separada por comas, aplicada en la herramienta `describe`. |

:::tip[Pon tu propia SAFETY_KEY]
El valor por defecto es público. Para cualquier uso no trivial, sustitúyelo:

```bash
export SAFETY_KEY=$(openssl rand -hex 16)
```
:::

## Ejemplos

### Permitido

```sql
SELECT * FROM products WHERE category = 'electronics' LIMIT 10
UPDATE orders SET status = 'shipped' WHERE order_id = 12345
WITH t AS (SELECT 1) SELECT * FROM t
SELECT SLEEP(1)              -- depuración legítima, permitido
```

### Permitido pero limitado

```sql
-- Afecta a más de MAX_SAFE_ROWS filas → exige confirm_key
UPDATE products SET discount = 0.1 WHERE category = 'clearance'

-- DELETE sin WHERE: también limitado por filas afectadas, no por sintaxis
DELETE FROM tabla_temporal
```

### Rechazado

```sql
-- Gestión de privilegios
GRANT ALL ON *.* TO 'evil'@'%'
CREATE USER 'foo'@'%' IDENTIFIED BY 'bar'
SET PASSWORD FOR 'root'@'localhost' = PASSWORD('x')
FLUSH PRIVILEGES

-- Sistema de archivos
LOAD DATA INFILE '/etc/passwd' INTO TABLE x
SELECT * FROM users INTO OUTFILE '/tmp/data'

-- Apiladas
SELECT 1; DROP DATABASE foo

-- DDL (salvo que ALLOW_DDL=true)
DROP TABLE users
ALTER TABLE users ADD COLUMN x INT

-- Verbo desconocido
FOOBAR users
```

## Validación

Los tests están en `test/security/integration_test.go`. Cubren cada categoría del clasificador: verbos permitidos (SQL legítimo debe pasar), verbos prohibidos (privilegios/sistema de archivos/control de servidor), gating de DDL, detección de sentencias apiladas, consultas precedidas por comentarios, y verbos desconocidos.

```bash
go test -v ./test/security/...
go test -bench=. ./test/security/...
```

El archivo `test/security/security_tests.go` además comprueba la integridad de `go.mod` / `go.sum` y la frescura de las dependencias.

## Escaneo de vulnerabilidades

```bash
govulncheck ./...
```

Ejecútalo periódicamente. Mantén actualizados Go y el driver de MySQL.
