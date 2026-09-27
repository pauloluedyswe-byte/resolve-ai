package httpapi

import (
	"context"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"resolveai/internal/domain"
	"resolveai/internal/service"
)

// FileReader lê arquivos enviados (imagens) para servi-los em /uploads.
type FileReader interface {
	Get(ctx context.Context, name string) (contentType string, data []byte, err error)
}

type Server struct {
	auth        *service.AuthService
	occurrences *service.OccurrenceService
	files       FileReader
	corsOrigins []string
}

func NewServer(auth *service.AuthService, occ *service.OccurrenceService, files FileReader, corsOrigins []string) *Server {
	return &Server{auth: auth, occurrences: occ, files: files, corsOrigins: corsOrigins}
}

func (s *Server) Routes() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.RealIP, skipHealth(middleware.Logger), middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(s.cors)

	r.Get("/uploads/{name}", s.serveFile)

	r.Route("/api", func(r chi.Router) {
		r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		})
		r.Post("/auth/register", s.register)
		r.Post("/auth/login", s.login)

		r.Group(func(r chi.Router) {
			r.Use(s.authenticate)
			r.Get("/auth/me", s.me)
			r.Get("/categories", s.listCategories)

			r.Get("/occurrences", s.listOccurrences)
			r.Post("/occurrences", s.createOccurrence)
			r.Get("/occurrences/{id}", s.getOccurrence)
			r.Post("/occurrences/{id}/image", s.uploadImage)
			r.Post("/occurrences/{id}/comments", s.addComment)
			r.Patch("/occurrences/{id}/status", s.changeStatus)
			r.Post("/occurrences/{id}/rating", s.rate)

			r.Group(func(r chi.Router) {
				r.Use(requireRole(domain.RoleGestor))
				r.Get("/users/gestores", s.listGestores)
				r.Get("/dashboard", s.dashboard)
				r.Patch("/occurrences/{id}/priority", s.updatePriority)
				r.Patch("/occurrences/{id}/assignee", s.assign)
				r.Patch("/occurrences/{id}/solution", s.registerSolution)
			})
		})
	})
	return r
}

// skipHealth evita poluir o log com o health check periódico do provedor de cloud.
func skipHealth(mw func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		logged := mw(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/api/health" {
				next.ServeHTTP(w, r)
				return
			}
			logged.ServeHTTP(w, r)
		})
	}
}

type ctxKey struct{}

func actorFrom(r *http.Request) service.Actor {
	return r.Context().Value(ctxKey{}).(service.Actor)
}

func (s *Server) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok || token == "" {
			writeError(w, http.StatusUnauthorized, "token ausente")
			return
		}
		actor, err := s.auth.ParseToken(token)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "token inválido ou expirado")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, *actor)))
	})
}

func requireRole(role domain.Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if actorFrom(r).Role != role {
				writeError(w, http.StatusForbidden, "acesso restrito a "+string(role))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (s *Server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && (slices.Contains(s.corsOrigins, origin) || slices.Contains(s.corsOrigins, "*")) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, OPTIONS")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
