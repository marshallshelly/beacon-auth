package chi_test

import (
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/marshallshelly/beacon-auth/adapters/memory"
	"github.com/marshallshelly/beacon-auth/auth"
	beaconauth_chi "github.com/marshallshelly/beacon-auth/integrations/chi"
	"github.com/marshallshelly/beacon-auth/session"
)

func Example_basicChiIntegration() {
	r := chi.NewRouter()

	dbAdapter := memory.New()
	sessionManager, _ := session.NewManager(&session.Config{
		CookieName: "session",
		Secret:     "secret-key-at-least-32-bytes-long",
		ExpiresIn:  24 * time.Hour,
	}, dbAdapter)

	// Middleware
	r.Use(beaconauth_chi.SessionMiddleware(sessionManager))

	// Handler
	authHandler := beaconauth_chi.NewHandler(dbAdapter, sessionManager, &auth.Config{})
	authHandler.RegisterRoutes(r)

	// Protected
	r.Group(func(r chi.Router) {
		r.Use(beaconauth_chi.RequireAuth(sessionManager))
		r.Get("/protected", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Protected"))
		})
	})

	log.Fatal(http.ListenAndServe(":8080", r))
}
