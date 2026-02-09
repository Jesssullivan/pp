package daemon

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
)

// IPCHandler processes incoming IPC commands. Implementations dispatch
// commands to the appropriate daemon subsystem.
type IPCHandler interface {
	HandleCommand(cmd string, args map[string]string) (string, error)
}

// IPCServer listens on a Unix domain socket for line-based text commands
// and returns JSON responses.
//
// Protocol:
//   - Client sends a single line: COMMAND [arg1] [arg2] ...
//   - Server responds with a JSON line followed by a newline.
//   - Supported commands: HEALTH, BANNER {width} {height} {protocol}, REFRESH, QUIT
type IPCServer struct {
	socketPath string
	handler    IPCHandler
	listener   net.Listener
	wg         sync.WaitGroup
	done       chan struct{}
}

// NewIPCServer creates an IPC server that will listen on socketPath and
// dispatch commands to handler.
func NewIPCServer(socketPath string, handler IPCHandler) *IPCServer {
	return &IPCServer{
		socketPath: socketPath,
		handler:    handler,
		done:       make(chan struct{}),
	}
}

// Start begins listening for connections on the Unix socket. The socket file
// is created with mode 0600 for security. Any existing socket file at the
// path is removed first.
func (s *IPCServer) Start() error {
	// Remove stale socket file.
	os.Remove(s.socketPath)

	ln, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", s.socketPath, err)
	}

	// Set socket permissions to owner-only.
	if err := os.Chmod(s.socketPath, 0o600); err != nil {
		ln.Close()
		return fmt.Errorf("chmod socket: %w", err)
	}

	s.listener = ln

	s.wg.Add(1)
	go s.acceptLoop()

	return nil
}

// Stop gracefully shuts down the IPC server. It closes the listener, waits
// for active connections to finish, and removes the socket file.
func (s *IPCServer) Stop() {
	select {
	case <-s.done:
		// Already stopped.
		return
	default:
	}

	close(s.done)

	if s.listener != nil {
		s.listener.Close()
	}

	s.wg.Wait()

	// Clean up socket file.
	os.Remove(s.socketPath)
}

// acceptLoop accepts connections until the server is stopped.
func (s *IPCServer) acceptLoop() {
	defer s.wg.Done()

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.done:
				return
			default:
				// Transient error, continue accepting.
				continue
			}
		}

		s.wg.Add(1)
		go s.handleConn(conn)
	}
}

// handleConn processes a single client connection. It reads one line,
// parses the command, dispatches it, and writes the response.
func (s *IPCServer) handleConn(conn net.Conn) {
	defer s.wg.Done()
	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		return
	}

	line := strings.TrimSpace(scanner.Text())
	if line == "" {
		return
	}

	cmd, args := parseIPCCommand(line)

	response, err := s.handler.HandleCommand(cmd, args)
	if err != nil {
		errResp := map[string]string{
			"error": err.Error(),
		}
		data, _ := json.Marshal(errResp)
		fmt.Fprintf(conn, "%s\n", data)
		return
	}

	// Compact the JSON response to a single line for the line-based protocol.
	// If compaction fails (response is not JSON), send as-is.
	compacted, compactErr := compactJSON(response)
	if compactErr == nil {
		response = compacted
	}

	fmt.Fprintf(conn, "%s\n", response)
}

// parseIPCCommand parses a line-based IPC command into the command name
// and a map of positional arguments.
//
// Format:
//
//	HEALTH                              -> cmd="HEALTH", args={}
//	BANNER 80 24 kitty                  -> cmd="BANNER", args={width:80, height:24, protocol:kitty}
//	REFRESH                             -> cmd="REFRESH", args={}
//	QUIT                                -> cmd="QUIT", args={}
func parseIPCCommand(line string) (string, map[string]string) {
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return "", nil
	}

	cmd := strings.ToUpper(parts[0])
	args := make(map[string]string)

	switch cmd {
	case "BANNER":
		if len(parts) >= 2 {
			args["width"] = parts[1]
		}
		if len(parts) >= 3 {
			args["height"] = parts[2]
		}
		if len(parts) >= 4 {
			args["protocol"] = parts[3]
		}
	}

	return cmd, args
}

// IPCClient connects to a running daemon via Unix socket to send commands.
type IPCClient struct {
	socketPath string
}

// NewIPCClient creates a client that will connect to the daemon at socketPath.
func NewIPCClient(socketPath string) *IPCClient {
	return &IPCClient{socketPath: socketPath}
}

// SendCommand sends a text command to the daemon and returns the response.
// Each call opens a new connection, sends the command, reads the response,
// and closes the connection.
func (c *IPCClient) SendCommand(cmd string) (string, error) {
	conn, err := net.Dial("unix", c.socketPath)
	if err != nil {
		return "", fmt.Errorf("connect to daemon: %w", err)
	}
	defer conn.Close()

	// Send command.
	fmt.Fprintf(conn, "%s\n", cmd)

	// Read response.
	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return "", fmt.Errorf("read response: %w", err)
		}
		return "", fmt.Errorf("empty response from daemon")
	}

	return scanner.Text(), nil
}

// compactJSON removes whitespace from JSON to produce a single-line string
// suitable for line-based IPC transport.
func compactJSON(s string) (string, error) {
	var buf bytes.Buffer
	if err := json.Compact(&buf, []byte(s)); err != nil {
		return "", err
	}
	return buf.String(), nil
}
