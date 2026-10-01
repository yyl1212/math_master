package auth

import (
	"errors"
	"strings"
	"testing"
)

func TestUsernameAndPasswordPolicy(t *testing.T) {
	for _, tt := range []struct {
		input, want string
		valid       bool
	}{
		{"Math_User", "math_user", true}, {"ABC", "abc", true}, {strings.Repeat("a", 32), strings.Repeat("a", 32), true},
		{" math_user ", "", false}, {"ab", "", false}, {strings.Repeat("a", 33), "", false}, {"数学用户", "", false}, {"abc-def", "", false},
	} {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ValidateUsername(tt.input)
			if tt.valid {
				if err != nil || got != tt.want {
					t.Fatal("username normalization failed")
				}
			} else if !errors.Is(err, ErrInvalidInput) {
				t.Fatal("invalid username accepted")
			}
		})
	}
	for i, tt := range []struct {
		input string
		valid bool
	}{
		{strings.Repeat("1", 15), true}, {strings.Repeat("1", 14), false}, {strings.Repeat("😀", 128), true}, {strings.Repeat("a", 129), false},
		{"这个密码包含中文以及末尾空格 ", true}, {strings.Repeat(" ", 15), true}, {string([]byte{0xff}), false},
	} {
		if err := ValidateNewPassword(tt.input); (err == nil) != tt.valid {
			t.Fatalf("password boundary %d failed", i)
		}
	}
	roles, err := NormalizeRoles([]Role{RoleAdmin, RoleLearner, RoleReviewer, RoleAdmin, RoleEditor})
	if err != nil || len(roles) != 4 || roles[0] != RoleLearner || roles[1] != RoleEditor || roles[2] != RoleReviewer || roles[3] != RoleAdmin {
		t.Fatal("role normalization failed")
	}
	for _, roles := range [][]Role{nil, {RoleAdmin}, {RoleLearner, "owner"}} {
		if _, err := NormalizeRoles(roles); !errors.Is(err, ErrInvalidInput) {
			t.Fatal("invalid role set accepted")
		}
	}
}
