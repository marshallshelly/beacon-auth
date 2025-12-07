package core

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
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

	// Initialize managers
	if cfg.DataManagerFactory == nil {
		return nil, errors.New("DataManagerFactory is required (use generic New or configure manually)")
	}
	a.ctx.DataManager = cfg.DataManagerFactory(cfg.Adapter)

	if cfg.PasswordHasherFactory == nil {
		return nil, errors.New("PasswordHasherFactory is required")
	}
	a.ctx.PasswordHasher = cfg.PasswordHasherFactory()

	if cfg.SessionManagerFactory == nil {
		return nil, errors.New("SessionManagerFactory is required")
	}
	sm, err := cfg.SessionManagerFactory(cfg, cfg.Adapter)
	if err != nil {
		return nil, fmt.Errorf("failed to create session manager: %w", err)
	}
	a.ctx.SessionManager = sm

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
	// Build router
	mux := http.NewServeMux()

	// Normalize base path
	basePath := cfg.BasePath
	if !strings.HasPrefix(basePath, "/") {
		basePath = "/" + basePath
	}
	basePath = strings.TrimRight(basePath, "/")

	// Default route
	rootPath := basePath
	if rootPath == "" {
		rootPath = "/"
	}
	mux.HandleFunc(rootPath, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("BeaconAuth"))
	})

	// Register plugin routes
	for _, p := range pm.plugins {
		for path, endpoint := range p.Endpoints() {
			if !strings.HasPrefix(path, "/") {
				path = "/" + path
			}
			fullPath := basePath + path

			// Capture closure variable
			handler := endpoint.Handler
			method := endpoint.Method

			mux.HandleFunc(fullPath, func(w http.ResponseWriter, r *http.Request) {
				if method != "" && r.Method != method {
					http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
					return
				}
				handler(w, r)
			})
		}
	}

	a.router = mux

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
	if a.ctx.SessionManager == nil {
		return nil, errors.New("session manager not initialized")
	}

	// Try to get from request cookie
	req := GetRequest(ctx)
	if req != nil {
		cookieName := a.ctx.Config.Session.CookieName
		if cookie, err := req.Cookie(cookieName); err == nil {
			session, _, err := a.ctx.SessionManager.Get(ctx, cookie.Value)
			return session, err
		}
	}

	// Try to get from session context (already populated by middleware?)
	// Or maybe GetSession(ctx) assumes context already has it?
	// The interface comment says "GetSession retrieves the session from a request".
	// The Middleware populates it.
	// If middleware ran, GetSession(ctx) should return ctx value?
	// But GetSession on struct usually implies logic to extract.

	// Let's stick to extraction logic.
	return nil, ErrSessionNotFound
}

func (a *beaconAuth) CreateSession(ctx context.Context, userID string, opts *SessionOptions) (*Session, error) {
	if a.ctx.SessionManager == nil {
		return nil, errors.New("session manager not initialized")
	}
	session, _, _, err := a.ctx.SessionManager.Create(ctx, userID, opts)
	return session, err
}

func (a *beaconAuth) RevokeSession(ctx context.Context, token string) error {
	if a.ctx.SessionManager == nil {
		return errors.New("session manager not initialized")
	}
	return a.ctx.SessionManager.Delete(ctx, token)
}

func (a *beaconAuth) Close() error {
	if a.ctx.Adapter != nil {
		return a.ctx.Adapter.Close()
	}
	return nil
}
