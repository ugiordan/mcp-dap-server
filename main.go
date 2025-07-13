package main

import (
	"log"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	// Create MCP server
	implementation := mcp.Implementation{
		Name:    "mcp-dap-server",
		Version: "v1.0.0",
	}
	server := mcp.NewServer(&implementation, nil)
	getServer := func(request *http.Request) *mcp.Server {
		return server
	}

	registerTools(server)

	sseHandler := mcp.NewSSEHandler(getServer)

	log.Printf("listening on port :8080")
	http.ListenAndServe(":8080", sseHandler)
}
