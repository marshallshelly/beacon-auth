package echo

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/marshallshelly/beacon-auth/auth"
	"github.com/marshallshelly/beacon-auth/core"
	"github.com/marshallshelly/beacon-auth/session"
)

// SessionMiddleware creates Echo middleware
func SessionMiddleware(manager *session.Manager) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Extract token from cookie
			cookie, err := c.Cookie(manager.Config().CookieName)
			if err != nil || cookie.Value == "" {
				return next(c)
			}

			// Get session from manager
			ctx := c.Request().Context()
			session, user, err := manager.Get(ctx, cookie.Value)
			if err != nil {
				return next(c)
			}

			// Store session and user in Echo context store
			c.Set("session", session)
			if user != nil {
				c.Set("user", user)
			}

			// Update request context
			ctx = core.WithSession(ctx, session)
			if user != nil {
				ctx = core.WithUser(ctx, user)
			}
			c.SetRequest(c.Request().WithContext(ctx))

			return next(c)
		}
	}
}

// RequireAuth middleware
func RequireAuth(manager *session.Manager) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			session := GetSession(c)
			if session == nil {
				return c.Redirect(http.StatusFound, "/auth/signin")
			}
			return next(c)
		}
	}
}

// RequireAuthJSON middleware
func RequireAuthJSON(manager *session.Manager) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			session := GetSession(c)
			if session == nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error":   "unauthorized",
					"message": "Authentication required",
				})
			}
			return next(c)
		}
	}
}

// GetSession retrieves the session from Echo context
func GetSession(c echo.Context) *core.Session {
	if v := c.Get("session"); v != nil {
		if session, ok := v.(*core.Session); ok {
			return session
		}
	}
	return nil
}

// GetUser retrieves the user from Echo context
func GetUser(c echo.Context) *core.User {
	if v := c.Get("user"); v != nil {
		if user, ok := v.(*core.User); ok {
			return user
		}
	}
	return nil
}

// Handler wraps the auth.Handler
type Handler struct {
	handler *auth.Handler
}

// NewHandler creates a new Echo auth handler
func NewHandler(dbAdapter core.Adapter, sessionManager *session.Manager, config *auth.Config) *Handler {
	return &Handler{
		handler: auth.NewHandler(dbAdapter, sessionManager, config),
	}
}

// RegisterRoutes registers routes on an Echo group or instance
// Since Echo has Group() returning *Group which is a struct, we take an interface similar to common echo interface
// But Echo doesn't share a common interface for Group and Echo for routing methods easily without referencing echo.Echo or echo.Group specifically or interface{}
// We'll follow standard practice of taking *echo.Group or *echo.Echo.
// Actually echo.Router is for low level.
// Let's use `*echo.Group` usually used. If user has `*echo.Echo`, they can pass `e.Group("")`.
func (h *Handler) RegisterRoutes(g *echo.Group) {
	g.POST("/auth/signup", h.SignUp)
	g.POST("/auth/signin", h.SignIn)
	g.POST("/auth/signout", h.SignOut)
	g.GET("/auth/session", h.GetSession)
}

// SignUp handler
func (h *Handler) SignUp(c echo.Context) error {
	h.handler.SignUp(c.Response().Writer, c.Request())
	return nil
}

// SignIn handler
func (h *Handler) SignIn(c echo.Context) error {
	h.handler.SignIn(c.Response().Writer, c.Request())
	return nil
}

// SignOut handler
func (h *Handler) SignOut(c echo.Context) error {
	h.handler.SignOut(c.Response().Writer, c.Request())
	return nil
}

// GetSession retrieves the current session
func (h *Handler) GetSession(c echo.Context) error {
	session := GetSession(c)
	user := GetUser(c)

	if session == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error":   "no_session",
			"message": "No active session",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"user":    user,
		"session": session,
	})
}
