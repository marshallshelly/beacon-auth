package gin

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// TenantConfig holds tenant extraction configuration
type TenantConfig struct {
	BaseDomain    string
	TenantHeader  string
	DefaultTenant string
	TenantKey     string
}

// DefaultTenantConfig returns default tenant configuration
func DefaultTenantConfig() *TenantConfig {
	return &TenantConfig{
		TenantHeader:  "X-Tenant-ID",
		DefaultTenant: "",
		TenantKey:     "tenant",
	}
}

// TenantMiddleware Gin middleware
func TenantMiddleware(config *TenantConfig) gin.HandlerFunc {
	if config == nil {
		config = DefaultTenantConfig()
	}

	return func(c *gin.Context) {
		var tenant string

		if config.TenantHeader != "" {
			tenant = c.GetHeader(config.TenantHeader)
		}

		if tenant == "" && config.BaseDomain != "" {
			tenant = extractTenantFromHost(c.Request.Host, config.BaseDomain)
		}

		if tenant == "" {
			tenant = config.DefaultTenant
		}

		c.Set(config.TenantKey, tenant)
		c.Next()
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

// GetTenant retrieves tenant from context
func GetTenant(c *gin.Context) string {
	if v, exists := c.Get("tenant"); exists {
		if t, ok := v.(string); ok {
			return t
		}
	}
	return ""
}

// RequireTenant middleware
func RequireTenant() gin.HandlerFunc {
	return func(c *gin.Context) {
		tenant := GetTenant(c)
		if tenant == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "tenant_required",
				"message": "Tenant identifier is required",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}
