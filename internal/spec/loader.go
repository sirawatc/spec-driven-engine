package spec

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

func Load(path string) (*Spec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("spec: read %s: %w", path, err)
	}

	var s Spec
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("spec: parse %s: %w", path, err)
	}

	if err := validate(&s); err != nil {
		return nil, fmt.Errorf("spec: validate %s: %w", path, err)
	}

	// Populate EngineCode from map key
	for key, def := range s.ResponseCodes {
		code, _ := strconv.Atoi(key)
		def.EngineCode = code
		s.ResponseCodes[key] = def
	}

	log.Printf("spec loaded: name=%q version=%q at=%s", s.Name, s.Version, time.Now().UTC().Format(time.RFC3339))
	return &s, nil
}

func validate(s *Spec) error {
	if s.Version == "" {
		return fmt.Errorf("missing required field: version")
	}
	if s.Name == "" {
		return fmt.Errorf("missing required field: name")
	}
	for i, ep := range s.Endpoints {
		if ep.Path == "" {
			return fmt.Errorf("endpoint[%d]: missing path", i)
		}
		if ep.Method == "" {
			return fmt.Errorf("endpoint[%d] %s: missing method", i, ep.Path)
		}
		for j, rule := range ep.Transform.Request {
			if rule.Name == "" {
				return fmt.Errorf("endpoint %s: request transform[%d]: missing name", ep.Path, j)
			}
			if rule.Length <= 0 {
				return fmt.Errorf("endpoint %s: request transform[%d] %s: length must be > 0", ep.Path, j, rule.Name)
			}
		}
		for j, rf := range ep.Transform.Response.Fields {
			if rf.Name == "" {
				return fmt.Errorf("endpoint %s: response field[%d]: missing name", ep.Path, j)
			}
			if rf.Length <= 0 {
				return fmt.Errorf("endpoint %s: response field[%d] %s: length must be > 0", ep.Path, j, rf.Name)
			}
		}
	}
	return nil
}
