package auth

import (
	"context"
	"database/sql"
	"errors"
	"log"

	"github.com/lib/pq"

	"github.com/tavis7/postalboard/internal/database"
)

type AuthenticatedUser struct {
	user     database.User
	Username string
}

func Authenticate(context context.Context, dbQueries *database.Queries, username string) (AuthenticatedUser, error) {
	log.Printf("Authenticating as %v", username)
	result := AuthenticatedUser{}
	user, err := dbQueries.GetUserByUsername(context, username)
	if err != nil {
		result.user = user

		if err == sql.ErrNoRows {
			log.Printf("sql error: %v", err)
			return result, nil
		}

		pqErr := new(pq.Error)
		log.Printf("authentication error")
		if errors.As(err, &pqErr) {
			log.Printf("pq error: %v", pqErr.Code.Name())
		}

		return result, err
	} else {
		result.user = user
		result.Username = user.Username
	}
	return result, nil
}

func ValidateToken(context context.Context, dbQueries *database.Queries, username string) (AuthenticatedUser, error) {
	log.Printf("Authenticating as %v", username)
	result := AuthenticatedUser{}
	user, err := dbQueries.GetUserByUsername(context, username)
	if err != nil {
		result.user = user

		if err == sql.ErrNoRows {
			log.Printf("sql error: %v", err)
			return result, nil
		}

		pqErr := new(pq.Error)
		log.Printf("authentication error")
		if errors.As(err, &pqErr) {
			log.Printf("pq error: %v", pqErr.Code.Name())
		}

		return result, err
	} else {
		result.user = user
		result.Username = user.Username
	}
	return result, nil
}
