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

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// testSetup holds the common test infrastructure
type testSetup struct {
	cwd        string
	binaryPath string
	server     *mcp.Server
	testServer *httptest.Server
	client     *mcp.Client
	session    *mcp.ClientSession
	ctx        context.Context
}

// compileTestProgram compiles the test Go program and returns the binary path
func compileTestProgram(t *testing.T, cwd, name string) (binaryPath string, cleanup func()) {
	t.Helper()

	programPath := filepath.Join(cwd, "testdata", "go", name)
	binaryPath = filepath.Join(programPath, "debugprog")

	// Remove old binary if exists
	os.Remove(binaryPath)

	// Compile with debugging flags
	cmd := exec.Command("go", "build", "-gcflags=all=-N -l", "-o", binaryPath, ".")
	cmd.Dir = programPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to compile program: %v\nOutput: %s", err, output)
	}

	cleanup = func() {
		os.Remove(binaryPath)
	}

	return binaryPath, cleanup
}

// setupMCPServerAndClient creates and connects MCP server and client
func setupMCPServerAndClient(t *testing.T) *testSetup {
	t.Helper()

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}

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

	return &testSetup{
		cwd:        cwd,
		server:     server,
		testServer: testServer,
		client:     client,
		session:    session,
		ctx:        ctx,
	}
}

// cleanup closes all resources
func (ts *testSetup) cleanup() {
	if ts.session != nil {
		ts.session.Close()
	}
	if ts.testServer != nil {
		ts.testServer.Close()
	}
}

// startDebuggerAndExecuteProgram starts the debugger and executes the test program
func (ts *testSetup) startDebuggerAndExecuteProgram(t *testing.T, port string, binaryPath string) {
	t.Helper()

	// Start debugger
	startResult, err := ts.session.CallTool(ts.ctx, &mcp.CallToolParams{
		Name: "start-debugger",
		Arguments: map[string]any{
			"port": port,
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

	// Execute program
	execResult, err := ts.session.CallTool(ts.ctx, &mcp.CallToolParams{
		Name: "exec-program",
		Arguments: map[string]any{
			"path": binaryPath,
		},
	})
	if err != nil {
		t.Fatalf("Failed to execute program: %v", err)
	}
	t.Logf("Execute program result: %v", execResult)
}

// setBreakpointAndContinue sets a breakpoint and continues execution
func (ts *testSetup) setBreakpointAndContinue(t *testing.T, file string, line int) {
	t.Helper()

	// Set breakpoint
	setBreakpointResult, err := ts.session.CallTool(ts.ctx, &mcp.CallToolParams{
		Name: "set-breakpoints",
		Arguments: map[string]any{
			"file":  file,
			"lines": []int{line},
		},
	})
	if err != nil {
		t.Fatalf("Failed to set breakpoint: %v", err)
	}
	t.Logf("Set breakpoint result: %v", setBreakpointResult)

	// Continue execution
	continueResult, err := ts.session.CallTool(ts.ctx, &mcp.CallToolParams{
		Name: "continue",
		Arguments: map[string]any{
			"threadID": 1,
		},
	})
	if err != nil {
		t.Fatalf("Failed to continue execution: %v", err)
	}
	t.Logf("Continue result: %v", continueResult)
}

// getStackTraceContent gets stacktrace and returns the content as a string
func (ts *testSetup) getStackTraceContent(t *testing.T) string {
	t.Helper()

	stacktraceResult, err := ts.session.CallTool(ts.ctx, &mcp.CallToolParams{
		Name: "stack-trace",
		Arguments: map[string]any{
			"threadID": 1,
		},
	})
	if err != nil {
		t.Fatalf("Failed to get stacktrace: %v", err)
	}
	t.Logf("Stacktrace result: %v", stacktraceResult)

	// Check if stacktrace returned an error
	if stacktraceResult.IsError {
		errorMsg := "Unknown error"
		if len(stacktraceResult.Content) > 0 {
			if textContent, ok := stacktraceResult.Content[0].(*mcp.TextContent); ok {
				errorMsg = textContent.Text
			}
		}
		t.Fatalf("Stacktrace returned error: %s", errorMsg)
	}

	// Verify we got stack frames
	if len(stacktraceResult.Content) == 0 {
		t.Fatalf("Expected stacktrace frames, got empty content")
	}

	// Extract stacktrace content
	stacktraceStr := ""
	for _, content := range stacktraceResult.Content {
		if textContent, ok := content.(*mcp.TextContent); ok {
			stacktraceStr += textContent.Text
		}
	}

	return stacktraceStr
}

// stopDebugger stops the debugger
func (ts *testSetup) stopDebugger(t *testing.T) {
	t.Helper()

	stopResult, err := ts.session.CallTool(ts.ctx, &mcp.CallToolParams{
		Name:      "stop-debugger",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("Failed to stop debugger: %v", err)
	}
	t.Logf("Stop debugger result: %v", stopResult)
}

func TestBasic(t *testing.T) {
	// Setup test infrastructure
	ts := setupMCPServerAndClient(t)
	defer ts.cleanup()

	// Compile test program
	binaryPath, cleanupBinary := compileTestProgram(t, ts.cwd, "helloworld")
	defer cleanupBinary()

	// Start debugger and execute program
	ts.startDebuggerAndExecuteProgram(t, "9090", binaryPath)

	// Set breakpoint and continue
	f := filepath.Join(ts.cwd, "testdata", "go", "helloworld", "main.go")
	ts.setBreakpointAndContinue(t, f, 7)

	// Get stacktrace
	stacktraceStr := ts.getStackTraceContent(t)

	// Verify stacktrace contains expected information
	if !strings.Contains(stacktraceStr, "main.main") {
		t.Errorf("Expected stacktrace to contain 'main.main', got: %s", stacktraceStr)
	}

	if !strings.Contains(stacktraceStr, "main.go") {
		t.Errorf("Expected stacktrace to contain 'main.go', got: %s", stacktraceStr)
	}

	// Evaluate expression
	evaluateResult, err := ts.session.CallTool(ts.ctx, &mcp.CallToolParams{
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

	// Stop debugger
	ts.stopDebugger(t)
}

func TestRestart(t *testing.T) {
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		t.Skip("Skipping test in Github CI: relies on unreleased feature of Delve DAP server.")
	}
	// Setup test infrastructure
	ts := setupMCPServerAndClient(t)
	defer ts.cleanup()

	// Compile test program
	binaryPath, cleanupBinary := compileTestProgram(t, ts.cwd, "restart")
	defer cleanupBinary()

	// Start debugger and execute program
	ts.startDebuggerAndExecuteProgram(t, "9092", binaryPath)

	// Set breakpoint and continue
	f := filepath.Join(ts.cwd, "testdata", "go", "restart", "main.go")
	ts.setBreakpointAndContinue(t, f, 15)

	// Restart debugger
	restartResult, err := ts.session.CallTool(ts.ctx, &mcp.CallToolParams{
		Name: "restart",
		Arguments: map[string]any{
			"args": []string{"me, its me again"},
		},
	})
	if err != nil {
		t.Fatalf("Failed to restart debugger: %v", err)
	}
	t.Logf("Restart result: %v", restartResult)

	// Check if restart returned an error
	if restartResult.IsError {
		errorMsg := "Unknown error"
		if len(restartResult.Content) > 0 {
			if textContent, ok := restartResult.Content[0].(*mcp.TextContent); ok {
				errorMsg = textContent.Text
			}
		}
		t.Fatalf("Restart returned error: %s", errorMsg)
	}

	// Continue to hit the breakpoint again
	continueResult, err := ts.session.CallTool(ts.ctx, &mcp.CallToolParams{
		Name: "continue",
		Arguments: map[string]any{
			"threadID": 1,
		},
	})
	if err != nil {
		t.Fatalf("Failed to continue after restart: %v", err)
	}
	t.Logf("Continue after restart result: %v", continueResult)

	// Get stacktrace again to verify we're at the breakpoint after restart
	stacktraceStr2 := ts.getStackTraceContent(t)
	if !strings.Contains(stacktraceStr2, "main.go:15") {
		t.Errorf("Expected to be at breakpoint main.go:15 after restart, got: %s", stacktraceStr2)
	}

	// Evaluate greeting variable again to ensure it's a fresh run
	evaluateResult2, err := ts.session.CallTool(ts.ctx, &mcp.CallToolParams{
		Name: "evaluate",
		Arguments: map[string]any{
			"expression": "greeting",
			"frameID":    1000,
			"context":    "repl",
		},
	})
	if err != nil {
		t.Fatalf("Failed to evaluate expression after restart: %v", err)
	}
	t.Logf("Evaluate after restart result: %v", evaluateResult2)

	// Verify the evaluation result still contains "hello, world"
	resultStr := ""
	for _, content := range evaluateResult2.Content {
		if textContent, ok := content.(*mcp.TextContent); ok {
			resultStr += textContent.Text
		}
	}

	if !strings.Contains(resultStr, "hello me, its me again") {
		t.Errorf("Expected evaluation after restart to contain 'hello, world', got: %s", resultStr)
	}

	// Stop debugger
	ts.stopDebugger(t)
}

func TestStacktrace(t *testing.T) {
	// Setup test infrastructure
	ts := setupMCPServerAndClient(t)
	defer ts.cleanup()

	// Compile test program
	binaryPath, cleanupBinary := compileTestProgram(t, ts.cwd, "helloworld")
	defer cleanupBinary()

	// Start debugger and execute program
	ts.startDebuggerAndExecuteProgram(t, "9091", binaryPath)

	// Set breakpoint and continue
	f := filepath.Join(ts.cwd, "testdata", "go", "helloworld", "main.go")
	ts.setBreakpointAndContinue(t, f, 7)

	// Get stacktrace
	stacktraceStr := ts.getStackTraceContent(t)

	t.Logf("Stacktrace output:\n%s", stacktraceStr)

	// Verify stacktrace contains expected information
	if !strings.Contains(stacktraceStr, "Stack trace for thread 1:") {
		t.Errorf("Expected stacktrace header, got: %s", stacktraceStr)
	}

	if !strings.Contains(stacktraceStr, "main.main") {
		t.Errorf("Expected stacktrace to contain 'main.main', got: %s", stacktraceStr)
	}

	if !strings.Contains(stacktraceStr, "main.go:7") {
		t.Errorf("Expected stacktrace to contain 'main.go:7' (breakpoint location), got: %s", stacktraceStr)
	}

	if !strings.Contains(stacktraceStr, "(runtime)") {
		t.Errorf("Expected stacktrace to contain runtime frames marked with '(runtime)', got: %s", stacktraceStr)
	}

	if !strings.Contains(stacktraceStr, "Total frames:") {
		t.Errorf("Expected stacktrace to contain 'Total frames:', got: %s", stacktraceStr)
	}

	// Stop debugger
	ts.stopDebugger(t)
}

func TestScopes(t *testing.T) {
	// Setup test infrastructure
	ts := setupMCPServerAndClient(t)
	defer ts.cleanup()

	// Compile test program
	binaryPath, cleanupBinary := compileTestProgram(t, ts.cwd, "helloworld")
	defer cleanupBinary()

	// Start debugger and execute program
	ts.startDebuggerAndExecuteProgram(t, "9093", binaryPath)

	// Set breakpoint and continue
	f := filepath.Join(ts.cwd, "testdata", "go", "helloworld", "main.go")
	ts.setBreakpointAndContinue(t, f, 7)

	// Get stacktrace first to ensure we have valid frame IDs
	stacktraceResult, err := ts.session.CallTool(ts.ctx, &mcp.CallToolParams{
		Name: "stack-trace",
		Arguments: map[string]any{
			"threadID": 1,
		},
	})
	if err != nil {
		t.Fatalf("Failed to get stacktrace: %v", err)
	}
	t.Logf("Stacktrace result: %v", stacktraceResult)

	// Test getting scopes for frame ID 1000 (the topmost frame)
	scopesResult, err := ts.session.CallTool(ts.ctx, &mcp.CallToolParams{
		Name: "scopes",
		Arguments: map[string]any{
			"frameId": 1000,
		},
	})
	if err != nil {
		t.Fatalf("Failed to get scopes: %v", err)
	}
	t.Logf("Scopes result: %v", scopesResult)

	// Check if scopes returned an error
	if scopesResult.IsError {
		errorMsg := "Unknown error"
		if len(scopesResult.Content) > 0 {
			if textContent, ok := scopesResult.Content[0].(*mcp.TextContent); ok {
				errorMsg = textContent.Text
			}
		}
		t.Fatalf("Scopes returned error: %s", errorMsg)
	}

	// Verify we got scopes data
	if len(scopesResult.Content) == 0 {
		t.Fatalf("Expected scopes data, got empty content")
	}

	// Extract scopes content
	scopesStr := ""
	for _, content := range scopesResult.Content {
		if textContent, ok := content.(*mcp.TextContent); ok {
			scopesStr += textContent.Text
		}
	}

	t.Logf("Scopes output:\n%s", scopesStr)

	// Verify scopes contains expected information
	// For the helloworld program at line 7, we should have locals scope
	if !strings.Contains(scopesStr, "Locals") {
		t.Errorf("Expected scopes to contain 'Locals', got: %s", scopesStr)
	}

	// The greeting variable should be in the locals scope
	if !strings.Contains(scopesStr, "greeting") {
		t.Errorf("Expected scopes to contain 'greeting' variable, got: %s", scopesStr)
	}

	// Stop debugger
	ts.stopDebugger(t)
}

func TestScopesComprehensive(t *testing.T) {
	// Setup test infrastructure
	ts := setupMCPServerAndClient(t)
	defer ts.cleanup()

	// Compile test program
	binaryPath, cleanupBinary := compileTestProgram(t, ts.cwd, "scopes")
	defer cleanupBinary()

	// Start debugger and execute program
	ts.startDebuggerAndExecuteProgram(t, "9094", binaryPath)

	// Set all breakpoints at once
	f := filepath.Join(ts.cwd, "testdata", "go", "scopes", "main.go")

	// Set breakpoint in greet function at line 42
	setBreakpointResult, err := ts.session.CallTool(ts.ctx, &mcp.CallToolParams{
		Name: "set-breakpoints",
		Arguments: map[string]any{
			"file":  f,
			"lines": []int{42, 54, 67},
		},
	})
	if err != nil {
		t.Fatalf("Failed to set breakpoints: %v", err)
	}
	t.Logf("Set breakpoints result: %v", setBreakpointResult)

	// Continue to first breakpoint
	_, err = ts.session.CallTool(ts.ctx, &mcp.CallToolParams{
		Name: "continue",
		Arguments: map[string]any{
			"threadID": 1,
		},
	})
	if err != nil {
		t.Fatalf("Failed to continue: %v", err)
	}

	// Get stacktrace to ensure we have valid frame IDs
	_, err = ts.session.CallTool(ts.ctx, &mcp.CallToolParams{
		Name: "stack-trace",
		Arguments: map[string]any{
			"threadID": 1,
		},
	})
	if err != nil {
		t.Fatalf("Failed to get stacktrace: %v", err)
	}

	// Get scopes for the greet function frame
	scopesResult, err := ts.session.CallTool(ts.ctx, &mcp.CallToolParams{
		Name: "scopes",
		Arguments: map[string]any{
			"frameId": 1000, // topmost frame (greet function)
		},
	})
	if err != nil {
		t.Fatalf("Failed to get scopes: %v", err)
	}

	scopesStr := ""
	for _, content := range scopesResult.Content {
		if textContent, ok := content.(*mcp.TextContent); ok {
			scopesStr += textContent.Text
		}
	}
	t.Logf("Scopes in greet function:\n%s", scopesStr)

	// Verify function arguments
	if !strings.Contains(scopesStr, "name") || !strings.Contains(scopesStr, "\"Alice\"") {
		t.Errorf("Expected to find argument 'name' with value 'Alice'")
	}
	if !strings.Contains(scopesStr, "age") || !strings.Contains(scopesStr, "30") {
		t.Errorf("Expected to find argument 'age' with value 30")
	}

	// Verify local variables
	if !strings.Contains(scopesStr, "greeting") {
		t.Errorf("Expected to find local variable 'greeting'")
	}
	if !strings.Contains(scopesStr, "prefix") && !strings.Contains(scopesStr, "Greeting: ") {
		t.Errorf("Expected to find local variable 'prefix' with value 'Greeting: '")
	}

	// Test 2: Struct parameter and local variables
	// Continue to next breakpoint in processPerson at line 54
	_, err = ts.session.CallTool(ts.ctx, &mcp.CallToolParams{
		Name: "continue",
		Arguments: map[string]any{
			"threadID": 1,
		},
	})
	if err != nil {
		t.Fatalf("Failed to continue: %v", err)
	}

	// Get stack trace for processPerson function
	_, err = ts.session.CallTool(ts.ctx, &mcp.CallToolParams{
		Name: "stack-trace",
		Arguments: map[string]any{
			"threadID": 1,
		},
	})
	if err != nil {
		t.Fatalf("Failed to get stacktrace: %v", err)
	}

	// Get scopes for processPerson function
	scopesResult2, err := ts.session.CallTool(ts.ctx, &mcp.CallToolParams{
		Name: "scopes",
		Arguments: map[string]any{
			"frameId": 1000, // topmost frame (processPerson function)
		},
	})
	if err != nil {
		t.Fatalf("Failed to get scopes: %v", err)
	}

	scopesStr2 := ""
	for _, content := range scopesResult2.Content {
		if textContent, ok := content.(*mcp.TextContent); ok {
			scopesStr2 += textContent.Text
		}
	}
	t.Logf("Scopes in processPerson function:\n%s", scopesStr2)

	// Verify struct parameter
	if !strings.Contains(scopesStr2, "p") {
		t.Errorf("Expected to find parameter 'p' (Person struct)")
	}
	if !strings.Contains(scopesStr2, "description") {
		t.Errorf("Expected to find local variable 'description'")
	}
	if !strings.Contains(scopesStr2, "isAdult") {
		t.Errorf("Expected to find local variable 'isAdult'")
	}

	// Test 3: Collection parameters
	// Continue to next breakpoint in processCollection at line 67
	_, err = ts.session.CallTool(ts.ctx, &mcp.CallToolParams{
		Name: "continue",
		Arguments: map[string]any{
			"threadID": 1,
		},
	})
	if err != nil {
		t.Fatalf("Failed to continue: %v", err)
	}

	// Get stack trace for processCollection function
	_, err = ts.session.CallTool(ts.ctx, &mcp.CallToolParams{
		Name: "stack-trace",
		Arguments: map[string]any{
			"threadID": 1,
		},
	})
	if err != nil {
		t.Fatalf("Failed to get stacktrace: %v", err)
	}

	// Get scopes for processCollection function
	scopesResult3, err := ts.session.CallTool(ts.ctx, &mcp.CallToolParams{
		Name: "scopes",
		Arguments: map[string]any{
			"frameId": 1000, // topmost frame (processCollection function)
		},
	})
	if err != nil {
		t.Fatalf("Failed to get scopes: %v", err)
	}

	scopesStr3 := ""
	for _, content := range scopesResult3.Content {
		if textContent, ok := content.(*mcp.TextContent); ok {
			scopesStr3 += textContent.Text
		}
	}
	t.Logf("Scopes in processCollection function:\n%s", scopesStr3)

	// Verify collection parameters and locals
	if !strings.Contains(scopesStr3, "nums") {
		t.Errorf("Expected to find parameter 'nums' (slice)")
	}
	if !strings.Contains(scopesStr3, "dict") {
		t.Errorf("Expected to find parameter 'dict' (map)")
	}
	if !strings.Contains(scopesStr3, "sum") {
		t.Errorf("Expected to find local variable 'sum'")
	}
	if !strings.Contains(scopesStr3, "count") {
		t.Errorf("Expected to find local variable 'count'")
	}

	// Stop debugger
	ts.stopDebugger(t)
}
