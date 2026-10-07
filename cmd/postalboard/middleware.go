package main

import (
	"log"
	"net/http"

	"github.com/tavis7/postalboard/internal/auth"
)

type authenticatedFunc func(auth.AuthenticatedUser, http.ResponseWriter, *http.Request)
type handler func(http.ResponseWriter, *http.Request)

func middlewareLoggedIn(handler authenticatedFunc) handler {
	result := func(w http.ResponseWriter, r *http.Request) {
		user := auth.AuthenticatedUser{}

		cookies := parseCookies(r.Header.Values("cookie"))

		username, ok := cookies["username"]
		if !ok || (len(username) != 0) {
			// If no credentials are given user is not logged in
			// @todo Notify user when authentication fails?
			// @todo Delete cookie when login fails?
			user_, err := auth.ValidateToken(r.Context(), dbQueries, username)
			if err != nil {
				log.Printf("Error: %v", err)
			}
			user = user_
		}

		handler(user, w, r)
	}
	return result
}
