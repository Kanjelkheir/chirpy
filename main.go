package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
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

func Handler_validate_chirp() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		type bodyType struct {
			Body string `json:"body"`
		}

		body := bodyType{}

		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&body); err != nil {
			w.WriteHeader(500)
			error := struct {
				Error string `json:"error"`
			}{
				Error: "Something went wrong",
			}
			response, err := json.Marshal(error)
			if err != nil {
				log.Printf("error marshaling json: %s", err)
				w.WriteHeader(500)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write(response)
		}

		if len(body.Body) > 140 {
			w.WriteHeader(400)
			w.Header().Set("Content-Type", "application/json")
			error := struct {
				Error string `json:"error"`
			}{
				Error: "Chirp is too long",
			}
			data, err := json.Marshal(error)
			if err != nil {
				w.WriteHeader(500)
				log.Printf("Error marshaling json: %s", err)
				return
			}
			w.Write(data)
			return
		}

		valid := struct {
			Valid bool `json:"valid"`
		}{
			Valid: true,
		}
		profanity := make([]string, 3)
		profanity[0] = "kerfuffle"
		profanity[1] = "sharbert"
		profanity[2] = "fornax"

		if strings.Contains(strings.ToLower(body.Body), profanity[0]) || strings.Contains(strings.ToLower(body.Body), profanity[1]) || strings.Contains(strings.ToLower(body.Body), profanity[2]) {
			for _, prof := range profanity {
				body.Body = strings.ReplaceAll(body.Body, prof, strings.Repeat("*", len(prof)))
			}

			w.WriteHeader(200)
			w.Header().Set("Content-Type", "application/json")
			jsonData := struct {
				Cleaned_body string `json:"cleaned_body"`
			}{
				Cleaned_body: body.Body,
			}

			data, err := json.Marshal(jsonData)
			if err != nil {
				w.WriteHeader(500)
				return
			}

			w.Write(data)
			return
		} else {

			data, err := json.Marshal(valid)
			if err != nil {
				w.WriteHeader(500)
				log.Printf("Error marshaling json: %s", err)
				return
			}

			w.Header().Set("Content-Type", "application/json")

			w.Write(data)
		}

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
	mux.Handle("GET /app/", config.middlewareMetricsInc(http.StripPrefix("/app/", fs)))

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

	mux.Handle("POST /api/validate_chirp", Handler_validate_chirp())

	config.metrics(mux)
	config.reset(mux)

	log.Println("Server starting on :8080")
	log.Fatal(server.ListenAndServe())
}
