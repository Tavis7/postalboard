package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/lib/pq/pqerror"

	"github.com/tavis7/postalboard/internal/config"
	"github.com/tavis7/postalboard/internal/database"
)

type AuthenticatedUser struct {
	user     database.User
	token    database.RefreshToken
	Username string
	UserID   uuid.UUID
}

func CreateUser(cfg config.Config, context context.Context, username, password string) (AuthenticatedUser, error) {
	result := AuthenticatedUser{}

	if len(password) < int(cfg.PasswordMinLength) ||
		len(password) > int(cfg.PasswordMaxLength) {
		return result, fmt.Errorf("Password must be between %d and %d characters long",
			cfg.PasswordMinLength, cfg.PasswordMaxLength)
	}

	hashed_password, err := hashPassword(password)

	if err != nil {
		return result, err
	}
	user, err := cfg.DbQueries.CreateUser(context,
		database.CreateUserParams{uuid.New(), time.Now().UTC(), username,
			fmt.Sprintf("%s@localhost" /* @todo */, username), hashed_password})
	if err != nil {
		if pq.As(err, pqerror.UniqueViolation) != nil {
			log.Printf("User already exists: %v", err)
		}
		// @todo Detect user already exists
		log.Printf("Error registering user: %v", err)
		return result, fmt.Errorf("Error registering user: %w", err)
	}

	result.user = user
	result.Username = user.Username
	result.UserID = user.ID
	getNewToken(context, cfg.DbQueries, &result)
	return result, err
}

func Authenticate(cfg config.Config, context context.Context, username, password string) (AuthenticatedUser, string, error) {
	log.Printf("Authenticating as %v", username)
	result := AuthenticatedUser{}
	user, err := cfg.DbQueries.GetUserByUsername(context, username)
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
	err = getNewToken(context, cfg.DbQueries, &result)
	return result, result.token.Token, err
}

func Logout(cfg config.Config, context context.Context, user AuthenticatedUser) error {
	return InvalidateRefreshToken(cfg, context, user.token.Token)
}

func InvalidateRefreshToken(cfg config.Config, context context.Context, token string) error {
	row, err := cfg.DbQueries.RevokeRefreshToken(context,
		database.RevokeRefreshTokenParams{token,
			sql.NullTime{time.Now().UTC(), true},
		})
	if err != nil {
		log.Printf("Failed to invalidate refresh token: '%s': %v: %v", token, row, err)
	}
	return err
}

func getNewToken(context context.Context, dbQueries *database.Queries, user *AuthenticatedUser) error {
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
		return fmt.Errorf("Failed to create refresh token: %w", err)
	}
	user.token = tokenRow
	return nil
}

func ValidateToken(cfg config.Config, context context.Context, token string) (AuthenticatedUser, error) {
	// @todo Currently accepts any registered user
	log.Printf("Authenticating with token: %v", token)
	result := AuthenticatedUser{}
	user, err := cfg.DbQueries.GetUserByRefreshToken(context, token)
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
		InvalidateRefreshToken(cfg, context, user.RefreshToken.Token)
		return result, fmt.Errorf("Refresh token '%s' expired at %v",
			user.RefreshToken.Token, user.RefreshToken.ExpiresAt)
	}

	result.user = user.User
	result.token = user.RefreshToken
	result.Username = result.user.Username
	return result, nil
}
