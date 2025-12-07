package providers

import (
	"context"
	"net/url"
	"time"
)

// OAuthProvider defines the interface for OAuth providers
type OAuthProvider interface {
	ID() string
	Name() string

	// CreateAuthorizationURL creates the authorization URL
	// options allows overriding defaults
	CreateAuthorizationURL(state, redirectURI string, options *AuthOptions) (*url.URL, error)

	// ExchangeCode exchanges authorization code for tokens
	ExchangeCode(ctx context.Context, code, codeVerifier, redirectURI string) (*OAuthTokens, error)

	// GetUserInfo retrieves user information
	GetUserInfo(ctx context.Context, accessToken string) (*OAuthUserInfo, error)

	// RefreshToken refreshes the access token
	RefreshToken(ctx context.Context, refreshToken string) (*OAuthTokens, error)

	// Init initializes the provider
	Init() error
}

// AuthOptions holds options for authorization URL
type AuthOptions struct {
	Scopes      []string
	UsePKCE     bool
	Nonce       string
	ExtraParams map[string]string
}

// OAuthTokens holds OAuth tokens
type OAuthTokens struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    *time.Time
	IDToken      string
	TokenType    string
}

// OAuthUserInfo holds user information from provider
type OAuthUserInfo struct {
	ID            string
	Email         string
	EmailVerified bool
	Name          string
	FirstName     string
	LastName      string
	Picture       string
	RawData       map[string]interface{}
}
