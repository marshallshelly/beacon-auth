package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type GitHubProvider struct {
	clientID     string
	clientSecret string
	scopes       []string
	httpClient   *http.Client
}

func NewGitHub(clientID, clientSecret string, scopes []string) *GitHubProvider {
	if len(scopes) == 0 {
		scopes = []string{"user:email"}
	}

	return &GitHubProvider{
		clientID:     clientID,
		clientSecret: clientSecret,
		scopes:       scopes,
		httpClient:   &http.Client{Timeout: 10 * time.Second},
	}
}

func (p *GitHubProvider) ID() string {
	return "github"
}

func (p *GitHubProvider) Name() string {
	return "GitHub"
}

func (p *GitHubProvider) Init() error {
	if p.clientID == "" || p.clientSecret == "" {
		return fmt.Errorf("GitHub client ID and secret are required")
	}
	return nil
}

func (p *GitHubProvider) CreateAuthorizationURL(state, redirectURI string, options *AuthOptions) (*url.URL, error) {
	authURL, _ := url.Parse("https://github.com/login/oauth/authorize")

	q := authURL.Query()
	q.Set("client_id", p.clientID)
	q.Set("redirect_uri", redirectURI)
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

func (p *GitHubProvider) ExchangeCode(ctx context.Context, code, codeVerifier, redirectURI string) (*OAuthTokens, error) {
	data := url.Values{}
	data.Set("client_id", p.clientID)
	data.Set("client_secret", p.clientSecret)
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)

	req, err := http.NewRequestWithContext(ctx, "POST", "https://github.com/login/oauth/access_token", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
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
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		Scope       string `json:"scope"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.AccessToken == "" {
		return nil, fmt.Errorf("empty access token received")
	}

	return &OAuthTokens{
		AccessToken: result.AccessToken,
		TokenType:   result.TokenType,
	}, nil
}

func (p *GitHubProvider) GetUserInfo(ctx context.Context, accessToken string) (*OAuthUserInfo, error) {
	// First get user profile
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user", nil)
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

	var userMap map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&userMap); err != nil {
		return nil, err
	}

	// Then get emails to find primary verified email
	emailReq, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user/emails", nil)
	if err != nil {
		return nil, err
	}
	emailReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))

	emailResp, err := p.httpClient.Do(emailReq)
	if err != nil {
		return nil, err
	}
	defer func() { _ = emailResp.Body.Close() }()

	var email string
	var emailVerified bool

	if emailResp.StatusCode == http.StatusOK {
		var emails []struct {
			Email    string `json:"email"`
			Primary  bool   `json:"primary"`
			Verified bool   `json:"verified"`
		}
		if err := json.NewDecoder(emailResp.Body).Decode(&emails); err == nil {
			for _, e := range emails {
				if e.Primary {
					email = e.Email
					emailVerified = e.Verified
					break
				}
			}
			// If no primary found, take the first verified
			if email == "" {
				for _, e := range emails {
					if e.Verified {
						email = e.Email
						emailVerified = true
						break
					}
				}
			}
		}
	}

	// Fallback if email API failed or returned nothing useful, check 'email' from profile
	if email == "" {
		if e, ok := userMap["email"].(string); ok {
			email = e
			emailVerified = false // GitHub public profile email isn't necessarily verified in this context
		}
	}

	id := fmt.Sprintf("%v", userMap["id"])
	name, _ := userMap["name"].(string)
	picture, _ := userMap["avatar_url"].(string)

	// Split name
	var firstName, lastName string
	if name != "" {
		parts := strings.SplitN(name, " ", 2)
		firstName = parts[0]
		if len(parts) > 1 {
			lastName = parts[1]
		}
	}

	return &OAuthUserInfo{
		ID:            id,
		Email:         email,
		EmailVerified: emailVerified,
		Name:          name,
		FirstName:     firstName,
		LastName:      lastName,
		Picture:       picture,
		RawData:       userMap,
	}, nil
}

func (p *GitHubProvider) RefreshToken(ctx context.Context, refreshToken string) (*OAuthTokens, error) {
	return nil, fmt.Errorf("refresh token not supported by GitHub")
}
