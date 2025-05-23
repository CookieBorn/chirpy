package healpers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/CookieBorn/chirpy/internal/database"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

type ReturnErr struct {
	Err string `json:"error"`
}

type ValidRet struct {
	Valid        bool   `json:"valid"`
	Cleaned_body string `json:"cleaned_body"`
}

type Chirp struct {
	Id         uuid.UUID `json:"id"`
	Created_at time.Time `json:"created_at"`
	Updated_at time.Time `json:"updated_at"`
	Body       string    `json:"body"`
	User_id    uuid.UUID `json:"user_id"`
}

type Chirps []struct {
	Id         uuid.UUID `json:"id"`
	Created_at time.Time `json:"created_at"`
	Updated_at time.Time `json:"updated_at"`
	Body       string    `json:"body"`
	User_id    uuid.UUID `json:"user_id"`
}

type User struct {
	Id            uuid.UUID `json:"id"`
	Created_at    time.Time `json:"created_at"`
	Updated_at    time.Time `json:"updated_at"`
	Email         string    `json:"email"`
	Token         string    `json:"token"`
	Refresh_token string    `json:"refresh_token"`
	Is_chirpy_red bool      `json:"is_chirpy_red"`
}

type PolkaWebHook struct {
	Event string `json:"event"`
	Data  struct {
		UserID string `json:"user_id"`
	} `json:"data"`
}

func StringCleaner(s string) string {
	slice := strings.Split(s, " ")
	for int, sli := range slice {
		if strings.ToLower(sli) == "kerfuffle" || strings.ToLower(sli) == "sharbert" || strings.ToLower(sli) == "fornax" {
			slice[int] = "****"
		}
	}
	string := strings.Join(slice, " ")
	return string
}

func GetEnv(Title string) string {
	godotenv.Load(".env")
	dbURL := os.Getenv(Title)
	return dbURL
}

func DatabaseConnection() *database.Queries {
	dbURL := GetEnv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		fmt.Printf("Open connection error: %v", err)
	}
	dbQueries := database.New(db)
	return dbQueries
}

func RespondWithError(w http.ResponseWriter, code int, msg string) {
	errRet := ReturnErr{
		Err: msg,
	}
	w.WriteHeader(code)
	dat, err := json.Marshal(errRet)
	if err != nil {
		fmt.Printf("Marshal error: %v", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(dat)
	return
}

func RespondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	tru, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(400)
		return
	}
	w.WriteHeader(code)
	w.Header().Set("Content-Type", "application/json")
	w.Write(tru)
}

func DecoderHealper(res http.ResponseWriter, req *http.Request, params any) (any, error) {
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&params)
	if err != nil {
		return params, err
	}
	return params, nil
}

func CreateRefreshToken(usrId uuid.UUID, token string) (database.CreateRefreshTokenParams, error) {
	emptyParam := database.CreateRefreshTokenParams{}
	expIn, err := time.ParseDuration("1440h")
	if err != nil {
		return emptyParam, err
	}
	CreateParam := database.CreateRefreshTokenParams{
		Token:     token,
		UserID:    usrId,
		ExpiresAt: time.Now().Add(expIn),
	}
	return CreateParam, err
}
