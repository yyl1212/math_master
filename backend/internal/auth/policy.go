package auth

import (
	"strings"
	"unicode/utf8"
)

func ValidateUsername(value string) (string, error) {
	if len(value) < 3 || len(value) > 32 {
		return "", ErrInvalidInput
	}
	for i := 0; i < len(value); i++ {
		c := value[i]
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_') {
			return "", ErrInvalidInput
		}
	}
	return strings.ToLower(value), nil
}
func ValidateNewPassword(value string) error {
	if len(value) > 512 || !utf8.ValidString(value) {
		return ErrInvalidInput
	}
	n := utf8.RuneCountInString(value)
	if n < 15 || n > 128 {
		return ErrInvalidInput
	}
	return nil
}
func NormalizeRoles(input []Role) ([]Role, error) {
	seen := make(map[Role]bool, 4)
	for _, role := range input {
		switch role {
		case RoleLearner, RoleEditor, RoleReviewer, RoleAdmin:
			seen[role] = true
		default:
			return nil, ErrInvalidInput
		}
	}
	if !seen[RoleLearner] {
		return nil, ErrInvalidInput
	}
	out := make([]Role, 0, 4)
	for _, role := range []Role{RoleLearner, RoleEditor, RoleReviewer, RoleAdmin} {
		if seen[role] {
			out = append(out, role)
		}
	}
	return out, nil
}
