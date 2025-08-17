package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

type test struct {
	password string
}

func TestHasher(t *testing.T) {
	tests := []test{
		{
			password: "Hello",
		},
		{
			password: "password123",
		},
		{
			password: "anotherpassword",
		},
	}
	for _, tt := range tests {
		hashedPassword, err := HashPassword(tt.password)
		if err != nil {
			t.Errorf("HashPassword(%q) failed: %v", tt.password, err)
			continue
		}

		// Test that the original password matches the hashed password
		err = CheckPasswordHash(tt.password, hashedPassword)
		if err != nil {
			t.Errorf("CheckPasswordHash(%q, %q) failed: %v", tt.password, hashedPassword, err)
		}

		// Test that a wrong password does not match the hashed password
		wrongPassword := tt.password + "wrong"
		err = CheckPasswordHash(wrongPassword, hashedPassword)
		if err == nil {
			t.Errorf("CheckPasswordHash with wrong password (%q) succeeded, but should have failed", wrongPassword)
		}
	}
}

func TestMakeJWT(t *testing.T) {
	userID := uuid.New()
	tokenSecret := "supersecretkey"
	expiresIn := time.Hour

	tokenString, err := MakeJWT(userID, tokenSecret, expiresIn)
	if err != nil {
		t.Errorf("MakeJWT failed: %v", err)
		return
	}
	if tokenString == "" {
		t.Error("MakeJWT returned an empty token string")
	}
}

func TestValidateJWT(t *testing.T) {
	userID := uuid.New()
	tokenSecret := "supersecretkey"
	expiresIn := time.Hour

	// Test with a valid token
	tokenString, err := MakeJWT(userID, tokenSecret, expiresIn)
	if err != nil {
		t.Fatalf("Failed to make JWT for validation test: %v", err)
	}

	validatedID, err := ValidateJWT(tokenString, tokenSecret)
	if err != nil {
		t.Errorf("ValidateJWT failed with valid token: %v", err)
	}
	if validatedID != userID {
		t.Errorf("Validated UserID does not match original. Expected: %s, Got: %s", userID, validatedID)
	}

	// Test with an invalid secret
	invalidSecret := "wrongsecret"
	_, err = ValidateJWT(tokenString, invalidSecret)
	if err == nil {
		t.Error("ValidateJWT should have failed with an invalid secret, but it didn't")
	}

	// Test with an expired token (requires a short expiry and time progression)
	expiredIn := -time.Hour // Make it expire in the past
	expiredTokenString, err := MakeJWT(userID, tokenSecret, expiredIn)
	if err != nil {
		t.Fatalf("Failed to make expired JWT for validation test: %v", err)
	}

	_, err = ValidateJWT(expiredTokenString, tokenSecret)
	if err == nil {
		t.Error("ValidateJWT should have failed with an expired token, but it didn't")
	}
}
