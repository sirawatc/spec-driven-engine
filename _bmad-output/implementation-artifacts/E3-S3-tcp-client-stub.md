# Story: E3-S3 — TCP Client Stub

**Epic:** E3 — backend Integration Layer  
**Status:** Ready for Development  
**Depends on:** E3-S2  
**Blocks:** nothing (not wired to main for PoC)

---

## Summary

Implement `TCPClient` — the real TCP implementation of `cbs.Client`. It is **not** used in the PoC runtime (MockClient is used instead), but it must compile and satisfy the interface so the upgrade path to real backend is a single swap in `main.go`.

---

## Acceptance Criteria

- [ ] `TCPClient.Send` dials `localhost:<port>`, writes message bytes, reads response bytes, closes connection
- [ ] Compiles and passes `go vet`
- [ ] Satisfies `cbs.Client` interface (compile-time verified)
- [ ] Not wired into `main.go` — no unit test required for PoC

---

## Implementation (`internal/cbs/tcp.go`)

```go
package cbs

import (
    "fmt"
    "net"
)

// TCPClient implements cbs.Client against a real TCP socket.
// Not used in the PoC — MockClient is used instead.
type TCPClient struct {
    // ResponseLength is the total expected byte length of a backend response.
    // Derived from the sum of all response field lengths in the spec.
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

    // msg here is CBSMessage (map[string]string — pre-serialization field values).
    // In the real implementation, the caller passes already-serialized bytes.
    // For now this is a compile stub — wire format serialization is done in the transformer.
    _ = msg // suppress unused warning in stub

    buf := make([]byte, c.ResponseLength)
    if _, err := conn.Read(buf); err != nil {
        return nil, fmt.Errorf("tcp client: read response: %w", err)
    }

    return BackendResponse(buf), nil
}
```

> **Design note:** In full production, `TCPClient.Send` would receive pre-serialized `[]byte` from the transformer rather than a `CBSMessage` map. The interface signature accepts `CBSMessage` to match the mock — the transformer serialization happens before calling Send in the handler pipeline. The `TCPClient` would need the serialized bytes passed through a wrapper or the interface adapted. This is an acceptable PoC trade-off; the interface refinement is a post-PoC concern.

---

## Compile-time Check

The line `var _ Client = (*TCPClient)(nil)` in `tcp.go` ensures the compiler catches interface drift. No test file needed.

---

## Verification

```bash
go build ./...   # must compile with zero errors
go vet ./...     # must pass
```
