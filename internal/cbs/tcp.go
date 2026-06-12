package cbs

import (
	"fmt"
	"net"
)

// TCPClient implements cbs.Client against a real TCP socket.
// Not used in the PoC — MockClient is used instead.
type TCPClient struct {
	// ResponseLength is the total expected byte length of a backend response.
	ResponseLength int
}

// Compile-time interface check.
var _ Client = (*TCPClient)(nil)

func (c *TCPClient) Send(port int, msg CBSMessage) (BackendResponse, error) {
	addr := fmt.Sprintf("localhost:%d", port)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("tcp client: dial %s: %w", addr, err)
	}
	defer conn.Close()

	_ = msg // serialization done by transformer before calling Send

	buf := make([]byte, c.ResponseLength)
	if _, err := conn.Read(buf); err != nil {
		return nil, fmt.Errorf("tcp client: read response: %w", err)
	}

	return BackendResponse(buf), nil
}
