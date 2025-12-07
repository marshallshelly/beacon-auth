package core

import (
	"time"
)

// User represents an authenticated user
type User struct {
	ID            string                 `json:"id"`
	Email         string                 `json:"email"`
	EmailVerified bool                   `json:"emailVerified"`
	Name          string                 `json:"name,omitempty"`
	Image         string                 `json:"image,omitempty"`
	CreatedAt     time.Time              `json:"createdAt"`
	UpdatedAt     time.Time              `json:"updatedAt"`
	Fields        map[string]interface{} `json:"-"` // Custom fields from plugins
}

// Session represents a user session
type Session struct {
	ID        string    `json:"id"`
	UserID    string    `json:"userId"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
	IPAddress string    `json:"ipAddress,omitempty"`
	UserAgent string    `json:"userAgent,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Account represents an authentication account (email, OAuth, etc.)
type Account struct {
	ID           string                 `json:"id"`
	UserID       string                 `json:"userId"`
	AccountID    string                 `json:"accountId"`
	Provider     string                 `json:"provider"`
	ProviderType string                 `json:"providerType"` // "oauth", "email", "credential"
	Password     string                 `json:"-"`
	AccessToken  string                 `json:"-"`
	RefreshToken string                 `json:"-"`
	ExpiresAt    *time.Time             `json:"expiresAt,omitempty"`
	CreatedAt    time.Time              `json:"createdAt"`
	UpdatedAt    time.Time              `json:"updatedAt"`
	Fields       map[string]interface{} `json:"-"` // Provider-specific fields
}

// Verification represents an email/phone verification token
type Verification struct {
	ID         string    `json:"id"`
	Identifier string    `json:"identifier"` // email or phone
	Token      string    `json:"token"`
	Type       string    `json:"type"` // "email", "phone", "reset_password"
	ExpiresAt  time.Time `json:"expiresAt"`
	CreatedAt  time.Time `json:"createdAt"`
}

// SessionOptions holds options for session creation
type SessionOptions struct {
	IPAddress  string
	UserAgent  string
	RememberMe bool
	ExpiresIn  *time.Duration
}
