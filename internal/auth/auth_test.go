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
		t.Run(fmt.Sprintf("password:%d,'%s'", i, password),
			func(t *testing.T) {
				if !validatePassword(password, hashed) {
					t.Errorf("Validation failed: '%s', '%v'", password, hashed)
				}
			})

		for j, fakePassword := range passwords {
			if i != j {
				t.Run(fmt.Sprintf("password:%d!=%d,'%s'!='%s'",
					i, j, password, fakePassword),
					func(t *testing.T) {
						if validatePassword(fakePassword, hashed) {
							t.Errorf("Validation succeeded incorrectly: '%s' ?= '%s', '%v'",
								password, fakePassword, hashed)
						}
					})
			}
		}
	}
}
