package auth

import (
	"context"
	"errors"
	"testing"
)

func TestShouldRefuseToSignInWithoutClientID(t *testing.T) {
	if _, err := Authenticate(context.Background(), ""); !errors.Is(err, ErrNoClientID) {
		t.Fatalf("expected ErrNoClientID, got %v", err)
	}
}
