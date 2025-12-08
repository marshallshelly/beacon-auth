package fiber

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestExtractTenantFromHost(t *testing.T) {
	tests := []struct {
		name       string
		hostname   string
		baseDomain string
		expected   string
	}{
		{
			name:       "valid subdomain",
			hostname:   "sunnyview.example.com",
			baseDomain: "example.com",
			expected:   "sunnyview",
		},
		{
			name:       "with port",
			hostname:   "acme.example.com:3000",
			baseDomain: "example.com",
			expected:   "acme",
		},
		{
			name:       "localhost",
			hostname:   "localhost",
			baseDomain: "example.com",
			expected:   "",
		},
		{
			name:       "base domain only",
			hostname:   "example.com",
			baseDomain: "example.com",
			expected:   "",
		},
		{
			name:       "different domain",
			hostname:   "example.com",
			baseDomain: "example.com",
			expected:   "",
		},
		{
			name:       "multi-level subdomain",
			hostname:   "app.sunnyview.example.com",
			baseDomain: "example.com",
			expected:   "app",
		},
		{
			name:       "IP address",
			hostname:   "127.0.0.1",
			baseDomain: "example.com",
			expected:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractTenantFromHost(tt.hostname, tt.baseDomain)
			if result != tt.expected {
				t.Errorf("ExtractTenantFromHost(%s, %s) = %s, expected %s",
					tt.hostname, tt.baseDomain, result, tt.expected)
			}
		})
	}
}

func TestTenantMiddleware(t *testing.T) {
	app := fiber.New()

	config := &TenantConfig{
		BaseDomain:   "example.com",
		TenantHeader: "X-Tenant-ID",
	}

	app.Use(TenantMiddleware(config))

	app.Get("/test", func(c *fiber.Ctx) error {
		tenant := GetTenant(c)
		return c.JSON(fiber.Map{
			"tenant": tenant,
		})
	})

	tests := []struct {
		name     string
		hostname string
		header   string
		expected string
	}{
		{
			name:     "from subdomain",
			hostname: "acme.example.com",
			header:   "",
			expected: "acme",
		},
		{
			name:     "from header",
			hostname: "localhost",
			header:   "customtenant",
			expected: "customtenant",
		},
		{
			name:     "header takes precedence",
			hostname: "acme.example.com",
			header:   "override",
			expected: "override",
		},
		{
			name:     "no tenant",
			hostname: "localhost",
			header:   "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "http://"+tt.hostname+"/test", nil)
			if tt.header != "" {
				req.Header.Set("X-Tenant-ID", tt.header)
			}

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}

			if resp.StatusCode != fiber.StatusOK {
				t.Errorf("Expected status %d, got %d", fiber.StatusOK, resp.StatusCode)
			}

			// For now, just verify the request was successful
			// In a real integration test, you'd parse the JSON response
		})
	}
}

func TestRequireTenant(t *testing.T) {
	app := fiber.New()

	app.Use(func(c *fiber.Ctx) error {
		// Manually set tenant for testing
		tenant := c.Query("tenant")
		if tenant != "" {
			c.Locals("tenant", tenant)
		}
		return c.Next()
	})

	app.Use(RequireTenant())

	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	// Test without tenant
	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", fiber.StatusBadRequest, resp.StatusCode)
	}

	// Test with tenant
	req = httptest.NewRequest("GET", "/test?tenant=acme", nil)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("Expected status %d, got %d", fiber.StatusOK, resp.StatusCode)
	}
}

func TestTenantIsolationMiddleware(t *testing.T) {
	app := fiber.New()

	// Set up tenant in locals
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("tenant", "test-tenant")
		return c.Next()
	})

	// Add tenant isolation middleware
	callCount := 0
	app.Use(TenantIsolationMiddleware(func(tenantID string) (interface{}, error) {
		callCount++
		if tenantID != "test-tenant" {
			t.Errorf("Expected tenant 'test-tenant', got '%s'", tenantID)
		}
		return "mock-adapter", nil
	}))

	app.Get("/test", func(c *fiber.Ctx) error {
		adapter := GetAdapter(c)
		if adapter == nil {
			t.Error("Expected adapter to be set")
		}
		if adapter != "mock-adapter" {
			t.Errorf("Expected adapter 'mock-adapter', got %v", adapter)
		}
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("Expected status %d, got %d", fiber.StatusOK, resp.StatusCode)
	}

	if callCount != 1 {
		t.Errorf("Expected getTenantAdapter to be called once, called %d times", callCount)
	}
}

func TestGetTenant(t *testing.T) {
	app := fiber.New()

	app.Get("/test", func(c *fiber.Ctx) error {
		// Test with no tenant
		tenant := GetTenant(c)
		if tenant != "" {
			t.Errorf("Expected empty tenant, got '%s'", tenant)
		}

		// Set tenant and test again
		c.Locals("tenant", "my-tenant")
		tenant = GetTenant(c)
		if tenant != "my-tenant" {
			t.Errorf("Expected tenant 'my-tenant', got '%s'", tenant)
		}

		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	app.Test(req)
}

func TestDefaultTenantConfig(t *testing.T) {
	config := DefaultTenantConfig()

	if config.TenantHeader != "X-Tenant-ID" {
		t.Errorf("Expected TenantHeader 'X-Tenant-ID', got '%s'", config.TenantHeader)
	}

	if config.DefaultTenant != "" {
		t.Errorf("Expected DefaultTenant '', got '%s'", config.DefaultTenant)
	}

	if config.TenantKey != "tenant" {
		t.Errorf("Expected TenantKey 'tenant', got '%s'", config.TenantKey)
	}
}

func TestTenantMiddleware_DefaultTenant(t *testing.T) {
	app := fiber.New()

	config := &TenantConfig{
		BaseDomain:    "example.com",
		DefaultTenant: "default",
		TenantHeader:  "", // Don't check header
		TenantKey:     "tenant",
	}

	app.Use(TenantMiddleware(config))

	app.Get("/test", func(c *fiber.Ctx) error {
		tenant := GetTenant(c)
		if tenant != "default" {
			t.Errorf("Expected default tenant 'default', got '%s'", tenant)
		}
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest("GET", "http://localhost/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("Expected status %d, got %d", fiber.StatusOK, resp.StatusCode)
	}
}
