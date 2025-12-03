package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestValidateJWT_HappyPath(t *testing.T) {
	userID := uuid.New()
	secret := "margareta"
	tokenString, err := MakeJWT(userID, secret, time.Hour)
	if err != nil {
		t.Fatalf("MakeJWT returned error: %v", err)
	}

	gotID, err := ValidateJWT(tokenString, secret)
	if err != nil {
		t.Errorf("expected err == nil, err == %v", err)
	}

	if gotID != userID {
		t.Fatalf("expected: %v, got: %v", userID, gotID)
	}
}

func TestValidateJWT_ExpiredToken(t *testing.T) {
	userID := uuid.New()
	secret := "hemlis"
	tokenString, err := MakeJWT(userID, secret, -time.Minute)
	if err != nil {
		t.Fatalf("MakeJWT returned error: %v", err)
	}

	gotID, err := ValidateJWT(tokenString, secret)
	if err == nil {
		t.Fatalf("expected error for expired token, got nil")
	}

	zeroID := uuid.UUID{}
	if gotID != zeroID {
		t.Fatalf("expected zero UUID for expired token, got %v", gotID)
	}
}

func TestValidateJWT_WrongSecret(t *testing.T) {
	userID := uuid.New()
	secret := "hundar dansar oftast inte mambo"
	tokenString, err := MakeJWT(userID, secret, -time.Minute)
	if err != nil {
		t.Fatalf("MakeJWT returned error: %v", err)
	}

	gotID, err := ValidateJWT(tokenString, "katter dansar alltid rumba")
	if err == nil {
		t.Errorf("expected err == nil, err == %v", err)
	}

	if gotID == userID {
		t.Fatal("userID & gotID should not be equal")
	}
}
