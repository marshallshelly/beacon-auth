package core

import (
	"errors"
	"time"
)

// Config holds the authentication configuration
type Config struct {
	// Core settings
	AppName  string
	BaseURL  string
	BasePath string
	Secret   string

	// Database
	Adapter Adapter

	// Email & Password
	EmailPassword *EmailPasswordConfig

	// OAuth providers
	OAuth *OAuthConfig

	// Session configuration
	Session *SessionConfig

	// Plugins
	Plugins []Plugin

	// Mailer
	Mailer Mailer

	// Rate limiting
	RateLimit *RateLimitConfig

	// Hooks
	Hooks *HooksConfig

	// Advanced settings
	Advanced *AdvancedConfig

	// Factories (dependency injection)
	SessionManagerFactory func(cfg *Config, adapter Adapter) (SessionManager, error)
	DataManagerFactory    func(adapter Adapter) DataManager
	PasswordHasherFactory func() PasswordHasher
}

// EmailPasswordConfig holds email/password authentication settings
type EmailPasswordConfig struct {
	Enabled             bool
	MinPasswordLength   int
	RequireVerification bool
	PasswordHashCost    int
	ResetPasswordExpiry time.Duration
}

// OAuthConfig holds OAuth configuration
type OAuthConfig struct {
	Providers []OAuthProvider
}

// SessionConfig holds session configuration
type SessionConfig struct {
	ExpiresIn        time.Duration
	UpdateAge        time.Duration
	CookieName       string
	CookieSecure     bool
	CookieHTTPOnly   bool
	CookieSameSite   string
	CookieDomain     string
	CookiePath       string
	SecondaryStorage SecondaryStorage // Redis, etc.
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	Enabled bool
	Storage RateLimitStorage
	Rules   []RateLimitRule
}

// RateLimitRule defines a rate limit rule
type RateLimitRule struct {
	Path    string
	Methods []string
	Limit   int
	Window  time.Duration
}

// HooksConfig holds hook configuration
type HooksConfig struct {
	BeforeRequest []Hook
	AfterRequest  []Hook
	OnError       []Hook
}

// AdvancedConfig holds advanced configuration options
type AdvancedConfig struct {
	UseSecureCookies bool
	DisableCSRFCheck bool
	TrustedOrigins   []string
	GenerateID       func() string
	Logger           Logger
}

// Option is a functional option for configuring BeaconAuth
type Option func(*Config) error

// defaultConfig returns the default configuration
func defaultConfig() *Config {
	return &Config{
		AppName:  "BeaconAuth App",
		BasePath: "/auth",
		EmailPassword: &EmailPasswordConfig{
			Enabled:             true,
			MinPasswordLength:   8,
			RequireVerification: false,
			PasswordHashCost:    10,
			ResetPasswordExpiry: 1 * time.Hour,
		},
		Session: &SessionConfig{
			ExpiresIn:      7 * 24 * time.Hour,
			UpdateAge:      1 * time.Hour,
			CookieName:     "beaconauth_session",
			CookieSecure:   true,
			CookieHTTPOnly: true,
			CookieSameSite: "lax",
			CookiePath:     "/",
		},
		Advanced: &AdvancedConfig{
			UseSecureCookies: true,
			GenerateID:       defaultIDGenerator,
		},
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Secret == "" {
		return errors.New("secret is required")
	}
	if c.Adapter == nil {
		return errors.New("adapter is required")
	}
	if c.BaseURL == "" {
		return errors.New("base URL is required")
	}
	return nil
}

// Configuration options

// WithSecret sets the secret key
func WithSecret(secret string) Option {
	return func(c *Config) error {
		c.Secret = secret
		return nil
	}
}

// WithBaseURL sets the base URL
func WithBaseURL(url string) Option {
	return func(c *Config) error {
		c.BaseURL = url
		return nil
	}
}

// WithBasePath sets the base path for auth routes
func WithBasePath(path string) Option {
	return func(c *Config) error {
		c.BasePath = path
		return nil
	}
}

// WithAdapter sets the database adapter
func WithAdapter(adapter Adapter) Option {
	return func(c *Config) error {
		c.Adapter = adapter
		return nil
	}
}

// WithPlugins adds plugins
func WithPlugins(plugins ...Plugin) Option {
	return func(c *Config) error {
		c.Plugins = append(c.Plugins, plugins...)
		return nil
	}
}

// WithMailer sets the mailer
func WithMailer(mailer Mailer) Option {
	return func(c *Config) error {
		c.Mailer = mailer
		return nil
	}
}

// WithOAuthProviders adds OAuth providers
func WithOAuthProviders(providers ...OAuthProvider) Option {
	return func(c *Config) error {
		if c.OAuth == nil {
			c.OAuth = &OAuthConfig{}
		}
		c.OAuth.Providers = append(c.OAuth.Providers, providers...)
		return nil
	}
}

// WithRateLimit sets rate limiting configuration
func WithRateLimit(storage RateLimitStorage, rules ...RateLimitRule) Option {
	return func(c *Config) error {
		c.RateLimit = &RateLimitConfig{
			Enabled: true,
			Storage: storage,
			Rules:   rules,
		}
		return nil
	}
}

// WithSessionConfig sets session configuration
func WithSessionConfig(session *SessionConfig) Option {
	return func(c *Config) error {
		c.Session = session
		return nil
	}
}

// WithEmailPassword configures email/password authentication
func WithEmailPassword(config *EmailPasswordConfig) Option {
	return func(c *Config) error {
		c.EmailPassword = config
		return nil
	}
}

// WithLogger sets a custom logger
func WithLogger(logger Logger) Option {
	return func(c *Config) error {
		if c.Advanced == nil {
			c.Advanced = &AdvancedConfig{}
		}
		c.Advanced.Logger = logger
		return nil
	}
}

// WithTrustedOrigins sets trusted origins for CORS
func WithTrustedOrigins(origins ...string) Option {
	return func(c *Config) error {
		if c.Advanced == nil {
			c.Advanced = &AdvancedConfig{}
		}
		c.Advanced.TrustedOrigins = origins
		return nil
	}
}

// defaultIDGenerator generates a default ID
func defaultIDGenerator() string {
	// Will be implemented with proper ID generation
	return ""
}
