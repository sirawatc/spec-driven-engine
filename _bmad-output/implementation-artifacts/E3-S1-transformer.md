# Story: E3-S1 — Fixed-Length Transformer

**Epic:** E3 — backend Integration Layer  
**Status:** Ready for Development  
**Depends on:** E1-S2  
**Blocks:** E5-S1

---

## Summary

Implement the `Transformer` — the component that converts between HTTP request fields and backend fixed-length TCP messages. This is the core translation engine. Template expressions (`{{request.body.field}}`) are evaluated at request time against the incoming JSON body.

---

## Acceptance Criteria

- [ ] `ToTCP`: evaluates templates, pads/truncates to exact `length`, serializes in declaration order
- [ ] `ToTCP`: returns error if a `{{request.body.<field>}}` template references a field absent from the body
- [ ] `ToHTTP`: parses backend response bytes by cumulative offset, trims padding, evaluates `{{cbs.<field>}}` mapping templates
- [ ] Left alignment pads on right; right alignment pads on left
- [ ] Default align: `left`; default pad: `" "` (space)
- [ ] `ResponseCode` extraction is the **handler's** responsibility — transformer does not special-case it
- [ ] Unit tests: serialization, left/right align, pad char, truncation, template substitution, offset-based parsing

---

## Types (`internal/transformer/transformer.go`)

```go
package transformer

import (
    "bytes"
    "fmt"
    "strings"

    "engine-poc/internal/cbs"
    "engine-poc/internal/spec"
)

// ToTCP converts a parsed HTTP request body into a fixed-length TCP message.
// Returns the serialized bytes and a map of field name → string value (for MockClient).
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
// Uses rules to locate fields by cumulative offset, then applies mapping templates.
func ToHTTP(raw cbs.BackendResponse, rt spec.ResponseTransform) (map[string]any, error) {
    // Parse fields by cumulative byte offset
    parsed := make(map[string]string)
    offset := 0
    for _, rf := range rt.Fields {
        end := offset + rf.Length
        if end > len(raw) {
            return nil, fmt.Errorf("transformer: response too short for field %s (need offset %d, have %d)", rf.Name, end, len(raw))
        }
        parsed[rf.Name] = strings.TrimRight(string(raw[offset:end]), " ")
        offset = end
    }

    // Apply mapping templates
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
            return nil, fmt.Errorf("transformer: response too short for field %s", rf.Name)
        }
        parsed[rf.Name] = strings.TrimRight(string(raw[offset:end]), " ")
        offset = end
    }
    return parsed, nil
}
```

---

## Helper Functions (`internal/transformer/helpers.go`)

```go
package transformer

import (
    "fmt"
    "strings"
)

const templatePrefix = "{{request.body."
const cbsTemplatePrefix = "{{cbs."
const templateSuffix = "}}"

// resolveValue evaluates a field rule value — either a literal or {{request.body.<field>}} template.
func resolveValue(tmpl string, body map[string]any) (string, error) {
    if strings.HasPrefix(tmpl, templatePrefix) && strings.HasSuffix(tmpl, templateSuffix) {
        fieldName := tmpl[len(templatePrefix) : len(tmpl)-len(templateSuffix)]
        val, ok := body[fieldName]
        if !ok {
            return "", fmt.Errorf("request body missing field %q", fieldName)
        }
        return fmt.Sprintf("%v", val), nil
    }
    return tmpl, nil // literal
}

// resolveCBSTemplate evaluates a {{cbs.<field>}} template against parsed backend fields.
func resolveCBSTemplate(tmpl string, parsed map[string]string) (string, error) {
    if strings.HasPrefix(tmpl, cbsTemplatePrefix) && strings.HasSuffix(tmpl, templateSuffix) {
        fieldName := tmpl[len(cbsTemplatePrefix) : len(tmpl)-len(templateSuffix)]
        val, ok := parsed[fieldName]
        if !ok {
            return "", fmt.Errorf("backend response missing field %q", fieldName)
        }
        return val, nil
    }
    return tmpl, nil // literal
}

// applyPadding pads or truncates value to exactly length bytes.
func applyPadding(value string, length int, align, pad string) string {
    if pad == "" {
        pad = " "
    }
    padChar := rune(pad[0])

    if len(value) >= length {
        return value[:length] // truncate
    }

    padding := strings.Repeat(string(padChar), length-len(value))
    if align == "right" {
        return padding + value
    }
    return value + padding // left align (default)
}
```

---

## Unit Tests (`internal/transformer/transformer_test.go`)

| Test | Description |
|------|-------------|
| `ToTCP_literal` | Field with literal value → exact bytes |
| `ToTCP_template` | `{{request.body.resourceId}}` → substituted + padded |
| `ToTCP_left_align` | Value shorter than length, left align → right-padded with space |
| `ToTCP_right_align` | Value shorter than length, right align → left-padded |
| `ToTCP_custom_pad` | `pad: "0"` → zero-padded |
| `ToTCP_truncate` | Value longer than length → truncated |
| `ToTCP_missing_field` | Template references absent body field → error |
| `ToHTTP_parse_offsets` | 3 fields with correct lengths → each parsed correctly |
| `ToHTTP_trim_padding` | Right-padded spaces trimmed from parsed values |
| `ToHTTP_short_response` | Response bytes too short → error |
| `ParseResponseFields` | Extracts `ResponseCode` at offset 0 |

---

## Verification

```bash
go test ./internal/transformer/...
go build ./...
```
