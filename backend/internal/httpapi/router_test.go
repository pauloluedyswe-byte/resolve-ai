package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"resolveai/internal/domain"
	"resolveai/internal/service"
)

const secret = "segredo-de-teste-123"

func token(t *testing.T, role domain.Role) string {
	t.Helper()
	claims := service.Claims{
		Name: "Teste",
		Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "1",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	s, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestRouterAuth(t *testing.T) {
	auth := service.NewAuthService(nil, secret, time.Hour)
	h := NewServer(auth, service.NewOccurrenceService(nil, nil, nil, nil), nil, []string{"http://localhost:5173"}).Routes()

	cases := []struct {
		name, method, path, token string
		want                      int
	}{
		{"health público", "GET", "/api/health", "", http.StatusOK},
		{"sem token", "GET", "/api/occurrences", "", http.StatusUnauthorized},
		{"token inválido", "GET", "/api/occurrences", "abc", http.StatusUnauthorized},
		{"solicitante no dashboard", "GET", "/api/dashboard", token(t, domain.RoleSolicitante), http.StatusForbidden},
		{"solicitante altera prioridade", "PATCH", "/api/occurrences/1/priority", token(t, domain.RoleSolicitante), http.StatusForbidden},
		{"id inválido", "GET", "/api/occurrences/abc", token(t, domain.RoleGestor), http.StatusBadRequest},
		{"filtro inválido", "GET", "/api/occurrences?status=xyz", token(t, domain.RoleGestor), http.StatusBadRequest},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := httptest.NewRequest(c.method, c.path, nil)
			if c.token != "" {
				req.Header.Set("Authorization", "Bearer "+c.token)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != c.want {
				t.Fatalf("got %d, want %d (%s)", rec.Code, c.want, rec.Body.String())
			}
		})
	}
}

func TestCORSPreflight(t *testing.T) {
	h := NewServer(service.NewAuthService(nil, secret, time.Hour), nil, nil, []string{"http://localhost:5173"}).Routes()
	req := httptest.NewRequest("OPTIONS", "/api/occurrences", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent || rec.Header().Get("Access-Control-Allow-Origin") != "http://localhost:5173" {
		t.Fatalf("preflight inesperado: %d %v", rec.Code, rec.Header())
	}
}
