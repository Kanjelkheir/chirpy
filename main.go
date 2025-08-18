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
	polka_key      string
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
			Body string `json:"body"`
		}

		tokenSecret := os.Getenv("SECRET")
		token, err := auth.GetBearerToken(&r.Header)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		userID, err := auth.ValidateJWT(token, tokenSecret)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		body := bodyType{}

		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&body); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			errorResponse := struct {
				Error string `json:"error"`
			}{
				Error: "Invalid request body",
			}
			response, marshalErr := json.Marshal(errorResponse)
			if marshalErr != nil {
				log.Printf("error marshaling json: %s", marshalErr)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write(response)
			return
		}

		if len(body.Body) > 140 {
			w.WriteHeader(http.StatusBadRequest)
			w.Header().Set("Content-Type", "application/json")
			errorResponse := struct {
				Error string `json:"error"`
			}{
				Error: "Chirp is too long",
			}
			data, marshalErr := json.Marshal(errorResponse)
			if marshalErr != nil {
				w.WriteHeader(http.StatusInternalServerError)
				log.Printf("Error marshaling json: %s", marshalErr)
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
			UserID:    sql.NullString{String: userID.String(), Valid: true},
		}
		chirp, err := cfg.queries.CreateChirp(context.Background(), chirpParams)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			errorResponse := struct {
				Error string `json:"error"`
			}{
				Error: "Failed to create chirp",
			}

			response, marshalErr := json.Marshal(errorResponse)
			if marshalErr != nil {
				w.WriteHeader(http.StatusInternalServerError)
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
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Type", "application/json")
		w.Write(response)

	})

}

func (cfg *apiConfig) HandlerAddUser() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		type requestBody struct {
			Email              string `json:"email"`
			Password           string `json:"password"`
			Expires_in_seconds int    `json:"expires_in_seconds"`
		}

		var reqBody requestBody

		decoder := json.NewDecoder(r.Body)
		defer r.Body.Close()
		if err := decoder.Decode(&reqBody); err != nil {
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

		hash, err := auth.HashPassword(reqBody.Password)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		params := database.CreateUserParams{
			ID:        uuid.New().String(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Email:     reqBody.Email,
			Password:  hash,
		}

		user, err := cfg.queries.CreateUser(context.Background(), params)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		user_uuid, err := uuid.Parse(user.ID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		tokenSecret := os.Getenv("SECRET")
		token, err := auth.MakeJWT(user_uuid, tokenSecret, time.Duration(reqBody.Expires_in_seconds)*time.Second)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		type responseBody struct {
			Id            string `json:"id"`
			Created_at    string `json:"created_at"`
			Updated_at    string `json:"updated_at"`
			Email         string `json:"email"`
			Token         string `json:"token"`
			Is_chirpy_red bool   `json:"is_chirpy_red"`
		}

		resp := responseBody{
			Id:            user.ID,
			Created_at:    user.CreatedAt.String(),
			Updated_at:    user.CreatedAt.String(),
			Email:         user.Email,
			Token:         token,
			Is_chirpy_red: user.IsChirpyRed.Bool,
		}

		jsonResponse, err := json.Marshal(resp)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated) // Use 201 Created for successful resource creation
		w.Write(jsonResponse)
	})
}

func (cfg *apiConfig) HandlerLoginUser() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		type requestBody struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}

		var reqBody requestBody
		decoder := json.NewDecoder(r.Body)
		defer r.Body.Close()

		if err := decoder.Decode(&reqBody); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			jsonResponse, _ := json.Marshal(map[string]string{"error": "Invalid request body"})
			w.Header().Set("Content-Type", "application/json")
			w.Write(jsonResponse)
			return
		}

		user, err := cfg.queries.GetUserByEmail(context.Background(), reqBody.Email)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized) // 401 for bad credentials
			jsonResponse, _ := json.Marshal(map[string]string{"error": "Invalid credentials"})
			w.Header().Set("Content-Type", "application/json")
			w.Write(jsonResponse)
			return
		}

		err = auth.CheckPasswordHash(reqBody.Password, user.Password)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized) // 401 for bad credentials
			jsonResponse, _ := json.Marshal(map[string]string{"error": "Invalid credentials"})
			w.Header().Set("Content-Type", "application/json")
			w.Write(jsonResponse)
			return
		}

		userUUID, err := uuid.Parse(user.ID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Printf("Error parsing user UUID: %v", err)
			return
		}

		tokenSecret := os.Getenv("SECRET")
		if tokenSecret == "" {
			w.WriteHeader(http.StatusInternalServerError)
			log.Println("JWT secret not set in environment")
			return
		}
		// Use the expires_in_seconds from the request, default to 1 hour if not provided or invalid
		expiresIn := time.Hour

		token, err := auth.MakeJWT(userUUID, tokenSecret, expiresIn)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Printf("Error making JWT: %v", err)
			return
		}

		refresh_token, err := auth.MakeRefreshTokens()
		if err != nil {
			error := struct {
				Error string `json:"error"`
			}{
				Error: "Failed to create refresh token",
			}

			errorResponse, err := json.Marshal(error)
			if err != nil {
				w.WriteHeader(500)
				return
			}

			w.WriteHeader(http.StatusInternalServerError)

			w.Header().Set("Content-Type", "application/json")

			w.Write(errorResponse)
		}

		type responseBody struct {
			Id            string `json:"id"`
			Email         string `json:"email"`
			Token         string `json:"token"`
			Refresh_token string `json:"refresh_token"`
			Is_chirpy_red bool   `json:"is_chirpy_red"`
		}

		resp := responseBody{
			Id:            user.ID,
			Email:         user.Email,
			Token:         token,
			Refresh_token: refresh_token,
			Is_chirpy_red: user.IsChirpyRed.Bool,
		}

		jsonResponse, err := json.Marshal(resp)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Printf("Error marshaling login response: %v", err)
			return
		}

		refresh_token_params := database.CreateRefreshTokenParams{
			Token:     resp.Refresh_token,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			UserID:    sql.NullString{String: resp.Id, Valid: true},
			ExpiresAt: sql.NullTime{Time: time.Now().Add(60 * 24 * time.Hour), Valid: true},
		}
		_, err = cfg.queries.CreateRefreshToken(context.Background(), refresh_token_params)
		if err != nil {
			error := struct {
				Error string `json:"error"`
			}{
				Error: "Failed to create refresh token",
			}

			responseError, err := json.Marshal(error)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusInternalServerError)
			w.Header().Set("Content-Type", "application/json")
			w.Write(responseError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK) // 200 OK for successful login
		w.Write(jsonResponse)
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

func (cfg *apiConfig) HandlerRefresh() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		godotenv.Load()
		token, err := auth.GetBearerToken(&r.Header)
		if err != nil {
			error := struct {
				Error string `json:"error"`
			}{
				Error: "Invalid authorization header",
			}

			errorResponse, err := json.Marshal(error)
			if err != nil {
				w.WriteHeader(500)
				return
			}

			w.WriteHeader(200)
			w.Header().Set("Content-Type", "application/json")
			w.Write(errorResponse)
			return
		}

		user_refresh_token, err := cfg.queries.GetRefreshToken(context.Background(), token)
		if err != nil {
			error := struct {
				Error string `json:"error"`
			}{
				Error: "Invalid refresh token",
			}

			errorResponse, err := json.Marshal(error)
			if err != nil {
				w.WriteHeader(500)
				return
			}

			w.WriteHeader(401)
			w.Header().Set("Content-Type", "application/json")
			w.Write(errorResponse)
			return
		}

		token_secret := os.Getenv("SECRET")

		user_uuid, err := uuid.Parse(user_refresh_token.UserID.String)
		if err != nil {
			w.WriteHeader(500)
			return
		}

		jwt, err := auth.MakeJWT(user_uuid, token_secret, time.Hour)
		if err != nil {
			error := struct {
				Error string `json:"error"`
			}{
				Error: "Failed to create token",
			}

			responseError, err := json.Marshal(error)
			if err != nil {
				w.WriteHeader(500)
				return
			}

			w.WriteHeader(500)
			w.Header().Set("Content-Type", "application/json")
			w.Write(responseError)
		}

		tokenStructure := struct {
			Token string `json:"token"`
		}{
			Token: jwt,
		}

		response, err := json.Marshal(tokenStructure)
		if err != nil {
			w.WriteHeader(500)
			return
		}

		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json")
		w.Write(response)
	})
}

func (cfg *apiConfig) HandlerRevokeRefreshToken() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		refresh_token, err := auth.GetBearerToken(&r.Header)
		if err != nil {
			error := struct {
				Error string `json:"error"`
			}{
				Error: "Invalid authorization header",
			}

			response, err := json.Marshal(error)
			if err != nil {
				w.WriteHeader(500)
				return
			}

			w.WriteHeader(400)
			w.Header().Set("Content-Type", "application/json")
			w.Write(response)
			return
		}

		params := database.RevokeRefreshTokenParams{
			RevokedAt: sql.NullTime{Time: time.Now(), Valid: true},
			UpdatedAt: time.Now(),
			Token:     refresh_token,
		}
		err = cfg.queries.RevokeRefreshToken(context.Background(), params)
		if err != nil {
			error := struct {
				Error string `json:"error"`
			}{
				Error: "Failed to revoke Refresh token",
			}

			response, err := json.Marshal(error)
			if err != nil {
				w.WriteHeader(500)
				return
			}

			w.WriteHeader(500)
			w.Header().Set("Content-Type", "application/json")
			w.Write(response)
		}

		w.WriteHeader(204)
	})
}

func (cfg *apiConfig) HandlerGetChirps() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		queryParams := r.URL.Query()
		var chirps []database.Chirp
		var err error
		author_id := queryParams.Get("author_id")
		if author_id != "" {
			chirps, err = cfg.queries.GetChirpByAuthor(context.Background(), sql.NullString{String: author_id, Valid: true})
		} else {
			chirps, err = cfg.queries.GetChirps(context.Background())
		}
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

func (cfg *apiConfig) HandlerUpdateUser() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		type HeaderParams struct {
			Password string `json:"password"`
			Email    string `json:"email"`
		}

		var creds HeaderParams

		decoder := json.NewDecoder(r.Body)

		if err := decoder.Decode(&creds); err != nil {
			error := struct {
				Error string `json:"error"`
			}{
				Error: "Invalid request body",
			}

			errorResponse, err := json.Marshal(error)
			if err != nil {
				w.WriteHeader(500)
				return
			}

			w.WriteHeader(401)
			w.Header().Set("Content-Type", "application/json")
			w.Write(errorResponse)
			return
		}

		refresh_token, err := auth.GetBearerToken(&r.Header)
		if err != nil {
			error := struct {
				Error string `json:"error"`
			}{
				Error: "Invalid authentication header",
			}

			errorResponse, err := json.Marshal(error)
			if err != nil {
				w.WriteHeader(500)
				return
			}

			w.WriteHeader(400)
			w.Header().Set("Content-Type", "application/json")
			w.Write(errorResponse)
			return
		}

		token, err := cfg.queries.GetRefreshToken(context.Background(), refresh_token)
		if err != nil {
			error := struct {
				Error string `json:"error"`
			}{
				Error: "Invalid refresh token",
			}

			errorResponse, err := json.Marshal(error)
			if err != nil {
				w.WriteHeader(500)
				return
			}

			w.WriteHeader(400)
			w.Header().Set("Content-Type", "application/json")
			w.Write(errorResponse)
			return
		}

		hashed_password, err := auth.HashPassword(creds.Password)
		if err != nil {
			error := struct {
				Error string `json:"error"`
			}{
				Error: "Failed to update user",
			}

			errorResponse, err := json.Marshal(error)
			if err != nil {
				w.WriteHeader(500)
				return
			}

			w.WriteHeader(500)
			w.Header().Set("Content-Type", "application/json")
			w.Write(errorResponse)
			return
		}
		updateUserParams := database.UpdateUserParams{
			Token:     token.Token,
			Email:     creds.Email,
			Password:  hashed_password,
			UpdatedAt: time.Now(),
		}
		user, err := cfg.queries.UpdateUser(context.Background(), updateUserParams)
		if err != nil {
			error := struct {
				Error string `json:"error"`
			}{
				Error: "Failed to update user",
			}

			errorResponse, err := json.Marshal(error)
			if err != nil {
				w.WriteHeader(500)
				return
			}

			w.WriteHeader(500)
			w.Header().Set("Content-Type", "application/json")
			w.Write(errorResponse)
			return
		}

		type userData struct {
			Id            string `json:"id"`
			Email         string `json:"email"`
			Created_at    string `json:"created_at"`
			Updated_at    string `json:"updated_at"`
			Is_chirpy_red bool   `json:"is_chirpy_red"`
		}

		responseData := userData{
			Id:            user.ID,
			Email:         user.Email,
			Created_at:    user.CreatedAt.String(),
			Updated_at:    user.UpdatedAt.String(),
			Is_chirpy_red: user.IsChirpyRed.Bool,
		}

		response, err := json.Marshal(responseData)
		if err != nil {
			w.WriteHeader(500)
			return
		}

		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json")
		w.Write(response)
	})
}

func (cfg *apiConfig) HandlerDeleteChirp() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		chirpId := r.PathValue("chirpID")
		token, err := auth.GetBearerToken(&r.Header)
		if err != nil {
			error := struct {
				Error string `json:"error"`
			}{
				Error: "Invalid authorization header",
			}

			errorResponse, err := json.Marshal(error)
			if err != nil {
				w.WriteHeader(500)
				return
			}

			w.WriteHeader(400)
			w.Header().Set("Content-Type", "application/json")
			w.Write(errorResponse)
			return
		}

		godotenv.Load()
		tokenSecret := os.Getenv("SECRET")
		user_uuid, err := auth.ValidateJWT(token, tokenSecret)
		if err != nil {
			error := struct {
				Error string `json:"error"`
			}{
				Error: "Invalid jwt token",
			}

			errorResponse, err := json.Marshal(error)
			if err != nil {
				w.WriteHeader(500)
				return
			}

			w.WriteHeader(400)
			w.Header().Set("Content-Type", "application/json")
			w.Write(errorResponse)
			return
		}

		chirp, err := cfg.queries.GetChirp(context.Background(), chirpId)
		if err != nil {
			error := struct {
				Error string `json:"error"`
			}{
				Error: "Chirp not found",
			}

			errorResponse, err := json.Marshal(error)
			if err != nil {
				w.WriteHeader(500)
				return
			}

			w.WriteHeader(404)
			w.Header().Set("Content-Type", "application/json")
			w.Write(errorResponse)
			return
		}

		if chirp.UserID.String != user_uuid.String() {
			error := struct {
				Error string `json:"error"`
			}{
				Error: "Not allowed",
			}

			errorResponse, err := json.Marshal(error)
			if err != nil {
				w.WriteHeader(500)
				return
			}

			w.WriteHeader(403)
			w.Header().Set("Content-Type", "application/json")
			w.Write(errorResponse)
			return
		}

		err = cfg.queries.DeleteChirp(context.Background(), sql.NullString{String: user_uuid.String(), Valid: true})
		if err != nil {
			error := struct {
				Error string `json:"error"`
			}{
				Error: "Failed to delete chirp",
			}

			errorResponse, err := json.Marshal(error)
			if err != nil {
				w.WriteHeader(500)
				return
			}

			w.WriteHeader(500)
			w.Header().Set("Content-Type", "application/json")
			w.Write(errorResponse)
			return
		}

		w.WriteHeader(204)
	})
}

func (cfg *apiConfig) HandlerPolkaWebhook() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		godotenv.Load()
		valid_api_key := os.Getenv("POLKA_KEY")
		api_key, err := auth.GetAPIKey(&r.Header)
		if err != nil {
			error := struct {
				Error string `json:"error"`
			}{
				Error: fmt.Sprintf("Invalid api key: %v", err),
			}

			errorResponse, err := json.Marshal(error)
			if err != nil {
				w.WriteHeader(500)
				return
			}

			w.WriteHeader(400)
			w.Header().Set("Content-Type", "application/json")
			w.Write(errorResponse)
		}

		if api_key != valid_api_key {
			w.WriteHeader(401)
			return
		}

		type data struct {
			User_id string `json:"user_id"`
		}
		type requestStruct struct {
			Event string `json:"event"`
			Data  data   `json:"data"`
		}

		var body requestStruct
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&body); err != nil {
			error := struct {
				Error string `json:"error"`
			}{
				Error: "Invalid request body",
			}

			errorResponse, err := json.Marshal(error)
			if err != nil {
				w.WriteHeader(500)
				return
			}

			w.WriteHeader(400)
			w.Header().Set("Content-Type", "application/json")
			w.Write(errorResponse)
			return
		}

		if body.Event != "user.upgraded" {
			w.WriteHeader(204)
			return
		}

		err = cfg.queries.UpgradeUser(context.Background(), body.Data.User_id)
		if err != nil {
			error := struct {
				Error string `json:"error"`
			}{
				Error: "Error upgrading user",
			}

			errorResponse, err := json.Marshal(error)
			if err != nil {
				w.WriteHeader(500)
				return
			}

			w.WriteHeader(400)
			w.Header().Set("Content-Type", "application/json")
			w.Write(errorResponse)
			return
		}

		w.WriteHeader(204)
	})
}

func main() {
	godotenv.Load()

	db_url := os.Getenv("DB_URL")
	polka_key := os.Getenv("POLKA_KEY")

	db, err := sql.Open("postgres", db_url)
	if err != nil {
		fmt.Printf("Error connecting to DB: %s", err)
		return
	}

	dbQueries := database.New(db)

	mux := http.NewServeMux()

	// Serve static files
	fs := http.FileServer(http.Dir("./static"))
	config := apiConfig{
		fileserverHits: atomic.Int32{},
		queries:        dbQueries,
		polka_key:      polka_key,
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
	mux.Handle("POST /api/users", config.HandlerAddUser()) // Existing user creation endpoint
	mux.Handle("POST /api/login", config.HandlerLoginUser())
	mux.Handle("POST /api/chirps", config.HandlerChirps())
	mux.Handle("GET /api/chirps", config.HandlerGetChirps())
	mux.Handle("GET /api/chirps/{chirp_id}", config.HandlerChirpsFilter())
	mux.Handle("POST /api/refresh", config.HandlerRefresh())
	mux.Handle("POST /api/revoke", config.HandlerRevokeRefreshToken())
	mux.Handle("PUT /api/users", config.HandlerUpdateUser())
	mux.Handle("DELETE /api/chirps/{chirpID}", config.HandlerDeleteChirp())
	mux.Handle("POST /api/polka/webhooks", config.HandlerPolkaWebhook())

	config.metrics(mux)
	config.reset(mux)
	config.HandlerAddUser()

	log.Println("Server starting on :8080")
	log.Fatal(server.ListenAndServe())
}
