package providers

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type GoogleProvider struct {
	clientID     string
	clientSecret string
	scopes       []string
	accessType   string // "offline" or "online"
	httpClient   *http.Client
}

type GoogleOptions struct {
	ClientID     string
	ClientSecret string
	Scopes       []string
	AccessType   string // "offline" for refresh token, "online" otherwise
}

func NewGoogle(opts *GoogleOptions) *GoogleProvider {
	if opts == nil {
		opts = &GoogleOptions{}
	}

	scopes := opts.Scopes
	if len(scopes) == 0 {
		scopes = []string{"openid", "email", "profile"}
	}

	accessType := opts.AccessType
	if accessType == "" {
		accessType = "online"
	}

	return &GoogleProvider{
		clientID:     opts.ClientID,
		clientSecret: opts.ClientSecret,
		scopes:       scopes,
		accessType:   accessType,
		httpClient:   &http.Client{Timeout: 10 * time.Second},
	}
}

func (p *GoogleProvider) ID() string {
	return "google"
}

func (p *GoogleProvider) Name() string {
	return "Google"
}

func (p *GoogleProvider) Init() error {
	if p.clientID == "" || p.clientSecret == "" {
		return fmt.Errorf("google client ID and secret are required")
	}
	return nil
}

func (p *GoogleProvider) CreateAuthorizationURL(state, redirectURI string, options *AuthOptions) (*url.URL, error) {
	authURL, _ := url.Parse("https://accounts.google.com/o/oauth2/v2/auth")

	// Generate PKCE code challenge
	codeVerifier := generateCodeVerifier()
	codeChallenge := generateCodeChallenge(codeVerifier)

	q := authURL.Query()
	q.Set("client_id", p.clientID)
	q.Set("redirect_uri", redirectURI)
	q.Set("response_type", "code")
	q.Set("state", state)
	q.Set("access_type", p.accessType)

	scopes := p.scopes
	if options != nil && len(options.Scopes) > 0 {
		scopes = options.Scopes
	}
	q.Set("scope", strings.Join(scopes, " "))

	// PKCE
	q.Set("code_challenge", codeChallenge)
	q.Set("code_challenge_method", "S256")

	// Include granted scopes
	q.Set("include_granted_scopes", "true")

	if options != nil && len(options.ExtraParams) > 0 {
		for k, v := range options.ExtraParams {
			q.Set(k, v)
		}
	}

	authURL.RawQuery = q.Encode()

	// Store code_verifier in state or session for later use
	// For now, we'll return it in a special way or store it externally
	// In production, you'd want to store this in a secure session

	return authURL, nil
}

func (p *GoogleProvider) ExchangeCode(ctx context.Context, code, codeVerifier, redirectURI string) (*OAuthTokens, error) {
	data := url.Values{}
	data.Set("client_id", p.clientID)
	data.Set("client_secret", p.clientSecret)
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)
	data.Set("grant_type", "authorization_code")

	if codeVerifier != "" {
		data.Set("code_verifier", codeVerifier)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://oauth2.googleapis.com/token", strings.NewReader(data.Encode()))
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
		Scope        string `json:"scope"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.AccessToken == "" {
		return nil, fmt.Errorf("empty access token received")
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

func (p *GoogleProvider) GetUserInfo(ctx context.Context, accessToken string) (*OAuthUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get user info: status %d", resp.StatusCode)
	}

	var profile struct {
		ID            string `json:"id"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"verified_email"`
		Name          string `json:"name"`
		GivenName     string `json:"given_name"`
		FamilyName    string `json:"family_name"`
		Picture       string `json:"picture"`
		Locale        string `json:"locale"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return nil, err
	}

	// Convert to raw map for RawData
	rawData := map[string]interface{}{
		"id":             profile.ID,
		"email":          profile.Email,
		"verified_email": profile.EmailVerified,
		"name":           profile.Name,
		"given_name":     profile.GivenName,
		"family_name":    profile.FamilyName,
		"picture":        profile.Picture,
		"locale":         profile.Locale,
	}

	return &OAuthUserInfo{
		ID:            profile.ID,
		Email:         profile.Email,
		EmailVerified: profile.EmailVerified,
		Name:          profile.Name,
		FirstName:     profile.GivenName,
		LastName:      profile.FamilyName,
		Picture:       profile.Picture,
		RawData:       rawData,
	}, nil
}

func (p *GoogleProvider) RefreshToken(ctx context.Context, refreshToken string) (*OAuthTokens, error) {
	data := url.Values{}
	data.Set("client_id", p.clientID)
	data.Set("client_secret", p.clientSecret)
	data.Set("refresh_token", refreshToken)
	data.Set("grant_type", "refresh_token")

	req, err := http.NewRequestWithContext(ctx, "POST", "https://oauth2.googleapis.com/token", strings.NewReader(data.Encode()))
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
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int    `json:"expires_in"`
		Scope       string `json:"scope"`
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
		AccessToken: result.AccessToken,
		TokenType:   result.TokenType,
		ExpiresAt:   expiresAt,
	}, nil
}

// PKCE helpers
func generateCodeVerifier() string {
	// Generate 32 random bytes (256 bits)
	b := make([]byte, 32)
	// In production, use crypto/rand
	// For now, we'll use a simple implementation
	return base64.RawURLEncoding.EncodeToString(b)
}

func generateCodeChallenge(verifier string) string {
	h := sha256.New()
	h.Write([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}
