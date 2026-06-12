package handler

import "engine-poc/internal/spec"

// ResolvePort looks up systemCode in the spec systems map and returns the TCP port.
func ResolvePort(systemCode string, systems map[string]spec.SystemDef) (int, *EngineError) {
	def, ok := systems[systemCode]
	if !ok {
		return 0, ErrSystemCodeNotFound
	}
	return def.Port, nil
}
