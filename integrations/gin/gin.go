package gin

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/marshallshelly/beacon-auth/auth"
	"github.com/marshallshelly/beacon-auth/core"
	"github.com/marshallshelly/beacon-auth/session"
)

// SessionMiddleware creates Gin middleware that loads session from request
func SessionMiddleware(manager *session.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract token from cookie
		token, err := c.Cookie(manager.Config().CookieName)
		if err != nil || token == "" {
			c.Next()
			return
		}

		// Get session from manager
		ctx := c.Request.Context()
		session, user, err := manager.Get(ctx, token)
		if err != nil {
			c.Next()
			return
		}

		// Store session and user in Gin locals
		c.Set("session", session)
		if user != nil {
			c.Set("user", user)
		}

		// Update request context for compatibility with standard helper functions
		ctx = core.WithSession(ctx, session)
		if user != nil {
			ctx = core.WithUser(ctx, user)
		}
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

// RequireAuth middleware
func RequireAuth(manager *session.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		session := GetSession(c)
		if session == nil {
			c.Redirect(http.StatusFound, "/auth/signin")
			c.Abort()
			return
		}
		c.Next()
	}
}

// RequireAuthJSON middleware
func RequireAuthJSON(manager *session.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		session := GetSession(c)
		if session == nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "Authentication required",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// GetSession retrieves the session from Gin context
func GetSession(c *gin.Context) *core.Session {
	if v, exists := c.Get("session"); exists {
		if session, ok := v.(*core.Session); ok {
			return session
		}
	}
	return nil
}

// GetUser retrieves the user from Gin context
func GetUser(c *gin.Context) *core.User {
	if v, exists := c.Get("user"); exists {
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

// NewHandler creates a new Gin auth handler
func NewHandler(dbAdapter core.Adapter, sessionManager *session.Manager, config *auth.Config) *Handler {
	return &Handler{
		handler: auth.NewHandler(dbAdapter, sessionManager, config),
	}
}

// RegisterRoutes registers routes on a Gin router group or engine
func (h *Handler) RegisterRoutes(r gin.IRouter) {
	r.POST("/auth/signup", h.SignUp)
	r.POST("/auth/signin", h.SignIn)
	r.POST("/auth/signout", h.SignOut)
	r.GET("/auth/session", h.GetSession)
}

// SignUp handler
func (h *Handler) SignUp(c *gin.Context) {
	h.handler.SignUp(c.Writer, c.Request)
}

// SignIn handler
func (h *Handler) SignIn(c *gin.Context) {
	h.handler.SignIn(c.Writer, c.Request)
}

// SignOut handler
func (h *Handler) SignOut(c *gin.Context) {
	h.handler.SignOut(c.Writer, c.Request)
}

// GetSession retrieves the current session
func (h *Handler) GetSession(c *gin.Context) {
	session := GetSession(c)
	user := GetUser(c)

	if session == nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "no_session",
			"message": "No active session",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user":    user,
		"session": session,
	})
}
