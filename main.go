package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	// Create MCP server
	implementation := mcp.Implementation{
		Name:    "mcp-dap-server",
		Version: "v1.0.0",
	}
	server := mcp.NewServer(&implementation, nil)
	registerTools(server)

	// Check transport mode from environment variable
	transportMode := os.Getenv("MCP_TRANSPORT")
	if transportMode == "" {
		transportMode = "sse" // Default to SSE for backward compatibility
	}

	switch transportMode {
	case "stdio":
		log.Println("Starting MCP server with stdio transport")
		stdioTransport := mcp.NewStdioTransport()
		err := server.Run(context.Background(), stdioTransport)
		if err != nil {
			log.Fatalf("Failed to serve stdio: %v", err)
		}
	case "sse":
		getServer := func(request *http.Request) *mcp.Server {
			return server
		}
		sseHandler := mcp.NewSSEHandler(getServer)

		// Get port from environment variable, default to 8080
		port := os.Getenv("PORT")
		if port == "" {
			port = "8080"
		}

		log.Printf("Starting MCP server with SSE transport on port :%s", port)
		err := http.ListenAndServe(":"+port, sseHandler)
		if err != nil {
			log.Fatalf("Failed to serve SSE: %v", err)
		}
	default:
		log.Fatalf("Unknown transport mode: %s. Supported modes: stdio, sse", transportMode)
	}
}
