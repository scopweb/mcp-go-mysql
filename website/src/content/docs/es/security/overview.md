---
title: Seguridad
description: Seguridad de calidad empresarial con 6 capas de protección
---

MCP Go MySQL implementa seguridad de nivel empresarial con 6 capas de protección.

## Características de Seguridad

| Fase | Componente | Estado |
|------|-----------|--------|
| 1 | Security Hardening | **Completa** |
| 2 | Database Compatibility | **Completa** |
| 3.1 | Timeout Management | **Completa** |
| 3.2 | Audit Logging | **Completa** |
| 3.3 | Rate Limiting | **Completa** |
| 3.4 | Error Sanitization | **Completa** |

## Fase 1: Security Hardening

### Protección contra SQL Injection

Detecta y bloquea **23+ patrones** de inyección SQL:

- Inyección clásica: `' OR '1'='1`
- UNION-based: `UNION SELECT`
- Comentarios: `--`, `#`, `/* */`
- Consultas apiladas: `;`
- Blind injection: `SLEEP()`, `BENCHMARK()`
- Codificación hexadecimal
- Funciones MySQL: `EXTRACTVALUE`, `UPDATEXML`

### Bloqueo de Operaciones Peligrosas

| Operación | Estado |
|-----------|--------|
| `DROP DATABASE` | **Bloqueada** |
| `TRUNCATE TABLE` | **Bloqueada** |
| `DELETE` sin WHERE | **Bloqueada** |
| `UPDATE` sin WHERE | **Bloqueada** |
| `INTO OUTFILE` | **Bloqueada** |
| `LOAD_FILE` | **Bloqueada** |

### Protección Path Traversal

Previene acceso no autorizado a archivos del sistema:

- `../../../etc/passwd` &rarr; Bloqueado
- `..\..\windows\system32` &rarr; Bloqueado
- Rutas absolutas no autorizadas &rarr; Bloqueadas
- URL encoding &rarr; Detectado y bloqueado

### Evaluación Inteligente de Riesgo

- **Operaciones pequeñas** (≤100 filas): Ejecutan libremente
- **Operaciones grandes** (>100 filas): Requieren confirmación
- **Operaciones DDL**: Siempre requieren confirmación

### Protección con Safety Key

La variable de entorno `SAFETY_KEY` protege operaciones destructivas (DROP, TRUNCATE, DELETE sin WHERE).

:::caution[Safety Key por Defecto]
Si `SAFETY_KEY` no está configurada, el servidor usa `PRODUCTION_CONFIRMED_2025` por defecto y registra una advertencia. Para entornos de producción, siempre configura una clave única:
```bash
export SAFETY_KEY=$(openssl rand -hex 16)
```
:::

Al ejecutar operaciones masivas (>100 filas) o sentencias destructivas, el cliente MCP debe proporcionar esta clave para confirmar la operación.

## Fase 3.1: Timeout Management

### Perfiles de Timeout

| Perfil | Timeout | Uso |
|--------|---------|-----|
| Query | 30 segundos | Consultas SELECT rápidas |
| Long Query | 5 minutos | Consultas complejas |
| Write | 2 minutos | INSERT, UPDATE, DELETE |
| Admin | 10 minutos | Operaciones DDL |
| Connection | 15 segundos | Establecer conexión |

**Beneficios:**

- Previene consultas que se ejecutan indefinidamente
- Libera recursos automáticamente
- Mejora la estabilidad del sistema

## Fase 3.2: Audit Logging

Registro detallado de todas las operaciones:

### Información Registrada

- Timestamp de la operación
- Usuario que ejecutó la operación
- Tipo de operación (SELECT, INSERT, UPDATE, DELETE, DDL)
- Consulta SQL ejecutada (sanitizada)
- Resultado (exito/error)
- Tiempo de ejecución
- Filas afectadas

### Categorias de Eventos

| Categoria | Severidad |
|-----------|-----------|
| Query Success | **Info** |
| Write Operation | **Warning** |
| Security Violation | **Critical** |
| Connection Error | **Error** |

:::note
Los logs son esenciales para auditorías de seguridad y troubleshooting. Configura la variable de entorno `LOG_PATH` para habilitar el registro de auditoría.
:::

## Fase 3.3: Rate Limiting

### Algoritmo Token Bucket

Implementación de algoritmo de cubetas de tokens para control de tasa:

| Tipo de Operación | Límite | Propósito |
|-------------------|--------|-----------|
| Queries (SELECT) | 1,000/segundo | Prevenir saturación de consultas |
| Writes (INSERT/UPDATE/DELETE) | 100/segundo | Proteger integridad de datos |
| Admin (DDL) | 10/segundo | Controlar cambios estructurales |

### Proteccion contra Ataques

- **DoS Prevention:** Limita consultas/escrituras masivas
- **Cascade Prevention:** Evita fallos en cascada
- **Fairness:** Distribucion equitativa de recursos
- **High Throughput:** Soporta 10,000+ ops/segundo

**Performance:** Overhead < 1 microsegundo por operacion.

## Fase 3.4: Error Sanitization

### Protección de Información Sensible

Los errores se sanitizan automáticamente antes de mostrarlos:

- Direcciones IP (IPv4/IPv6)
- Rutas de archivos del sistema
- Nombres de base de datos
- Nombres de host
- Números de puerto
- Patrones de consultas SQL

### Ejemplo de Sanitización

:::danger[Error Original (interno)]
```
Error conectando a 192.168.1.100:3306, database 'production_db' en /var/lib/mysql/data
```
:::

:::tip[Error Sanitizado (cliente)]
```
Error de conexion a la base de datos. Codigo: DB_CONN_001
```
:::

### Categorias de Error

| Categoria | Codigo | Ejemplo |
|-----------|--------|---------|
| User Error | USR_* | Error de sintaxis SQL |
| System Error | SYS_* | Error interno del servidor |
| Network Error | NET_* | Fallo de conexion |
| Auth Error | AUTH_* | Credenciales incorrectas |
| Timeout Error | TO_* | Operación expiró |

## Validación de Seguridad

### Tests Implementados

| Categoria | Tests | Estado |
|-----------|-------|--------|
| SQL Injection | 23 patrones | **100%** |
| Path Traversal | 9 patrones | **100%** |
| Command Injection | 10 patrones | **100%** |
| Dangerous SQL | 9 operaciones | **100%** |
| Client Validation | 22 casos | **100%** |

**Total:** 170 tests, 100% aprobación.

## Cobertura CWE

| CWE | Descripción | Protección |
|-----|-------------|------------|
| CWE-89 | SQL Injection | **Protegido** |
| CWE-22 | Path Traversal | **Protegido** |
| CWE-78 | Command Injection | **Protegido** |
| CWE-287 | Improper Authentication | **Protegido** |
| CWE-311 | Missing Encryption | **TLS Soportado** |
| CWE-522 | Credential Protection | **Protegido** |
| CWE-400 | Resource Consumption | **Rate Limiting** |

## Mejores Prácticas

1. **Nunca uses el usuario root** para conexiones MCP
2. **Crea usuarios dedicados** con permisos mínimos necesarios
3. **Usa ALLOWED_TABLES** para restringir acceso en producción
4. **Habilita logs de auditoría** y revísalos periódicamente
5. **Ejecuta govulncheck** regularmente para detectar vulnerabilidades
6. **Mantén Go actualizado** a la última versión estable
7. **Usa TLS/SSL** para conexiones a bases de datos remotas
8. **Ajusta rate limiting** según tu caso de uso
9. **Revisa errores sanitizados** en los logs internos
10. **Haz backups** antes de operaciones de escritura importantes

## Escaneo de Vulnerabilidades

**Estado actual:** 0 vulnerabilidades detectadas.

Ejecutar escaneo manual:

```bash
govulncheck ./...
```

**Última actualización:** Go 1.24.12 (2026-02-01)
