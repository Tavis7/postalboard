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

func generateDebugAsHTML(r *http.Request) (string, error) {
	w := &strings.Builder{}
	acceptedMimeTypes, err := parseAccept(r.Header.Get("Accept"))
	if err != nil {
		log.Printf("Error parsing mimetypes: %v", err)
		fmt.Fprintf(w, "<p>Couldn't parse mimetypes: %v</p>", err)
	}

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
		return "", fmt.Errorf("Accept header does not contain text/html")
	}
	return w.String(), nil
}

type post struct {
	user string
	text string
}

type board struct {
	children map[string]*board
	canPost  bool
	posts    []post
}

var boards map[string]*board

func MakeBoard(canPost bool) *board {
	result := board{
		children: make(map[string]*board),
	}
	result.canPost = canPost
	return &result
}

func main() {
	fmt.Println("Starting")

	boards = make(map[string]*board)
	boards["test-board"] = MakeBoard(true)
	testBoard := boards["test-board"]
	testBoard.posts = append(testBoard.posts, post{
		user: "me",
		text: "test post please ignore",
	})
	testBoard.posts = append(testBoard.posts, post{
		user: "you",
		text: "no",
	})
	boards["test-board"] = testBoard

	server := &http.Server{
		Addr:         ":8080",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	exitCode := 0

	shutdownChan := make(chan struct{})

	getDebugger := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "<html><head></head><body>\n")
		fmt.Fprintf(w, "<p>Get handler says \"Hello\"</p>\n")

		fmt.Fprintf(w, "<form method=\"post\">\n")
		fmt.Fprintf(w, "<div>\n")
		fmt.Fprintf(w, "<input type=\"text\" name=\"text_input\"/>\n")
		fmt.Fprintf(w, "</div>\n")
		fmt.Fprintf(w, "<div>\n")
		fmt.Fprintf(w, "<input type=\"text\" name=\"text_input\"/>\n")
		fmt.Fprintf(w, "</div>\n")
		fmt.Fprintf(w, "<div>\n")
		fmt.Fprintf(w, "<textarea name=\"text_area\">\n")
		fmt.Fprintf(w, "</textarea>\n")
		fmt.Fprintf(w, "</div>\n")
		fmt.Fprintf(w, "<input type=\"submit\" value=\"Submit\" />\n")
		fmt.Fprintf(w, "</form>\n")

		debugHTML, err := generateDebugAsHTML(r)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
		} else {
			fmt.Fprintf(w, "<div>")
			fmt.Fprintf(w, debugHTML)
			fmt.Fprintf(w, "</div>")
		}

		fmt.Fprintf(w, "</body></html>\n")
	}

	httpGetBoard := func(w http.ResponseWriter, r *http.Request) {
		boardString := strings.TrimSuffix(r.PathValue("board"), "/")
		currentBoard := &board{
			children: boards,
		}
		sb := &strings.Builder{}
		if len(boardString) > 0 {
			boardPath := strings.Split(boardString, "/")
			for _, s := range boardPath {
				fmt.Fprintf(sb, "<p>---> %v</p>\n", s)
				b, ok := currentBoard.children[s]
				if !ok {
					w.WriteHeader(http.StatusNotFound)
					fmt.Fprintf(w, "<html><head></head><body>\n")
					fmt.Fprintf(w, "<p>Not found</p>\n")
					fmt.Fprintf(w, boardString)
					fmt.Fprintf(w, sb.String())
					fmt.Fprintf(w, "</body></html>\n")
					return
				}
				currentBoard = b
			}
		}
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "<html><head></head><body>\n")
		fmt.Fprintf(w, sb.String())
		fmt.Fprintf(w, "<p>Board: %v</p>\n", boardString)
		fmt.Fprintf(w, "<p>Boards:</p>\n")
		for key, _ := range currentBoard.children {
			fmt.Fprintf(w, "<p><a href=%s>%s</a></p>\n", key, key)
		}
		fmt.Fprintf(w, "<p>Can post: %v</p>\n", currentBoard.canPost)

		if currentBoard.canPost || (len(currentBoard.posts) > 1) {
			fmt.Fprintf(w, "<p>Posts: </p>\n")
			for _, post := range currentBoard.posts {
				fmt.Fprintf(w, "<p>%v: %v</p>\n", post.user, post.text)
			}
			fmt.Fprintf(w, "<form method=\"post\">")
			fmt.Fprintf(w, "<textarea name=\"post\"></textarea>")
			fmt.Fprintf(w, "<div>")
			fmt.Fprintf(w, "<input type=\"submit\" value=\"post\" />")
			fmt.Fprintf(w, "</div>")
			fmt.Fprintf(w, "</form>")
		}
		fmt.Fprintf(w, "</body></html>\n")
	}

	postDebugger := func(w http.ResponseWriter, r *http.Request) {
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

	postBoardPost := func(w http.ResponseWriter, r *http.Request) {
		boardString := strings.TrimSuffix(r.PathValue("board"), "/")
		currentBoard := &board{
			children: boards,
		}
		sb := &strings.Builder{}
		if len(boardString) > 0 {
			boardPath := strings.Split(boardString, "/")
			for _, s := range boardPath {
				fmt.Fprintf(sb, "<p>---> %v</p>\n", s)
				b, ok := currentBoard.children[s]
				if !ok {
					w.WriteHeader(http.StatusNotFound)
					fmt.Fprintf(w, "<html><head></head><body>\n")
					fmt.Fprintf(w, "<p>Not found</p>\n")
					fmt.Fprintf(w, boardString)
					fmt.Fprintf(w, sb.String())
					fmt.Fprintf(w, "</body></html>\n")
					return
				}
				currentBoard = b
			}
		}
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "<html><head></head><body>\n")
		fmt.Fprintf(w, "<h1>Posted</h1>\n")

		fmt.Fprintf(w, "<a href=/app/boards/>Boards</a>\n")
		fmt.Fprintf(w, "<a href=%v>%v</a>\n",
			html.EscapeString(r.URL.Path), html.EscapeString(boardString))

		fmt.Fprintf(w, "<div>")
		fmt.Fprintf(w, "<form method=\"post\" action=\"/admin/restart\">")
		fmt.Fprintf(w, "<input type=\"submit\" value=\"Restart server\" /a>\n")
		fmt.Fprintf(w, "</form>")
		fmt.Fprintf(w, "</div>")
		err := r.ParseForm()
		if err != nil {
			log.Printf("Error parsing form: %v", err)
		}

		fmt.Fprintf(w, "form values:\n")
		for key, val := range r.PostForm {
			fmt.Fprintf(w, "    %v: '%v'\n", key, val)
		}

		postText, ok := r.PostForm["post"]
		if !ok || len(postText) != 1 {
			fmt.Fprintf(w, "<div>")
			fmt.Fprintf(w, "No post")
			fmt.Fprintf(w, "</div>")
		} else {
			currentBoard.posts = append(currentBoard.posts, post{
				user: "whoever",
				text: postText[0],
			})
			fmt.Fprintf(w, "<div>")
			fmt.Fprintf(w, "Posted '%v'", postText[0])
			fmt.Fprintf(w, "</div>")
		}

		debugHTML, err := generateDebugAsHTML(r)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
		} else {
			fmt.Fprintf(w, "<div>")
			fmt.Fprintf(w, debugHTML)
			fmt.Fprintf(w, "</div>")
		}

		fmt.Fprintf(w, "<div>")
		fmt.Fprintf(w, "<p>path: %q</p>\n", html.EscapeString(r.URL.Path))
		fmt.Fprintf(w, "</body></html>\n")
	}

	getHome := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "<html><head></head><body>\n")
		fmt.Fprintf(w, "<h1>Home</h1>\n")

		fmt.Fprintf(w, "<a href=/app/boards/>Boards</a>\n")
		fmt.Fprintf(w, "<form method=\"post\" action=\"/admin/restart\">")
		fmt.Fprintf(w, "<input type=\"submit\" value=\"Restart server\" /a>\n")
		fmt.Fprintf(w, "</form>")

		debugHTML, err := generateDebugAsHTML(r)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
		} else {
			fmt.Fprintf(w, "<div>")
			fmt.Fprintf(w, debugHTML)
			fmt.Fprintf(w, "</div>")
		}

		fmt.Fprintf(w, "</body></html>\n")
	}

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
