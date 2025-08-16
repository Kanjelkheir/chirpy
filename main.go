package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/kanjelkheir/chirpy/internal/auth"
	"github.com/kanjelkheir/chirpy/internal/database"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	queries        *database.Queries
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

func (cfg *apiConfig) HandlerChirps() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		type bodyType struct {
			Body    string `json:"body"`
			User_id string `json:"user_id"`
		}

		body := bodyType{}

		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&body); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			error := struct {
				Error string `json:"error"`
			}{
				Error: "Invalid request body",
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

		profanity := make([]string, 3)
		profanity[0] = "kerfuffle"
		profanity[1] = "sharbert"
		profanity[2] = "fornax"

		for _, prof := range profanity {
			body.Body = strings.ReplaceAll(body.Body, prof, strings.Repeat("*", len(prof)))
		}

		w.Header().Set("Content-Type", "application/json")
		chirpParams := database.CreateChirpParams{
			ID:        uuid.New().String(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Body:      body.Body,
			UserID:    sql.NullString{String: body.User_id, Valid: true},
		}
		chirp, err := cfg.queries.CreateChirp(context.Background(), chirpParams)
		if err != nil {
			w.WriteHeader(500)
			error := struct {
				Error string `json:"error"`
			}{
				Error: "Failed to create chirp",
			}

			response, err := json.Marshal(error)
			if err != nil {
				w.WriteHeader(500)
				return
			}

			w.Write(response)
			return
		}

		chirpData := struct {
			ID        string `json:"id"`
			CreatedAt string `json:"created_at"`
			UpdatedAt string `json:"updated_at"`
			Body      string `json:"body"`
			UserID    string `json:"user_id"`
		}{
			ID:        chirp.ID,
			CreatedAt: chirp.CreatedAt.String(),
			UpdatedAt: chirp.UpdatedAt.String(),
			Body:      chirp.Body,
			UserID:    chirp.UserID.String,
		}

		response, err := json.Marshal(chirpData)
		if err != nil {
			w.WriteHeader(500)
			return
		}

		w.WriteHeader(201)
		w.Header().Set("Content-Type", "application/json")
		w.Write(response)

	})

}

func (cfg *apiConfig) HandlerAddUser() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		type emailStruct struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}

		var email emailStruct

		decoder := json.NewDecoder(r.Body)
		defer r.Body.Close()
		if err := decoder.Decode(&email); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			errorResponse := struct {
				Error string `json:"error"`
			}{
				Error: "Invalid request body",
			}
			jsonResponse, marshalErr := json.Marshal(errorResponse)
			if marshalErr != nil {
				log.Printf("Error marshaling error response: %s", marshalErr)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write(jsonResponse)
			return
		}
		defer r.Body.Close()

		hash, err := auth.HashPassword(email.Password)
		if err != nil {
			w.WriteHeader(500)
			return
		}

		params := database.CreateUserParams{
			ID:        uuid.New().String(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Email:     email.Email,
			Password:  hash,
		}

		user, err := cfg.queries.CreateUser(context.Background(), params)
		if err != nil {
			w.WriteHeader(500)
			return
		}

		type DataStruct struct {
			Id         string `json:"id"`
			Created_at string `json:"created_at"`
			Updated_at string `json:"updated_at"`
			Email      string `json:"email"`
		}

		data := DataStruct{
			Id:         user.ID,
			Created_at: user.CreatedAt.String(),
			Updated_at: user.CreatedAt.String(),
			Email:      user.Email,
		}

		response, err := json.Marshal(data)
		if err != nil {
			w.WriteHeader(500)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write(response)
	})
}

func (cfg *apiConfig) reset(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/reset", func(w http.ResponseWriter, r *http.Request) {
		platform := os.Getenv("PLATFORM")

		if platform == "dev" {
			type emailStruct struct {
				Email string `json:"email"`
			}

			var email emailStruct

			decoder := json.NewDecoder(r.Body)
			if err := decoder.Decode(&email); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			err := cfg.queries.DeleteUser(context.Background(), email.Email)

			if err != nil {
				w.WriteHeader(500)
				return
			}

			w.WriteHeader(200)
			w.Write([]byte("OK"))
		} else {
			w.WriteHeader(http.StatusForbidden)
			return
		}

		cfg.fileserverHits.Swap(0)
		w.WriteHeader(200)
	})
}

func (cfg *apiConfig) HandlerGetChirps() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		chirps, err := cfg.queries.GetChirps(context.Background())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Header().Set("Content-Type", "application/json")
			error := struct {
				Error string `json:"error"`
			}{
				Error: "Failed to get chirps",
			}
			data, err := json.Marshal(error)
			if err != nil {
				w.WriteHeader(500)
				return
			}
			w.Write(data)
			return
		}

		type ResponseFormat struct {
			ID        string `json:"id"`
			CreatedAt string `json:"created_at"`
			UpdatedAt string `json:"updated_at"`
			Body      string `json:"body"`
			UserID    string `json:"user_id"`
		}

		response := make([]ResponseFormat, len(chirps))

		for index, chirp := range chirps {
			response[index] = ResponseFormat{
				ID:        chirp.ID,
				CreatedAt: chirp.CreatedAt.String(),
				UpdatedAt: chirp.UpdatedAt.String(),
				Body:      chirp.Body,
				UserID:    chirp.UserID.String,
			}
		}

		data, err := json.Marshal(response)
		if err != nil {
			w.WriteHeader(500)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write(data)
	})
}

func (cfg *apiConfig) HandlerChirpsFilter() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		chirp_id := r.PathValue("chirp_id")
		chirp, err := cfg.queries.GetChirp(context.Background(), chirp_id)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		response := struct {
			Id         string `json:"id"`
			Created_at string `json:"created_at"`
			Updated_at string `json:"updated_at"`
			Body       string `json:"body"`
			User_id    string `json:"user_id"`
		}{
			Id:         chirp.ID,
			Created_at: chirp.CreatedAt.String(),
			Updated_at: chirp.UpdatedAt.String(),
			Body:       chirp.Body,
			User_id:    chirp.UserID.String,
		}

		data, err := json.Marshal(response)
		if err != nil {
			w.WriteHeader(500)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write(data)
	})
}

func main() {
	godotenv.Load()

	db_url := os.Getenv("DB_URL")

	db, err := sql.Open("postgres", db_url)
	if err != nil {
		fmt.Println("Error connecting to DB: %s", err)
		return
	}

	dbQueries := database.New(db)

	mux := http.NewServeMux()

	// Serve static files
	fs := http.FileServer(http.Dir("./static"))
	config := apiConfig{
		fileserverHits: atomic.Int32{},
		queries:        dbQueries,
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

	mux.Handle("POST /api/validate_chirp", config.HandlerChirps())
	mux.Handle("POST /api/users", config.HandlerAddUser())
	mux.Handle("POST /api/chirps", config.HandlerChirps())
	mux.Handle("GET /api/chirps", config.HandlerGetChirps())
	mux.Handle("GET /api/chirps/{chirp_id}", config.HandlerChirpsFilter())

	config.metrics(mux)
	config.reset(mux)
	config.HandlerAddUser()

	log.Println("Server starting on :8080")
	log.Fatal(server.ListenAndServe())
}
