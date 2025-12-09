package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/marshallshelly/beacon-auth/adapter"
	"github.com/marshallshelly/beacon-auth/core"
	"github.com/marshallshelly/beacon-auth/crypto"
	"github.com/marshallshelly/beacon-auth/session"
)

// Handler provides HTTP handlers for authentication
type Handler struct {
	internal       *adapter.InternalAdapter
	sessionManager *session.Manager
	hasher         crypto.PasswordHasher
	config         *Config
}

// Config holds authentication handler configuration
type Config struct {
	MinPasswordLength   int
	RequireVerification bool
	AllowSignup         bool
}

// NewHandler creates a new authentication handler
func NewHandler(dbAdapter core.Adapter, sessionManager *session.Manager, config *Config) *Handler {
	if config == nil {
		config = &Config{
			MinPasswordLength:   8,
			RequireVerification: false,
			AllowSignup:         true,
		}
	}

	return &Handler{
		internal:       adapter.NewInternalAdapter(dbAdapter, nil),
		sessionManager: sessionManager,
		hasher:         crypto.NewArgon2Hasher(),
		config:         config,
	}
}

// SignUpRequest represents a sign up request
type SignUpRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name,omitempty"`
}

// SignInRequest represents a sign in request
type SignInRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AuthResponse represents an authentication response
type AuthResponse struct {
	User    *core.User    `json:"user"`
	Session *core.Session `json:"session,omitempty"`
	Token   string        `json:"token,omitempty"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// SignUp handles user registration
func (h *Handler) SignUp(w http.ResponseWriter, r *http.Request) {
	if !h.config.AllowSignup {
		h.writeError(w, http.StatusForbidden, "signup_disabled", "Sign up is disabled")
		return
	}

	var req SignUpRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_request", "Invalid request body")
		return
	}

	// Validate input
	if err := h.validateSignUpRequest(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	ctx := r.Context()

	// Check if user already exists
	existingUser, err := h.internal.FindUserByEmail(ctx, req.Email)
	if err != nil && err != core.ErrUserNotFound {
		h.writeError(w, http.StatusInternalServerError, "database_error", "Failed to check existing user")
		return
	}

	if existingUser != nil {
		h.writeError(w, http.StatusConflict, "user_exists", "User with this email already exists")
		return
	}

	// Hash password
	hashedPassword, err := h.hasher.Hash(req.Password)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "hash_error", "Failed to hash password")
		return
	}

	// Create user with hashed password
	user, err := h.createUserWithPassword(ctx, req.Email, req.Name, hashedPassword)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "create_error", "Failed to create user")
		return
	}

	// Create session
	session, _, token, err := h.sessionManager.Create(ctx, user.ID, &core.SessionOptions{
		IPAddress: getIPAddress(r),
		UserAgent: r.UserAgent(),
	})
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "session_error", "Failed to create session")
		return
	}

	// Set session cookie
	h.setSessionCookie(w, token, session.ExpiresAt)

	// Send response
	h.writeJSON(w, http.StatusCreated, &AuthResponse{
		User:    user,
		Session: session,
		Token:   token,
	})
}

// SignIn handles user authentication
func (h *Handler) SignIn(w http.ResponseWriter, r *http.Request) {
	var req SignInRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid_request", "Invalid request body")
		return
	}

	// Validate input
	if req.Email == "" || req.Password == "" {
		h.writeError(w, http.StatusBadRequest, "validation_error", "Email and password are required")
		return
	}

	ctx := r.Context()

	// Find user by email
	user, err := h.internal.FindUserByEmail(ctx, req.Email)
	if err != nil {
		if err == core.ErrUserNotFound {
			h.writeError(w, http.StatusUnauthorized, "invalid_credentials", "Invalid email or password")
			return
		}
		h.writeError(w, http.StatusInternalServerError, "database_error", "Failed to find user")
		return
	}

	// Get user's password hash
	passwordHash, err := h.getUserPasswordHash(ctx, user.ID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "database_error", "Failed to retrieve credentials")
		return
	}

	// Verify password
	valid, err := h.hasher.Verify(req.Password, passwordHash)
	if err != nil || !valid {
		h.writeError(w, http.StatusUnauthorized, "invalid_credentials", "Invalid email or password")
		return
	}

	// Check if email verification is required
	if h.config.RequireVerification && !user.EmailVerified {
		h.writeError(w, http.StatusForbidden, "email_not_verified", "Please verify your email before signing in")
		return
	}

	// Create session
	session, _, token, err := h.sessionManager.Create(ctx, user.ID, &core.SessionOptions{
		IPAddress: getIPAddress(r),
		UserAgent: r.UserAgent(),
	})
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "session_error", "Failed to create session")
		return
	}

	// Set session cookie
	h.setSessionCookie(w, token, session.ExpiresAt)

	// Send response
	h.writeJSON(w, http.StatusOK, &AuthResponse{
		User:    user,
		Session: session,
		Token:   token,
	})
}

// SignOut handles user logout
func (h *Handler) SignOut(w http.ResponseWriter, r *http.Request) {
	// Get session from cookie
	cookie, err := r.Cookie(h.sessionManager.Config().CookieName)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "no_session", "No session found")
		return
	}

	ctx := r.Context()

	// Delete the session
	_ = h.sessionManager.Delete(ctx, cookie.Value)
	// Ignore error as session cookie is being cleared anyway

	// Clear session cookie
	h.clearSessionCookie(w)

	// Send response
	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Signed out successfully",
	})
}

// GetSession retrieves the current session
func (h *Handler) GetSession(w http.ResponseWriter, r *http.Request) {
	session := core.GetSession(r.Context())
	user := core.GetUser(r.Context())

	if session == nil {
		h.writeError(w, http.StatusUnauthorized, "no_session", "No active session")
		return
	}

	h.writeJSON(w, http.StatusOK, &AuthResponse{
		User:    user,
		Session: session,
	})
}

// Helper methods

func (h *Handler) validateSignUpRequest(req *SignUpRequest) error {
	if req.Email == "" {
		return fmt.Errorf("email is required")
	}

	if req.Password == "" {
		return fmt.Errorf("password is required")
	}

	if len(req.Password) < h.config.MinPasswordLength {
		return fmt.Errorf("password must be at least %d characters", h.config.MinPasswordLength)
	}

	// Basic email validation
	if !isValidEmail(req.Email) {
		return fmt.Errorf("invalid email format")
	}

	return nil
}

func (h *Handler) createUserWithPassword(ctx context.Context, email, name, hashedPassword string) (*core.User, error) {
	// Create user
	user, err := h.internal.CreateUser(ctx, email, name)
	if err != nil {
		return nil, err
	}

	// Generate account ID
	accountID, err := crypto.GenerateID()
	if err != nil {
		return nil, err
	}

	// Store password hash in accounts table
	_, err = h.internal.Adapter().Create(ctx, "accounts", map[string]interface{}{
		"id":            accountID,
		"user_id":       user.ID,
		"provider":      "credential",
		"provider_type": "email",
		"password_hash": hashedPassword,
		"created_at":    time.Now(),
		"updated_at":    time.Now(),
	})
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (h *Handler) getUserPasswordHash(ctx context.Context, userID string) (string, error) {
	query := &core.Query{
		Model: "accounts",
		Where: []core.WhereClause{
			{Field: "user_id", Operator: core.OpEqual, Value: userID},
			{Field: "provider_type", Operator: core.OpEqual, Value: "email"},
		},
	}

	result, err := h.internal.Adapter().FindOne(ctx, query)
	if err != nil {
		return "", err
	}

	if result == nil {
		return "", core.ErrUserNotFound
	}

	hash, ok := result["password_hash"].(string)
	if !ok {
		return "", fmt.Errorf("password hash not found")
	}

	return hash, nil
}

func (h *Handler) setSessionCookie(w http.ResponseWriter, token string, expiresAt time.Time) {
	config := h.sessionManager.Config()
	cookie := &http.Cookie{
		Name:     config.CookieName,
		Value:    token,
		Path:     config.CookiePath,
		Domain:   config.CookieDomain,
		Expires:  expiresAt,
		Secure:   config.CookieSecure,
		HttpOnly: config.CookieHTTPOnly,
		SameSite: parseSameSite(config.CookieSameSite),
	}

	http.SetCookie(w, cookie)
}

func (h *Handler) clearSessionCookie(w http.ResponseWriter) {
	config := h.sessionManager.Config()
	cookie := &http.Cookie{
		Name:     config.CookieName,
		Value:    "",
		Path:     config.CookiePath,
		Domain:   config.CookieDomain,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		Secure:   config.CookieSecure,
		HttpOnly: config.CookieHTTPOnly,
	}

	http.SetCookie(w, cookie)
}

func (h *Handler) writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		// Log error but don't return since headers are already written
		// In production, you'd use a proper logger here
		_ = err
	}
}

func (h *Handler) writeError(w http.ResponseWriter, status int, code, message string) {
	h.writeJSON(w, status, &ErrorResponse{
		Error:   code,
		Message: message,
	})
}

func getIPAddress(r *http.Request) string {
	// Check X-Forwarded-For header first
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	return r.RemoteAddr
}

func parseSameSite(s string) http.SameSite {
	switch s {
	case "strict":
		return http.SameSiteStrictMode
	case "lax":
		return http.SameSiteLaxMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}

func isValidEmail(email string) bool {
	// Basic email validation
	// In production, use a proper email validation library
	return len(email) > 3 &&
		contains(email, "@") &&
		contains(email, ".")
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
