package mux_test

import (
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/marshallshelly/beacon-auth/adapters/memory"
	"github.com/marshallshelly/beacon-auth/auth"
	beaconauth_mux "github.com/marshallshelly/beacon-auth/integrations/mux"
	"github.com/marshallshelly/beacon-auth/session"
)

func Example_basicMuxIntegration() {
	r := mux.NewRouter()

	dbAdapter := memory.New()
	sessionManager, _ := session.NewManager(&session.Config{
		CookieName: "session",
		Secret:     "secret-key-at-least-32-bytes-long",
		ExpiresIn:  24 * time.Hour,
	}, dbAdapter)

	r.Use(beaconauth_mux.SessionMiddleware(sessionManager))

	authHandler := beaconauth_mux.NewHandler(dbAdapter, sessionManager, &auth.Config{})
	authHandler.RegisterRoutes(r)

	// Protected subrouter
	protected := r.PathPrefix("/api").Subrouter()
	protected.Use(beaconauth_mux.RequireAuth(sessionManager))
	protected.HandleFunc("/protected", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Protected"))
	})

	log.Fatal(http.ListenAndServe(":8080", r))
}
