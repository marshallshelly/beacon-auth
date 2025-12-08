package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type DiscordProvider struct {
	clientID     string
	clientSecret string
	scopes       []string
	prompt       string // "none" or "consent"
	httpClient   *http.Client
}

type DiscordOptions struct {
	ClientID     string
	ClientSecret string
	Scopes       []string
	Prompt       string
}

func NewDiscord(opts *DiscordOptions) *DiscordProvider {
	if opts == nil {
		opts = &DiscordOptions{}
	}

	scopes := opts.Scopes
	if len(scopes) == 0 {
		scopes = []string{"identify", "email"}
	}

	prompt := opts.Prompt
	if prompt == "" {
		prompt = "none"
	}

	return &DiscordProvider{
		clientID:     opts.ClientID,
		clientSecret: opts.ClientSecret,
		scopes:       scopes,
		prompt:       prompt,
		httpClient:   &http.Client{Timeout: 10 * time.Second},
	}
}

func (p *DiscordProvider) ID() string {
	return "discord"
}

func (p *DiscordProvider) Name() string {
	return "Discord"
}

func (p *DiscordProvider) Init() error {
	if p.clientID == "" || p.clientSecret == "" {
		return fmt.Errorf("discord client ID and secret are required")
	}
	return nil
}

func (p *DiscordProvider) CreateAuthorizationURL(state, redirectURI string, options *AuthOptions) (*url.URL, error) {
	scopes := p.scopes
	if options != nil && len(options.Scopes) > 0 {
		scopes = options.Scopes
	}

	// Discord uses + as scope separator instead of space
	scopeStr := strings.Join(scopes, "+")

	authURL := fmt.Sprintf(
		"https://discord.com/api/oauth2/authorize?scope=%s&response_type=code&client_id=%s&redirect_uri=%s&state=%s&prompt=%s",
		scopeStr,
		url.QueryEscape(p.clientID),
		url.QueryEscape(redirectURI),
		url.QueryEscape(state),
		p.prompt,
	)

	if options != nil && len(options.ExtraParams) > 0 {
		for k, v := range options.ExtraParams {
			authURL += fmt.Sprintf("&%s=%s", url.QueryEscape(k), url.QueryEscape(v))
		}
	}

	return url.Parse(authURL)
}

func (p *DiscordProvider) ExchangeCode(ctx context.Context, code, codeVerifier, redirectURI string) (*OAuthTokens, error) {
	data := url.Values{}
	data.Set("client_id", p.clientID)
	data.Set("client_secret", p.clientSecret)
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)
	data.Set("grant_type", "authorization_code")

	req, err := http.NewRequestWithContext(ctx, "POST", "https://discord.com/api/oauth2/token", strings.NewReader(data.Encode()))
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
	}, nil
}

func (p *DiscordProvider) GetUserInfo(ctx context.Context, accessToken string) (*OAuthUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://discord.com/api/users/@me", nil)
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
		Username      string `json:"username"`
		Discriminator string `json:"discriminator"`
		GlobalName    string `json:"global_name"`
		Avatar        string `json:"avatar"`
		Email         string `json:"email"`
		Verified      bool   `json:"verified"`
		Locale        string `json:"locale"`
		MFAEnabled    bool   `json:"mfa_enabled"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return nil, err
	}

	// Generate avatar URL
	imageURL := p.getAvatarURL(&profile)

	// Use global_name if available, otherwise username
	name := profile.GlobalName
	if name == "" {
		name = profile.Username
	}

	// Convert to raw map
	rawData := map[string]interface{}{
		"id":            profile.ID,
		"username":      profile.Username,
		"discriminator": profile.Discriminator,
		"global_name":   profile.GlobalName,
		"avatar":        profile.Avatar,
		"email":         profile.Email,
		"verified":      profile.Verified,
		"locale":        profile.Locale,
		"mfa_enabled":   profile.MFAEnabled,
		"image_url":     imageURL,
	}

	return &OAuthUserInfo{
		ID:            profile.ID,
		Email:         profile.Email,
		EmailVerified: profile.Verified,
		Name:          name,
		Picture:       imageURL,
		RawData:       rawData,
	}, nil
}

func (p *DiscordProvider) RefreshToken(ctx context.Context, refreshToken string) (*OAuthTokens, error) {
	data := url.Values{}
	data.Set("client_id", p.clientID)
	data.Set("client_secret", p.clientSecret)
	data.Set("refresh_token", refreshToken)
	data.Set("grant_type", "refresh_token")

	req, err := http.NewRequestWithContext(ctx, "POST", "https://discord.com/api/oauth2/token", strings.NewReader(data.Encode()))
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
		Scope        string `json:"scope"`
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
	}, nil
}

// getAvatarURL generates the Discord avatar URL based on the profile
func (p *DiscordProvider) getAvatarURL(profile *struct {
	ID            string `json:"id"`
	Username      string `json:"username"`
	Discriminator string `json:"discriminator"`
	GlobalName    string `json:"global_name"`
	Avatar        string `json:"avatar"`
	Email         string `json:"email"`
	Verified      bool   `json:"verified"`
	Locale        string `json:"locale"`
	MFAEnabled    bool   `json:"mfa_enabled"`
}) string {
	if profile.Avatar == "" {
		// No custom avatar, calculate default avatar
		var defaultAvatarNumber int
		if profile.Discriminator == "0" {
			// New username system (no discriminator)
			// Use BigInt arithmetic for large user IDs
			userIDBig := new(big.Int)
			userIDBig.SetString(profile.ID, 10)

			// (user_id >> 22) % 6
			shifted := new(big.Int).Rsh(userIDBig, 22)
			mod := new(big.Int).Mod(shifted, big.NewInt(6))
			defaultAvatarNumber = int(mod.Int64())
		} else {
			// Old discriminator system
			discriminator, _ := strconv.Atoi(profile.Discriminator)
			defaultAvatarNumber = discriminator % 5
		}
		return fmt.Sprintf("https://cdn.discordapp.com/embed/avatars/%d.png", defaultAvatarNumber)
	}

	// Has custom avatar
	format := "png"
	if strings.HasPrefix(profile.Avatar, "a_") {
		format = "gif" // Animated avatar
	}
	return fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.%s", profile.ID, profile.Avatar, format)
}
