package main

import (
	"fmt"
	"log"
	"net/http"
	"html"
	"os"
)

func main() {
	fmt.Println("rebuilt")

	port := ":8080"

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "...Hello, %q\n", html.EscapeString(r.URL.Path))
	})

	http.HandleFunc("/admin/restart", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hard restart: %q", html.EscapeString(r.URL.Path))
		os.Exit(2)
	})

	http.HandleFunc("/admin/kill", func(w http.ResponseWriter, r *http.Request) {
		// @todo Clean shutdown
		os.Exit(1)
	})

	log.Fatal(http.ListenAndServe(port, nil))
}
