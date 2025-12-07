package fiber_test

import (
	"context"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/marshallshelly/beacon-auth/adapters/postgres"
	"github.com/marshallshelly/beacon-auth/auth"
	"github.com/marshallshelly/beacon-auth/core"
	beaconauth_fiber "github.com/marshallshelly/beacon-auth/integrations/fiber"
	"github.com/marshallshelly/beacon-auth/session"
)

// Example_basicFiberIntegration demonstrates basic BeaconAuth integration with Fiber
func Example_basicFiberIntegration() {
	app := fiber.New()

	// Create database adapter
	dbAdapter, err := postgres.New(context.Background(), &postgres.Config{
		Host:     "localhost",
		Port:     5432,
		Database: "myapp",
		Username: "postgres",
		Password: "password",
	})
	if err != nil {
		log.Fatal(err)
	}
	defer dbAdapter.Close()

	// Create session manager
	sessionConfig := &session.Config{
		CookieName:     "app_session",
		CookieSecure:   true,
		CookieHTTPOnly: true,
		CookieSameSite: "lax",
		ExpiresIn:      7 * 24 * time.Hour,
		EnableDBStore:  true,
		Secret:         "your-secret-key-at-least-32-bytes",
		Issuer:         "myapp",
	}

	sessionManager, err := session.NewManager(sessionConfig, dbAdapter)
	if err != nil {
		log.Fatal(err)
	}

	// Add session middleware
	app.Use(beaconauth_fiber.SessionMiddleware(sessionManager))

	// Create auth handler
	authHandler := beaconauth_fiber.NewHandler(dbAdapter, sessionManager, &auth.Config{
		MinPasswordLength:   8,
		RequireVerification: false,
		AllowSignup:         true,
	})

	// Auth routes
	app.Post("/auth/signup", authHandler.SignUp)
	app.Post("/auth/signin", authHandler.SignIn)
	app.Post("/auth/signout", authHandler.SignOut)
	app.Get("/auth/session", authHandler.GetSession)

	// Protected routes
	protected := app.Group("/api")
	protected.Use(beaconauth_fiber.RequireAuthJSON(sessionManager))

	protected.Get("/profile", func(c *fiber.Ctx) error {
		user := beaconauth_fiber.GetUser(c)
		return c.JSON(fiber.Map{
			"user": user,
		})
	})

	app.Listen(":3000")
}

// Example_multiTenantFiberIntegration demonstrates multi-tenant BeaconAuth with Fiber
func Example_multiTenantFiberIntegration() {
	app := fiber.New()

	// CORS configuration for multi-tenant
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "https://*.soulcareuk.com",
		AllowCredentials: true,
	}))

	app.Use(logger.New())

	// Tenant extraction middleware
	tenantConfig := &beaconauth_fiber.TenantConfig{
		BaseDomain:    "soulcareuk.com",
		TenantHeader:  "X-Tenant-ID",
		DefaultTenant: "",
	}
	app.Use(beaconauth_fiber.TenantMiddleware(tenantConfig))

	// Tenant-specific database adapter middleware
	app.Use(beaconauth_fiber.TenantIsolationMiddleware(func(tenantID string) (interface{}, error) {
		// Connect to tenant-specific database
		dbAdapter, err := postgres.New(context.Background(), &postgres.Config{
			Host:     "localhost",
			Port:     5432,
			Database: "tenant_" + tenantID, // Each tenant has its own database
			Username: "postgres",
			Password: "password",
		})
		return dbAdapter, err
	}))

	// Session manager per tenant
	sessionManagerCache := make(map[string]*session.Manager)

	app.Use(func(c *fiber.Ctx) error {
		tenant := beaconauth_fiber.GetTenant(c)
		adapter := beaconauth_fiber.GetAdapter(c).(core.Adapter)

		// Get or create session manager for tenant
		sessionManager, exists := sessionManagerCache[tenant]
		if !exists {
			sessionConfig := &session.Config{
				CookieName:     "session_" + tenant,
				CookieSecure:   true,
				CookieHTTPOnly: true,
				CookieSameSite: "lax",
				CookieDomain:   ".soulcareuk.com",
				ExpiresIn:      7 * 24 * time.Hour,
				EnableDBStore:  true,
				Secret:         getTenantSecret(tenant),
				Issuer:         "soulcareuk-" + tenant,
			}

			var err error
			sessionManager, err = session.NewManager(sessionConfig, adapter)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Failed to initialize session manager",
				})
			}

			sessionManagerCache[tenant] = sessionManager
		}

		c.Locals("sessionManager", sessionManager)
		return c.Next()
	})

	// Session middleware using tenant-specific manager
	app.Use(func(c *fiber.Ctx) error {
		sessionManager := c.Locals("sessionManager").(*session.Manager)
		return beaconauth_fiber.SessionMiddleware(sessionManager)(c)
	})

	// Auth routes
	authRoutes := app.Group("/auth")
	authRoutes.Post("/signup", func(c *fiber.Ctx) error {
		adapter := beaconauth_fiber.GetAdapter(c).(core.Adapter)
		sessionManager := c.Locals("sessionManager").(*session.Manager)

		handler := beaconauth_fiber.NewHandler(adapter, sessionManager, &auth.Config{
			MinPasswordLength:   8,
			RequireVerification: true,
			AllowSignup:         true,
		})

		return handler.SignUp(c)
	})

	authRoutes.Post("/signin", func(c *fiber.Ctx) error {
		adapter := beaconauth_fiber.GetAdapter(c).(core.Adapter)
		sessionManager := c.Locals("sessionManager").(*session.Manager)

		handler := beaconauth_fiber.NewHandler(adapter, sessionManager, nil)
		return handler.SignIn(c)
	})

	// Protected API routes
	api := app.Group("/api")
	api.Use(beaconauth_fiber.RequireTenant())
	api.Use(func(c *fiber.Ctx) error {
		sessionManager := c.Locals("sessionManager").(*session.Manager)
		return beaconauth_fiber.RequireAuthJSON(sessionManager)(c)
	})

	api.Get("/profile", func(c *fiber.Ctx) error {
		user := beaconauth_fiber.GetUser(c)
		tenant := beaconauth_fiber.GetTenant(c)

		return c.JSON(fiber.Map{
			"tenant": tenant,
			"user":   user,
		})
	})

	api.Get("/users", func(c *fiber.Ctx) error {
		tenant := beaconauth_fiber.GetTenant(c)
		adapter := beaconauth_fiber.GetAdapter(c).(core.Adapter)

		// Query users from tenant-specific database
		users, err := adapter.FindMany(c.Context(), &core.Query{
			Model: "users",
		})
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to fetch users",
			})
		}

		return c.JSON(fiber.Map{
			"tenant": tenant,
			"users":  users,
		})
	})

	app.Listen(":3000")
}

// getTenantSecret retrieves or generates a secret for the tenant
func getTenantSecret(tenantID string) string {
	// In production, fetch from secure storage (vault, secret manager, etc.)
	return "tenant-secret-" + tenantID + "-at-least-32-bytes-long"
}

// Example_fiberCustomRoutes demonstrates custom route patterns with BeaconAuth
func Example_fiberCustomRoutes() {
	app := fiber.New()

	// ... setup adapters and session manager ...

	// Custom middleware that only allows verified users
	requireVerified := func(c *fiber.Ctx) error {
		user := beaconauth_fiber.GetUser(c)
		if user == nil || !user.EmailVerified {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error":   "email_not_verified",
				"message": "Please verify your email address",
			})
		}
		return c.Next()
	}

	// Admin-only routes
	requireAdmin := func(c *fiber.Ctx) error {
		user := beaconauth_fiber.GetUser(c)
		if user == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "unauthorized",
			})
		}

		// Check if user is admin (you'd implement this based on your schema)
		// For example, check a role field or query an admin table
		return c.Next()
	}

	admin := app.Group("/admin")
	admin.Use(requireVerified)
	admin.Use(requireAdmin)

	admin.Get("/dashboard", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "Admin dashboard",
		})
	})

	app.Listen(":3000")
}
