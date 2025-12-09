package echo_test

import (
	"log"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/marshallshelly/beacon-auth/adapters/memory"
	"github.com/marshallshelly/beacon-auth/auth"
	beaconauth_echo "github.com/marshallshelly/beacon-auth/integrations/echo"
	"github.com/marshallshelly/beacon-auth/session"
)

func Example_basicEchoIntegration() {
	e := echo.New()

	dbAdapter := memory.New()
	sessionManager, _ := session.NewManager(&session.Config{
		CookieName: "session",
		Secret:     "secret-key-at-least-32-bytes-long",
		ExpiresIn:  24 * time.Hour,
	}, dbAdapter)

	e.Use(beaconauth_echo.SessionMiddleware(sessionManager))

	authHandler := beaconauth_echo.NewHandler(dbAdapter, sessionManager, &auth.Config{})
	authHandler.RegisterRoutes(e.Group("")) // Register on root group

	protected := e.Group("/api")
	protected.Use(beaconauth_echo.RequireAuth(sessionManager))
	protected.GET("/protected", func(c echo.Context) error {
		user := beaconauth_echo.GetUser(c)
		return c.JSON(http.StatusOK, map[string]interface{}{"user": user})
	})

	log.Fatal(e.Start(":8080"))
}
