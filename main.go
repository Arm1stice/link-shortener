package main

import (
	"net/http"

	"github.com/gorilla/mux"
)

func main() {
	// Initialize the main router
	r := mux.NewRouter()

	http.ListenAndServe(":8000", r)
}
