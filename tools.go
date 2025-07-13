package main

import (
	"context"
	"os"
	"os/exec"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var cmd *exec.Cmd

func registerTools(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "start-debugger",
		Description: "Starts a debugger exposed via a DAP server. You can provide the port you would like the debugger DAP server to listen on.",
	}, startDebugger)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "stop-debugger",
		Description: "Stops an already running debugger.",
	}, stopDebugger)
}

type StartDebuggerParams struct {
	Port string `json:"port" mcp:"the port for the DAP server to listen on"`
}

func startDebugger(ctx context.Context, _ *mcp.ServerSession, params *mcp.CallToolParamsFor[StartDebuggerParams]) (*mcp.CallToolResultFor[any], error) {
	port := params.Arguments.Port
	if !strings.HasPrefix(port, ":") {
		port = ":" + port
	}
	cmd = exec.Command("dlv", "dap", "--listen", port)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{&mcp.TextContent{Text: "Started debugger at: " + port}},
	}, nil
}

type StopDebuggerParams struct {
}

func stopDebugger(ctx context.Context, _ *mcp.ServerSession, _ *mcp.CallToolParamsFor[StopDebuggerParams]) (*mcp.CallToolResultFor[any], error) {
	if cmd == nil {
		return &mcp.CallToolResultFor[any]{
			Content: []mcp.Content{&mcp.TextContent{Text: "No debugger currently executing."}},
		}, nil
	}
	if err := cmd.Process.Kill(); err != nil {
		return nil, err
	}
	if err := cmd.Wait(); err != nil {
		return nil, err
	}
	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{&mcp.TextContent{Text: "Debugger stopped."}},
	}, nil
}
