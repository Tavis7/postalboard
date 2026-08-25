package main

import (
	"strings"
	"mime"
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
		mimetypes := make([]struct{t string; params map[string]string}, 0, 8)
		acceptHeader := r.Header.Get("Accept")
		acceptHeaderList := strings.Split(acceptHeader, ",")
		for _, t := range acceptHeaderList {
			accepting, params, err := mime.ParseMediaType(t)
			if (err != nil) {
				fmt.Fprintf(w, "Error: %v\n", err)
			}
			mimetypes = append(mimetypes, struct{
				t string;
				params map[string]string;
			} {
				t : strings.ToLower(accepting),
				params : params,
			})
		}

		fmt.Fprintf(w, "Get handler says \"Hello\"\n")
		fmt.Fprintf(w, "path: %q\n", html.EscapeString(r.URL.Path))
		fmt.Fprintf(w, "raw query: %q\n", html.EscapeString(r.URL.RawQuery))
		fmt.Fprintf(w, "query:\n")
		for k, v := range r.URL.Query() {
			fmt.Fprintf(w, "    %v=%v\n", k, v)
		}
		fmt.Fprintf(w, "accept: %q\n", acceptHeader)
		fmt.Fprintf(w, "    %q\n", mimetypes)
		found := false
		for _, val := range mimetypes {
			if val.t == "text/html" {
				fmt.Fprintf(w, "contains text/html: %v\n", val.params)
				found = true
				break
			}
			if val.t == "text/*" {
				fmt.Fprintf(w, "contains text/*: %v\n", val.params)
				found = true
				break
			}
			if val.t == "*/*" {
				fmt.Fprintf(w, "contains */*: %v\n", val.params)
				found = true
			}
		}
		if !found {
			fmt.Fprintf(w, "does not contain text/html\n")
		}
	})

	http.HandleFunc("POST /app/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Post handler says \"Hello\"\n")
		fmt.Fprintf(w, "path: %q\n", html.EscapeString(r.URL.Path))
		fmt.Fprintf(w, "raw query: %q\n", html.EscapeString(r.URL.RawQuery))
		fmt.Fprintf(w, "query:\n")
		for k, v := range r.URL.Query() {
			fmt.Fprintf(w, "    %v: %v\n", k, len(v))
			for _, val := range v {
				fmt.Fprintf(w, "        %v\n", val)
			}
		}
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
