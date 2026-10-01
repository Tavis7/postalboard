package main

import (
	"context"
	"database/sql"
	"fmt"
	"html"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"

	_ "github.com/lib/pq"

	"github.com/tavis7/postalboard/internal/database"
	"github.com/tavis7/postalboard/internal/htmlgen"
)

var debugRedirect bool

func testHTMLGenerator() {
	log.Printf("testHMTLGenerator")
	testNode := htmlgen.MakeNode("p", htmlgen.AttribList{{"class", "something"}, {"id", "foobar"}})
	testNode.AppendChildren(htmlgen.MakeTextNode("hello "))
	testSpan := htmlgen.MakeLeafNode("span")
	testSpan.AppendChildren(htmlgen.MakeTextNode("world"))
	testNode.AppendChildren(testSpan)
	log.Printf("testNode: %v\n", testNode)
	testRoot := htmlgen.MakeNode("html", nil,
		htmlgen.MakeNode("body", nil, testNode))
	log.Printf("testRoot: %v\n", testRoot)
	rendered, err := htmlgen.Render(*testRoot)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
	fmt.Printf("%s\n", rendered)
}

func postDebugger(w http.ResponseWriter, r *http.Request) {
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

	err := r.ParseForm()
	if err != nil {
		log.Printf("Error parsing form: %v", err)
	}

	fmt.Fprintf(w, "form values:\n")
	for key, val := range r.PostForm {
		fmt.Fprintf(w, "    %v: '%v'\n", key, val)
	}
}

func main() {
	godotenv.Load()

	db_url := os.Getenv("DB_URL")
	fmt.Printf("DB_URL: '%s'\n", db_url)

	db, err := sql.Open("postgres", db_url)
	if err != nil {
		log.Printf("Error opening database: %v", err)
	}

	fmt.Printf("db: %v\n", db)

	dbQueries := database.New(db)

	fmt.Printf("db queries: %v\n", dbQueries)

	debugRedirect = true
	fmt.Println("Starting")

	testHTMLGenerator()
	initializeBoards()
	createBoard("test-board")
	postMessage("test-board", "me", "test post please ignore")
	postMessage("test-board", "you", "no")

	server := &http.Server{
		Addr:         ":8080",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	exitCode := 0

	shutdownChan := make(chan struct{})

	http.HandleFunc("GET /app/boards/{board...}", httpGetBoard)
	http.HandleFunc("GET /app/login", httpLoginPage)
	http.HandleFunc("GET /debug", getDebugger)
	http.HandleFunc("GET /{$}", getHome)

	http.HandleFunc("POST /app", postDebugger)
	http.HandleFunc("POST /app/login", postLogin)
	http.HandleFunc("POST /app/logout", postLogout)
	http.HandleFunc("POST /app/boards/{board...}", postBoardPost)

	doShutdown := func(code int) {
		exitCode = code
		server.Shutdown(context.Background())
		log.Printf("Shutdown finished")
		shutdownChan <- struct{}{}
	}

	http.HandleFunc("POST /admin/restart", func(w http.ResponseWriter, r *http.Request) {
		// @todo html
		fmt.Fprintf(w, "Restart: %q", html.EscapeString(r.URL.Path))

		go doShutdown(2)
	})

	http.HandleFunc("POST /admin/kill", func(w http.ResponseWriter, r *http.Request) {
		// @todo html
		fmt.Fprintf(w, "Quitting: %q", html.EscapeString(r.URL.Path))

		go doShutdown(1)
	})

	err = server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
	<-shutdownChan
	log.Printf("Exiting with code %v", exitCode)
	os.Exit(exitCode)
}
