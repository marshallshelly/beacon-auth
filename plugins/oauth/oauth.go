package oauth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/marshallshelly/beacon-auth/core"
	"github.com/marshallshelly/beacon-auth/plugin"
	"github.com/marshallshelly/beacon-auth/plugins/oauth/providers"
)

// OAuthPlugin implements the OAuth 2.0 plugin
type OAuthPlugin struct {
	*plugin.BasePlugin
	providers map[string]providers.OAuthProvider
	ctx       *core.AuthContext
}

// New creates a new OAuth plugin with the given providers
func New(provs ...providers.OAuthProvider) *OAuthPlugin {
	p := &OAuthPlugin{
		BasePlugin: plugin.NewBasePlugin("oauth"),
		providers:  make(map[string]providers.OAuthProvider),
	}

	for _, prov := range provs {
		p.providers[prov.ID()] = prov
	}

	return p
}

// Init initializes the plugin
func (p *OAuthPlugin) Init(ctx *core.AuthContext) error {
	p.ctx = ctx
	for _, prov := range p.providers {
		if err := prov.Init(); err != nil {
			return err
		}
	}
	return nil
}

// Endpoints returns the OAuth endpoints
func (p *OAuthPlugin) Endpoints() map[string]plugin.Endpoint {
	endpoints := make(map[string]plugin.Endpoint)

	for id, prov := range p.providers {
		// Capture closure variables
		providerID := id
		provider := prov

		endpoints["/oauth/"+providerID+"/login"] = plugin.Endpoint{
			Method: "GET",
			Handler: func(w http.ResponseWriter, r *http.Request) {
				p.handleLogin(w, r, provider)
			},
		}
		endpoints["/oauth/"+providerID+"/callback"] = plugin.Endpoint{
			Method: "GET",
			Handler: func(w http.ResponseWriter, r *http.Request) {
				p.handleCallback(w, r, provider)
			},
		}
	}

	return endpoints
}

func (p *OAuthPlugin) handleLogin(w http.ResponseWriter, r *http.Request, provider providers.OAuthProvider) {
	state, err := generateRandomString(32)
	if err != nil {
		p.ctx.Logger.Error("Failed to generate state: %v", err)
		http.Error(w, "Failed to generate state", http.StatusInternalServerError)
		return
	}

	// Determine redirect URI
	baseURL := p.ctx.Config.BaseURL
	basePath := p.ctx.Config.BasePath
	// Ensure basePath starts with / if not empty, and doesn't end with /
	if basePath != "" && !strings.HasPrefix(basePath, "/") {
		basePath = "/" + basePath
	}
	basePath = strings.TrimRight(basePath, "/")

	redirectURI := fmt.Sprintf("%s%s/oauth/%s/callback", baseURL, basePath, provider.ID())

	// Set state cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		Secure:   p.ctx.Config.Advanced.UseSecureCookies,
		MaxAge:   300, // 5 minutes
	})

	authURL, err := provider.CreateAuthorizationURL(state, redirectURI, nil)
	if err != nil {
		p.ctx.Logger.Error("Failed to create authorization URL: %v", err)
		http.Error(w, "Failed to create authorization URL", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, authURL.String(), http.StatusTemporaryRedirect)
}

func (p *OAuthPlugin) handleCallback(w http.ResponseWriter, r *http.Request, provider providers.OAuthProvider) {
	// Verify state
	state := r.URL.Query().Get("state")
	cookie, err := r.Cookie("oauth_state")
	if err != nil || cookie.Value != state {
		http.Error(w, "Invalid state param", http.StatusBadRequest)
		return
	}

	// Clear state cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   p.ctx.Config.Advanced.UseSecureCookies,
		MaxAge:   -1,
	})

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Missing code param", http.StatusBadRequest)
		return
	}

	baseURL := p.ctx.Config.BaseURL
	basePath := p.ctx.Config.BasePath
	if basePath != "" && !strings.HasPrefix(basePath, "/") {
		basePath = "/" + basePath
	}
	basePath = strings.TrimRight(basePath, "/")

	redirectURI := fmt.Sprintf("%s%s/oauth/%s/callback", baseURL, basePath, provider.ID())

	tokens, err := provider.ExchangeCode(r.Context(), code, "", redirectURI)
	if err != nil {
		p.ctx.Logger.Error("Failed to exchange code: %v", err)
		http.Error(w, "Failed to exchange code", http.StatusInternalServerError)
		return
	}

	userInfo, err := provider.GetUserInfo(r.Context(), tokens.AccessToken)
	if err != nil {
		p.ctx.Logger.Error("Failed to get user info: %v", err)
		http.Error(w, "Failed to get user info", http.StatusInternalServerError)
		return
	}

	// Check if account exists
	account, err := p.ctx.DataManager.FindAccountByProvider(r.Context(), provider.ID(), userInfo.ID)
	if err != nil {
		p.ctx.Logger.Error("Database error finding account: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	var userID string
	if account != nil {
		userID = account.UserID
	} else {
		// Link or create user
		var user *core.User
		if userInfo.Email != "" {
			user, err = p.ctx.DataManager.FindUserByEmail(r.Context(), userInfo.Email)
			if err != nil && err != core.ErrUserNotFound {
				p.ctx.Logger.Error("Database error finding user: %v", err)
				http.Error(w, "Database error", http.StatusInternalServerError)
				return
			}
		}

		if user == nil {
			// Create new user
			user, err = p.ctx.DataManager.CreateUser(r.Context(), userInfo.Email, userInfo.Name)
			if err != nil {
				p.ctx.Logger.Error("Failed to create user: %v", err)
				http.Error(w, "Failed to create user", http.StatusInternalServerError)
				return
			}
			// Update image if available
			if userInfo.Picture != "" {
				// Note: UpdateUser expects map of updates.
				// For MVP we just create. We can add update logic later if needed.
				p.ctx.Logger.Debug("User has picture (%s), skipping update for now", userInfo.Picture)
			}
		}
		userID = user.ID

		// Create account
		_, err = p.ctx.DataManager.CreateOAuthAccount(r.Context(), userID, provider.ID(), userInfo.ID, tokens.AccessToken, tokens.RefreshToken, tokens.ExpiresAt)
		if err != nil {
			p.ctx.Logger.Error("Failed to create account: %v", err)
			http.Error(w, "Failed to create account", http.StatusInternalServerError)
			return
		}
	}

	// Create Session
	// Note: CreateSession takes SessionOptions.
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

	// Redirect to home
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

func generateRandomString(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
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
