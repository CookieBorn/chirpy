package main

import (
	"strings"
)

type returnErr struct {
	Err string `json:"error"`
}

type validRet struct {
	Valid        bool   `json:"valid"`
	Cleaned_body string `json:"cleaned_body"`
}

func stringCleaner(s string) string {
	slice := strings.Split(s, " ")
	for int, sli := range slice {
		if strings.ToLower(sli) == "kerfuffle" || strings.ToLower(sli) == "sharbert" || strings.ToLower(sli) == "fornax" {
			slice[int] = "****"
		}
	}
	string := strings.Join(slice, " ")
	return string
}
