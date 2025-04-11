package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync/atomic"

	"github.com/CookieBorn/chirpy/internal/auth"
	"github.com/CookieBorn/chirpy/internal/database"
	healpers "github.com/CookieBorn/chirpy/internal/helpers"
	"github.com/google/uuid"
	"github.com/joho/godotenv"

	_ "github.com/lib/pq"
)

func main() {
	dbQueries := healpers.DatabaseConnection()
	apiC := ApiConfig{
		DB: dbQueries,
	}
	apiC.FileserverHits.Store(0)
	servMux := http.NewServeMux()
	servMux.Handle("/app/", apiC.middlewareMetricsInc(http.FileServer(http.Dir("."))))
	servMux.HandleFunc("GET /api/healthz", ReadinessHandeler)
	servMux.HandleFunc("GET /admin/metrics", apiC.metricHandle)
	servMux.HandleFunc("POST /admin/reset", apiC.metricReset)
	servMux.HandleFunc("POST /api/chirps", apiC.postHandle)
	servMux.HandleFunc("POST /api/users", apiC.createUserHandle)
	servMux.HandleFunc("GET /api/chirps", apiC.getChirpsHandle)
	servMux.HandleFunc("GET /api/chirps/", apiC.getChirpHandle)
	servMux.HandleFunc("POST /api/login", apiC.postLoginHandle)
	http.StripPrefix("app/", servMux)
	servStruct := http.Server{
		Addr:    ":8081",
		Handler: servMux,
	}
	err := servStruct.ListenAndServe()
	if err != nil {
		fmt.Printf("%v", err)
	}
}

func ReadinessHandeler(res http.ResponseWriter, req *http.Request) {
	req.Header.Set("Content-Type", "text/plain")
	res.WriteHeader(200)
	write := []byte("OK")
	int, err := res.Write(write)
	if err != nil {
		fmt.Printf("%d: %v", int, err)
	}
}

type ApiConfig struct {
	FileserverHits atomic.Int32
	DB             *database.Queries
}

func (cfg *ApiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.FileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *ApiConfig) metricHandle(res http.ResponseWriter, req *http.Request) {
	req.Header.Set("Content-Type", "text/html")
	res.WriteHeader(200)
	write := []byte(fmt.Sprintf("<html><body><h1>Welcome, Chirpy Admin</h1><p>Chirpy has been visited %d times!</p></body></html>", cfg.FileserverHits.Load()))
	int, err := res.Write(write)
	if err != nil {
		fmt.Printf("%d: %v", int, err)
	}
}

func (cfg *ApiConfig) metricReset(res http.ResponseWriter, req *http.Request) {
	godotenv.Load(".env")
	dev := os.Getenv("PLATFORM")
	if dev != "dev" {
		healpers.RespondWithError(res, 403, "Forbidden")
	}
	cfg.DB.Reset(req.Context())
	cfg.FileserverHits.Add(-cfg.FileserverHits.Load())
	req.Header.Set("Content-Type", "text/plain")
	res.WriteHeader(200)
	write := []byte(fmt.Sprintf("Reset Successful hits: %v\n Users deleted", cfg.FileserverHits.Load()))
	int, err := res.Write(write)
	if err != nil {
		fmt.Printf("%d: %v", int, err)
	}
}

func (cfg *ApiConfig) postHandle(res http.ResponseWriter, req *http.Request) {
	type parameters struct {
		Body    string    `json:"body"`
		User_id uuid.UUID `json:"user_id"`
	}
	decoder := json.NewDecoder(req.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		fmt.Printf("Decoding error: %v", err)
		res.WriteHeader(400)
		return
	}
	if len([]rune(params.Body)) > 140 {
		healpers.RespondWithError(res, 400, "Chirpy is too long")
	}
	clean := healpers.StringCleaner(params.Body)
	chirpsParam := database.CreateChirpParams{
		Body:   clean,
		UserID: params.User_id,
	}
	chirp, err := cfg.DB.CreateChirp(req.Context(), chirpsParam)
	if err != nil {
		fmt.Printf("Create Error: %v", err)
		return
	}
	jsonChirp := healpers.Chirp{
		Id:         chirp.ID,
		Created_at: chirp.CreatedAt,
		Updated_at: chirp.UpdatedAt,
		Body:       chirp.Body,
		User_id:    chirp.UserID,
	}
	healpers.RespondWithJSON(res, 201, jsonChirp)
}

func (cfg *ApiConfig) createUserHandle(res http.ResponseWriter, req *http.Request) {
	type parameters struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	decoder := json.NewDecoder(req.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		fmt.Printf("Decoding error: %v", err)
		res.WriteHeader(400)
		return
	}
	passw, err := auth.HashPassword(params.Password)
	if err != nil {
		fmt.Printf("Password hash error: %v", err)
		res.WriteHeader(400)
		return
	}
	userParam := database.CreateUserParams{
		Email:    params.Email,
		Password: passw,
	}
	usr, err := cfg.DB.CreateUser(req.Context(), userParam)
	UserStruct := healpers.User{
		Id:         usr.ID,
		Created_at: usr.CreatedAt,
		Updated_at: usr.UpdatedAt,
		Email:      usr.Email,
	}
	healpers.RespondWithJSON(res, 201, UserStruct)
}

func (cfg *ApiConfig) getChirpsHandle(res http.ResponseWriter, req *http.Request) {
	chirps, err := cfg.DB.GetChirpsAll(req.Context())
	if err != nil {
		healpers.RespondWithError(res, 400, fmt.Sprintf("Get chirps failed: %v", err))
	}
	jsonChirps := healpers.Chirps{}
	for _, chirp := range chirps {
		jsonChirp := healpers.Chirp{
			Id:         chirp.ID,
			Created_at: chirp.CreatedAt,
			Updated_at: chirp.UpdatedAt,
			Body:       chirp.Body,
			User_id:    chirp.UserID,
		}
		jsonChirps = append(jsonChirps, jsonChirp)
	}
	healpers.RespondWithJSON(res, 200, jsonChirps)
}

func (cfg *ApiConfig) getChirpHandle(res http.ResponseWriter, req *http.Request) {
	elements := strings.Split(req.RequestURI, "/")
	fmt.Printf("%v", elements[3])
	idP, err := uuid.Parse(elements[3])
	if err != nil {
		fmt.Printf("Parse error: %v\n", err)
		healpers.RespondWithError(res, 404, "User not found")
		return
	}
	chirp, err := cfg.DB.GetChirp(req.Context(), idP)
	if err != nil {
		fmt.Printf("Get chirp error: %v\n", err)
		healpers.RespondWithError(res, 404, "User not found")
		return
	}
	jsonChirp := healpers.Chirp{
		Id:         chirp.ID,
		Created_at: chirp.CreatedAt,
		Updated_at: chirp.UpdatedAt,
		Body:       chirp.Body,
		User_id:    chirp.UserID,
	}
	healpers.RespondWithJSON(res, 200, jsonChirp)
}

func (cfg *ApiConfig) postLoginHandle(res http.ResponseWriter, req *http.Request) {
	type parameters struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	decoder := json.NewDecoder(req.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		fmt.Printf("Decoding error: %v", err)
		healpers.RespondWithError(res, 400, "Decoding Error")
		return
	}
	usr, err := cfg.DB.GetUserEmail(req.Context(), params.Email)
	if err != nil {
		fmt.Printf("User error: %v", err)
		healpers.RespondWithError(res, 400, "User Does Not Exist")
		return
	}
	err = auth.CheckPasswordHash(usr.Password, params.Password)
	if err != nil {
		healpers.RespondWithError(res, 401, "Unauthorized")
		return
	}
	userJson := healpers.User{
		Id:         usr.ID,
		Created_at: usr.CreatedAt,
		Updated_at: usr.UpdatedAt,
		Email:      usr.Email,
	}
	healpers.RespondWithJSON(res, 200, userJson)
}
