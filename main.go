package main

import (
	"time"
	"fmt"
	"log"
	"net/http"
	"html"
	"os"
	"context"
)

func main() {
	fmt.Println("Starting")

	server := &http.Server {
		Addr: ":8080",
		ReadTimeout: 10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	exitCode := 0

	shutdownChan := make(chan struct{})

	http.HandleFunc("GET /app/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "GetHandler says \"Hello, %q\"\n", html.EscapeString(r.URL.Path))
	})

	http.HandleFunc("POST /app/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "PostHandler says \"%q\"\n", html.EscapeString(r.URL.Path))
	})


	doShutdown := func (code int) {
		exitCode = code
		server.Shutdown(context.Background())
		log.Printf("Shutdown finished")
		shutdownChan <- struct{}{}
	}

	http.HandleFunc("/admin/restart", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Restart: %q", html.EscapeString(r.URL.Path))

		go doShutdown(2)
	})

	http.HandleFunc("/admin/kill", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Quitting: %q", html.EscapeString(r.URL.Path))

		go doShutdown(1)
	})

	err := server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
	<- shutdownChan
	log.Printf("Exiting with code %v", exitCode)
	os.Exit(exitCode)
}
