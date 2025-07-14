# MCP DAP Server

A Model Context Protocol (MCP) server that provides debugging capabilities through the Debug Adapter Protocol (DAP). This server enables AI assistants and other MCP clients to interact with debuggers for various programming languages.

## Overview

The MCP DAP Server acts as a bridge between MCP clients and DAP-compatible debuggers, allowing programmatic control of debugging sessions. It provides a comprehensive set of debugging tools that can be used to:

- Start and stop debugging sessions
- Set breakpoints (line-based and function-based)
- Control program execution (continue, step in/out/over, pause)
- Inspect program state (threads, stack traces, variables, scopes)
- Evaluate expressions
- Attach to running processes
- Handle exceptions

## Features

### Core Debugging Operations
- **Session Management**: Start, stop, restart, and terminate debugging sessions
- **Breakpoint Management**: Set line and function breakpoints
- **Execution Control**: Continue, pause, step operations (in/out/next)
- **State Inspection**: View threads, stack traces, scopes, and variables
- **Expression Evaluation**: Evaluate expressions in the current debugging context
- **Variable Modification**: Change variable values during debugging

### Advanced Features
- **Process Attachment**: Attach to already running processes
- **Module Information**: List loaded modules
- **Source Information**: View loaded source files
- **Disassembly**: View disassembled code
- **Exception Handling**: Get information about exceptions

## Installation

### Prerequisites
- Go 1.24.4 or later
- A DAP-compatible debugger for your target language

### Building from Source

```bash
git clone https://github.com/go-delve/mcp-dap-server
cd mcp-dap-server
go build -o bin/mcp-dap-server
```

## Usage

### Starting the Server

The server listens on port 8080 by default:

```bash
./bin/mcp-dap-server
```

### Connecting via MCP

Configure your MCP client to connect to the server at `http://localhost:8080` using the SSE (Server-Sent Events) transport.

### Example MCP Client Configuration

```json
{
  "mcpServers": {
    "dap-debugger": {
      "command": "mcp-dap-server",
      "args": [],
      "env": {}
    }
  }
}
```

## Available Tools

### Session Management

#### `start_debugger`
Starts a new debugging session.
- **Parameters**:
  - `port` (number): The port number for the DAP server

#### `stop_debugger`
Stops the current debugging session.

#### `restart_debugger`
Restarts the current debugging session.

#### `terminate_debugger`
Terminates the debugger and the debuggee process.

### Program Control

#### `debug_program`
Launches a program in debug mode.
- **Parameters**:
  - `path` (string): Path to the program to debug

#### `exec_program`
Executes a program without debugging.
- **Parameters**:
  - `path` (string): Path to the program to execute

#### `attach_debugger`
Attaches to a running process.
- **Parameters**:
  - `mode` (string): Attachment mode
  - `processId` (number, optional): Process ID to attach to

### Breakpoints

#### `set_breakpoints`
Sets line breakpoints in a file.
- **Parameters**:
  - `file` (string): Source file path
  - `lines` (array): Line numbers for breakpoints

#### `set_function_breakpoints`
Sets function breakpoints.
- **Parameters**:
  - `functions` (array): Function names

### Execution Control

#### `continue`
Continues program execution.
- **Parameters**:
  - `threadId` (number, optional): Thread ID to continue

#### `next`
Steps over the current line.
- **Parameters**:
  - `threadId` (number): Thread ID

#### `step_in`
Steps into function calls.
- **Parameters**:
  - `threadId` (number): Thread ID

#### `step_out`
Steps out of the current function.
- **Parameters**:
  - `threadId` (number): Thread ID

#### `pause`
Pauses program execution.
- **Parameters**:
  - `threadId` (number): Thread ID

### State Inspection

#### `threads`
Lists all threads in the debugged process.

#### `stack_trace`
Gets the stack trace for a thread.
- **Parameters**:
  - `threadId` (number): Thread ID
  - `startFrame` (number, optional): Starting frame index
  - `levels` (number, optional): Number of frames to retrieve

#### `scopes`
Gets variable scopes for a stack frame.
- **Parameters**:
  - `frameId` (number): Stack frame ID

#### `variables`
Gets variables in a scope.
- **Parameters**:
  - `variablesReference` (number): Variables reference

#### `evaluate`
Evaluates an expression.
- **Parameters**:
  - `expression` (string): Expression to evaluate
  - `frameId` (number, optional): Frame context
  - `context` (string, optional): Evaluation context

#### `set_variable`
Sets a variable value.
- **Parameters**:
  - `variablesReference` (number): Variables reference
  - `name` (string): Variable name
  - `value` (string): New value

### Additional Tools

#### `loaded_sources`
Lists all loaded source files.

#### `modules`
Lists loaded modules.

#### `disassemble`
Disassembles code at a memory location.
- **Parameters**:
  - `memoryReference` (string): Memory address
  - `instructionOffset` (number, optional): Instruction offset
  - `instructionCount` (number): Number of instructions

#### `exception_info`
Gets exception information.
- **Parameters**:
  - `threadId` (number): Thread ID

#### `disconnect`
Disconnects from the debugger.
- **Parameters**:
  - `terminateDebuggee` (boolean, optional): Whether to terminate the debuggee

#### `configuration_done`
Signals that configuration is complete.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT

## Acknowledgments

- Built with the [Model Context Protocol SDK for Go](https://github.com/modelcontextprotocol/go-sdk)
- Uses the [Google DAP implementation for Go](https://github.com/google/go-dap)
