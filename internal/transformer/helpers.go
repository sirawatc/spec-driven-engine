package transformer

import (
	"fmt"
	"strings"
)

const templatePrefix = "{{request.body."
const cbsTemplatePrefix = "{{cbs."
const templateSuffix = "}}"

func resolveValue(tmpl string, body map[string]any) (string, error) {
	if strings.HasPrefix(tmpl, templatePrefix) && strings.HasSuffix(tmpl, templateSuffix) {
		fieldName := tmpl[len(templatePrefix) : len(tmpl)-len(templateSuffix)]
		val, ok := body[fieldName]
		if !ok {
			return "", fmt.Errorf("request body missing field %q", fieldName)
		}
		return fmt.Sprintf("%v", val), nil
	}
	return tmpl, nil
}

func resolveCBSTemplate(tmpl string, parsed map[string]string) (string, error) {
	if strings.HasPrefix(tmpl, cbsTemplatePrefix) && strings.HasSuffix(tmpl, templateSuffix) {
		fieldName := tmpl[len(cbsTemplatePrefix) : len(tmpl)-len(templateSuffix)]
		val, ok := parsed[fieldName]
		if !ok {
			return "", fmt.Errorf("backend response missing field %q", fieldName)
		}
		return val, nil
	}
	return tmpl, nil
}

func applyPadding(value string, length int, align, pad string) string {
	if pad == "" {
		pad = " "
	}
	padChar := string(pad[0])

	if len(value) >= length {
		return value[:length]
	}

	padding := strings.Repeat(padChar, length-len(value))
	if align == "right" {
		return padding + value
	}
	return value + padding
}
