package main

import (
	"log"
	"net/http"
)

func main() {
	mux := http.NewServeMux()

	// Serve static files
	fs := http.FileServer(http.Dir("./static"))
	mux.Handle("/", fs)

	// Serve assets with proper strip prefix
	assetFS := http.FileServer(http.Dir("."))
	mux.Handle("/assets/", assetFS)

	// Start the server
	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	log.Println("Server starting on :8080")
	log.Fatal(server.ListenAndServe())
}
