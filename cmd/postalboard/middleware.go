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
		cookies := parseCookies(r.Header.Values("cookie"))

		//refresh_token := cookies["auth_refresh"]
		token := cookies["auth_refresh"]
		//refresh_token := cookies["auth_refresh"]
		// If no credentials are given user is not logged in
		// @todo Notify user when authentication fails?
		// @todo Delete cookie when login fails?
		user, err := auth.ValidateToken(r.Context(), dbQueries, token)
		if err != nil {
			log.Printf("Error: %v", err)
			// @todo Maybe reset auth_refresh cookie
		}

		handler(user, w, r)
	}
	return result
}
