package main

import (
	"fmt"
	"net/http"
	"time"
	"log"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/volatiletech/authboss"
	"github.com/joho/godotenv"
	_ "github.com/volatiletech/authboss/auth"
	_ "github.com/volatiletech/authboss/confirm"
	_ "github.com/volatiletech/authboss/lock"
	_ "github.com/volatiletech/authboss/recover"
	_ "github.com/volatiletech/authboss/register"
	_ "github.com/volatiletech/authboss/remember"
	"os"
)

var ab = authboss.New()

func main() {
	// Read the .env file and parse it into the local environment
	if err := godotenv.Load(); err != nil{
		log.Fatal("Error loading .env file")
	}else{
		log.Println("Successfully loaded .env file")
	}

	// Initialize the main router
	r := chi.NewRouter()

	// Initialize the middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// Basic get on the index
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello")
	})

	// Handle all 404
	r.NotFound(func(w http.ResponseWriter, r *http.Request){
		fmt.Fprintf(w, "This page was unable to be found")
	})
	port := ":8000"
	if value, ok := os.LookupEnv("PORT"); ok{
		port = ":" + value
	}
	log.Println("Running on port " + port)
	http.ListenAndServe(port, r)
}

/*func authbossSetup() {
	ab := authboss.New()

}*/
