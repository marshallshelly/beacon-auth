package session

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/marshallshelly/beacon-auth/core"
)

// CookieStore implements Store using signed JWT-like tokens
// This is a stateless store that embeds session data in the cookie
type CookieStore struct {
	secret []byte
	issuer string
}

// NewCookieStore creates a new cookie-based session store
func NewCookieStore(secret, issuer string) *CookieStore {
	return &CookieStore{
		secret: []byte(secret),
		issuer: issuer,
	}
}

// cookiePayload represents the data stored in the cookie
type cookiePayload struct {
	Session  *core.Session `json:"session"`
	User     *core.User    `json:"user,omitempty"`
	Issuer   string        `json:"iss"`
	IssuedAt int64         `json:"iat"`
}

// Get retrieves a session from a signed cookie token
func (c *CookieStore) Get(ctx context.Context, token string) (*core.Session, *core.User, error) {
	// Token format: base64(payload).base64(signature)
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return nil, nil, fmt.Errorf("invalid token format")
	}

	payloadB64 := parts[0]
	signatureB64 := parts[1]

	// Verify signature
	expectedSig := c.sign(payloadB64)
	if signatureB64 != expectedSig {
		return nil, nil, fmt.Errorf("invalid token signature")
	}

	// Decode payload
	payloadBytes, err := base64.RawURLEncoding.DecodeString(payloadB64)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode payload: %w", err)
	}

	var payload cookiePayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	// Verify issuer
	if payload.Issuer != c.issuer {
		return nil, nil, fmt.Errorf("invalid issuer")
	}

	// Check expiration
	if time.Now().After(payload.Session.ExpiresAt) {
		return nil, nil, nil // Session expired
	}

	return payload.Session, payload.User, nil
}

// Set creates a signed cookie token
func (c *CookieStore) Set(ctx context.Context, session *core.Session) error {
	// Cookie store doesn't need to store anything server-side
	// The token is returned and stored client-side
	return nil
}

// CreateToken creates a signed token for a session
func (c *CookieStore) CreateToken(session *core.Session, user *core.User) (string, error) {
	payload := cookiePayload{
		Session:  session,
		User:     user,
		Issuer:   c.issuer,
		IssuedAt: time.Now().Unix(),
	}

	// Marshal payload
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Base64 encode payload
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadBytes)

	// Sign payload
	signature := c.sign(payloadB64)

	// Return token
	return payloadB64 + "." + signature, nil
}

// sign creates an HMAC signature for the payload
func (c *CookieStore) sign(data string) string {
	h := hmac.New(sha256.New, c.secret)
	h.Write([]byte(data))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

// Delete removes a session (no-op for cookie store)
func (c *CookieStore) Delete(ctx context.Context, token string) error {
	// Cookie store is stateless, deletion happens client-side
	return nil
}

// DeleteByUserID removes all sessions for a user (no-op for cookie store)
func (c *CookieStore) DeleteByUserID(ctx context.Context, userID string) error {
	// Cookie store is stateless, can't revoke tokens server-side
	// This is a limitation of stateless sessions
	return nil
}

// Cleanup removes expired sessions (no-op for cookie store)
func (c *CookieStore) Cleanup(ctx context.Context) error {
	// No server-side storage to clean up
	return nil
}

// Close closes the store (no-op for cookie store)
func (c *CookieStore) Close() error {
	return nil
}
