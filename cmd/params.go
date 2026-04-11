package main

import "fmt"

// Parameter extraction helpers for MCP tool arguments

// getStringArg extracts a required string argument
func getStringArg(args map[string]interface{}, key string) (string, error) {
	v, ok := args[key].(string)
	if !ok || v == "" {
		return "", fmt.Errorf("missing or invalid '%s' parameter", key)
	}
	return v, nil
}

// getOptionalString extracts an optional string argument with default
func getOptionalString(args map[string]interface{}, key, defaultVal string) string {
	if v, ok := args[key].(string); ok && v != "" {
		return v
	}
	return defaultVal
}

// getIntArg extracts an integer argument with default (JSON numbers are float64)
func getIntArg(args map[string]interface{}, key string, defaultVal int) int {
	if v, ok := args[key].(float64); ok {
		return int(v)
	}
	return defaultVal
}

// getIntArgClamped extracts an integer clamped to min/max range
func getIntArgClamped(args map[string]interface{}, key string, defaultVal, minVal, maxVal int) int {
	v := getIntArg(args, key, defaultVal)
	if v < minVal {
		return minVal
	}
	if v > maxVal {
		return maxVal
	}
	return v
}
