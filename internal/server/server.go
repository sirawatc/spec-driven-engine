package server

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"engine-poc/internal/dashboard"
	"engine-poc/internal/spec"
)

type Server struct {
	router *chi.Mux
	port   string
}

// HandlerFactory produces an http.HandlerFunc for a given spec version and endpoint.
type HandlerFactory func(version string, endpoint spec.Endpoint) http.HandlerFunc

func New(registry spec.SpecRegistry, factory HandlerFactory, dash *dashboard.Handler) *Server {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Dashboard
	r.Route("/dashboard", func(r chi.Router) {
		dash.Routes(r)
	})

	// Register versioned routes from registry
	versions := sortedVersions(registry)
	for _, version := range versions {
		s := registry[version]
		v := version
		r.Route("/"+v, func(r chi.Router) {
			for _, ep := range s.Endpoints {
				method := ep.Method
				path := ep.Path
				handler := factory(v, ep)
				r.Method(method, path, handler)
			}
		})
		log.Printf("registered version %s: %d endpoint(s)", v, len(s.Endpoints))
	}

	port := os.Getenv("ENGINE_PORT")
	if port == "" {
		port = "8080"
	}

	return &Server{router: r, port: port}
}

func (s *Server) Start() error {
	addr := ":" + s.port
	fmt.Printf("server listening on %s\n", addr)
	return http.ListenAndServe(addr, s.router)
}

// Router returns the underlying http.Handler for use with httptest.
func (s *Server) Router() http.Handler {
	return s.router
}

func sortedVersions(registry spec.SpecRegistry) []string {
	versions := make([]string, 0, len(registry))
	for v := range registry {
		versions = append(versions, v)
	}
	sort.Strings(versions)
	return versions
}
