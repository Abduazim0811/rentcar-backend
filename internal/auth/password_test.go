package auth

import "testing"

func TestPasswordHashAndCheck(t *testing.T) {
	hash, err := HashPassword("secret-password")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	if hash == "secret-password" {
		t.Fatal("hash must not equal raw password")
	}

	if !CheckPassword(hash, "secret-password") {
		t.Fatal("expected password to match hash")
	}

	if CheckPassword(hash, "wrong-password") {
		t.Fatal("expected wrong password to fail")
	}
}
