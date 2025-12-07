package fiber

import (
	"github.com/gofiber/fiber/v2"
	"github.com/marshallshelly/beaconauth/auth"
	"github.com/marshallshelly/beaconauth/core"
	"github.com/marshallshelly/beaconauth/session"
)

// SessionMiddleware creates Fiber middleware that loads session from request
func SessionMiddleware(manager *session.Manager) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Extract token from cookie
		token := c.Cookies(manager.Config().CookieName)
		if token == "" {
			return c.Next()
		}

		// Get session from manager
		ctx := c.Context()
		session, user, err := manager.Get(ctx, token)
		if err != nil {
			return c.Next()
		}

		// Store session and user in Fiber locals
		c.Locals("session", session)
		if user != nil {
			c.Locals("user", user)
		}

		return c.Next()
	}
}

// RequireAuth creates Fiber middleware that requires a valid session
func RequireAuth(manager *session.Manager) fiber.Handler {
	return func(c *fiber.Ctx) error {
		session := GetSession(c)
		if session == nil {
			return c.Status(fiber.StatusUnauthorized).Redirect("/auth/signin")
		}

		return c.Next()
	}
}

// RequireAuthJSON creates Fiber middleware that requires authentication and returns JSON errors
func RequireAuthJSON(manager *session.Manager) fiber.Handler {
	return func(c *fiber.Ctx) error {
		session := GetSession(c)
		if session == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":   "unauthorized",
				"message": "Authentication required",
			})
		}

		return c.Next()
	}
}

// GetSession retrieves the session from Fiber context
func GetSession(c *fiber.Ctx) *core.Session {
	session, ok := c.Locals("session").(*core.Session)
	if !ok {
		return nil
	}
	return session
}

// GetUser retrieves the user from Fiber context
func GetUser(c *fiber.Ctx) *core.User {
	user, ok := c.Locals("user").(*core.User)
	if !ok {
		return nil
	}
	return user
}

// GetUserID retrieves the user ID from Fiber context
func GetUserID(c *fiber.Ctx) string {
	user := GetUser(c)
	if user == nil {
		return ""
	}
	return user.ID
}

// Handler wraps the auth.Handler for use with Fiber
type Handler struct {
	handler *auth.Handler
}

// NewHandler creates a new Fiber auth handler
func NewHandler(dbAdapter core.Adapter, sessionManager *session.Manager, config *auth.Config) *Handler {
	return &Handler{
		handler: auth.NewHandler(dbAdapter, sessionManager, config),
	}
}

// SignUp handles user registration in Fiber
func (h *Handler) SignUp(c *fiber.Ctx) error {
	var req auth.SignUpRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request",
			"message": "Invalid request body",
		})
	}

	// Convert to standard HTTP request/response for the handler
	return h.convertAndHandle(c, func(w *responseAdapter, r *requestAdapter) {
		h.handler.SignUp(w, r.Request)
	})
}

// SignIn handles user authentication in Fiber
func (h *Handler) SignIn(c *fiber.Ctx) error {
	var req auth.SignInRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request",
			"message": "Invalid request body",
		})
	}

	return h.convertAndHandle(c, func(w *responseAdapter, r *requestAdapter) {
		h.handler.SignIn(w, r.Request)
	})
}

// SignOut handles user logout in Fiber
func (h *Handler) SignOut(c *fiber.Ctx) error {
	return h.convertAndHandle(c, func(w *responseAdapter, r *requestAdapter) {
		h.handler.SignOut(w, r.Request)
	})
}

// GetSession retrieves the current session in Fiber
func (h *Handler) GetSession(c *fiber.Ctx) error {
	session := GetSession(c)
	user := GetUser(c)

	if session == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":   "no_session",
			"message": "No active session",
		})
	}

	return c.JSON(fiber.Map{
		"user":    user,
		"session": session,
	})
}

// convertAndHandle converts Fiber context to standard HTTP and calls the handler
func (h *Handler) convertAndHandle(c *fiber.Ctx, handlerFunc func(*responseAdapter, *requestAdapter)) error {
	w := newResponseAdapter(c)
	r := newRequestAdapter(c)

	handlerFunc(w, r)

	return w.flush()
}
