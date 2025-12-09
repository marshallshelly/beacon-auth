package http_test

import (
	"log"
	"net/http"
	"time"

	"github.com/marshallshelly/beacon-auth/adapters/memory"
	"github.com/marshallshelly/beacon-auth/auth"
	beaconauth_http "github.com/marshallshelly/beacon-auth/integrations/http"
	"github.com/marshallshelly/beacon-auth/session"
)

func Example_basicHTTPIntegration() {
	// Setup dependencies
	dbAdapter := memory.New()
	sessionConfig := &session.Config{
		CookieName: "app_session",
		Secret:     "secret-key-at-least-32-bytes-long",
		ExpiresIn:  24 * time.Hour,
	}
	sessionManager, _ := session.NewManager(sessionConfig, dbAdapter)

	// Create handler
	authHandler := beaconauth_http.NewHandler(dbAdapter, sessionManager, &auth.Config{
		AllowSignup: true,
	})

	// Setup mux
	mux := http.NewServeMux()

	// Apply middleware globally or per route
	// In standard net/http, we wrap the mux
	var handler http.Handler = mux
	handler = beaconauth_http.SessionMiddleware(sessionManager)(handler)

	// Routes
	mux.HandleFunc("/auth/signup", authHandler.SignUp)
	mux.HandleFunc("/auth/signin", authHandler.SignIn)
	mux.HandleFunc("/auth/signout", authHandler.SignOut)
	mux.HandleFunc("/auth/session", authHandler.GetSession)

	// Protected route
	protectedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Protected content"))
	})
	mux.Handle("/protected", beaconauth_http.RequireAuth(sessionManager)(protectedHandler))

	log.Fatal(http.ListenAndServe(":8080", handler))
}
