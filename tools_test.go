package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestBasic(t *testing.T) {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}

	// Compile the test program
	programPath := filepath.Join(cwd, "testdata", "go", "helloworld")
	binaryPath := filepath.Join(programPath, "helloworld")

	// Remove old binary if exists
	os.Remove(binaryPath)

	// Compile with debugging flags
	cmd := exec.Command("go", "build", "-gcflags=all=-N -l", "-o", binaryPath, ".")
	cmd.Dir = programPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to compile program: %v\nOutput: %s", err, output)
	}
	defer os.Remove(binaryPath) // Clean up after test

	// Create MCP server
	implementation := mcp.Implementation{
		Name:    "mcp-dap-server",
		Version: "v1.0.0",
	}
	server := mcp.NewServer(&implementation, nil)
	registerTools(server)

	// Create httptest server
	getServer := func(request *http.Request) *mcp.Server {
		return server
	}
	sseHandler := mcp.NewSSEHandler(getServer)
	testServer := httptest.NewServer(sseHandler)
	defer testServer.Close()

	// Create MCP client
	clientImplementation := mcp.Implementation{
		Name:    "test-client",
		Version: "v1.0.0",
	}
	client := mcp.NewClient(&clientImplementation, nil)

	// Connect client to server
	ctx := context.Background()
	transport := mcp.NewSSEClientTransport(testServer.URL, &mcp.SSEClientTransportOptions{})
	session, err := client.Connect(ctx, transport)
	if err != nil {
		t.Fatalf("Failed to connect client to server: %v", err)
	}
	defer session.Close()

	// Execute tool calls
	// 1. Start debugger on port 9090
	startResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "start-debugger",
		Arguments: map[string]any{
			"port": "9090",
		},
	})
	if err != nil {
		t.Fatalf("Failed to start debugger: %v", err)
	}
	t.Logf("Start debugger result: %v", startResult)

	// Check if the result indicates an error
	if startResult.IsError {
		errorMsg := "Unknown error"
		if len(startResult.Content) > 0 {
			if textContent, ok := startResult.Content[0].(*mcp.TextContent); ok {
				errorMsg = textContent.Text
			}
		}
		t.Fatalf("Start debugger returned error: %s", errorMsg)
	}

	// Give debugger time to start
	time.Sleep(2 * time.Second)

	// 2. Execute program
	execResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "exec-program",
		Arguments: map[string]any{
			"path": binaryPath,
		},
	})
	if err != nil {
		t.Fatalf("Failed to execute program: %v", err)
	}
	t.Logf("Execute program result: %v", execResult)

	// Give program time to start
	time.Sleep(1 * time.Second)

	// 3. Set breakpoint
	breakpointFile := filepath.Join(cwd, "testdata", "go", "helloworld", "main.go")
	setBreakpointResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "set-breakpoints",
		Arguments: map[string]any{
			"file":  breakpointFile,
			"lines": []int{7},
		},
	})
	if err != nil {
		t.Fatalf("Failed to set breakpoint: %v", err)
	}
	t.Logf("Set breakpoint result: %v", setBreakpointResult)

	// 4. Continue execution
	continueResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "continue",
		Arguments: map[string]any{
			"threadID": 1,
		},
	})
	if err != nil {
		t.Fatalf("Failed to continue execution: %v", err)
	}
	t.Logf("Continue result: %v", continueResult)

	// Give time for breakpoint to hit
	time.Sleep(1 * time.Second)

	// 5. Evaluate expression
	evaluateResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "evaluate",
		Arguments: map[string]any{
			"expression": "greeting",
			"frameID":    1000,
			"context":    "repl",
		},
	})
	if err != nil {
		t.Fatalf("Failed to evaluate expression: %v", err)
	}
	t.Logf("Evaluate result: %v", evaluateResult)

	// Check if evaluate returned an error
	if evaluateResult.IsError {
		errorMsg := "Unknown error"
		if len(evaluateResult.Content) > 0 {
			if textContent, ok := evaluateResult.Content[0].(*mcp.TextContent); ok {
				errorMsg = textContent.Text
			}
		}
		t.Fatalf("Evaluate returned error: %s", errorMsg)
	}

	// Verify the evaluation result
	if len(evaluateResult.Content) == 0 {
		t.Fatalf("Expected evaluation result, got empty content")
	}

	// Check if the result contains "hello, world"
	resultStr := ""
	for _, content := range evaluateResult.Content {
		if textContent, ok := content.(*mcp.TextContent); ok {
			resultStr += textContent.Text
		}
	}

	if !strings.Contains(resultStr, "hello, world") {
		t.Errorf("Expected evaluation to contain 'hello, world', got: %s", resultStr)
	}

	// 6. Stop debugger
	stopResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "stop-debugger",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("Failed to stop debugger: %v", err)
	}
	t.Logf("Stop debugger result: %v", stopResult)
}
