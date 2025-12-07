package core

import (
	"context"
	"testing"
	"time"
)

// mockAdapter is a simple mock adapter for testing
type mockAdapter struct{}

func (m *mockAdapter) Create(ctx context.Context, model string, data map[string]interface{}) (map[string]interface{}, error) {
	return data, nil
}

func (m *mockAdapter) FindOne(ctx context.Context, query *Query) (map[string]interface{}, error) {
	return nil, nil
}

func (m *mockAdapter) FindMany(ctx context.Context, query *Query) ([]map[string]interface{}, error) {
	return nil, nil
}

func (m *mockAdapter) Update(ctx context.Context, query *Query, data map[string]interface{}) (map[string]interface{}, error) {
	return nil, nil
}

func (m *mockAdapter) UpdateMany(ctx context.Context, query *Query, data map[string]interface{}) (int64, error) {
	return 0, nil
}

func (m *mockAdapter) Delete(ctx context.Context, query *Query) error {
	return nil
}

func (m *mockAdapter) DeleteMany(ctx context.Context, query *Query) (int64, error) {
	return 0, nil
}

func (m *mockAdapter) Count(ctx context.Context, query *Query) (int64, error) {
	return 0, nil
}

func (m *mockAdapter) Transaction(ctx context.Context, fn func(Adapter) error) error {
	return fn(m)
}

func (m *mockAdapter) Ping(ctx context.Context) error {
	return nil
}

func (m *mockAdapter) Close() error {
	return nil
}

func (m *mockAdapter) ID() string {
	return "mock"
}

// Mocks for managers
type mockSessionManager struct{}

func (m *mockSessionManager) Create(ctx context.Context, userID string, opts *SessionOptions) (*Session, *User, string, error) {
	return &Session{}, &User{}, "token", nil
}

type mockDataManager struct{}

func (m *mockDataManager) FindAccountByProvider(ctx context.Context, provider, accountID string) (*Account, error) {
	return nil, nil
}
func (m *mockDataManager) FindUserByEmail(ctx context.Context, email string) (*User, error) {
	return nil, nil
}
func (m *mockDataManager) CreateUser(ctx context.Context, email, name string) (*User, error) {
	return &User{ID: "id"}, nil
}
func (m *mockDataManager) CreateOAuthAccount(ctx context.Context, userID, provider, accountID, accessToken, refreshToken string, expiresAt *time.Time) (*Account, error) {
	return &Account{}, nil
}

func withMockFactories() Option {
	return func(c *Config) error {
		c.DataManagerFactory = func(a Adapter) DataManager { return &mockDataManager{} }
		c.SessionManagerFactory = func(cfg *Config, a Adapter) (SessionManager, error) { return &mockSessionManager{}, nil }
		return nil
	}
}

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		opts    []Option
		wantErr bool
	}{
		{
			name: "valid configuration",
			opts: []Option{
				WithSecret("test-secret"),
				WithBaseURL("http://localhost:3000"),
				WithAdapter(&mockAdapter{}),
				withMockFactories(),
			},
			wantErr: false,
		},
		{
			name: "missing secret",
			opts: []Option{
				WithBaseURL("http://localhost:3000"),
				WithAdapter(&mockAdapter{}),
				withMockFactories(),
			},
			wantErr: true,
		},
		{
			name: "missing adapter",
			opts: []Option{
				WithSecret("test-secret"),
				WithBaseURL("http://localhost:3000"),
				withMockFactories(),
			},
			wantErr: true,
		},
		{
			name: "missing base URL",
			opts: []Option{
				WithSecret("test-secret"),
				WithAdapter(&mockAdapter{}),
				withMockFactories(),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth, err := New(tt.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && auth == nil {
				t.Error("New() returned nil auth instance")
			}
			if auth != nil {
				auth.Close()
			}
		})
	}
}

func TestAuthContext(t *testing.T) {
	adapter := &mockAdapter{}
	auth, err := New(
		WithSecret("test-secret"),
		WithBaseURL("http://localhost:3000"),
		WithAdapter(adapter),
		withMockFactories(),
	)
	if err != nil {
		t.Fatalf("Failed to create auth: %v", err)
	}
	defer auth.Close()

	ctx := auth.Context()
	if ctx == nil {
		t.Error("Context() returned nil")
	}

	if ctx.Config == nil {
		t.Error("Context config is nil")
	}

	if ctx.Adapter != adapter {
		t.Error("Context adapter does not match provided adapter")
	}
}

func TestWithOptions(t *testing.T) {
	tests := []struct {
		name string
		opt  Option
		test func(*testing.T, *Config)
	}{
		{
			name: "WithSecret",
			opt:  WithSecret("my-secret"),
			test: func(t *testing.T, c *Config) {
				if c.Secret != "my-secret" {
					t.Errorf("Secret = %q, want %q", c.Secret, "my-secret")
				}
			},
		},
		{
			name: "WithBaseURL",
			opt:  WithBaseURL("http://example.com"),
			test: func(t *testing.T, c *Config) {
				if c.BaseURL != "http://example.com" {
					t.Errorf("BaseURL = %q, want %q", c.BaseURL, "http://example.com")
				}
			},
		},
		{
			name: "WithBasePath",
			opt:  WithBasePath("/api/auth"),
			test: func(t *testing.T, c *Config) {
				if c.BasePath != "/api/auth" {
					t.Errorf("BasePath = %q, want %q", c.BasePath, "/api/auth")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := defaultConfig()
			if err := tt.opt(cfg); err != nil {
				t.Fatalf("Option failed: %v", err)
			}
			tt.test(t, cfg)
		})
	}
}
