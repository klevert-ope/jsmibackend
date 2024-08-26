package controllers

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

// RootHandler handles requests to the root path
func rootHandler(w http.ResponseWriter, r *http.Request) {
	// Set response status to 200 OK
	w.WriteHeader(http.StatusOK)

	// Write response body
	if _, err := w.Write([]byte("Welcome to the root route!")); err != nil {
		log.Fatalf("Error writing response: %v", err)
	}
}

// SetupRootRoute sets up routes for the application
func SetupRootRoute(router *mux.Router) {
	// Define routes here
	router.HandleFunc("/", rootHandler).Methods("GET")
}
