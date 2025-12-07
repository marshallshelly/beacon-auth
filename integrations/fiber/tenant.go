package fiber

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

// TenantConfig holds tenant extraction configuration
type TenantConfig struct {
	// BaseDomain is the base domain (e.g., "soulcareuk.com")
	BaseDomain string

	// TenantHeader is an optional header to check for tenant ID
	TenantHeader string

	// DefaultTenant is the tenant to use if none is found
	DefaultTenant string

	// TenantKey is the key used to store tenant in Fiber locals
	TenantKey string
}

// DefaultTenantConfig returns default tenant configuration
func DefaultTenantConfig() *TenantConfig {
	return &TenantConfig{
		TenantHeader:  "X-Tenant-ID",
		DefaultTenant: "",
		TenantKey:     "tenant",
	}
}

// TenantMiddleware creates middleware that extracts tenant from subdomain or header
func TenantMiddleware(config *TenantConfig) fiber.Handler {
	if config == nil {
		config = DefaultTenantConfig()
	}

	return func(c *fiber.Ctx) error {
		var tenant string

		// Check header first
		if config.TenantHeader != "" {
			tenant = c.Get(config.TenantHeader)
		}

		// Extract from subdomain if not in header
		if tenant == "" && config.BaseDomain != "" {
			tenant = ExtractTenantFromHost(c.Hostname(), config.BaseDomain)
		}

		// Use default if still empty
		if tenant == "" {
			tenant = config.DefaultTenant
		}

		// Store tenant in locals
		c.Locals(config.TenantKey, tenant)

		return c.Next()
	}
}

// ExtractTenantFromHost extracts tenant subdomain from hostname
// Example: "sunnyview.soulcareuk.com" with base "soulcareuk.com" returns "sunnyview"
func ExtractTenantFromHost(hostname, baseDomain string) string {
	// Remove port if present
	if idx := strings.IndexByte(hostname, ':'); idx != -1 {
		hostname = hostname[:idx]
	}

	// Handle localhost and IP addresses
	if hostname == "localhost" || strings.Contains(hostname, "127.0.0.1") || strings.Contains(hostname, "::1") {
		return ""
	}

	// Remove base domain
	if !strings.HasSuffix(hostname, baseDomain) {
		return ""
	}

	// Extract subdomain
	subdomain := strings.TrimSuffix(hostname, "."+baseDomain)
	if subdomain == baseDomain {
		return ""
	}

	// Handle multi-level subdomains (take first part only)
	parts := strings.Split(subdomain, ".")
	if len(parts) > 0 {
		return parts[0]
	}

	return subdomain
}

// GetTenant retrieves the tenant from Fiber context
func GetTenant(c *fiber.Ctx) string {
	tenant, ok := c.Locals("tenant").(string)
	if !ok {
		return ""
	}
	return tenant
}

// RequireTenant creates middleware that requires a tenant to be present
func RequireTenant() fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenant := GetTenant(c)
		if tenant == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "tenant_required",
				"message": "Tenant identifier is required",
			})
		}

		return c.Next()
	}
}

// TenantIsolationMiddleware ensures database adapter is tenant-specific
// This is useful for multi-tenant architectures with per-tenant databases
func TenantIsolationMiddleware(getTenantAdapter func(tenantID string) (interface{}, error)) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tenant := GetTenant(c)
		if tenant == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "tenant_required",
				"message": "Tenant identifier is required",
			})
		}

		// Get tenant-specific adapter
		adapter, err := getTenantAdapter(tenant)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error":   "tenant_error",
				"message": "Failed to load tenant configuration",
			})
		}

		// Store adapter in locals for use by handlers
		c.Locals("adapter", adapter)

		return c.Next()
	}
}

// GetAdapter retrieves the adapter from Fiber context
func GetAdapter(c *fiber.Ctx) interface{} {
	return c.Locals("adapter")
}
