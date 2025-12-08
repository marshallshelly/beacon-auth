package core

import (
	"time"
)

// User represents an authenticated user
type User struct {
	ID               string                 `json:"id"`
	Email            string                 `json:"email"`
	EmailVerified    bool                   `json:"emailVerified"`
	Name             string                 `json:"name,omitempty"`
	Image            string                 `json:"image,omitempty"`
	TwoFactorEnabled bool                   `json:"twoFactorEnabled"`
	CreatedAt        time.Time              `json:"createdAt"`
	UpdatedAt        time.Time              `json:"updatedAt"`
	Role             string                 `json:"role,omitempty"`
	Banned           bool                   `json:"banned"`
	BanReason        string                 `json:"banReason,omitempty"`
	BanExpires       *time.Time             `json:"banExpires,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"` // Custom fields from plugins
}

// HasRole checks if the user has the specified role
func (u *User) HasRole(role string) bool {
	return u.Role == role
}

// Session represents a user session
type Session struct {
	ID             string                 `json:"id"`
	UserID         string                 `json:"userId"`
	Token          string                 `json:"token"`
	ExpiresAt      time.Time              `json:"expiresAt"`
	IPAddress      string                 `json:"ipAddress,omitempty"`
	UserAgent      string                 `json:"userAgent,omitempty"`
	CreatedAt      time.Time              `json:"createdAt"`
	UpdatedAt      time.Time              `json:"updatedAt"`
	ImpersonatedBy string                 `json:"impersonatedBy,omitempty"` // ID of the admin impersonating this session
	Metadata       map[string]interface{} `json:"metadata,omitempty"`       // Custom fields from plugins
}

// Account represents an authentication account (email, OAuth, etc.)
type Account struct {
	ID                    string                 `json:"id"`
	UserID                string                 `json:"userId"`
	AccountID             string                 `json:"accountId"`
	ProviderID            string                 `json:"providerId"`
	ProviderType          string                 `json:"providerType,omitempty"` // "oauth", "email", "credential"
	Password              string                 `json:"-"`
	AccessToken           string                 `json:"-"`
	RefreshToken          string                 `json:"-"`
	AccessTokenExpiresAt  *time.Time             `json:"accessTokenExpiresAt,omitempty"`
	RefreshTokenExpiresAt *time.Time             `json:"refreshTokenExpiresAt,omitempty"`
	Scope                 string                 `json:"scope,omitempty"`
	IDToken               string                 `json:"idToken,omitempty"`
	CreatedAt             time.Time              `json:"createdAt"`
	UpdatedAt             time.Time              `json:"updatedAt"`
	Metadata              map[string]interface{} `json:"metadata,omitempty"` // Provider-specific fields
}

// Verification represents an email/phone verification token
type Verification struct {
	ID         string    `json:"id"`
	Identifier string    `json:"identifier"` // email or phone
	Value      string    `json:"value"`      // The value to be verified (token/otp)
	ExpiresAt  time.Time `json:"expiresAt"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

// SessionOptions holds options for session creation
type SessionOptions struct {
	IPAddress  string
	UserAgent  string
	RememberMe bool
	ExpiresIn  *time.Duration
}
