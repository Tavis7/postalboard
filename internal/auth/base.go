package auth

import (
	"crypto/rand"
	"encoding/hex"
	"log"

	"github.com/alexedwards/argon2id"
)

func hashPassword(password string) (string, error) {
	return argon2id.CreateHash(password, argon2id.DefaultParams)
}

func validatePassword(password, hashed_password string) bool {
	match, err := argon2id.ComparePasswordAndHash(password, hashed_password)
	if err != nil {
		log.Printf("Error validating password: %v", err)
		return false
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
