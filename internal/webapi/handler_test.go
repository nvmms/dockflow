package webapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealth(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	w := httptest.NewRecorder()
	NewHandler().ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if w.Header().Get("Content-Type") != "application/json; charset=utf-8" {
		t.Fatalf("content type = %q", w.Header().Get("Content-Type"))
	}
}

func TestOpenAPIIsValidJSON(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/openapi.json", nil)
	w := httptest.NewRecorder()
	NewHandler().ServeHTTP(w, r)
	var spec map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &spec); err != nil {
		t.Fatal(err)
	}
	if spec["openapi"] != "3.1.0" {
		t.Fatalf("unexpected version: %v", spec["openapi"])
	}
}

func TestCreateNamespaceValidation(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/api/v1/namespaces", strings.NewReader(`{"name":"","unknown":true}`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	NewHandler().ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestRepositoryProviderValidation(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/api/v1/repositories/bitbucket", strings.NewReader(`{"name":"me","token":"secret"}`))
	w := httptest.NewRecorder()
	NewHandler().ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestMethodNotAllowed(t *testing.T) {
	r := httptest.NewRequest(http.MethodPatch, "/api/v1/namespaces", nil)
	w := httptest.NewRecorder()
	NewHandler().ServeHTTP(w, r)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d", w.Code)
	}
	if w.Header().Get("Allow") != "GET, POST" {
		t.Fatalf("allow = %q", w.Header().Get("Allow"))
	}
}

func TestListAppDeploymentJobsRoute(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/namespaces/gb/apps/gb.api/deploy/jobs", nil)
	w := httptest.NewRecorder()
	NewHandler().ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
}

func TestManagementUI(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	NewHandler().ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "Dockflow") {
		t.Fatalf("management UI index was not served")
	}
}
