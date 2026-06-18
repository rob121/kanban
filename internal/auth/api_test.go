package auth

import (
	"testing"

	"github.com/rob121/kanban/internal/models"
)

func TestHashAPITokenDeterministic(t *testing.T) {
	a := hashAPIToken("kbn_deadbeef")
	b := hashAPIToken("kbn_deadbeef")
	if a != b {
		t.Fatalf("hash mismatch")
	}
	if hashAPIToken("kbn_other") == a {
		t.Fatal("expected different hash")
	}
}

func TestTokenPrefixLength(t *testing.T) {
	if tokenPrefix("kbn_short") != "kbn_short" {
		t.Fatal("short token prefix")
	}
	long := "kbn_" + string(make([]byte, 40))
	if len(tokenPrefix(long)) != 12 {
		t.Fatalf("expected 12 char prefix, got %d", len(tokenPrefix(long)))
	}
}

func TestIsAPIUser(t *testing.T) {
	u := models.User{UserType: models.UserTypeAPI}
	if !u.IsAPIUser() {
		t.Fatal("expected api user")
	}
}
