package chi

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

// TenantMiddleware Chi middleware
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

			// Extract from subdomain
			if tenant == "" && config.BaseDomain != "" {
				tenant = extractTenantFromHost(r.Host, config.BaseDomain)
			}

			if tenant == "" {
				tenant = config.DefaultTenant
			}

			// Store in context (Chi compatible since it's just context)
			ctx := context.WithValue(r.Context(), tenantKey{}, tenant)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
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
func GetTenant(ctx context.Context) string {
	tenant, ok := ctx.Value(tenantKey{}).(string)
	if !ok {
		return ""
	}
	return tenant
}

// RequireTenant middleware
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
