package emailpassword

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/marshallshelly/beacon-auth/core"
	"github.com/marshallshelly/beacon-auth/plugin"
)

// EmailPasswordPlugin implements email/password authentication
type EmailPasswordPlugin struct {
	*plugin.BasePlugin
	ctx *core.AuthContext
}

// New creates a new EmailPassword plugin
func New() *EmailPasswordPlugin {
	return &EmailPasswordPlugin{
		BasePlugin: plugin.NewBasePlugin("email_password"),
	}
}

// Init initializes the plugin
func (p *EmailPasswordPlugin) Init(ctx *core.AuthContext) error {
	p.ctx = ctx
	return nil
}

// Endpoints returns the plugin endpoints
func (p *EmailPasswordPlugin) Endpoints() map[string]plugin.Endpoint {
	return map[string]plugin.Endpoint{
		"/register": {
			Method:  "POST",
			Handler: p.handleRegister,
		},
		"/login": {
			Method:  "POST",
			Handler: p.handleLogin,
		},
	}
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (p *EmailPasswordPlugin) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Password == "" {
		http.Error(w, "Email and password are required", http.StatusBadRequest)
		return
	}

	if len(req.Password) < p.ctx.Config.EmailPassword.MinPasswordLength {
		http.Error(w, "Password too short", http.StatusBadRequest)
		return
	}

	// Check if user exists (optional, CreateUser might fail later)
	// But giving distinct error is nice.
	existingUser, _ := p.ctx.DataManager.FindUserByEmail(r.Context(), req.Email)
	if existingUser != nil {
		http.Error(w, "Email already exists", http.StatusConflict)
		return
	}

	// Hash password
	hash, err := p.ctx.PasswordHasher.Hash(req.Password)
	if err != nil {
		p.ctx.Logger.Error("Failed to hash password: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Create User
	user, err := p.ctx.DataManager.CreateUser(r.Context(), req.Email, req.Name)
	if err != nil {
		p.ctx.Logger.Error("Failed to create user: %v", err)
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	// Create Credential Account
	_, err = p.ctx.DataManager.CreateCredentialAccount(r.Context(), user.ID, req.Email, hash)
	if err != nil {
		p.ctx.Logger.Error("Failed to create account: %v", err)
		http.Error(w, "Failed to create account", http.StatusInternalServerError)
		return
	}

	// Create Session
	p.createSessionAndResponse(w, r, user.ID, user)
}

func (p *EmailPasswordPlugin) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Find account
	// We use "local" provider and email as account ID
	account, err := p.ctx.DataManager.FindAccountByProvider(r.Context(), "local", req.Email)
	if err != nil {
		p.ctx.Logger.Error("Database error finding account: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if account == nil {
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}

	// Verify password
	valid, err := p.ctx.PasswordHasher.Verify(req.Password, account.Password)
	if err != nil {
		p.ctx.Logger.Error("Error verifying password: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if !valid {
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}

	// Get user (for response)
	user, err := p.ctx.DataManager.FindUserByEmail(r.Context(), req.Email)
	if err != nil {
		p.ctx.Logger.Warn("Could not find user details for valid account: %v", err)
	}

	p.createSessionAndResponse(w, r, account.UserID, user)
}

func (p *EmailPasswordPlugin) createSessionAndResponse(w http.ResponseWriter, r *http.Request, userID string, user *core.User) {
	_, _, token, err := p.ctx.SessionManager.Create(r.Context(), userID, nil)
	if err != nil {
		p.ctx.Logger.Error("Failed to create session: %v", err)
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	// Set Session Cookie
	sessionConfig := p.ctx.Config.Session
	http.SetCookie(w, &http.Cookie{
		Name:     sessionConfig.CookieName,
		Value:    token,
		Path:     sessionConfig.CookiePath,
		Domain:   sessionConfig.CookieDomain,
		Expires:  time.Now().Add(sessionConfig.ExpiresIn),
		Secure:   sessionConfig.CookieSecure,
		HttpOnly: sessionConfig.CookieHTTPOnly,
		SameSite: parseSameSite(sessionConfig.CookieSameSite),
	})

	w.Header().Set("Content-Type", "application/json")
	if user != nil {
		_ = json.NewEncoder(w).Encode(user)
	} else {
		// Should not happen if logic is correct
		_, _ = w.Write([]byte(`{"success":true}`))
	}
}

func parseSameSite(s string) http.SameSite {
	switch strings.ToLower(s) {
	case "lax":
		return http.SameSiteLaxMode
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}
