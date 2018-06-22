package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/volatiletech/authboss"
	_ "github.com/volatiletech/authboss/auth"
	_ "github.com/volatiletech/authboss/confirm"
	_ "github.com/volatiletech/authboss/lock"
	_ "github.com/volatiletech/authboss/recover"
	_ "github.com/volatiletech/authboss/register"
	_ "github.com/volatiletech/authboss/remember"
)

var ab = authboss.New()

func main() {
	// Initialize the main router
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello")
	})
	http.ListenAndServe(":8000", r)
}

func authbossSetup() {
	ab := authboss.New()

}
