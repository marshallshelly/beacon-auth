// Package beaconauth provides a modular, plugin-based authentication library for Go
package beaconauth

import (
	"github.com/marshallshelly/beacon-auth/adapter"
	"github.com/marshallshelly/beacon-auth/core"
	"github.com/marshallshelly/beacon-auth/crypto"
	"github.com/marshallshelly/beacon-auth/session"
)

// Auth is the main authentication interface
type Auth = core.Auth

// User represents an authenticated user
type User = core.User

// Session represents a user session
type Session = core.Session

// Account represents an authentication account
type Account = core.Account

// Config holds the authentication configuration
type Config = core.Config

// Option is a functional option for configuring BeaconAuth
type Option = core.Option

// Configuration options
var (
	WithSecret         = core.WithSecret
	WithBaseURL        = core.WithBaseURL
	WithBasePath       = core.WithBasePath
	WithAdapter        = core.WithAdapter
	WithPlugins        = core.WithPlugins
	WithMailer         = core.WithMailer
	WithOAuthProviders = core.WithOAuthProviders
	WithRateLimit      = core.WithRateLimit
	WithSessionConfig  = core.WithSessionConfig
	WithEmailPassword  = core.WithEmailPassword
	WithLogger         = core.WithLogger
	WithTrustedOrigins = core.WithTrustedOrigins
)

// Common errors
var (
	ErrInvalidCredentials = core.ErrInvalidCredentials
	ErrUserNotFound       = core.ErrUserNotFound
	ErrSessionNotFound    = core.ErrSessionNotFound
	ErrSessionExpired     = core.ErrSessionExpired
	ErrEmailTaken         = core.ErrEmailTaken
	ErrInvalidEmail       = core.ErrInvalidEmail
	ErrInvalidPassword    = core.ErrInvalidPassword
	ErrEmailNotVerified   = core.ErrEmailNotVerified
	ErrUnauthorized       = core.ErrUnauthorized
	ErrForbidden          = core.ErrForbidden
	ErrNotFound           = core.ErrNotFound
	ErrBadRequest         = core.ErrBadRequest
	ErrInternalServer     = core.ErrInternalServer
)

// New creates a new BeaconAuth instance
func New(opts ...Option) (Auth, error) {
	// Add default factory configuration
	factoryOpt := func(c *core.Config) error {
		c.DataManagerFactory = func(adapterInstance core.Adapter) core.DataManager {
			return adapter.NewInternalAdapter(adapterInstance)
		}

		c.PasswordHasherFactory = func() core.PasswordHasher {
			return crypto.NewArgon2Hasher()
		}

		c.SessionManagerFactory = func(cfg *core.Config, adapterInstance core.Adapter) (core.SessionManager, error) {
			// Map core config to session config
			sessCfg := &session.Config{
				Secret:            cfg.Secret,
				Issuer:            cfg.AppName,
				CookieName:        cfg.Session.CookieName,
				CookieDomain:      cfg.Session.CookieDomain,
				CookiePath:        cfg.Session.CookiePath,
				CookieSecure:      cfg.Session.CookieSecure,
				CookieHTTPOnly:    cfg.Session.CookieHTTPOnly,
				CookieSameSite:    cfg.Session.CookieSameSite,
				ExpiresIn:         cfg.Session.ExpiresIn,
				UpdateAge:         cfg.Session.UpdateAge,
				EnableCookieStore: true,
				EnableDBStore:     true,
				// Redis support requires advanced config parsing not implemented in this bridge yet
				EnableRedisStore: false,
			}

			return session.NewManager(sessCfg, adapterInstance)
		}
		return nil
	}

	// Prepend factory option so it runs (but user opts can override if they want?)
	// Actually append is better so it sets them. If user wants to override, they should pass override AFTER.
	// But usually defaults are set inside core.New via defaultConfig.
	// Options run sequentially.
	// So we add this option to the list.
	opts = append(opts, factoryOpt)

	return core.New(opts...)
}
