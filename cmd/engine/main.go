package main

import (
	"log"
	"net/http"
	"os"

	"engine-poc/internal/cbs"
	"engine-poc/internal/codemap"
	"engine-poc/internal/dashboard"
	"engine-poc/internal/handler"
	"engine-poc/internal/server"
	"engine-poc/internal/spec"
)

func main() {
	specDir := spec.SpecDirFromEnv()
	registry, err := spec.LoadRegistry(specDir)
	if err != nil {
		log.Printf("failed to load spec registry: %v", err)
		os.Exit(1)
	}

	fixturePath := os.Getenv("ENGINE_MOCK_FIXTURES")
	if fixturePath == "" {
		fixturePath = "mock/fixtures.yaml"
	}

	handlerMap := make(map[string]*handler.Handler, len(registry))
	for version, s := range registry {
		mapper := codemap.NewWithOverrides(s.ResponseCodes)

		var respFields []spec.ResponseField
		if len(s.Endpoints) > 0 {
			respFields = s.Endpoints[0].Transform.Response.Fields
		}

		mockClient, err := cbs.LoadMockClient(fixturePath, respFields)
		if err != nil {
			log.Printf("failed to load mock fixtures from %s: %v", fixturePath, err)
			os.Exit(1)
		}

		handlerMap[version] = handler.New(s, mockClient, mapper)
	}

	factory := func(version string, endpoint spec.Endpoint) http.HandlerFunc {
		return handlerMap[version].Factory()(version, endpoint)
	}

	dash := dashboard.New(registry)
	srv := server.New(registry, factory, dash)
	if err := srv.Start(); err != nil {
		log.Printf("server error: %v", err)
		os.Exit(1)
	}
}
