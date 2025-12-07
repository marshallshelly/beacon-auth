// Package beaconauth provides a modular, plugin-based authentication library for Go
package beaconauth

import (
	"github.com/marshallshelly/beacon-auth/core"
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

// New creates a new BeaconAuth instance
var New = core.New

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
