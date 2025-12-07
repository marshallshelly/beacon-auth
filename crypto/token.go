package crypto

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// TokenGenerator generates secure random tokens
type TokenGenerator interface {
	Generate(length int) (string, error)
	GenerateHex(length int) (string, error)
}

// DefaultTokenGenerator implements TokenGenerator
type DefaultTokenGenerator struct{}

// NewTokenGenerator creates a new token generator
func NewTokenGenerator() *DefaultTokenGenerator {
	return &DefaultTokenGenerator{}
}

// Generate generates a URL-safe base64 encoded token
func (g *DefaultTokenGenerator) Generate(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(b), nil
}

// GenerateHex generates a hex-encoded token
func (g *DefaultTokenGenerator) GenerateHex(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	return fmt.Sprintf("%x", b), nil
}

// GenerateID generates a unique ID
func GenerateID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate ID: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(b), nil
}

// GenerateSessionToken generates a session token
func GenerateSessionToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate session token: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(b), nil
}

// GenerateVerificationToken generates a verification token
func GenerateVerificationToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate verification token: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(b), nil
}
