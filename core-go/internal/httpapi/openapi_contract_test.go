package httpapi

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
	"gopkg.in/yaml.v3"
)

type openAPISpec struct {
	Paths map[string]map[string]any `yaml:"paths"`
}

func TestOpenAPIDoesNotDriftFromRouter(t *testing.T) {
	// Load canonical OpenAPI spec from repo root.
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))
	openAPIPath := filepath.Join(repoRoot, "api", "openapi.yaml")

	b, err := os.ReadFile(openAPIPath)
	if err != nil {
		t.Fatalf("read openapi spec %q: %v", openAPIPath, err)
	}

	var spec openAPISpec
	if err := yaml.Unmarshal(b, &spec); err != nil {
		t.Fatalf("parse openapi spec %q: %v", openAPIPath, err)
	}

	expected := expectedRoutesFromOpenAPI(t, spec)
	actual := actualRoutesFromRouter(t)

	missing := diff(expected, actual)
	extra := diff(actual, expected)

	// This gate is intentionally strict: we want the spec and router to match.
	if len(missing) > 0 || len(extra) > 0 {
		var sb strings.Builder
		if len(missing) > 0 {
			sb.WriteString("missing routes (in OpenAPI but not registered in chi router):\n")
			for _, k := range missing {
				sb.WriteString("  - ")
				sb.WriteString(k)
				sb.WriteString("\n")
			}
		}
		if len(extra) > 0 {
			sb.WriteString("extra routes (registered in chi router but not present in OpenAPI):\n")
			for _, k := range extra {
				sb.WriteString("  - ")
				sb.WriteString(k)
				sb.WriteString("\n")
			}
		}
		t.Fatalf("OpenAPI drift detected. Update api/openapi.yaml or the router.\n\n%s", sb.String())
	}
}

func expectedRoutesFromOpenAPI(t *testing.T, spec openAPISpec) map[string]struct{} {
	t.Helper()

	validMethods := map[string]struct{}{
		"get": {}, "post": {}, "put": {}, "patch": {}, "delete": {}, "head": {}, "options": {},
	}

	out := make(map[string]struct{})
	for p, ops := range spec.Paths {
		for m := range ops {
			mLower := strings.ToLower(m)
			if _, ok := validMethods[mLower]; !ok {
				continue
			}
			method := strings.ToUpper(mLower)
			// OpenAPI paths are rooted at /v1/... with servers.url=/api.
			route := normalizeRoute("/api" + p)
			out[method+" "+route] = struct{}{}
		}
	}

	return out
}

func actualRoutesFromRouter(t *testing.T) map[string]struct{} {
	t.Helper()

	log := zerolog.New(io.Discard)
	h := NewHandler(log, nil)
	raw := h.Router()

	mux, ok := raw.(*chi.Mux)
	if !ok {
		t.Fatalf("expected *chi.Mux from Handler.Router(), got %T", raw)
	}

	validMethods := map[string]struct{}{
		http.MethodGet: {}, http.MethodPost: {}, http.MethodPut: {}, http.MethodPatch: {}, http.MethodDelete: {}, http.MethodHead: {}, http.MethodOptions: {},
	}

	out := make(map[string]struct{})
	if err := chi.Walk(mux, func(method string, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		if _, ok := validMethods[method]; !ok {
			return nil
		}
		route = normalizeRoute(route)
		if !strings.HasPrefix(route, "/api/") {
			return nil
		}
		out[method+" "+route] = struct{}{}
		return nil
	}); err != nil {
		t.Fatalf("walk chi router: %v", err)
	}

	return out
}

func normalizeRoute(route string) string {
	if route == "" {
		return route
	}
	if len(route) > 1 {
		route = strings.TrimSuffix(route, "/")
	}
	return route
}

func diff(a, b map[string]struct{}) []string {
	out := make([]string, 0)
	for k := range a {
		if _, ok := b[k]; !ok {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}
