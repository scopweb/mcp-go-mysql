package main

// Version constants for the MCP MySQL server
const (
	Version     = "2.0.3"
	ServerName  = "mysql-mcp-advanced"
	JSONRPCVer  = "2.0"
)

// SupportedProtocolVersions lists the MCP protocol versions this server supports.
// Order: newest first. The server will negotiate the best match with the client.
var SupportedProtocolVersions = []string{
	"2025-11-25",
	"2025-03-26",
	"2024-11-05",
}

// LatestProtocolVersion is the most recent protocol version supported.
var LatestProtocolVersion = SupportedProtocolVersions[0]

// isSupportedProtocolVersion checks if a given version is in the supported list.
func isSupportedProtocolVersion(version string) bool {
	for _, v := range SupportedProtocolVersions {
		if v == version {
			return true
		}
	}
	return false
}

// Tool configuration constants
const (
	MaxSampleRows = 100
	DefaultLimit  = 10
	MinLimit      = 1
)
