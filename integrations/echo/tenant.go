package echo

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

// TenantConfig configuration
type TenantConfig struct {
	BaseDomain    string
	TenantHeader  string
	DefaultTenant string
	TenantKey     string
}

// DefaultTenantConfig defaults
func DefaultTenantConfig() *TenantConfig {
	return &TenantConfig{
		TenantHeader:  "X-Tenant-ID",
		DefaultTenant: "",
		TenantKey:     "tenant",
	}
}

// TenantMiddleware Echo middleware
func TenantMiddleware(config *TenantConfig) echo.MiddlewareFunc {
	if config == nil {
		config = DefaultTenantConfig()
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			var tenant string

			if config.TenantHeader != "" {
				tenant = c.Request().Header.Get(config.TenantHeader)
			}

			if tenant == "" && config.BaseDomain != "" {
				tenant = extractTenantFromHost(c.Request().Host, config.BaseDomain)
			}

			if tenant == "" {
				tenant = config.DefaultTenant
			}

			c.Set(config.TenantKey, tenant)
			return next(c)
		}
	}
}

func extractTenantFromHost(hostname, baseDomain string) string {
	if idx := strings.IndexByte(hostname, ':'); idx != -1 {
		hostname = hostname[:idx]
	}

	if hostname == "localhost" || strings.Contains(hostname, "127.0.0.1") || strings.Contains(hostname, "::1") {
		return ""
	}

	if !strings.HasSuffix(hostname, baseDomain) {
		return ""
	}

	subdomain := strings.TrimSuffix(hostname, "."+baseDomain)
	if subdomain == baseDomain {
		return ""
	}

	parts := strings.Split(subdomain, ".")
	if len(parts) > 0 {
		return parts[0]
	}

	return subdomain
}

// GetTenant helper
func GetTenant(c echo.Context) string {
	if v := c.Get("tenant"); v != nil {
		if t, ok := v.(string); ok {
			return t
		}
	}
	return ""
}

// RequireTenant middleware
func RequireTenant() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			tenant := GetTenant(c)
			if tenant == "" {
				return c.JSON(http.StatusBadRequest, map[string]string{
					"error":   "tenant_required",
					"message": "Tenant identifier is required",
				})
			}
			return next(c)
		}
	}
}
