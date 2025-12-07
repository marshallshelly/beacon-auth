package twofa

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/marshallshelly/beacon-auth/core"
	"github.com/marshallshelly/beacon-auth/plugin"
	"github.com/pquerna/otp/totp"
)

// TwoFAPlugin implements Two-Factor Authentication
type TwoFAPlugin struct {
	*plugin.BasePlugin
	ctx *core.AuthContext
}

// New creates a new TwoFA plugin
func New() *TwoFAPlugin {
	return &TwoFAPlugin{
		BasePlugin: plugin.NewBasePlugin("two_factor"),
	}
}

// Init initializes the plugin
func (p *TwoFAPlugin) Init(ctx *core.AuthContext) error {
	p.ctx = ctx
	return nil
}

// Endpoints returns the plugin endpoints
func (p *TwoFAPlugin) Endpoints() map[string]core.Endpoint {
	return map[string]core.Endpoint{
		"/2fa/generate": {Method: "POST", Handler: p.auth(p.handleGenerate)},
		"/2fa/enable":   {Method: "POST", Handler: p.auth(p.handleEnable)},
		"/2fa/verify":   {Method: "POST", Handler: p.handleVerify}, // No auth check as it might be used during login process
		"/2fa/disable":  {Method: "POST", Handler: p.auth(p.handleDisable)},
	}
}

type generateResponse struct {
	Secret  string `json:"secret"`
	TotpURI string `json:"totpURI"`
}

type enableRequest struct {
	Secret string `json:"secret"` // user confirms secret
	Code   string `json:"code"`
}

type verifyRequest struct {
	Code string `json:"code"`
}

// Helper middleware for plugin routes
func (p *TwoFAPlugin) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookieName := p.ctx.Config.Session.CookieName
		c, err := r.Cookie(cookieName)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		session, _, err := p.ctx.SessionManager.Get(r.Context(), c.Value)
		if err != nil || session == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
}

func (p *TwoFAPlugin) getSession(r *http.Request) (*core.Session, *core.User) {
	cookieName := p.ctx.Config.Session.CookieName
	c, err := r.Cookie(cookieName)
	if err != nil {
		return nil, nil
	}
	session, user, _ := p.ctx.SessionManager.Get(r.Context(), c.Value)
	return session, user
}

func (p *TwoFAPlugin) handleGenerate(w http.ResponseWriter, r *http.Request) {
	_, user := p.getSession(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      p.ctx.Config.AppName,
		AccountName: user.Email,
	})
	if err != nil {
		http.Error(w, "Failed to generate TOTP", http.StatusInternalServerError)
		return
	}

	secret := key.Secret()

	// Upsert secret
	err = p.saveSecret(r.Context(), user.ID, secret, false)
	if err != nil {
		http.Error(w, "Failed to save secret", http.StatusInternalServerError)
		return
	}

	// Explicitly ignore error to satisfy lint
	_ = json.NewEncoder(w).Encode(generateResponse{
		Secret:  secret,
		TotpURI: key.String(),
	})
}

func (p *TwoFAPlugin) handleEnable(w http.ResponseWriter, r *http.Request) {
	var req enableRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	_, user := p.getSession(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Retrieve stored secret
	record, err := p.getSecret(r.Context(), user.ID)
	if err != nil || record == nil {
		http.Error(w, "No pending 2FA setup found", http.StatusBadRequest)
		return
	}

	// Verify code
	valid := totp.Validate(req.Code, record["secret"].(string))
	if !valid {
		http.Error(w, "Invalid code", http.StatusUnauthorized)
		return
	}

	// Backup codes generation could happen here

	// Mark confirmed
	err = p.saveSecret(r.Context(), user.ID, record["secret"].(string), true)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	// Update user fields
	_, err = p.ctx.DataManager.UpdateUser(r.Context(), user.ID, map[string]interface{}{
		"two_factor_enabled": true,
	})
	if err != nil {
		p.ctx.Logger.Error("Failed to update user 2fa status: %v", err)
	}

	_, _ = w.Write([]byte(`{"success":true}`))
}

func (p *TwoFAPlugin) handleDisable(w http.ResponseWriter, r *http.Request) {
	_, user := p.getSession(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Delete secret logic omitted

	// Update user
	_, err := p.ctx.DataManager.UpdateUser(r.Context(), user.ID, map[string]interface{}{
		"two_factor_enabled": false,
	})
	if err != nil {
		p.ctx.Logger.Error("Failed to disable 2fa: %v", err)
	}

	_, _ = w.Write([]byte(`{"success":true}`))
}

func (p *TwoFAPlugin) handleVerify(w http.ResponseWriter, r *http.Request) {
	// Logic for login verification
	_, _ = w.Write([]byte(`{"status":"pending implementation"}`))
}

// DB Helpers
func (p *TwoFAPlugin) saveSecret(ctx context.Context, userID, secret string, confirmed bool) error {
	query := &core.Query{
		Model: "two_factors",
		Where: []core.WhereClause{{Field: "user_id", Operator: core.OpEqual, Value: userID}},
	}
	existing, _ := p.ctx.Adapter.FindOne(ctx, query)

	data := map[string]interface{}{
		"user_id":    userID,
		"secret":     secret,
		"confirmed":  confirmed,
		"updated_at": time.Now(),
	}

	if existing != nil {
		_, err := p.ctx.Adapter.Update(ctx, query, data)
		return err
	}

	// Create
	data["id"] = "2fa_" + userID
	data["created_at"] = time.Now()
	_, err := p.ctx.Adapter.Create(ctx, "two_factors", data)
	return err
}

func (p *TwoFAPlugin) getSecret(ctx context.Context, userID string) (map[string]interface{}, error) {
	query := &core.Query{
		Model: "two_factors",
		Where: []core.WhereClause{{Field: "user_id", Operator: core.OpEqual, Value: userID}},
	}
	return p.ctx.Adapter.FindOne(ctx, query)
}
