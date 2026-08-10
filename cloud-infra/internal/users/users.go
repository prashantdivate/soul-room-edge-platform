package users

import (
	"errors"
	"strings"
)

func NormalizeEmail(email string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if !strings.Contains(email, "@") || strings.ContainsAny(email, "\r\n\t ") {
		return "", errors.New("invalid email")
	}
	return email, nil
}
