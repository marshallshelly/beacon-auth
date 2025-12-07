package middleware

import (
	"net/http"

	"github.com/marshallshelly/beacon-auth/core"
	"github.com/marshallshelly/beacon-auth/session"
)

// SessionMiddleware creates middleware that loads session from request
func SessionMiddleware(manager *session.Manager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract token from cookie
			cookie, err := r.Cookie(manager.Config().CookieName)
			if err != nil {
				// No session cookie, continue without session
				next.ServeHTTP(w, r)
				return
			}

			// Get session from manager
			ctx := r.Context()
			session, user, err := manager.Get(ctx, cookie.Value)
			if err != nil {
				// Invalid or expired session, continue without session
				next.ServeHTTP(w, r)
				return
			}

			// Add session and user to context
			ctx = core.WithSession(ctx, session)
			if user != nil {
				ctx = core.WithUser(ctx, user)
			}

			// Continue with updated context
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireAuth creates middleware that requires a valid session
func RequireAuth(manager *session.Manager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if session exists in context
			session := core.GetSession(r.Context())
			if session == nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Session exists, continue
			next.ServeHTTP(w, r)
		})
	}
}

// RequireAuthJSON creates middleware that requires auth and returns JSON error
func RequireAuthJSON(manager *session.Manager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			session := core.GetSession(r.Context())
			if session == nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":"unauthorized","message":"Authentication required"}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
