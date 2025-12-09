package chi

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/marshallshelly/beacon-auth/auth"
	"github.com/marshallshelly/beacon-auth/core"
	"github.com/marshallshelly/beacon-auth/session"
)

// SessionMiddleware is a standard HTTP middleware compatible with Chi
func SessionMiddleware(manager *session.Manager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract token from cookie
			cookie, err := r.Cookie(manager.Config().CookieName)
			if err != nil || cookie.Value == "" {
				next.ServeHTTP(w, r)
				return
			}

			// Get session from manager
			ctx := r.Context()
			session, user, err := manager.Get(ctx, cookie.Value)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			// Store session and user in context
			ctx = core.WithSession(ctx, session)
			if user != nil {
				ctx = core.WithUser(ctx, user)
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireAuth middleware
func RequireAuth(manager *session.Manager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			session := core.GetSession(r.Context())
			if session == nil {
				http.Redirect(w, r, "/auth/signin", http.StatusFound)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireAuthJSON middleware
func RequireAuthJSON(manager *session.Manager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			session := core.GetSession(r.Context())
			if session == nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error":   "unauthorized",
					"message": "Authentication required",
				})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// Handler wraps the auth.Handler
type Handler struct {
	handler *auth.Handler
}

// NewHandler creates a new handler
func NewHandler(dbAdapter core.Adapter, sessionManager *session.Manager, config *auth.Config) *Handler {
	return &Handler{
		handler: auth.NewHandler(dbAdapter, sessionManager, config),
	}
}

// RegisterRoutes registers the authentication routes on a Chi router
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/auth/signup", h.SignUp)
	r.Post("/auth/signin", h.SignIn)
	r.Post("/auth/signout", h.SignOut)
	r.Get("/auth/session", h.GetSession)
}

// SignUp handler
func (h *Handler) SignUp(w http.ResponseWriter, r *http.Request) {
	h.handler.SignUp(w, r)
}

// SignIn handler
func (h *Handler) SignIn(w http.ResponseWriter, r *http.Request) {
	h.handler.SignIn(w, r)
}

// SignOut handler
func (h *Handler) SignOut(w http.ResponseWriter, r *http.Request) {
	h.handler.SignOut(w, r)
}

// GetSession handler
func (h *Handler) GetSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	session := core.GetSession(ctx)
	user := core.GetUser(ctx)

	if session == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error":   "no_session",
			"message": "No active session",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"user":    user,
		"session": session,
	})
}
