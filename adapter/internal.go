package adapter

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"strconv"
	"time"

	"github.com/marshallshelly/beacon-auth/core"
)

// IDStrategy defines how IDs are generated
type IDStrategy string

const (
	// IDStrategyApplication generates unique string IDs in the application (default)
	IDStrategyApplication IDStrategy = "application"
	// IDStrategyDatabase defers ID generation to the database (UUID or Serial)
	IDStrategyDatabase IDStrategy = "database"
)

// InternalAdapter provides high-level database operations
type InternalAdapter struct {
	adapter    core.Adapter
	idStrategy IDStrategy
}

// InternalAdapterConfig configuration for InternalAdapter
type InternalAdapterConfig struct {
	IDStrategy IDStrategy
}

// Adapter returns the underlying adapter
func (ia *InternalAdapter) Adapter() core.Adapter {
	return ia.adapter
}

// NewInternalAdapter creates a new internal adapter
func NewInternalAdapter(adapter core.Adapter, config *InternalAdapterConfig) *InternalAdapter {
	strategy := IDStrategyApplication
	if config != nil && config.IDStrategy != "" {
		strategy = config.IDStrategy
	}
	return &InternalAdapter{
		adapter:    adapter,
		idStrategy: strategy,
	}
}

// generateID returns a new ID if using Application strategy, or nil if Database strategy
func (ia *InternalAdapter) generateID() interface{} {
	if ia.idStrategy == IDStrategyDatabase {
		return nil // Let the DB handle it
	}
	// Application strategy
	return generateRandomStringID()
}

// CreateUser creates a new user
func (ia *InternalAdapter) CreateUser(ctx context.Context, email, name string) (*core.User, error) {
	now := time.Now()
	data := map[string]interface{}{
		"email":          email,
		"name":           name,
		"email_verified": false,
		"created_at":     now,
		"updated_at":     now,
	}

	if id := ia.generateID(); id != nil {
		data["id"] = id
	}

	result, err := ia.adapter.Create(ctx, "users", data)
	if err != nil {
		return nil, err
	}

	return mapToUser(result), nil
}

// FindUserByEmail finds a user by email
func (ia *InternalAdapter) FindUserByEmail(ctx context.Context, email string) (*core.User, error) {
	query := &core.Query{
		Model: "users",
		Where: []core.WhereClause{
			{Field: "email", Operator: core.OpEqual, Value: email},
		},
	}

	result, err := ia.adapter.FindOne(ctx, query)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, core.ErrUserNotFound
	}

	return mapToUser(result), nil
}

// FindUserByID finds a user by ID
func (ia *InternalAdapter) FindUserByID(ctx context.Context, id string) (*core.User, error) {
	query := &core.Query{
		Model: "users",
		Where: []core.WhereClause{
			{Field: "id", Operator: core.OpEqual, Value: id},
		},
	}

	result, err := ia.adapter.FindOne(ctx, query)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, core.ErrUserNotFound
	}

	return mapToUser(result), nil
}

// UpdateUser updates a user
func (ia *InternalAdapter) UpdateUser(ctx context.Context, userID string, data map[string]interface{}) (*core.User, error) {
	data["updated_at"] = time.Now()

	query := &core.Query{
		Model: "users",
		Where: []core.WhereClause{
			{Field: "id", Operator: core.OpEqual, Value: userID},
		},
	}

	result, err := ia.adapter.Update(ctx, query, data)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, core.ErrUserNotFound
	}

	return mapToUser(result), nil
}

// CreateSession creates a new session
func (ia *InternalAdapter) CreateSession(ctx context.Context, userID string, opts *core.SessionOptions) (*core.Session, error) {
	now := time.Now()
	expiresAt := now.Add(7 * 24 * time.Hour)

	if opts != nil && opts.ExpiresIn != nil {
		expiresAt = now.Add(*opts.ExpiresIn)
	}

	token, err := generateSessionToken()
	if err != nil {
		return nil, err
	}

	data := map[string]interface{}{
		"user_id":    userID,
		"token":      token,
		"expires_at": expiresAt,
		"created_at": now,
		"updated_at": now,
	}

	if id := ia.generateID(); id != nil {
		data["id"] = id
	}

	if opts != nil {
		if opts.IPAddress != "" {
			data["ip_address"] = opts.IPAddress
		}
		if opts.UserAgent != "" {
			data["user_agent"] = opts.UserAgent
		}
	}

	result, err := ia.adapter.Create(ctx, "sessions", data)
	if err != nil {
		return nil, err
	}

	return mapToSession(result), nil
}

// FindSessionWithUser finds a session and joins with user
func (ia *InternalAdapter) FindSessionWithUser(ctx context.Context, token string) (*core.Session, *core.User, error) {
	// First find the session
	sessionQuery := &core.Query{
		Model: "sessions",
		Where: []core.WhereClause{
			{Field: "token", Operator: core.OpEqual, Value: token},
			{Field: "expires_at", Operator: core.OpGreaterThan, Value: time.Now()},
		},
	}

	sessionResult, err := ia.adapter.FindOne(ctx, sessionQuery)
	if err != nil {
		return nil, nil, err
	}
	if sessionResult == nil {
		return nil, nil, core.ErrSessionNotFound
	}

	session := mapToSession(sessionResult)

	// Then find the user
	userQuery := &core.Query{
		Model: "users",
		Where: []core.WhereClause{
			{Field: "id", Operator: core.OpEqual, Value: session.UserID},
		},
	}

	userResult, err := ia.adapter.FindOne(ctx, userQuery)
	if err != nil {
		return nil, nil, err
	}
	if userResult == nil {
		return session, nil, core.ErrUserNotFound
	}

	user := mapToUser(userResult)

	return session, user, nil
}

// RevokeSession revokes a session by token
func (ia *InternalAdapter) RevokeSession(ctx context.Context, token string) error {
	query := &core.Query{
		Model: "sessions",
		Where: []core.WhereClause{
			{Field: "token", Operator: core.OpEqual, Value: token},
		},
	}

	return ia.adapter.Delete(ctx, query)
}

// RevokeAllUserSessions revokes all sessions for a user
func (ia *InternalAdapter) RevokeAllUserSessions(ctx context.Context, userID string) (int64, error) {
	query := &core.Query{
		Model: "sessions",
		Where: []core.WhereClause{
			{Field: "user_id", Operator: core.OpEqual, Value: userID},
		},
	}

	return ia.adapter.DeleteMany(ctx, query)
}

// CreateAccount creates an authentication account
func (ia *InternalAdapter) CreateAccount(ctx context.Context, userID, provider, accountID string) (*core.Account, error) {
	now := time.Now()
	data := map[string]interface{}{
		"user_id":       userID,
		"account_id":    accountID,
		"provider_id":   provider,
		"provider_type": "credential",
		"created_at":    now,
		"updated_at":    now,
	}

	if id := ia.generateID(); id != nil {
		data["id"] = id
	}

	result, err := ia.adapter.Create(ctx, "accounts", data)
	if err != nil {
		return nil, err
	}

	return mapToAccount(result), nil
}

// CreateOAuthAccount creates an OAuth account with tokens
func (ia *InternalAdapter) CreateOAuthAccount(ctx context.Context, userID, provider, accountID, accessToken, refreshToken string, expiresAt *time.Time) (*core.Account, error) {
	now := time.Now()
	data := map[string]interface{}{
		"user_id":                 userID,
		"account_id":              accountID,
		"provider_id":             provider,
		"provider_type":           "oauth",
		"access_token":            accessToken,
		"refresh_token":           refreshToken,
		"access_token_expires_at": expiresAt,
		"created_at":              now,
		"updated_at":              now,
	}

	if id := ia.generateID(); id != nil {
		data["id"] = id
	}

	result, err := ia.adapter.Create(ctx, "accounts", data)
	if err != nil {
		return nil, err
	}

	return mapToAccount(result), nil
}

// CreateCredentialAccount creates a credential account
func (ia *InternalAdapter) CreateCredentialAccount(ctx context.Context, userID, identifier, passwordHash string) (*core.Account, error) {
	now := time.Now()
	data := map[string]interface{}{
		"user_id":       userID,
		"account_id":    identifier,
		"provider_id":   "local",
		"provider_type": "credential",
		"password":      passwordHash,
		"created_at":    now,
		"updated_at":    now,
	}

	if id := ia.generateID(); id != nil {
		data["id"] = id
	}

	result, err := ia.adapter.Create(ctx, "accounts", data)
	if err != nil {
		return nil, err
	}

	return mapToAccount(result), nil
}

// FindAccountByProvider finds an account by provider and account ID
func (ia *InternalAdapter) FindAccountByProvider(ctx context.Context, provider, accountID string) (*core.Account, error) {
	query := &core.Query{
		Model: "accounts",
		Where: []core.WhereClause{
			{Field: "provider_id", Operator: core.OpEqual, Value: provider},
			{Field: "account_id", Operator: core.OpEqual, Value: accountID},
		},
	}

	result, err := ia.adapter.FindOne(ctx, query)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}

	return mapToAccount(result), nil
}

// CreateVerification creates a verification token
func (ia *InternalAdapter) CreateVerification(ctx context.Context, identifier, verifyType string, expiresIn time.Duration) (*core.Verification, error) {
	token, err := generateVerificationToken()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	data := map[string]interface{}{
		"identifier": identifier,
		"token":      token,
		"type":       verifyType,
		"expires_at": now.Add(expiresIn),
		"created_at": now,
	}

	if id := ia.generateID(); id != nil {
		data["id"] = id
	}

	result, err := ia.adapter.Create(ctx, "verifications", data)
	if err != nil {
		return nil, err
	}

	return mapToVerification(result), nil
}

// FindVerification finds a verification by token
func (ia *InternalAdapter) FindVerification(ctx context.Context, token string) (*core.Verification, error) {
	query := &core.Query{
		Model: "verifications",
		Where: []core.WhereClause{
			{Field: "token", Operator: core.OpEqual, Value: token},
			{Field: "expires_at", Operator: core.OpGreaterThan, Value: time.Now()},
		},
	}

	result, err := ia.adapter.FindOne(ctx, query)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}

	return mapToVerification(result), nil
}

// Helper functions

func mapToUser(data map[string]interface{}) *core.User {
	user := &core.User{
		Metadata: make(map[string]interface{}),
	}

	knownFields := map[string]bool{
		"id":                 true,
		"email":              true,
		"email_verified":     true,
		"name":               true,
		"image":              true,
		"two_factor_enabled": true,
		"created_at":         true,
		"updated_at":         true,
		"role":               true,
		"banned":             true,
		"ban_reason":         true,
		"ban_expires":        true,
	}

	if id, ok := data["id"]; ok {
		user.ID = toString(id)
	}
	if email, ok := data["email"].(string); ok {
		user.Email = email
	}
	if emailVerified, ok := data["email_verified"].(bool); ok {
		user.EmailVerified = emailVerified
	}
	if name, ok := data["name"].(string); ok {
		user.Name = name
	}
	if image, ok := data["image"].(string); ok {
		user.Image = image
	}
	if twoFactor, ok := data["two_factor_enabled"].(bool); ok {
		user.TwoFactorEnabled = twoFactor
	}
	if role, ok := data["role"].(string); ok {
		user.Role = role
	}
	if banned, ok := data["banned"].(bool); ok {
		user.Banned = banned
	}
	if banReason, ok := data["ban_reason"].(string); ok {
		user.BanReason = banReason
	}
	if banExpires, ok := data["ban_expires"].(time.Time); ok {
		user.BanExpires = &banExpires
	} else if banExpires, ok := data["ban_expires"].(*time.Time); ok {
		user.BanExpires = banExpires
	}
	if createdAt, ok := data["created_at"].(time.Time); ok {
		user.CreatedAt = createdAt
	}
	if updatedAt, ok := data["updated_at"].(time.Time); ok {
		user.UpdatedAt = updatedAt
	}

	// Map remaining fields to Metadata
	for k, v := range data {
		if !knownFields[k] {
			user.Metadata[k] = v
		}
	}

	return user
}

func mapToSession(data map[string]interface{}) *core.Session {
	session := &core.Session{
		Metadata: make(map[string]interface{}),
	}

	knownFields := map[string]bool{
		"id":              true,
		"user_id":         true,
		"token":           true,
		"expires_at":      true,
		"ip_address":      true,
		"user_agent":      true,
		"created_at":      true,
		"updated_at":      true,
		"impersonated_by": true,
	}

	if id, ok := data["id"]; ok {
		session.ID = toString(id)
	}
	if userID, ok := data["user_id"]; ok {
		session.UserID = toString(userID)
	}
	if token, ok := data["token"].(string); ok {
		session.Token = token
	}
	if expiresAt, ok := data["expires_at"].(time.Time); ok {
		session.ExpiresAt = expiresAt
	}
	if ipAddress, ok := data["ip_address"].(string); ok {
		session.IPAddress = ipAddress
	}
	if userAgent, ok := data["user_agent"].(string); ok {
		session.UserAgent = userAgent
	}
	if createdAt, ok := data["created_at"].(time.Time); ok {
		session.CreatedAt = createdAt
	}
	if updatedAt, ok := data["updated_at"].(time.Time); ok {
		session.UpdatedAt = updatedAt
	}
	if impersonatedBy, ok := data["impersonated_by"].(string); ok {
		session.ImpersonatedBy = impersonatedBy
	}

	for k, v := range data {
		if !knownFields[k] {
			session.Metadata[k] = v
		}
	}

	return session
}

func mapToAccount(data map[string]interface{}) *core.Account {
	account := &core.Account{
		Metadata: make(map[string]interface{}),
	}

	knownFields := map[string]bool{
		"id":                       true,
		"user_id":                  true,
		"account_id":               true,
		"provider_id":              true,
		"provider_type":            true,
		"password":                 true,
		"access_token":             true,
		"refresh_token":            true,
		"access_token_expires_at":  true,
		"refresh_token_expires_at": true,
		"scope":                    true,
		"id_token":                 true,
		"created_at":               true,
		"updated_at":               true,
	}

	if id, ok := data["id"]; ok {
		account.ID = toString(id)
	}
	if userID, ok := data["user_id"]; ok {
		account.UserID = toString(userID)
	}
	if accountID, ok := data["account_id"].(string); ok {
		account.AccountID = accountID
	}
	if providerID, ok := data["provider_id"].(string); ok {
		account.ProviderID = providerID
	}
	if providerType, ok := data["provider_type"].(string); ok {
		account.ProviderType = providerType
	}
	if password, ok := data["password"].(string); ok {
		account.Password = password
	}
	if accessToken, ok := data["access_token"].(string); ok {
		account.AccessToken = accessToken
	}
	if refreshToken, ok := data["refresh_token"].(string); ok {
		account.RefreshToken = refreshToken
	}
	if expiresAt, ok := data["access_token_expires_at"].(time.Time); ok {
		account.AccessTokenExpiresAt = &expiresAt
	} else if expiresAt, ok := data["access_token_expires_at"].(*time.Time); ok {
		account.AccessTokenExpiresAt = expiresAt
	}
	if refreshExpiresAt, ok := data["refresh_token_expires_at"].(time.Time); ok {
		account.RefreshTokenExpiresAt = &refreshExpiresAt
	} else if refreshExpiresAt, ok := data["refresh_token_expires_at"].(*time.Time); ok {
		account.RefreshTokenExpiresAt = refreshExpiresAt
	}
	if scope, ok := data["scope"].(string); ok {
		account.Scope = scope
	}
	if idToken, ok := data["id_token"].(string); ok {
		account.IDToken = idToken
	}

	if createdAt, ok := data["created_at"].(time.Time); ok {
		account.CreatedAt = createdAt
	}
	if updatedAt, ok := data["updated_at"].(time.Time); ok {
		account.UpdatedAt = updatedAt
	}

	// Populate Metadata with any fields not explicitly mapped
	for k, v := range data {
		if !knownFields[k] {
			account.Metadata[k] = v
		}
	}

	return account
}

func mapToVerification(data map[string]interface{}) *core.Verification {
	verification := &core.Verification{}
	if id, ok := data["id"]; ok {
		verification.ID = toString(id)
	}
	if identifier, ok := data["identifier"].(string); ok {
		verification.Identifier = identifier
	}
	if value, ok := data["value"].(string); ok {
		verification.Value = value
	}
	if expiresAt, ok := data["expires_at"].(time.Time); ok {
		verification.ExpiresAt = expiresAt
	}
	if createdAt, ok := data["created_at"].(time.Time); ok {
		verification.CreatedAt = createdAt
	}
	if updatedAt, ok := data["updated_at"].(time.Time); ok {
		verification.UpdatedAt = updatedAt
	}
	return verification
}

func generateRandomStringID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp-based ID if random read fails
		return base64.URLEncoding.EncodeToString([]byte(time.Now().String()))[:22]
	}
	return base64.URLEncoding.EncodeToString(b)[:22]
}

func generateSessionToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func generateVerificationToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// toString safely converts an interface{} (usually from DB) to a string
func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case []byte:
		return string(val)
	case int:
		return strconv.Itoa(val)
	case int64:
		return strconv.FormatInt(val, 10)
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	default:
		return ""
	}
}
