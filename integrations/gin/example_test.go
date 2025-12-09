package gin_test

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/marshallshelly/beacon-auth/adapters/memory"
	"github.com/marshallshelly/beacon-auth/auth"
	beaconauth_gin "github.com/marshallshelly/beacon-auth/integrations/gin"
	"github.com/marshallshelly/beacon-auth/session"
)

func Example_basicGinIntegration() {
	r := gin.New()

	dbAdapter := memory.New()
	sessionManager, _ := session.NewManager(&session.Config{
		CookieName: "session",
		Secret:     "secret-key-at-least-32-bytes-long",
		ExpiresIn:  24 * time.Hour,
	}, dbAdapter)

	r.Use(beaconauth_gin.SessionMiddleware(sessionManager))

	authHandler := beaconauth_gin.NewHandler(dbAdapter, sessionManager, &auth.Config{})
	authHandler.RegisterRoutes(r)

	protected := r.Group("/api")
	protected.Use(beaconauth_gin.RequireAuth(sessionManager))
	protected.GET("/protected", func(c *gin.Context) {
		user := beaconauth_gin.GetUser(c)
		c.JSON(200, gin.H{"user": user})
	})

	log.Fatal(r.Run(":8080"))
}
