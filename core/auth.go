package core

import (
	"context"
	"net/http"
)

// Auth is the main authentication interface
type Auth interface {
	// Handler returns an http.Handler for the auth routes
	Handler() http.Handler

	// Context returns the auth context
	Context() *AuthContext

	// Middleware returns middleware for protecting routes
	Middleware() func(http.Handler) http.Handler

	// GetSession retrieves the session from a request
	GetSession(ctx context.Context) (*Session, error)

	// CreateSession creates a new session for a user
	CreateSession(ctx context.Context, userID string, options *SessionOptions) (*Session, error)

	// RevokeSession revokes a session
	RevokeSession(ctx context.Context, token string) error

	// Close cleans up resources
	Close() error
}

// beaconAuth implements the Auth interface
type beaconAuth struct {
	config        *Config
	ctx           *AuthContext
	pluginManager *PluginManager
	router        http.Handler
}

// PluginManager manages plugins
type PluginManager struct {
	plugins []Plugin
}

// New creates a new BeaconAuth instance
func New(opts ...Option) (Auth, error) {
	cfg := defaultConfig()
	for _, opt := range opts {
		if err := opt(cfg); err != nil {
			return nil, err
		}
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	a := &beaconAuth{
		config: cfg,
	}

	// Set default logger if not provided
	if cfg.Advanced.Logger == nil {
		cfg.Advanced.Logger = NewDefaultLogger()
	}

	// Initialize context
	a.ctx = NewAuthContext(cfg)

	// Initialize plugin manager
	pm := &PluginManager{
		plugins: cfg.Plugins,
	}
	a.pluginManager = pm

	// Initialize plugins
	for _, plugin := range pm.plugins {
		if err := plugin.Init(a.ctx); err != nil {
			return nil, err
		}
	}

	// Build router (placeholder for now)
	a.router = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("BeaconAuth"))
	})

	cfg.Advanced.Logger.Info("BeaconAuth initialized successfully")

	return a, nil
}

func (a *beaconAuth) Handler() http.Handler {
	return a.router
}

func (a *beaconAuth) Context() *AuthContext {
	return a.ctx
}

func (a *beaconAuth) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			session, _ := a.GetSession(r.Context())
			if session == nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			ctx := WithSession(r.Context(), session)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func (a *beaconAuth) GetSession(ctx context.Context) (*Session, error) {
	// Placeholder implementation
	return nil, ErrSessionNotFound
}

func (a *beaconAuth) CreateSession(ctx context.Context, userID string, opts *SessionOptions) (*Session, error) {
	// Placeholder implementation
	return nil, nil
}

func (a *beaconAuth) RevokeSession(ctx context.Context, token string) error {
	// Placeholder implementation
	return nil
}

func (a *beaconAuth) Close() error {
	if a.ctx.Adapter != nil {
		return a.ctx.Adapter.Close()
	}
	return nil
}
