package auth

import (
	"testing"

	"github.com/markbates/goth"
)

func TestOAuthDisplayName(t *testing.T) {
	tests := []struct {
		name string
		user goth.User
		want string
	}{
		{
			name: "full name",
			user: goth.User{Name: "Jane Doe"},
			want: "Jane Doe",
		},
		{
			name: "given and family",
			user: goth.User{FirstName: "Jane", LastName: "Doe"},
			want: "Jane Doe",
		},
		{
			name: "nickname fallback",
			user: goth.User{NickName: "jane"},
			want: "jane",
		},
		{
			name: "empty",
			user: goth.User{Email: "jane@example.com"},
			want: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := oauthDisplayName(tc.user); got != tc.want {
				t.Fatalf("oauthDisplayName() = %q, want %q", got, tc.want)
			}
		})
	}
}
