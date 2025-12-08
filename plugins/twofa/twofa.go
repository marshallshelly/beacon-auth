package twofa

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
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
func (p *TwoFAPlugin) Endpoints() map[string]plugin.Endpoint {
	return map[string]plugin.Endpoint{
		"/2fa/generate": {Method: "POST", Handler: p.auth(p.handleGenerate)},
		"/2fa/enable":   {Method: "POST", Handler: p.auth(p.handleEnable)},
		"/2fa/verify":   {Method: "POST", Handler: p.handleVerify}, // No auth check as it might be used during login process
		"/2fa/disable":  {Method: "POST", Handler: p.auth(p.handleDisable)},
	}
}

type generateResponse struct {
	Secret      string   `json:"secret"`
	TotpURI     string   `json:"totpURI"`
	BackupCodes []string `json:"backupCodes,omitempty"`
}

type enableRequest struct {
	Secret string `json:"secret"` // user confirms secret
	Code   string `json:"code"`
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

	// Generate backup codes
	backupCodes, err := p.generateBackupCodes(r.Context(), user.ID, 10)
	if err != nil {
		p.ctx.Logger.Warn("Failed to generate backup codes: %v", err)
		// Continue without backup codes
	}

	// Upsert secret
	err = p.saveSecret(r.Context(), user.ID, secret, false)
	if err != nil {
		http.Error(w, "Failed to save secret", http.StatusInternalServerError)
		return
	}

	// Explicitly ignore error to satisfy lint
	_ = json.NewEncoder(w).Encode(generateResponse{
		Secret:      secret,
		TotpURI:     key.String(),
		BackupCodes: backupCodes,
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

	// Delete secret
	query := &core.Query{
		Model: "two_factors",
		Where: []core.WhereClause{{Field: "user_id", Operator: core.OpEqual, Value: user.ID}},
	}
	_ = p.ctx.Adapter.Delete(r.Context(), query)

	// Delete backup codes
	backupQuery := &core.Query{
		Model: "two_factor_backup_codes",
		Where: []core.WhereClause{{Field: "user_id", Operator: core.OpEqual, Value: user.ID}},
	}
	_, _ = p.ctx.Adapter.DeleteMany(r.Context(), backupQuery)

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
	var req struct {
		Email string `json:"email"`
		Code  string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Code == "" {
		http.Error(w, "Email and code are required", http.StatusBadRequest)
		return
	}

	// Find user
	user, err := p.ctx.DataManager.FindUserByEmail(r.Context(), req.Email)
	if err != nil || user == nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Check if 2FA is enabled
	if !user.TwoFactorEnabled {
		http.Error(w, "2FA not enabled for this user", http.StatusBadRequest)
		return
	}

	// Get stored secret
	record, err := p.getSecret(r.Context(), user.ID)
	if err != nil || record == nil {
		p.ctx.Logger.Error("Failed to get 2FA secret: %v", err)
		http.Error(w, "2FA not configured", http.StatusBadRequest)
		return
	}

	// Check if confirmed
	confirmed, ok := record["confirmed"].(bool)
	if !ok || !confirmed {
		http.Error(w, "2FA not confirmed", http.StatusBadRequest)
		return
	}

	// Verify TOTP code
	secret, ok := record["secret"].(string)
	if !ok {
		http.Error(w, "Invalid 2FA configuration", http.StatusInternalServerError)
		return
	}

	valid := totp.Validate(req.Code, secret)
	if !valid {
		// Check backup codes if TOTP fails
		if p.checkBackupCode(r.Context(), user.ID, req.Code) {
			// Backup code is valid, consume it
			_ = p.consumeBackupCode(r.Context(), user.ID, req.Code)
		} else {
			http.Error(w, "Invalid code", http.StatusUnauthorized)
			return
		}
	}

	// Create full session
	_, _, token, err := p.ctx.SessionManager.Create(r.Context(), user.ID, nil)
	if err != nil {
		p.ctx.Logger.Error("Failed to create session: %v", err)
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	// Set session cookie
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
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"user":    user,
	})
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

// Backup code helpers
func (p *TwoFAPlugin) generateBackupCodes(ctx context.Context, userID string, count int) ([]string, error) {
	codes := make([]string, count)

	for i := 0; i < count; i++ {
		// Generate a random 8-character hex code
		b := make([]byte, 4)
		if _, err := rand.Read(b); err != nil {
			return nil, err
		}
		codes[i] = hex.EncodeToString(b)

		// Store in database
		data := map[string]interface{}{
			"id":         "backup_" + userID + "_" + codes[i],
			"user_id":    userID,
			"code":       codes[i],
			"used":       false,
			"created_at": time.Now(),
		}

		_, _ = p.ctx.Adapter.Create(ctx, "two_factor_backup_codes", data)
	}

	return codes, nil
}

func (p *TwoFAPlugin) checkBackupCode(ctx context.Context, userID, code string) bool {
	query := &core.Query{
		Model: "two_factor_backup_codes",
		Where: []core.WhereClause{
			{Field: "user_id", Operator: core.OpEqual, Value: userID},
			{Field: "code", Operator: core.OpEqual, Value: code},
			{Field: "used", Operator: core.OpEqual, Value: false},
		},
	}

	result, err := p.ctx.Adapter.FindOne(ctx, query)
	return err == nil && result != nil
}

func (p *TwoFAPlugin) consumeBackupCode(ctx context.Context, userID, code string) error {
	query := &core.Query{
		Model: "two_factor_backup_codes",
		Where: []core.WhereClause{
			{Field: "user_id", Operator: core.OpEqual, Value: userID},
			{Field: "code", Operator: core.OpEqual, Value: code},
		},
	}

	data := map[string]interface{}{
		"used":       true,
		"used_at":    time.Now(),
		"updated_at": time.Now(),
	}

	_, err := p.ctx.Adapter.Update(ctx, query, data)
	return err
}
