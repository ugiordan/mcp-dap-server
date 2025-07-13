package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"

	"github.com/google/go-dap"
)

// DAPClient is a debugger service client that uses Debug Adaptor Protocol.
// It does not (yet?) implement service.DAPClient interface.
// All client methods are synchronous.
type DAPClient struct {
	conn   net.Conn
	reader *bufio.Reader
	// seq is used to track the sequence number of each
	// requests that the client sends to the server
	seq int
}

// newDAPClient creates a new Client over a TCP connection.
// Call Close() to close the connection.
func newDAPClient(addr string) *DAPClient {
	fmt.Println("Connecting to server at:", addr)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	return newDAPClientFromConn(conn)
}

// newDAPClientFromConn creates a new Client with the given TCP connection.
// Call Close to close the connection.
func newDAPClientFromConn(conn net.Conn) *DAPClient {
	c := &DAPClient{conn: conn, reader: bufio.NewReader(conn)}
	c.seq = 1 // match VS Code numbering
	return c
}

// Close closes the client connection.
func (c *DAPClient) Close() {
	c.conn.Close()
}

// InitializeRequest sends an 'initialize' request.
func (c *DAPClient) InitializeRequest() error {
	request := &dap.InitializeRequest{Request: *c.newRequest("initialize")}
	request.Arguments = dap.InitializeRequestArguments{
		AdapterID:                    "go",
		PathFormat:                   "path",
		LinesStartAt1:                true,
		ColumnsStartAt1:              true,
		SupportsVariableType:         true,
		SupportsVariablePaging:       true,
		SupportsRunInTerminalRequest: true,
		Locale:                       "en-us",
	}
	return c.send(request)
}

func (c *DAPClient) ReadMessage() (dap.Message, error) {
	return dap.ReadProtocolMessage(c.reader)
}

// LaunchRequest sends a 'launch' request with the specified args.
func (c *DAPClient) LaunchRequest(mode, program string, stopOnEntry bool) error {
	request := &dap.LaunchRequest{Request: *c.newRequest("launch")}
	request.Arguments = toRawMessage(map[string]any{
		"request":     "launch",
		"mode":        mode,
		"program":     program,
		"stopOnEntry": stopOnEntry,
	})
	return c.send(request)
}

func (c *DAPClient) newRequest(command string) *dap.Request {
	request := &dap.Request{}
	request.Type = "request"
	request.Command = command
	request.Seq = c.seq
	c.seq++
	return request
}

func (c *DAPClient) send(request dap.Message) error {
	return dap.WriteProtocolMessage(c.conn, request)
}

func toRawMessage(in any) json.RawMessage {
	out, _ := json.Marshal(in)
	return out
}
