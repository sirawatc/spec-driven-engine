package cbs

import (
	"fmt"
	"os"
	"strings"

	"engine-poc/internal/spec"

	"gopkg.in/yaml.v3"
)

// Fixture is one entry in mock/fixtures.yaml.
type Fixture struct {
	Match    map[string]string `yaml:"match"`
	Response map[string]string `yaml:"response"`
}

// MockClient matches incoming messages against loaded fixtures.
type MockClient struct {
	fixtures   []Fixture
	respFields []spec.ResponseField // ordered, matches spec definition
}

// LoadMockClient reads fixture YAML and returns a MockClient.
// respFields must be in spec-defined order for correct response serialization.
func LoadMockClient(fixturePath string, respFields []spec.ResponseField) (*MockClient, error) {
	data, err := os.ReadFile(fixturePath)
	if err != nil {
		return nil, fmt.Errorf("mock client: read fixtures %s: %w", fixturePath, err)
	}
	var fixtures []Fixture
	if err := yaml.Unmarshal(data, &fixtures); err != nil {
		return nil, fmt.Errorf("mock client: parse fixtures %s: %w", fixturePath, err)
	}
	return &MockClient{fixtures: fixtures, respFields: respFields}, nil
}

func (m *MockClient) Send(port int, msg CBSMessage) (BackendResponse, error) {
	for _, f := range m.fixtures {
		if matchesAll(msg, f.Match) {
			return m.serialize(f.Response), nil
		}
	}
	return nil, fmt.Errorf("mock client: no fixture matched message: %v", msg)
}

func matchesAll(msg CBSMessage, match map[string]string) bool {
	for k, v := range match {
		if msg[k] != v {
			return false
		}
	}
	return true
}

// serialize builds fixed-length response bytes from the fixture response map,
// iterating respFields in spec-defined order.
func (m *MockClient) serialize(response map[string]string) BackendResponse {
	var buf strings.Builder
	for _, rf := range m.respFields {
		value := response[rf.Name]
		if len(value) >= rf.Length {
			buf.WriteString(value[:rf.Length])
		} else {
			buf.WriteString(value + strings.Repeat(" ", rf.Length-len(value)))
		}
	}
	return BackendResponse(buf.String())
}
