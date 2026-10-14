package main

import (
	"context"
	"database/sql"
	"fmt"
	"html"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"

	_ "github.com/lib/pq"

	"github.com/tavis7/postalboard/internal/config"
	"github.com/tavis7/postalboard/internal/database"
)

var debugRedirect bool

var cfg config.Config

func main() {
	godotenv.Load()

	cfg.PasswordMinLength = 8
	cfg.PasswordMaxLength = 1024

	db_url := os.Getenv("DB_URL")
	passwordMinLength := os.Getenv("MIN_PASSWORD_LENGTH")
	passwordMaxLength := os.Getenv("MAX_PASSWORD_LENGTH")
	if len(passwordMinLength) > 0 {
		passwordMinLengthVal, err := strconv.ParseUint(passwordMinLength, 10, 31)
		if err != nil {
			log.Fatalf("Invalid MIN_PASSWORD_LENGTH: %v", passwordMinLength)
		}
		cfg.PasswordMinLength = int32(passwordMinLengthVal)
	}
	if len(passwordMaxLength) > 0 {
		passwordMaxLengthVal, err := strconv.ParseUint(passwordMaxLength, 10, 31)
		if err != nil {
			log.Fatalf("Invalid MAX_PASSWORD_LENGTH: %v", passwordMaxLength)
		}
		cfg.PasswordMaxLength = int32(passwordMaxLengthVal)
	}

	if cfg.PasswordMaxLength < cfg.PasswordMinLength {
		log.Fatalf("MAX_PASSWORD_LENGTH (%v) must not be less than MAX_PASSWORD_LENGTH (%v)", cfg.PasswordMaxLength, cfg.PasswordMinLength)
	}

	db, err := sql.Open("postgres", db_url)
	if err != nil {
		// @todo Fatal?
		log.Printf("Error opening database: %v", err)
	}

	cfg.DbQueries = database.New(db)

	debugRedirect = true
	fmt.Println("Starting")

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

	http.HandleFunc("GET /{$}", middlewareLoggedIn(handlerHomePage))
	http.HandleFunc("GET /app/login", middlewareLoggedIn(handlerLoginPage))
	http.HandleFunc("GET /app/logout", middlewareLoggedIn(handlerLogoutPage))
	http.HandleFunc("GET /app/register", middlewareLoggedIn(handlerRegisterPage))
	http.HandleFunc("GET /app/boards/{board...}", middlewareLoggedIn(handlerBoardPage))

	http.HandleFunc("GET /debug", middlewareLoggedIn(handlerDebugGetRequest))
	http.HandleFunc("POST /debug", middlewareLoggedIn(handlerDebugPostRequest))

	http.HandleFunc("POST /app/login", handlerLogin)
	http.HandleFunc("POST /app/register", handlerRegister)
	http.HandleFunc("POST /app/logout", middlewareLoggedIn(handlerLogout))
	http.HandleFunc("POST /app/boards/{board...}", middlewareLoggedIn(handlerMakePost))

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
