package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

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
		DB:        dbQueries,
		JWTSecret: healpers.GetEnv("JWT_SECRET"),
		PolkaKey:  healpers.GetEnv("POLKA_KEY"),
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
	servMux.HandleFunc("POST /api/refresh", apiC.postRefres)
	servMux.HandleFunc("POST /api/revoke", apiC.postRevoke)
	servMux.HandleFunc("PUT /api/users", apiC.putUserUpdate)
	servMux.HandleFunc("DELETE /api/chirps/", apiC.deleteChirp)
	servMux.HandleFunc("POST /api/polka/webhooks", apiC.postPolkaWebhook)
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
	JWTSecret      string
	PolkaKey       string
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
	token, err := auth.GetBearerToken(req.Header)
	if err != nil {
		healpers.RespondWithError(res, 401, "Unauthorized")
		return
	}
	usr, err := auth.ValidateJWT(token, cfg.JWTSecret)
	if err != nil {
		healpers.RespondWithError(res, 401, "Unauthorized")
		return
	}
	if len([]rune(params.Body)) > 140 {
		healpers.RespondWithError(res, 400, "Chirpy is too long")
	}
	clean := healpers.StringCleaner(params.Body)
	chirpsParam := database.CreateChirpParams{
		Body:   clean,
		UserID: usr,
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
		Id:            usr.ID,
		Created_at:    usr.CreatedAt,
		Updated_at:    usr.UpdatedAt,
		Email:         usr.Email,
		Is_chirpy_red: usr.IsChirpyRed,
	}
	healpers.RespondWithJSON(res, 201, UserStruct)
}

func (cfg *ApiConfig) getChirpsHandle(res http.ResponseWriter, req *http.Request) {
	Aid := req.URL.Query().Get("author_id")
	chirps := []database.Chirp{}
	var err error
	if Aid != "" {
		ID, err := uuid.Parse(Aid)
		if err != nil {
			healpers.RespondWithError(res, 400, fmt.Sprintf("Parse error: %v", err))
		}
		chirps, err = cfg.DB.GetChirpsAllAuthor(req.Context(), ID)
		if err != nil {
			healpers.RespondWithError(res, 400, fmt.Sprintf("Get chirps failed: %v", err))
		}
	} else {
		chirps, err = cfg.DB.GetChirpsAll(req.Context())
		if err != nil {
			healpers.RespondWithError(res, 400, fmt.Sprintf("Get chirps failed: %v", err))
		}
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
	sortP := req.URL.Query().Get("sort")
	if sortP == "desc" {
		sort.Slice(jsonChirps, func(i int, j int) bool { return jsonChirps[i].Created_at.Compare(jsonChirps[j].Created_at) > 0 })
	}
	healpers.RespondWithJSON(res, 200, jsonChirps)
}

func (cfg *ApiConfig) getChirpHandle(res http.ResponseWriter, req *http.Request) {
	elements := strings.Split(req.RequestURI, "/")
	idP, err := uuid.Parse(elements[3])
	if err != nil {
		healpers.RespondWithError(res, 404, "User not found")
		return
	}
	chirp, err := cfg.DB.GetChirp(req.Context(), idP)
	if err != nil {
		healpers.RespondWithError(res, 404, "Get chirp error")
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
	expiersIn := 3600
	sec, err := time.ParseDuration(strconv.Itoa(expiersIn) + "s")
	if err != nil {
		healpers.RespondWithError(res, 400, "Parse Time Error")
		return
	}
	token, err := auth.MakeJWT(usr.ID, cfg.JWTSecret, sec)
	if err != nil {
		healpers.RespondWithError(res, 400, "Make JWT error")
		return
	}
	refToke, err := auth.MakeRefreshToken()
	if err != nil {
		healpers.RespondWithError(res, 400, "Make RefreshT error")
		return
	}
	refCretTok, err := healpers.CreateRefreshToken(usr.ID, refToke)
	if err != nil {
		healpers.RespondWithError(res, 400, "Make RefreshTDB error")
		return
	}
	_, err = cfg.DB.CreateRefreshToken(req.Context(), refCretTok)
	if err != nil {
		healpers.RespondWithError(res, 400, "Make RefreshTDB error")
		return
	}
	userJson := healpers.User{
		Id:            usr.ID,
		Created_at:    usr.CreatedAt,
		Updated_at:    usr.UpdatedAt,
		Email:         usr.Email,
		Token:         token,
		Refresh_token: refToke,
		Is_chirpy_red: usr.IsChirpyRed,
	}
	healpers.RespondWithJSON(res, 200, userJson)
}

func (cfg *ApiConfig) postRefres(res http.ResponseWriter, req *http.Request) {
	type params struct {
		Token string `json:"token"`
	}
	toke, err := auth.GetBearerToken(req.Header)
	if err != nil {
		healpers.RespondWithError(res, 401, "Unauthorized")
		return
	}
	usrID, err := cfg.DB.GetUserFromRefreshToken(req.Context(), toke)
	if err != nil {
		healpers.RespondWithError(res, 401, "Unauthorized")
		return
	}
	if usrID.RevokedAt.Valid {
		healpers.RespondWithError(res, 401, "Unauthorized")
		return
	}
	JWTToke, err := auth.MakeJWT(usrID.UserID, cfg.JWTSecret, time.Hour)
	if err != nil {
		healpers.RespondWithError(res, 401, "Unauthorized")
		return
	}
	parap := params{
		Token: JWTToke,
	}
	healpers.RespondWithJSON(res, 200, parap)
}

func (cfg *ApiConfig) postRevoke(res http.ResponseWriter, req *http.Request) {
	toke, err := auth.GetBearerToken(req.Header)
	if err != nil {
		healpers.RespondWithError(res, 400, "Header missing Token")
		return
	}
	usrID, err := cfg.DB.GetUserFromRefreshToken(req.Context(), toke)
	if err != nil {
		healpers.RespondWithError(res, 400, "Token not Valid")
		return
	}
	err = cfg.DB.RevokeRefreshToken(req.Context(), usrID.UserID)
	if err != nil {
		healpers.RespondWithError(res, 400, "Revoke Token Error")
		return
	}
	res.WriteHeader(204)
}

func (cfg *ApiConfig) putUserUpdate(res http.ResponseWriter, req *http.Request) {
	type parameters struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	decoder := json.NewDecoder(req.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		healpers.RespondWithError(res, 400, "Decoding Error")
		return
	}
	toke, err := auth.GetBearerToken(req.Header)
	if err != nil {
		healpers.RespondWithError(res, 401, "Unauthorized")
		return
	}
	usrID, err := auth.ValidateJWT(toke, cfg.JWTSecret)
	if err != nil {
		healpers.RespondWithError(res, 401, "Unauthorized")
		return
	}
	HashPass, err := auth.HashPassword(params.Password)
	if err != nil {
		healpers.RespondWithError(res, 401, "Unauthorized")
		return
	}
	UpParam := database.UpdateUserEmailPasswordParams{
		Email:    params.Email,
		Password: HashPass,
		ID:       usrID,
	}
	err = cfg.DB.UpdateUserEmailPassword(req.Context(), UpParam)
	if err != nil {
		healpers.RespondWithError(res, 401, "Unauthorized")
		return
	}
	type retStruct struct {
		Email string `json:"email"`
	}
	Ret := retStruct{Email: params.Email}
	healpers.RespondWithJSON(res, 200, Ret)
}

func (cfg *ApiConfig) deleteChirp(res http.ResponseWriter, req *http.Request) {
	toke, err := auth.GetBearerToken(req.Header)
	if err != nil {
		healpers.RespondWithError(res, 401, "Unauthorized")
		return
	}
	usrID, err := auth.ValidateJWT(toke, cfg.JWTSecret)
	if err != nil {
		healpers.RespondWithError(res, 401, "Unauthorized")
		return
	}
	elements := strings.Split(req.RequestURI, "/")
	idP, err := uuid.Parse(elements[3])
	if err != nil {
		healpers.RespondWithError(res, 404, "User not found")
		return
	}
	chirp, err := cfg.DB.GetChirp(req.Context(), idP)
	if err != nil {
		healpers.RespondWithError(res, 404, "Get chirp error")
		return
	}
	if chirp.UserID != usrID {
		healpers.RespondWithError(res, 403, "User not creator")
		return
	}
	err = cfg.DB.DeleteChirp(req.Context(), chirp.ID)
	if err != nil {
		healpers.RespondWithError(res, 404, "Delete chirp error")
		return
	}
	res.WriteHeader(204)
}

func (cfg *ApiConfig) postPolkaWebhook(res http.ResponseWriter, req *http.Request) {
	api, err := auth.GetAPIKey(req.Header)
	if err != nil {
		healpers.RespondWithError(res, 401, "API Error")
		return
	}
	if api != cfg.PolkaKey {
		healpers.RespondWithError(res, 401, "Unautherized")
		return
	}
	decoder := json.NewDecoder(req.Body)
	params := healpers.PolkaWebHook{}
	err = decoder.Decode(&params)
	if err != nil {
		healpers.RespondWithError(res, 400, "Decoding Error")
		return
	}
	if params.Event != "user.upgraded" {
		healpers.RespondWithError(res, 204, "Decoding Error")
		return
	}
	id, err := uuid.Parse(params.Data.UserID)
	if err != nil {
		healpers.RespondWithError(res, 400, "Parse Error")
		return
	}
	err = cfg.DB.SetUserToRed(req.Context(), id)
	if err != nil {
		healpers.RespondWithError(res, 404, "User Not Found")
		return
	}
	res.WriteHeader(204)
}
