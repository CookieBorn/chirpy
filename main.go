package main

import (
	"fmt"
	"net/http"
)

func main() {
	servMux := http.NewServeMux()
	servMux.Handle("/", http.FileServer(http.Dir(".")))
	servStruct := http.Server{
		Addr:    ":8081",
		Handler: servMux,
	}
	err := servStruct.ListenAndServe()
	if err != nil {
		fmt.Printf("%v", err)
	}
}
