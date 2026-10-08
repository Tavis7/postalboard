package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/tavis7/postalboard/internal/database"
)

type AuthenticatedUser struct {
	user     database.User
	token    database.RefreshToken
	Username string
	UserID   uuid.UUID
}

func CreateUser(context context.Context, dbQueries *database.Queries, username, password string) (AuthenticatedUser, error) {
	result := AuthenticatedUser{}
	hashed_password, err := hashPassword(password)
	if err != nil {
		return result, err
	}
	user, err := dbQueries.CreateUser(context,
		database.CreateUserParams{uuid.New(), time.Now().UTC(), username,
			fmt.Sprintf("%s@localhost" /* @todo */, username), hashed_password})
	if err != nil {
		// @todo Detect user already exists
		log.Printf("Error registering user: %v", err)
		return result, fmt.Errorf("Error registering user: %w", err)
	}
	result.user = user
	result.Username = user.Username
	result.UserID = user.ID
	getNewToken(context, dbQueries, &result)
	return result, err
}

func Authenticate(context context.Context, dbQueries *database.Queries, username, password string) (AuthenticatedUser, string, error) {
	log.Printf("Authenticating as %v", username)
	result := AuthenticatedUser{}
	user, err := dbQueries.GetUserByUsername(context, username)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("sql error: %v", err)
			return result, "", fmt.Errorf("Authentication failed")
		}

		pqErr := new(pq.Error)
		log.Printf("authentication error")
		if errors.As(err, &pqErr) {
			log.Printf("pq error: %v", pqErr.Code.Name())
		}

		return result, "", fmt.Errorf("Authentication failed: %w", err)
	}

	if !validatePassword(password, user.HashedPassword) {
		return result, "", fmt.Errorf("Authentication failed")
	}

	result.user = user
	result.Username = user.Username
	result.UserID = user.ID
	getNewToken(context, dbQueries, &result)
	return result, result.token.Token, nil
}

func Logout(context context.Context, dbQueries *database.Queries, user AuthenticatedUser) error {
	return InvalidateRefreshToken(context, dbQueries, user.token.Token)
}

func InvalidateRefreshToken(context context.Context, dbQueries *database.Queries, token string) error {
	row, err := dbQueries.RevokeRefreshToken(context,
		database.RevokeRefreshTokenParams{token,
			sql.NullTime{time.Now().UTC(), true},
		})
	if err != nil {
		log.Printf("Failed to invalidate refresh token: '%s': %v: %v", token, row, err)
	}
	return err
}

func getNewToken(context context.Context, dbQueries *database.Queries, user *AuthenticatedUser) {
	token := generateRefreshToken()

	// @todo @configuration login duration
	timeout := time.Hour * 24 * 30
	now := time.Now().UTC()
	expires := now.Add(timeout)

	log.Printf("now:     %v", now)
	log.Printf("expires: %v", expires)

	tokenRow, err := dbQueries.CreateRefreshToken(context,
		database.CreateRefreshTokenParams{token, now, expires, user.UserID})
	if err != nil {
		log.Printf("Error storing new refresh token: %v", err)
		return
	}
	user.token = tokenRow
}

func ValidateToken(context context.Context, dbQueries *database.Queries, token string) (AuthenticatedUser, error) {
	// @todo Currently accepts any registered user
	log.Printf("Authenticating with token: %v", token)
	result := AuthenticatedUser{}
	user, err := dbQueries.GetUserByRefreshToken(context, token)
	log.Printf("Username %v", user.User.Username)
	log.Printf("User: %v", user)
	if err != nil {
		result.user = user.User

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
	}

	now := time.Now().UTC()

	if user.RefreshToken.RevokedAt.Valid {
		// @todo Reset login cookie
		return result, fmt.Errorf("Refresh token '%v' revoked at %v",
			user.RefreshToken.Token, user.RefreshToken.RevokedAt)
	}

	if now.Compare(user.RefreshToken.ExpiresAt) >= 0 {
		// @todo Reset login cookie
		InvalidateRefreshToken(context, dbQueries, user.RefreshToken.Token)
		return result, fmt.Errorf("Refresh token '%s' expired at %v",
			user.RefreshToken.Token, user.RefreshToken.ExpiresAt)
	}

	result.user = user.User
	result.token = user.RefreshToken
	result.Username = result.user.Username
	return result, nil
}

func hashPassword(password string) (string, error) {
	return argon2id.CreateHash(password, argon2id.DefaultParams)
}

func validatePassword(password, hashed_password string) bool {
	match, err := argon2id.ComparePasswordAndHash(password, hashed_password)
	if err != nil {
		log.Printf("Error validating password: %v", err)
	}
	return match
}

func generateRefreshToken() string {
	data := [32]byte{}
	_, err := rand.Read(data[:])
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	return hex.EncodeToString(data[:])
}
