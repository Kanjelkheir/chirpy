package main

import (
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) metrics(mux *http.ServeMux) {
	mux.HandleFunc("GET /admin/metrics", func(w http.ResponseWriter, r *http.Request) {
		hits := cfg.fileserverHits.Load()

		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(200)
		body := fmt.Sprintf(`

			<html>
			<body>
			<h1>Welcome, Chirpy Admin</h1>
			<p>Chirpy has been visited %d times!</p>
			</body>
			</html>
			`, hits)
		w.Write([]byte(body))
	})
}

func (cfg *apiConfig) reset(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/reset", func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Swap(0)
		w.WriteHeader(200)
	})
}

func main() {
	mux := http.NewServeMux()

	// Serve static files
	fs := http.FileServer(http.Dir("./static"))
	config := apiConfig{
		fileserverHits: atomic.Int32{},
	}
	mux.Handle("/app/", config.middlewareMetricsInc(http.StripPrefix("/app/", fs)))

	// Serve assets with proper strip prefix
	assetFS := http.FileServer(http.Dir("./assets"))
	mux.Handle("/assets/", http.StripPrefix("/assets/", assetFS))

	// Start the server
	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	mux.HandleFunc("GET /api/healthz", func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Add("Content-Type", "text/plain")
		writer.WriteHeader(200)
		writer.Write([]byte("OK"))
	})

	config.metrics(mux)
	config.reset(mux)

	log.Println("Server starting on :8080")
	log.Fatal(server.ListenAndServe())
}
