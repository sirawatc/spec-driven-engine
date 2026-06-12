package transformer

import (
	"bytes"
	"fmt"
	"strings"

	"engine-poc/internal/cbs"
	"engine-poc/internal/spec"
)

// ToTCP converts a parsed HTTP request body into a fixed-length TCP message.
// Returns the named field map (for MockClient) and the serialized bytes.
func ToTCP(body map[string]any, rules []spec.FieldRule) (cbs.CBSMessage, []byte, error) {
	msg := make(cbs.CBSMessage, len(rules))
	var buf bytes.Buffer

	for _, rule := range rules {
		value, err := resolveValue(rule.Value, body)
		if err != nil {
			return nil, nil, fmt.Errorf("transformer: field %s: %w", rule.Name, err)
		}

		padded := applyPadding(value, rule.Length, rule.Align, rule.Pad)
		msg[rule.Name] = strings.TrimRight(padded, " ")
		buf.WriteString(padded)
	}

	return msg, buf.Bytes(), nil
}

// ToHTTP parses fixed-length backend response bytes into an HTTP response body map.
func ToHTTP(raw cbs.BackendResponse, rt spec.ResponseTransform) (map[string]any, error) {
	parsed, err := ParseResponseFields(raw, rt.Fields)
	if err != nil {
		return nil, err
	}

	result := make(map[string]any, len(rt.Mapping))
	for httpKey, tmpl := range rt.Mapping {
		value, err := resolveCBSTemplate(tmpl, parsed)
		if err != nil {
			return nil, fmt.Errorf("transformer: mapping %s: %w", httpKey, err)
		}
		result[httpKey] = value
	}

	return result, nil
}

// ParseResponseFields parses fixed-length backend response bytes into a named field map.
// Used by the handler to extract ResponseCode before calling ToHTTP.
func ParseResponseFields(raw cbs.BackendResponse, fields []spec.ResponseField) (map[string]string, error) {
	parsed := make(map[string]string, len(fields))
	offset := 0
	for _, rf := range fields {
		end := offset + rf.Length
		if end > len(raw) {
			return nil, fmt.Errorf("transformer: response too short for field %s (need offset %d, have %d)", rf.Name, end, len(raw))
		}
		parsed[rf.Name] = strings.TrimRight(string(raw[offset:end]), " ")
		offset = end
	}
	return parsed, nil
}
