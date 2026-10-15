package main

import (
	"errors"
	"strings"
	"unicode"
)

func validateName(s string) error {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return errors.New("name must contain a letter or digit")
	}
	for _, r := range trimmed {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && !unicode.IsSpace(r) {
			return errors.New("name contains an unsupported character")
		}
	}
	return nil
}
