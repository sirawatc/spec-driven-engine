package cbs

// CBSMessage is the pre-serialization representation of a backend request:
// field name → string value (already resolved from templates, not yet padded into bytes).
type CBSMessage map[string]string

// BackendResponse is the raw fixed-length byte slice received from the backend service.
type BackendResponse []byte

// Client sends a backend message to the given TCP port and returns the raw response.
type Client interface {
	Send(port int, msg CBSMessage) (BackendResponse, error)
}
