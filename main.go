package main

import (
	"fmt"
	"net/http"
	"sync/atomic"
)

func main() {
	apiC := apiConfig{}
	apiC.fileserverHits.Store(0)
	servMux := http.NewServeMux()
	servMux.Handle("/app/", apiC.middlewareMetricsInc(http.FileServer(http.Dir("."))))
	servMux.HandleFunc("/healthz", ReadinessHandeler)
	servMux.HandleFunc("/metrics", apiC.metricHandle)
	servMux.HandleFunc("/reset", apiC.metricReset)
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
	req.Header.Set("Content-Type", "text/plain")
	res.WriteHeader(200)
	write := []byte(fmt.Sprintf("Hits: %v", cfg.fileserverHits.Load()))
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
