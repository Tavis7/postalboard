package auth

import "testing"

import (
	"fmt"
)

func TestGenerateRefreshToken(t *testing.T) {
	a := generateRefreshToken()
	b := generateRefreshToken()
	if a == b {
		t.Errorf("generateRefreshToken() generated two identical tokens: '%s', '%s'", a, b)
	}
}

func TestValidatePassword(t *testing.T) {
	// @note Testing is quadratic in the length of passwords array
	passwords := []string{
		"",
		"x",
		"san,83fys*FU!#(*%Y^nhtduos8cr",
		"abc",
		"foobar",
		"...",
	}

	for i, password := range passwords {
		hashed, err := hashPassword(password)
		if err != nil {
			t.Errorf("Hashing password failed: '%s': %v", password, err)
			continue
		}

		for j, testPassword := range passwords {
			if i != j {
				t.Run(fmt.Sprintf("password:%d!=%d,'%s'!='%s'",
					i, j, password, testPassword),
					func(t *testing.T) {
						if validatePassword(testPassword, hashed) {
							t.Errorf("Validation succeeded incorrectly: '%s' ?= '%s', '%v'",
								password, testPassword, hashed)
						}
					})
			} else {
				t.Run(fmt.Sprintf("password:%d,'%s'", i, password),
					func(t *testing.T) {
						if !validatePassword(password, hashed) {
							t.Errorf("Validation failed: '%s', '%v'", password, hashed)
						}
					})
			}
		}
	}
}
