package main

import (
	"context"
	"fmt"
	"github.com/tavis7/postalboard/internal/htmlgen"
	"html"
	"log"
	"net/http"
	"os"
	"time"
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

func main() {
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

	err := server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
	<-shutdownChan
	log.Printf("Exiting with code %v", exitCode)
	os.Exit(exitCode)
}
