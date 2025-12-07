package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/marshallshelly/beacon-auth/adapters/memory"
	"github.com/marshallshelly/beacon-auth/core"
	"github.com/marshallshelly/beacon-auth/crypto"
	"github.com/marshallshelly/beacon-auth/session"
)

func setupTestHandler(t *testing.T) (*Handler, *session.Manager) {
	// Create in-memory adapter
	dbAdapter := memory.New()

	// Create session manager
	sessionConfig := &session.Config{
		CookieName:        "test_session",
		CookiePath:        "/",
		CookieDomain:      "",
		CookieSecure:      false,
		CookieHTTPOnly:    true,
		CookieSameSite:    "lax",
		ExpiresIn:         24 * time.Hour,
		UpdateAge:         1 * time.Hour,
		AbsoluteExpiry:    false,
		EnableCookieStore: false,
		EnableRedisStore:  false,
		EnableDBStore:     true,
		Secret:            "test-secret-key-at-least-32-bytes-long",
		Issuer:            "beaconauth-test",
	}

	sessionManager, err := session.NewManager(sessionConfig, dbAdapter)
	if err != nil {
		t.Fatalf("Failed to create session manager: %v", err)
	}

	// Create handler
	handlerConfig := &Config{
		MinPasswordLength:   8,
		RequireVerification: false,
		AllowSignup:         true,
	}

	handler := NewHandler(dbAdapter, sessionManager, handlerConfig)

	return handler, sessionManager
}

func TestSignUp_Success(t *testing.T) {
	handler, _ := setupTestHandler(t)

	reqBody := SignUpRequest{
		Email:    "test@example.com",
		Password: "secure-password-123",
		Name:     "Test User",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/auth/signup", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.SignUp(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, w.Code)
		t.Logf("Response body: %s", w.Body.String())
	}

	var resp AuthResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.User == nil {
		t.Error("Expected user in response")
	}

	if resp.User.Email != "test@example.com" {
		t.Errorf("Expected email %s, got %s", "test@example.com", resp.User.Email)
	}

	if resp.Session == nil {
		t.Error("Expected session in response")
	}

	if resp.Token == "" {
		t.Error("Expected token in response")
	}

	// Check that session cookie was set
	cookies := w.Result().Cookies()
	if len(cookies) == 0 {
		t.Error("Expected session cookie to be set")
	}
}

func TestSignUp_DuplicateUser(t *testing.T) {
	handler, _ := setupTestHandler(t)

	// Create first user
	reqBody := SignUpRequest{
		Email:    "duplicate@example.com",
		Password: "secure-password-123",
		Name:     "First User",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/auth/signup", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.SignUp(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("First signup failed: %d", w.Code)
	}

	// Try to create duplicate user
	body, _ = json.Marshal(reqBody)
	req = httptest.NewRequest(http.MethodPost, "/auth/signup", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	handler.SignUp(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("Expected status %d, got %d", http.StatusConflict, w.Code)
	}

	var errResp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}

	if errResp.Error != "user_exists" {
		t.Errorf("Expected error code 'user_exists', got '%s'", errResp.Error)
	}
}

func TestSignUp_ValidationErrors(t *testing.T) {
	handler, _ := setupTestHandler(t)

	tests := []struct {
		name           string
		request        SignUpRequest
		expectedStatus int
		expectedError  string
	}{
		{
			name: "missing email",
			request: SignUpRequest{
				Password: "secure-password-123",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "validation_error",
		},
		{
			name: "missing password",
			request: SignUpRequest{
				Email: "test@example.com",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "validation_error",
		},
		{
			name: "short password",
			request: SignUpRequest{
				Email:    "test@example.com",
				Password: "short",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "validation_error",
		},
		{
			name: "invalid email",
			request: SignUpRequest{
				Email:    "not-an-email",
				Password: "secure-password-123",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "validation_error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.request)
			req := httptest.NewRequest(http.MethodPost, "/auth/signup", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.SignUp(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			var errResp ErrorResponse
			if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
				t.Fatalf("Failed to decode error response: %v", err)
			}

			if errResp.Error != tt.expectedError {
				t.Errorf("Expected error code '%s', got '%s'", tt.expectedError, errResp.Error)
			}
		})
	}
}

func TestSignUp_DisabledSignup(t *testing.T) {
	handler, _ := setupTestHandler(t)
	handler.config.AllowSignup = false

	reqBody := SignUpRequest{
		Email:    "test@example.com",
		Password: "secure-password-123",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/auth/signup", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.SignUp(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status %d, got %d", http.StatusForbidden, w.Code)
	}

	var errResp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}

	if errResp.Error != "signup_disabled" {
		t.Errorf("Expected error code 'signup_disabled', got '%s'", errResp.Error)
	}
}

func TestSignIn_Success(t *testing.T) {
	handler, _ := setupTestHandler(t)

	// First, create a user
	signupReq := SignUpRequest{
		Email:    "signin@example.com",
		Password: "secure-password-123",
		Name:     "SignIn User",
	}

	body, _ := json.Marshal(signupReq)
	req := httptest.NewRequest(http.MethodPost, "/auth/signup", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.SignUp(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Signup failed: %d", w.Code)
	}

	// Now sign in
	signinReq := SignInRequest{
		Email:    "signin@example.com",
		Password: "secure-password-123",
	}

	body, _ = json.Marshal(signinReq)
	req = httptest.NewRequest(http.MethodPost, "/auth/signin", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	handler.SignIn(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		t.Logf("Response body: %s", w.Body.String())
	}

	var resp AuthResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.User == nil {
		t.Error("Expected user in response")
	}

	if resp.Session == nil {
		t.Error("Expected session in response")
	}

	if resp.Token == "" {
		t.Error("Expected token in response")
	}
}

func TestSignIn_InvalidCredentials(t *testing.T) {
	handler, _ := setupTestHandler(t)

	// Create a user
	signupReq := SignUpRequest{
		Email:    "wrongpass@example.com",
		Password: "correct-password-123",
		Name:     "Test User",
	}

	body, _ := json.Marshal(signupReq)
	req := httptest.NewRequest(http.MethodPost, "/auth/signup", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.SignUp(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Signup failed: %d", w.Code)
	}

	tests := []struct {
		name           string
		request        SignInRequest
		expectedStatus int
		expectedError  string
	}{
		{
			name: "wrong password",
			request: SignInRequest{
				Email:    "wrongpass@example.com",
				Password: "wrong-password-123",
			},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "invalid_credentials",
		},
		{
			name: "non-existent user",
			request: SignInRequest{
				Email:    "nonexistent@example.com",
				Password: "any-password-123",
			},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "invalid_credentials",
		},
		{
			name: "missing email",
			request: SignInRequest{
				Password: "password-123",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "validation_error",
		},
		{
			name: "missing password",
			request: SignInRequest{
				Email: "test@example.com",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "validation_error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.request)
			req := httptest.NewRequest(http.MethodPost, "/auth/signin", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.SignIn(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			var errResp ErrorResponse
			if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
				t.Fatalf("Failed to decode error response: %v", err)
			}

			if errResp.Error != tt.expectedError {
				t.Errorf("Expected error code '%s', got '%s'", tt.expectedError, errResp.Error)
			}
		})
	}
}

func TestSignIn_RequireVerification(t *testing.T) {
	handler, _ := setupTestHandler(t)
	handler.config.RequireVerification = true

	// Create a user (will be unverified by default)
	signupReq := SignUpRequest{
		Email:    "unverified@example.com",
		Password: "secure-password-123",
		Name:     "Unverified User",
	}

	body, _ := json.Marshal(signupReq)
	req := httptest.NewRequest(http.MethodPost, "/auth/signup", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.SignUp(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Signup failed: %d", w.Code)
	}

	// Try to sign in with unverified email
	signinReq := SignInRequest{
		Email:    "unverified@example.com",
		Password: "secure-password-123",
	}

	body, _ = json.Marshal(signinReq)
	req = httptest.NewRequest(http.MethodPost, "/auth/signin", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	handler.SignIn(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status %d, got %d", http.StatusForbidden, w.Code)
	}

	var errResp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}

	if errResp.Error != "email_not_verified" {
		t.Errorf("Expected error code 'email_not_verified', got '%s'", errResp.Error)
	}
}

func TestSignOut_Success(t *testing.T) {
	handler, sessionManager := setupTestHandler(t)

	// Create a user and session
	signupReq := SignUpRequest{
		Email:    "signout@example.com",
		Password: "secure-password-123",
		Name:     "SignOut User",
	}

	body, _ := json.Marshal(signupReq)
	req := httptest.NewRequest(http.MethodPost, "/auth/signup", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.SignUp(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Signup failed: %d", w.Code)
	}

	// Get the session cookie
	cookies := w.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("No session cookie found")
	}

	sessionCookie := cookies[0]

	// Sign out
	req = httptest.NewRequest(http.MethodPost, "/auth/signout", nil)
	req.AddCookie(sessionCookie)
	w = httptest.NewRecorder()

	handler.SignOut(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Verify session was deleted
	ctx := context.Background()
	_, _, err := sessionManager.Get(ctx, sessionCookie.Value)
	if err == nil {
		t.Error("Expected session to be deleted")
	}

	// Check that cookie was cleared
	responseCookies := w.Result().Cookies()
	if len(responseCookies) == 0 {
		t.Error("Expected cookie to be cleared")
	} else {
		clearedCookie := responseCookies[0]
		if clearedCookie.MaxAge != -1 {
			t.Errorf("Expected MaxAge -1, got %d", clearedCookie.MaxAge)
		}
	}
}

func TestSignOut_NoCookie(t *testing.T) {
	handler, _ := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/auth/signout", nil)
	w := httptest.NewRecorder()

	handler.SignOut(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var errResp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}

	if errResp.Error != "no_session" {
		t.Errorf("Expected error code 'no_session', got '%s'", errResp.Error)
	}
}

func TestGetSession_Success(t *testing.T) {
	handler, _ := setupTestHandler(t)

	// Create a user
	signupReq := SignUpRequest{
		Email:    "getsession@example.com",
		Password: "secure-password-123",
		Name:     "GetSession User",
	}

	body, _ := json.Marshal(signupReq)
	req := httptest.NewRequest(http.MethodPost, "/auth/signup", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.SignUp(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Signup failed: %d", w.Code)
	}

	// Decode signup response to get session
	var signupResp AuthResponse
	if err := json.NewDecoder(bytes.NewReader(w.Body.Bytes())).Decode(&signupResp); err != nil {
		t.Fatalf("Failed to decode signup response: %v", err)
	}

	// Create request with session in context
	req = httptest.NewRequest(http.MethodGet, "/auth/session", nil)
	ctx := core.WithSession(req.Context(), signupResp.Session)
	ctx = core.WithUser(ctx, signupResp.User)
	req = req.WithContext(ctx)
	w = httptest.NewRecorder()

	handler.GetSession(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp AuthResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.User == nil {
		t.Error("Expected user in response")
	}

	if resp.Session == nil {
		t.Error("Expected session in response")
	}
}

func TestGetSession_NoSession(t *testing.T) {
	handler, _ := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/auth/session", nil)
	w := httptest.NewRecorder()

	handler.GetSession(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}

	var errResp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}

	if errResp.Error != "no_session" {
		t.Errorf("Expected error code 'no_session', got '%s'", errResp.Error)
	}
}

func TestPasswordHashing(t *testing.T) {
	handler, _ := setupTestHandler(t)

	password := "test-password-123"

	// Hash password
	hash, err := handler.hasher.Hash(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	// Verify correct password
	valid, err := handler.hasher.Verify(password, hash)
	if err != nil {
		t.Fatalf("Failed to verify password: %v", err)
	}

	if !valid {
		t.Error("Expected password to be valid")
	}

	// Verify incorrect password
	valid, err = handler.hasher.Verify("wrong-password", hash)
	if err != nil {
		t.Fatalf("Failed to verify password: %v", err)
	}

	if valid {
		t.Error("Expected password to be invalid")
	}
}

func TestHelperFunctions(t *testing.T) {
	t.Run("isValidEmail", func(t *testing.T) {
		tests := []struct {
			email string
			valid bool
		}{
			{"test@example.com", true},
			{"user@domain.org", true},
			{"invalid", false},
			{"@example.com", true}, // Basic validator accepts this
			{"test@", false},
			{"", false},
		}

		for _, tt := range tests {
			result := isValidEmail(tt.email)
			if result != tt.valid {
				t.Errorf("isValidEmail(%s) = %v, expected %v", tt.email, result, tt.valid)
			}
		}
	})

	t.Run("getIPAddress", func(t *testing.T) {
		tests := []struct {
			name       string
			setupReq   func() *http.Request
			expectedIP string
		}{
			{
				name: "X-Forwarded-For header",
				setupReq: func() *http.Request {
					req := httptest.NewRequest(http.MethodGet, "/", nil)
					req.Header.Set("X-Forwarded-For", "1.2.3.4")
					return req
				},
				expectedIP: "1.2.3.4",
			},
			{
				name: "X-Real-IP header",
				setupReq: func() *http.Request {
					req := httptest.NewRequest(http.MethodGet, "/", nil)
					req.Header.Set("X-Real-IP", "5.6.7.8")
					return req
				},
				expectedIP: "5.6.7.8",
			},
			{
				name: "RemoteAddr fallback",
				setupReq: func() *http.Request {
					req := httptest.NewRequest(http.MethodGet, "/", nil)
					req.RemoteAddr = "9.10.11.12:12345"
					return req
				},
				expectedIP: "9.10.11.12:12345",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				req := tt.setupReq()
				ip := getIPAddress(req)
				if ip != tt.expectedIP {
					t.Errorf("getIPAddress() = %s, expected %s", ip, tt.expectedIP)
				}
			})
		}
	})

	t.Run("parseSameSite", func(t *testing.T) {
		tests := []struct {
			input    string
			expected http.SameSite
		}{
			{"strict", http.SameSiteStrictMode},
			{"lax", http.SameSiteLaxMode},
			{"none", http.SameSiteNoneMode},
			{"invalid", http.SameSiteLaxMode},
			{"", http.SameSiteLaxMode},
		}

		for _, tt := range tests {
			result := parseSameSite(tt.input)
			if result != tt.expected {
				t.Errorf("parseSameSite(%s) = %v, expected %v", tt.input, result, tt.expected)
			}
		}
	})
}

func TestIDGeneration(t *testing.T) {
	// Test that GenerateID produces unique IDs
	id1, err := crypto.GenerateID()
	if err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}

	id2, err := crypto.GenerateID()
	if err != nil {
		t.Fatalf("Failed to generate ID: %v", err)
	}

	if id1 == id2 {
		t.Error("Expected unique IDs, got duplicates")
	}

	if len(id1) == 0 {
		t.Error("Expected non-empty ID")
	}
}
