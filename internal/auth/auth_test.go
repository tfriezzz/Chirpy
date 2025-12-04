package auth

import (
	"net/http"
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

func TestGetBearerToken_HappyPath(t *testing.T) {
	headers := make(http.Header)
	headers.Add("Authorization", "Bearer TOKEN_STRING")

	wantString := "TOKEN_STRING"

	gotString, err := GetBearerToken(headers)
	if err != nil {
		t.Fatalf("GetBearerToken returned err: %v", err)
	}

	if gotString != wantString {
		t.Fatalf("want: %v, got: %v", wantString, gotString)
	}
}

func TestGetBearerToken_MissingHeader(t *testing.T) {
	headers := make(http.Header)
	headers.Add("gobbeldyguck", "Bearer TOKEN_STRING")

	_, err := GetBearerToken(headers)
	if err == nil {
		t.Fatalf("should return error")
	}
}
