package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/google/go-dap"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var (
	cmd    *exec.Cmd
	client *DAPClient
)

// registerTools registers the debugger tools with the MCP server.
// It adds two tools: start-debugger for starting a DAP server and stop-debugger for stopping it.
func registerTools(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "start-debugger",
		Description: "Starts a debugger exposed via a DAP server. You can provide the port you would like the debugger DAP server to listen on.",
	}, startDebugger)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "stop-debugger",
		Description: "Stops an already running debugger.",
	}, stopDebugger)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "debug-program",
		Description: "Tells the debugger running via DAP to debug a local program.",
	}, debugProgram)
}

// StartDebuggerParams defines the parameters for starting a debugger.
type StartDebuggerParams struct {
	Port string `json:"port" mcp:"the port for the DAP server to listen on"`
}

// startDebugger starts a debugger DAP server on the specified port.
// It launches the delve debugger in DAP mode and configures it to listen on the given port.
// If the port doesn't start with ":", it will be prefixed automatically.
func startDebugger(ctx context.Context, _ *mcp.ServerSession, params *mcp.CallToolParamsFor[StartDebuggerParams]) (*mcp.CallToolResultFor[any], error) {
	port := params.Arguments.Port
	if !strings.HasPrefix(port, ":") {
		port = ":" + port
	}
	cmd = exec.Command("dlv", "dap", "--listen", port, "--log", "--log-output", "dap")
	cmd.Stderr = os.Stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	r := bufio.NewReader(stdout)
	for {
		s, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		// Check if server has started
		if strings.HasPrefix(s, "DAP server listening at") {
			break
		}
	}

	client = newDAPClient("localhost" + port)
	if err := client.InitializeRequest(); err != nil {
		return nil, err
	}
	// TODO(deparker): read response to discovery server capabilities
	_, err = client.ReadMessage()
	if err != nil {
		return nil, err
	}

	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{&mcp.TextContent{Text: "Started debugger at: " + port}},
	}, nil
}

// StopDebuggerParams defines the parameters for stopping a debugger.
// Currently no parameters are needed to stop the debugger.
type StopDebuggerParams struct {
}

// stopDebugger stops the currently running debugger process.
// It kills the debugger process and waits for it to exit.
// If no debugger is running, it returns a message indicating this.
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

type DebugProgramParams struct {
	Path string `json:"path" mcp:"path to the program we want to start debugging."`
}

func debugProgram(ctx context.Context, _ *mcp.ServerSession, params *mcp.CallToolParamsFor[DebugProgramParams]) (*mcp.CallToolResultFor[any], error) {
	path := params.Arguments.Path
	if err := client.LaunchRequest("debug", path, true); err != nil {
		return nil, err
	}
	msg, err := client.ReadMessage()
	if err != nil {
		return nil, err
	}
	switch resp := msg.(type) {
	case *dap.LaunchResponse:
		if !resp.Success {
			return nil, fmt.Errorf("unable to launch program to debug via DAP server: %s", resp.Message)
		}
	case *dap.ErrorResponse:
		return nil, fmt.Errorf("unable to launch program to debug via DAP server: %s", resp.Message)
	}
	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{&mcp.TextContent{Text: "Started debugging: " + path}},
	}, nil
}
