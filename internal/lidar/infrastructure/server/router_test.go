package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRouter_HealthWithoutAuth(t *testing.T) {
	handler := NewRouter(nil, nil, nil, testJWTSecret)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	assert.Equal(t, "ok", resp["status"])
}

func TestRouter_APIv1WithoutAuth(t *testing.T) {
	handler := NewRouter(nil, nil, nil, testJWTSecret)

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{"create experiment", http.MethodPost, "/api/v1/experiments/create"},
		{"create task", http.MethodPost, "/api/v1/experiments/task"},
		{"get task status", http.MethodGet, "/api/v1/tasks/00000000-0000-0000-0000-000000000000"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnauthorized, w.Code)

			var resp map[string]string
			json.NewDecoder(w.Body).Decode(&resp)
			assert.Contains(t, resp["error"], "missing Authorization header")
		})
	}
}

func TestRouter_APIv1WithValidToken(t *testing.T) {
	token := generateTestToken(t, testJWTSecret, "user-123", time.Now().Add(time.Hour))

	handler := NewRouter(nil, nil, nil, testJWTSecret)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks/00000000-0000-0000-0000-000000000000", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// With auth but nil handlers → should get 501 Not Implemented
	// (because expHandler and taskHandler are nil),
	// NOT 401, which means auth passed.
	assert.Equal(t, http.StatusNotImplemented, w.Code)
}

func TestRouter_APIv1WithExpiredToken(t *testing.T) {
	token := generateTestToken(t, testJWTSecret, "user-123", time.Now().Add(-time.Hour))

	handler := NewRouter(nil, nil, nil, testJWTSecret)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks/00000000-0000-0000-0000-000000000000", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
