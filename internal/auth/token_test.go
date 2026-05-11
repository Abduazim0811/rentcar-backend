package auth

import (
	"testing"
	"time"

	"car-rental-system/internal/models"
)

func TestGenerateAndParseToken(t *testing.T) {
	user := &models.User{
		ID:   42,
		Role: models.RoleAdmin,
	}

	token, err := GenerateToken(user, "test-secret", 24*time.Hour)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	claims, err := ParseToken(token, "test-secret")
	if err != nil {
		t.Fatalf("parse token: %v", err)
	}

	if claims.UserID != user.ID {
		t.Fatalf("expected user id %d, got %d", user.ID, claims.UserID)
	}
	if claims.Role != user.Role {
		t.Fatalf("expected role %q, got %q", user.Role, claims.Role)
	}
	if claims.Subject != "42" {
		t.Fatalf("expected subject %q, got %q", "42", claims.Subject)
	}
}

func TestParseTokenRejectsWrongSecret(t *testing.T) {
	user := &models.User{
		ID:   42,
		Role: models.RoleCustomer,
	}

	token, err := GenerateToken(user, "correct-secret", 24*time.Hour)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	if _, err := ParseToken(token, "wrong-secret"); err == nil {
		t.Fatal("expected wrong secret to fail")
	}
}
