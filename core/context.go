package core

import (
	"context"
	"net/http"
)

type contextKey int

const (
	authContextKey contextKey = iota
	sessionContextKey
	userContextKey
	requestContextKey
)

// AuthContext holds the authentication context
type AuthContext struct {
	Config           *Config
	Adapter          Adapter
	InternalAdapter  *InternalAdapter
	SessionManager   *SessionManager
	PasswordHasher   PasswordHasher
	TokenGenerator   TokenGenerator
	Logger           Logger
	Plugins          []Plugin
	PluginEndpoints  map[string]http.HandlerFunc
	PluginMiddleware []MiddlewareConfig
	PluginHooks      *HookRegistry
}

// InternalAdapter provides high-level database operations
type InternalAdapter struct {
	adapter Adapter
}

// SessionManager manages user sessions
type SessionManager struct {
	config  *SessionConfig
	adapter Adapter
}

// PasswordHasher defines the interface for password hashing
type PasswordHasher interface {
	Hash(password string) (string, error)
	Verify(password, hash string) (bool, error)
}

// TokenGenerator defines the interface for token generation
type TokenGenerator interface {
	Generate(length int) (string, error)
}

// MiddlewareConfig defines middleware to inject
type MiddlewareConfig struct {
	Path       string
	Middleware func(http.Handler) http.Handler
	Priority   int // Lower numbers run first
}

// HookRegistry manages lifecycle hooks
type HookRegistry struct {
	beforeHooks map[string][]Hook
	afterHooks  map[string][]Hook
}

// NewAuthContext creates a new auth context
func NewAuthContext(cfg *Config) *AuthContext {
	return &AuthContext{
		Config:          cfg,
		Adapter:         cfg.Adapter,
		Logger:          cfg.Advanced.Logger,
		PluginEndpoints: make(map[string]http.HandlerFunc),
	}
}

// WithAuthContext adds auth context to the request context
func WithAuthContext(ctx context.Context, authCtx *AuthContext) context.Context {
	return context.WithValue(ctx, authContextKey, authCtx)
}

// GetAuthContext retrieves auth context from request context
func GetAuthContext(ctx context.Context) *AuthContext {
	if authCtx, ok := ctx.Value(authContextKey).(*AuthContext); ok {
		return authCtx
	}
	return nil
}

// WithSession adds session to the request context
func WithSession(ctx context.Context, session *Session) context.Context {
	return context.WithValue(ctx, sessionContextKey, session)
}

// GetSession retrieves session from request context
func GetSession(ctx context.Context) *Session {
	if session, ok := ctx.Value(sessionContextKey).(*Session); ok {
		return session
	}
	return nil
}

// WithUser adds user to the request context
func WithUser(ctx context.Context, user *User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

// GetUser retrieves user from request context
func GetUser(ctx context.Context) *User {
	if user, ok := ctx.Value(userContextKey).(*User); ok {
		return user
	}
	return nil
}

// WithRequest adds HTTP request to context
func WithRequest(ctx context.Context, req *http.Request) context.Context {
	return context.WithValue(ctx, requestContextKey, req)
}

// GetRequest retrieves HTTP request from context
func GetRequest(ctx context.Context) *http.Request {
	if req, ok := ctx.Value(requestContextKey).(*http.Request); ok {
		return req
	}
	return nil
}
