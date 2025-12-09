package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

type tenantKey struct{}

// TenantConfig holds tenant extraction configuration
type TenantConfig struct {
	BaseDomain    string
	TenantHeader  string
	DefaultTenant string
}

// DefaultTenantConfig returns default tenant configuration
func DefaultTenantConfig() *TenantConfig {
	return &TenantConfig{
		TenantHeader:  "X-Tenant-ID",
		DefaultTenant: "",
	}
}

// TenantMiddleware creates middleware that extracts tenant from subdomain or header
func TenantMiddleware(config *TenantConfig) func(http.Handler) http.Handler {
	if config == nil {
		config = DefaultTenantConfig()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var tenant string

			// Check header first
			if config.TenantHeader != "" {
				tenant = r.Header.Get(config.TenantHeader)
			}

			// Extract from subdomain if not in header
			if tenant == "" && config.BaseDomain != "" {
				tenant = extractTenantFromHost(r.Host, config.BaseDomain)
			}

			// Use default if still empty
			if tenant == "" {
				tenant = config.DefaultTenant
			}

			// Store tenant in context
			ctx := context.WithValue(r.Context(), tenantKey{}, tenant)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// extractTenantFromHost extracts tenant subdomain from hostname
func extractTenantFromHost(hostname, baseDomain string) string {
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

// GetTenant retrieves the tenant from context
func GetTenant(ctx context.Context) string {
	tenant, ok := ctx.Value(tenantKey{}).(string)
	if !ok {
		return ""
	}
	return tenant
}

// RequireTenant creates middleware that requires a tenant to be present
func RequireTenant() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tenant := GetTenant(r.Context())
			if tenant == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error":   "tenant_required",
					"message": "Tenant identifier is required",
				})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
