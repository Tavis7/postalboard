package main

import (
	"context"
	"fmt"
	"html"
	"log"
	"math"
	"mime"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

func parseAccept(acceptHeader string) ([]struct {
	mimetype string
	params   map[string]string
	q        float32
}, error) {
	accepted := make([]struct {
		mimetype string
		params   map[string]string
		q        float32
	}, 0, 8)
	acceptHeaderList := strings.Split(acceptHeader, ",")
	for _, t := range acceptHeaderList {
		accepting, params, err := mime.ParseMediaType(t)
		if err != nil {
			return accepted, err
		}
		qString, ok := params["q"]
		if !ok {
			qString = "1"
		}
		q, err := strconv.ParseFloat(qString, 32)
		if err != nil {
			q = 1
		}
		q = math.Max(0.0, math.Min(1.0, q))
		accepted = append(accepted, struct {
			mimetype string
			params   map[string]string
			q        float32
		}{
			mimetype: strings.ToLower(accepting),
			params:   params,
			q:        float32(q),
		})
	}
	return accepted, nil
}

func main() {
	fmt.Println("Starting")

	server := &http.Server{
		Addr:         ":8080",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	exitCode := 0

	shutdownChan := make(chan struct{})

	http.HandleFunc("GET /app/", func(w http.ResponseWriter, r *http.Request) {
		acceptedMimeTypes, err := parseAccept(r.Header.Get("Accept"))
		if err != nil {
			log.Printf("Error parsing mimetypes: %v", err)
		}
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "<html><head></head><body>\n")
		fmt.Fprintf(w, "<p>Get handler says \"Hello\"</p>\n")

		fmt.Fprintf(w, "<p>protocol: %q</p>\n", html.EscapeString(r.Proto))
		fmt.Fprintf(w, "<p>headers:</p>")
		fmt.Fprintf(w, "<ul>")
		for header, val := range r.Header {
			fmt.Fprintf(w, "<li><pre style=\"display:inline\">%v: %v</pre></li>\n",
				html.EscapeString(fmt.Sprintf("%v", header)),
				html.EscapeString(fmt.Sprintf("%v", val)))
		}
		fmt.Fprintf(w, "</ul>")
		fmt.Fprintf(w, "<p>path: %q</p>\n", html.EscapeString(r.URL.Path))
		fmt.Fprintf(w, "<p>raw query: <pre style=\"display:inline\">%q</pre></p>\n",
			html.EscapeString(r.URL.RawQuery))
		fmt.Fprintf(w, "<p>query:</p>\n")
		fmt.Fprintf(w, "<ul>\n")
		for k, v := range r.URL.Query() {
			fmt.Fprintf(w, "<li>\n")
			fmt.Fprintf(w, "<pre style=\"display:inline\">%v=%v</pre>",
				html.EscapeString(fmt.Sprintf("%v", k)),
				html.EscapeString(fmt.Sprintf("%v", v)))
			fmt.Fprintf(w, "</li>\n")
		}
		fmt.Fprintf(w, "</ul>\n")
		fmt.Fprintf(w, "<p>accepted mimetypes:</p>\n")
		fmt.Fprintf(w, "<ul>")
		found := -1
		for i, val := range acceptedMimeTypes {
			fmt.Fprintf(w, "<li>%v</li>\n", html.EscapeString(fmt.Sprintf("%v", val)))
			if found == -1 {
				if val.mimetype == "text/html" ||
					val.mimetype == "text/*" ||
					val.mimetype == "*/*" {
					found = i
				}
			}
		}
		fmt.Fprintf(w, "</ul>")
		fmt.Fprintf(w, "<p>index: %v</p>\n",
			html.EscapeString(fmt.Sprintf("%v", found)))
		if found != -1 {
			fmt.Fprintf(w, "<p>Found: %v</p>\n",
				html.EscapeString(fmt.Sprintf("%v", acceptedMimeTypes[found])))
		} else {
			fmt.Fprintf(w, "does not contain text/html\n")
			// 406 Not Acceptable
		}
		fmt.Fprintf(w, "</body></html>\n")
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

	doShutdown := func(code int) {
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
	<-shutdownChan
	log.Printf("Exiting with code %v", exitCode)
	os.Exit(exitCode)
}
