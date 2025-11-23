package main

import (
	"fmt"
	"log"
	mysql "mcp-gp-mysql/internal"
)

func callClientMethod(client *mysql.Client, toolName string, arguments map[string]interface{}) (string, error) {
	switch toolName {
	// BÁSICAS
	case "query":
		if sql, ok := arguments["sql"].(string); ok {
			log.Printf("Ejecutando SQL: %s", sql)
			return client.ExecuteQuerySimple(sql)
		}
		return "", fmt.Errorf("parámetro 'sql' requerido")

	case "tables":
		log.Println("Listando tablas")
		return client.ListTablesSimple()

	case "describe":
		if name, ok := arguments["name"].(string); ok {
			log.Printf("Describiendo: %s", name)
			return client.DescribeSimple(name)
		}
		return "", fmt.Errorf("parámetro 'name' requerido")

	// VISTAS
	case "list_views":
		log.Println("Listando vistas")
		return client.ListViews(mysql.DBArgs{})

	case "view_definition":
		if name, ok := arguments["name"].(string); ok {
			log.Printf("Definición vista: %s", name)
			return client.ShowViewDefinition(mysql.ViewArgs{Name: name})
		}
		return "", fmt.Errorf("parámetro 'name' requerido")

	case "create_view":
		name, nameOk := arguments["name"].(string)
		query, queryOk := arguments["query"].(string)
		if !nameOk || !queryOk {
			return "", fmt.Errorf("parámetros 'name' y 'query' requeridos")
		}
		replace, _ := arguments["replace"].(bool)
		log.Printf("Creando vista: %s", name)
		return client.CreateView(mysql.CreateViewArgs{Name: name, Query: query, Replace: replace})

	case "drop_view":
		if name, ok := arguments["name"].(string); ok {
			log.Printf("Eliminando vista: %s", name)
			return client.DropView(mysql.ViewArgs{Name: name})
		}
		return "", fmt.Errorf("parámetro 'name' requerido")

	case "view_dependencies":
		if name, ok := arguments["name"].(string); ok {
			log.Printf("Dependencias vista: %s", name)
			return client.AnalyzeViewDependencies(mysql.ViewArgs{Name: name})
		}
		return "", fmt.Errorf("parámetro 'name' requerido")

	// ANÁLISIS
	case "explain_query":
		if query, ok := arguments["query"].(string); ok {
			explainType, _ := arguments["type"].(string)
			log.Printf("Explicando consulta: %s", query)
			return client.ExplainQuery(mysql.ExplainArgs{Query: query, Type: explainType})
		}
		return "", fmt.Errorf("parámetro 'query' requerido")

	case "analyze_object":
		if target, ok := arguments["target"].(string); ok {
			analysisType, _ := arguments["type"].(string)
			log.Printf("Analizando objeto: %s", target)
			return client.AnalyzeObject(mysql.AnalysisArgs{Target: target, Type: analysisType})
		}
		return "", fmt.Errorf("parámetro 'target' requerido")

	case "optimize_tables":
		if tablesArg, ok := arguments["tables"].([]interface{}); ok {
			var tables []string
			for _, t := range tablesArg {
				if table, ok := t.(string); ok {
					tables = append(tables, table)
				}
			}
			log.Printf("Optimizando tablas: %v", tables)
			return client.OptimizeTables(mysql.OptimizeArgs{Tables: tables})
		}
		return "", fmt.Errorf("parámetro 'tables' requerido")

	case "process_list":
		log.Println("Listando procesos")
		return client.ShowProcessList(mysql.DBArgs{})

	// OPERACIONES DE ESCRITURA
	case "show_safety_info":
		return fmt.Sprintf(`CONFIGURACIÓN DE SEGURIDAD:

🔑 Clave de confirmación: [oculta]
📊 Límite seguro: %d filas
	
📋 REGLAS:
• SELECT/SHOW/DESCRIBE → Siempre libre
• INSERT/UPDATE/DELETE ≤%d filas → Libre  
• INSERT/UPDATE/DELETE >%d filas → Requiere confirm_key
• CREATE/DROP/ALTER → Siempre requiere confirm_key
• DROP DATABASE/SCHEMA → Siempre bloqueado

💡 USO:
execute_write: "UPDATE tabla SET campo=valor WHERE id=123" (libre)
execute_write: "UPDATE tabla SET campo=valor" + confirm_key (requiere confirmación)
execute_ddl: "CREATE TABLE test (...)" + confirm_key (siempre requiere confirmación)`,
			MAX_SAFE_ROWS, MAX_SAFE_ROWS, MAX_SAFE_ROWS), nil

	case "execute_write":
		return executeWrite(client, arguments)

	case "execute_ddl":
		return executeDDL(client, arguments)

	// INFORMES
	case "create_report":
		log.Printf("=== INICIO create_report ===")
		if query, ok := arguments["query"].(string); ok {
			format, _ := arguments["format"].(string)
			limit, _ := arguments["limit"].(float64)
			log.Printf("Ejecutando create_report con query: %s", query)
			return client.GenerateReport(mysql.ReportArgs{
				Query:  query,
				Format: format,
				Limit:  int(limit),
			})
		}
		return "", fmt.Errorf("parámetro 'query' requerido")

	case "view_report":
		if viewName, ok := arguments["view_name"].(string); ok {
			include, _ := arguments["include"].(string)
			log.Printf("Informe vista: %s", viewName)
			return client.ViewReport(mysql.ViewReportArgs{ViewName: viewName, Include: include})
		}
		return "", fmt.Errorf("parámetro 'view_name' requerido")

	case "database_report":
		include, _ := arguments["include"].(string)
		log.Println("Informe base de datos")
		return client.DatabaseReport(mysql.DatabaseReportArgs{Include: include})

	default:
		return "", fmt.Errorf("herramienta '%s' no implementada", toolName)
	}
}
