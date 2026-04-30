package utils

import (
	"strings"
	"unicode"
)

func TrimSpaces(s string) string {
	return strings.TrimSpace(s)
}

func ToUpper(s string) string {
	return strings.ToUpper(s)
}

func IsLetter(ch rune) bool {
	return unicode.IsLetter(ch) || ch == '_'
}

func IsDigit(ch rune) bool {
	return unicode.IsDigit(ch)
}

func IsLetterOrDigit(ch rune) bool {
	return IsLetter(ch) || IsDigit(ch)
}
