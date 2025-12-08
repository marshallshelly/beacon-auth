package providers

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type AppleProvider struct {
	clientID     string
	clientSecret string // For Apple, this is actually a JWT we generate
	teamID       string
	keyID        string
	privateKey   string
	scopes       []string
	httpClient   *http.Client
}

type AppleOptions struct {
	ClientID     string   // Service ID
	TeamID       string   // Apple Team ID
	KeyID        string   // Key ID from Apple
	PrivateKey   string   // Private key content (PEM format)
	Scopes       []string // Optional scopes
	ClientSecret string   // Pre-generated client secret (optional, will generate if not provided)
}

// AppleProfile represents the user data from Apple ID token
type AppleProfile struct {
	Sub            string `json:"sub"`
	Email          string `json:"email"`
	EmailVerified  bool   `json:"email_verified"`
	IsPrivateEmail bool   `json:"is_private_email"`
	RealUserStatus int    `json:"real_user_status"`
	Name           string `json:"name,omitempty"`
	GivenName      string `json:"given_name,omitempty"`
	FamilyName     string `json:"family_name,omitempty"`
}

func NewApple(opts *AppleOptions) *AppleProvider {
	if opts == nil {
		opts = &AppleOptions{}
	}

	scopes := opts.Scopes
	if len(scopes) == 0 {
		scopes = []string{"name", "email"}
	}

	// Use provided client secret or we'll generate it on-demand
	clientSecret := opts.ClientSecret

	return &AppleProvider{
		clientID:     opts.ClientID,
		clientSecret: clientSecret,
		teamID:       opts.TeamID,
		keyID:        opts.KeyID,
		privateKey:   opts.PrivateKey,
		scopes:       scopes,
		httpClient:   &http.Client{Timeout: 10 * time.Second},
	}
}

func (p *AppleProvider) ID() string {
	return "apple"
}

func (p *AppleProvider) Name() string {
	return "Apple"
}

func (p *AppleProvider) Init() error {
	if p.clientID == "" {
		return fmt.Errorf("apple client ID (Service ID) is required")
	}
	// Client secret can be generated on-demand if not provided
	if p.clientSecret == "" {
		if p.teamID == "" || p.keyID == "" || p.privateKey == "" {
			return fmt.Errorf("apple Team ID, Key ID, and Private Key are required to generate client secret")
		}
	}
	return nil
}

func (p *AppleProvider) CreateAuthorizationURL(state, redirectURI string, options *AuthOptions) (*url.URL, error) {
	authURL, _ := url.Parse("https://appleid.apple.com/auth/authorize")

	q := authURL.Query()
	q.Set("client_id", p.clientID)
	q.Set("redirect_uri", redirectURI)
	q.Set("response_type", "code")
	q.Set("response_mode", "form_post")
	q.Set("state", state)

	scopes := p.scopes
	if options != nil && len(options.Scopes) > 0 {
		scopes = options.Scopes
	}
	q.Set("scope", strings.Join(scopes, " "))

	if options != nil && len(options.ExtraParams) > 0 {
		for k, v := range options.ExtraParams {
			q.Set(k, v)
		}
	}

	authURL.RawQuery = q.Encode()
	return authURL, nil
}

func (p *AppleProvider) ExchangeCode(ctx context.Context, code, codeVerifier, redirectURI string) (*OAuthTokens, error) {
	// Generate client secret if not provided
	clientSecret := p.clientSecret
	if clientSecret == "" {
		var err error
		clientSecret, err = p.generateClientSecret()
		if err != nil {
			return nil, fmt.Errorf("failed to generate client secret: %w", err)
		}
	}

	data := url.Values{}
	data.Set("client_id", p.clientID)
	data.Set("client_secret", clientSecret)
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)
	data.Set("grant_type", "authorization_code")

	req, err := http.NewRequestWithContext(ctx, "POST", "https://appleid.apple.com/auth/token", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to exchange token (status %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
		IDToken      string `json:"id_token"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.AccessToken == "" && result.IDToken == "" {
		return nil, fmt.Errorf("neither access token nor ID token received")
	}

	var expiresAt *time.Time
	if result.ExpiresIn > 0 {
		t := time.Now().Add(time.Duration(result.ExpiresIn) * time.Second)
		expiresAt = &t
	}

	return &OAuthTokens{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		TokenType:    result.TokenType,
		ExpiresAt:    expiresAt,
		IDToken:      result.IDToken,
	}, nil
}

func (p *AppleProvider) GetUserInfo(ctx context.Context, accessToken string) (*OAuthUserInfo, error) {
	// Apple doesn't have a traditional userinfo endpoint
	// User data comes from the ID token
	return nil, fmt.Errorf("apple provider requires ID token for user info")
}

// GetUserInfoFromIDToken extracts user info from Apple's ID token
func (p *AppleProvider) GetUserInfoFromIDToken(idToken string) (*OAuthUserInfo, error) {
	// Parse the JWT without verification for now (verification should be done separately)
	// In production, you should verify the signature against Apple's public keys
	token, _, err := new(jwt.Parser).ParseUnverified(idToken, jwt.MapClaims{})
	if err != nil {
		return nil, fmt.Errorf("failed to parse ID token: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	profile := &AppleProfile{}

	if sub, ok := claims["sub"].(string); ok {
		profile.Sub = sub
	}
	if email, ok := claims["email"].(string); ok {
		profile.Email = email
	}

	// email_verified can be string or bool
	if emailVerified, ok := claims["email_verified"].(bool); ok {
		profile.EmailVerified = emailVerified
	} else if emailVerifiedStr, ok := claims["email_verified"].(string); ok {
		profile.EmailVerified = emailVerifiedStr == "true"
	}

	if isPrivate, ok := claims["is_private_email"].(bool); ok {
		profile.IsPrivateEmail = isPrivate
	}

	// Build name from available fields
	name := ""
	if givenName, ok := claims["given_name"].(string); ok {
		profile.GivenName = givenName
		name = givenName
	}
	if familyName, ok := claims["family_name"].(string); ok {
		profile.FamilyName = familyName
		if name != "" {
			name += " " + familyName
		} else {
			name = familyName
		}
	}

	// Fallback to email if no name
	if name == "" {
		name = profile.Email
	}
	profile.Name = name

	// Convert to raw map
	rawData := make(map[string]interface{})
	for k, v := range claims {
		rawData[k] = v
	}

	return &OAuthUserInfo{
		ID:            profile.Sub,
		Email:         profile.Email,
		EmailVerified: profile.EmailVerified,
		Name:          profile.Name,
		FirstName:     profile.GivenName,
		LastName:      profile.FamilyName,
		RawData:       rawData,
	}, nil
}

func (p *AppleProvider) RefreshToken(ctx context.Context, refreshToken string) (*OAuthTokens, error) {
	clientSecret := p.clientSecret
	if clientSecret == "" {
		var err error
		clientSecret, err = p.generateClientSecret()
		if err != nil {
			return nil, fmt.Errorf("failed to generate client secret: %w", err)
		}
	}

	data := url.Values{}
	data.Set("client_id", p.clientID)
	data.Set("client_secret", clientSecret)
	data.Set("refresh_token", refreshToken)
	data.Set("grant_type", "refresh_token")

	req, err := http.NewRequestWithContext(ctx, "POST", "https://appleid.apple.com/auth/token", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to refresh token (status %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
		IDToken      string `json:"id_token"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var expiresAt *time.Time
	if result.ExpiresIn > 0 {
		t := time.Now().Add(time.Duration(result.ExpiresIn) * time.Second)
		expiresAt = &t
	}

	return &OAuthTokens{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		TokenType:    result.TokenType,
		ExpiresAt:    expiresAt,
		IDToken:      result.IDToken,
	}, nil
}

// generateClientSecret creates a JWT client secret for Apple Sign In
func (p *AppleProvider) generateClientSecret() (string, error) {
	// Parse the private key
	block := []byte(p.privateKey)
	if !strings.Contains(p.privateKey, "BEGIN") {
		// Assume it's base64 encoded
		decoded, err := base64.StdEncoding.DecodeString(p.privateKey)
		if err != nil {
			return "", fmt.Errorf("failed to decode private key: %w", err)
		}
		block = decoded
	}

	key, err := jwt.ParseECPrivateKeyFromPEM(block)
	if err != nil {
		return "", fmt.Errorf("failed to parse private key: %w", err)
	}

	// Create JWT claims
	now := time.Now()
	claims := jwt.MapClaims{
		"iss": p.teamID,
		"iat": now.Unix(),
		"exp": now.Add(6 * 30 * 24 * time.Hour).Unix(), // 6 months
		"aud": "https://appleid.apple.com",
		"sub": p.clientID,
	}

	// Create token
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	token.Header["kid"] = p.keyID

	// Sign the token
	signedToken, err := token.SignedString(key)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return signedToken, nil
}

// VerifyApplePublicKey fetches and verifies Apple's public key
func VerifyApplePublicKey(kid string) (*rsa.PublicKey, error) {
	resp, err := http.Get("https://appleid.apple.com/auth/keys")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Apple public keys: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var jwks struct {
		Keys []struct {
			Kid string `json:"kid"`
			Alg string `json:"alg"`
			Kty string `json:"kty"`
			Use string `json:"use"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, fmt.Errorf("failed to decode JWKS: %w", err)
	}

	// Find the key with matching kid
	for _, key := range jwks.Keys {
		if key.Kid == kid {
			// Decode N and E
			nBytes, err := base64.RawURLEncoding.DecodeString(key.N)
			if err != nil {
				return nil, fmt.Errorf("failed to decode N: %w", err)
			}
			eBytes, err := base64.RawURLEncoding.DecodeString(key.E)
			if err != nil {
				return nil, fmt.Errorf("failed to decode E: %w", err)
			}

			// Convert to big.Int
			n := new(big.Int).SetBytes(nBytes)
			e := new(big.Int).SetBytes(eBytes)

			return &rsa.PublicKey{
				N: n,
				E: int(e.Int64()),
			}, nil
		}
	}

	return nil, fmt.Errorf("key with kid %s not found", kid)
}
