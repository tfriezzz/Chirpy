package main

import (
	"fmt"
	"net/http"
)

func main() {
	serveMux := http.NewServeMux()
	s := &http.Server{
		Addr:    ":8080",
		Handler: serveMux,
	}
	serveMux.Handle("/", http.FileServer(http.Dir(".")))

	if err := s.ListenAndServe(); err != nil {
		fmt.Print(err)
	}
}
