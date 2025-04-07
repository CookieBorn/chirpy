package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"
)

func main() {
	apiC := apiConfig{}
	apiC.fileserverHits.Store(0)
	servMux := http.NewServeMux()
	servMux.Handle("/app/", apiC.middlewareMetricsInc(http.FileServer(http.Dir("."))))
	servMux.HandleFunc("GET /api/healthz", ReadinessHandeler)
	servMux.HandleFunc("GET /admin/metrics", apiC.metricHandle)
	servMux.HandleFunc("POST /admin/reset", apiC.metricReset)
	servMux.HandleFunc("POST /api/validate_chirp", postHandle)
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

type apiConfig struct {
	fileserverHits atomic.Int32
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) metricHandle(res http.ResponseWriter, req *http.Request) {
	req.Header.Set("Content-Type", "text/html")
	res.WriteHeader(200)
	write := []byte(fmt.Sprintf("<html><body><h1>Welcome, Chirpy Admin</h1><p>Chirpy has been visited %d times!</p></body></html>", cfg.fileserverHits.Load()))
	int, err := res.Write(write)
	if err != nil {
		fmt.Printf("%d: %v", int, err)
	}
}

func (cfg *apiConfig) metricReset(res http.ResponseWriter, req *http.Request) {
	cfg.fileserverHits.Add(-cfg.fileserverHits.Load())
	req.Header.Set("Content-Type", "text/plain")
	res.WriteHeader(200)
	write := []byte(fmt.Sprintf("Reset Successful hits: %v", cfg.fileserverHits.Load()))
	int, err := res.Write(write)
	if err != nil {
		fmt.Printf("%d: %v", int, err)
	}
}

func postHandle(res http.ResponseWriter, req *http.Request) {
	type parameters struct {
		Body string `json:"body"`
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
		errRet := returnErr{
			Err: "Chirp is too long",
		}
		res.WriteHeader(400)
		dat, err := json.Marshal(errRet)
		if err != nil {
			fmt.Printf("Marshal error: %v", err)
			return
		}
		res.Header().Set("Content-Type", "application/json")
		res.Write(dat)
		return
	}
	clean := stringCleaner(params.Body)
	valid := validRet{
		Valid:        true,
		Cleaned_body: clean,
	}
	tru, err := json.Marshal(valid)
	if err != nil {
		fmt.Printf("Error marshalling JSON: %s", err)
		res.WriteHeader(500)
		return
	}
	res.WriteHeader(200)
	res.Header().Set("Content-Type", "application/json")
	res.Write(tru)
}
