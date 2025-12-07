package fiber

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/marshallshelly/beacon-auth/adapters/memory"
	"github.com/marshallshelly/beacon-auth/auth"
	"github.com/marshallshelly/beacon-auth/core"
	"github.com/marshallshelly/beacon-auth/session"
)

func setupTestApp(t *testing.T) (*fiber.App, *session.Manager, *Handler) {
	app := fiber.New()

	// Create in-memory adapter
	dbAdapter := memory.New()

	// Create session manager
	sessionConfig := &session.Config{
		CookieName:     "test_session",
		CookieSecure:   false,
		CookieHTTPOnly: true,
		CookieSameSite: "lax",
		ExpiresIn:      24 * time.Hour,
		EnableDBStore:  true,
		Secret:         "test-secret-key-at-least-32-bytes-long",
		Issuer:         "test",
	}

	sessionManager, err := session.NewManager(sessionConfig, dbAdapter)
	if err != nil {
		t.Fatalf("Failed to create session manager: %v", err)
	}

	// Create auth handler
	authHandler := NewHandler(dbAdapter, sessionManager, &auth.Config{
		MinPasswordLength:   8,
		RequireVerification: false,
		AllowSignup:         true,
	})

	// Add session middleware
	app.Use(SessionMiddleware(sessionManager))

	return app, sessionManager, authHandler
}

func TestFiberIntegration_SignUp(t *testing.T) {
	app, _, authHandler := setupTestApp(t)

	app.Post("/auth/signup", authHandler.SignUp)

	reqBody := auth.SignUpRequest{
		Email:    "fiber@example.com",
		Password: "secure-password-123",
		Name:     "Fiber User",
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/auth/signup", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	if resp.StatusCode != fiber.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Errorf("Expected status %d, got %d. Body: %s", fiber.StatusCreated, resp.StatusCode, string(bodyBytes))
	}

	var authResp auth.AuthResponse
	bodyBytes, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(bodyBytes, &authResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if authResp.User == nil {
		t.Error("Expected user in response")
	}

	if authResp.User.Email != "fiber@example.com" {
		t.Errorf("Expected email %s, got %s", "fiber@example.com", authResp.User.Email)
	}

	// Check session cookie was set
	cookies := resp.Cookies()
	if len(cookies) == 0 {
		t.Error("Expected session cookie to be set")
	}
}

func TestFiberIntegration_SignIn(t *testing.T) {
	app, _, authHandler := setupTestApp(t)

	app.Post("/auth/signup", authHandler.SignUp)
	app.Post("/auth/signin", authHandler.SignIn)

	// First, sign up
	signupReq := auth.SignUpRequest{
		Email:    "signin@example.com",
		Password: "secure-password-123",
		Name:     "SignIn User",
	}

	body, _ := json.Marshal(signupReq)
	req := httptest.NewRequest("POST", "/auth/signup", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Signup failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("Signup failed with status %d", resp.StatusCode)
	}

	// Now sign in
	signinReq := auth.SignInRequest{
		Email:    "signin@example.com",
		Password: "secure-password-123",
	}

	body, _ = json.Marshal(signinReq)
	req = httptest.NewRequest("POST", "/auth/signin", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("Failed to send signin request: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Errorf("Expected status %d, got %d. Body: %s", fiber.StatusOK, resp.StatusCode, string(bodyBytes))
	}

	var authResp auth.AuthResponse
	bodyBytes, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(bodyBytes, &authResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if authResp.User == nil {
		t.Error("Expected user in response")
	}

	if authResp.Session == nil {
		t.Error("Expected session in response")
	}
}

func TestFiberIntegration_RequireAuth(t *testing.T) {
	app, sessionManager, authHandler := setupTestApp(t)

	app.Post("/auth/signup", authHandler.SignUp)

	protected := app.Group("/api")
	protected.Use(RequireAuthJSON(sessionManager))

	protected.Get("/profile", func(c *fiber.Ctx) error {
		user := GetUser(c)
		return c.JSON(fiber.Map{
			"user": user,
		})
	})

	// Test without authentication
	req := httptest.NewRequest("GET", "/api/profile", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", fiber.StatusUnauthorized, resp.StatusCode)
	}

	// Sign up to get a session
	signupReq := auth.SignUpRequest{
		Email:    "protected@example.com",
		Password: "secure-password-123",
		Name:     "Protected User",
	}

	body, _ := json.Marshal(signupReq)
	req = httptest.NewRequest("POST", "/auth/signup", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("Signup failed: %v", err)
	}

	// Get session cookie
	var sessionCookie string
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "test_session" {
			sessionCookie = cookie.Value
			break
		}
	}

	if sessionCookie == "" {
		t.Fatal("No session cookie found")
	}

	// Test with authentication
	req = httptest.NewRequest("GET", "/api/profile", nil)
	req.Header.Set("Cookie", "test_session="+sessionCookie)

	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("Failed to send authenticated request: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Errorf("Expected status %d, got %d. Body: %s", fiber.StatusOK, resp.StatusCode, string(bodyBytes))
	}
}

func TestFiberIntegration_SessionMiddleware(t *testing.T) {
	app, _, authHandler := setupTestApp(t)

	app.Post("/auth/signup", authHandler.SignUp)
	app.Get("/test", func(c *fiber.Ctx) error {
		user := GetUser(c)
		session := GetSession(c)

		return c.JSON(fiber.Map{
			"hasUser":    user != nil,
			"hasSession": session != nil,
		})
	})

	// Test without session
	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	var result map[string]bool
	bodyBytes, _ := io.ReadAll(resp.Body)
	json.Unmarshal(bodyBytes, &result)

	if result["hasUser"] || result["hasSession"] {
		t.Error("Expected no user or session without authentication")
	}

	// Sign up to create a session
	signupReq := auth.SignUpRequest{
		Email:    "middleware@example.com",
		Password: "secure-password-123",
		Name:     "Middleware User",
	}

	body, _ := json.Marshal(signupReq)
	req = httptest.NewRequest("POST", "/auth/signup", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("Signup failed: %v", err)
	}

	// Get session cookie
	var sessionCookie string
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "test_session" {
			sessionCookie = cookie.Value
			break
		}
	}

	// Test with session
	req = httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Cookie", "test_session="+sessionCookie)

	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("Failed to send request with session: %v", err)
	}

	bodyBytes, _ = io.ReadAll(resp.Body)
	json.Unmarshal(bodyBytes, &result)

	if !result["hasUser"] || !result["hasSession"] {
		t.Error("Expected user and session to be loaded by middleware")
	}
}

func TestFiberIntegration_GetSessionHandler(t *testing.T) {
	app, _, authHandler := setupTestApp(t)

	app.Post("/auth/signup", authHandler.SignUp)
	app.Get("/auth/session", authHandler.GetSession)

	// Create user and session
	signupReq := auth.SignUpRequest{
		Email:    "getsession@example.com",
		Password: "secure-password-123",
		Name:     "GetSession User",
	}

	body, _ := json.Marshal(signupReq)
	req := httptest.NewRequest("POST", "/auth/signup", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Signup failed: %v", err)
	}

	// Get session cookie
	var sessionCookie string
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "test_session" {
			sessionCookie = cookie.Value
			break
		}
	}

	// Get session endpoint
	req = httptest.NewRequest("GET", "/auth/session", nil)
	req.Header.Set("Cookie", "test_session="+sessionCookie)

	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("Failed to get session: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Errorf("Expected status %d, got %d. Body: %s", fiber.StatusOK, resp.StatusCode, string(bodyBytes))
	}

	var result map[string]interface{}
	bodyBytes, _ := io.ReadAll(resp.Body)
	json.Unmarshal(bodyBytes, &result)

	if result["user"] == nil {
		t.Error("Expected user in response")
	}

	if result["session"] == nil {
		t.Error("Expected session in response")
	}
}

func TestFiberIntegration_GetUserID(t *testing.T) {
	app := fiber.New()

	app.Get("/test", func(c *fiber.Ctx) error {
		// Test with no user
		userID := GetUserID(c)
		if userID != "" {
			t.Error("Expected empty user ID when no user present")
		}

		// Set a user in locals
		c.Locals("user", &core.User{ID: "test-user-123"})

		userID = GetUserID(c)
		if userID != "test-user-123" {
			t.Errorf("Expected user ID 'test-user-123', got '%s'", userID)
		}

		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	_, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
}
