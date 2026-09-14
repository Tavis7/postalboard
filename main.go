package main

import (
	"context"
	"fmt"
	"html"
	"log"
	"net/http"
	"os"
	"time"
)

func debugHTMLNode() {
	testNode := makeHTMLNode("p", attribList{{"class", "something"}, {"id", "foobar"}})
	fmt.Printf("Testnode: %v, %v\n", testNode, testNode.attributes)
	testNode.appendChildren(&htmlNode{text: "hello "})
	testSpan := makeHTMLNode("span", nil)
	testSpan.appendChildren(&htmlNode{text: "world"})
	testNode.appendChildren(testSpan)
	fmt.Printf("%s\n", renderHTMLNode(testNode))
}

func main() {
	fmt.Println("Starting")

	debugHTMLNode()
	initializeBoards()
	createBoard("test-board")
	postMessage("test-board", "me", "test post please ignore")
	postMessage("test-board", "you", "no")
	/*
		testBoard := boards.children["test-board"]
		testBoard.posts = append(testBoard.posts, post{
			user: "me",
			text: "test post please ignore",
		})
		testBoard.posts = append(testBoard.posts, post{
			user: "you",
			text: "no",
		})
		boards.children["test-board"] = testBoard
	*/

	server := &http.Server{
		Addr:         ":8080",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	exitCode := 0

	shutdownChan := make(chan struct{})

	http.HandleFunc("GET /app/boards/{board...}", httpGetBoard)
	http.HandleFunc("GET /app/", getDebugger)
	http.HandleFunc("GET /{$}", getHome)

	http.HandleFunc("POST /app/", postDebugger)
	http.HandleFunc("POST /app/boards/{board...}", postBoardPost)

	doShutdown := func(code int) {
		exitCode = code
		server.Shutdown(context.Background())
		log.Printf("Shutdown finished")
		shutdownChan <- struct{}{}
	}

	http.HandleFunc("POST /admin/restart", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Restart: %q", html.EscapeString(r.URL.Path))

		go doShutdown(2)
	})

	http.HandleFunc("POST /admin/kill", func(w http.ResponseWriter, r *http.Request) {
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
