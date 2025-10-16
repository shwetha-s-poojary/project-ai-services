package utils

import (
	"strings"
	"unicode"
)

// CapitalizeAndFormat replaces '_' and '-' with spaces, then capitalizes each word.
func CapitalizeAndFormat(s string) string {
	// Replace _ and - with spaces
	s = strings.ReplaceAll(s, "_", " ")
	s = strings.ReplaceAll(s, "-", " ")

	// Split into words
	words := strings.Fields(s)

	// Capitalize each word
	for i, w := range words {
		if len(w) > 0 {
			runes := []rune(w)
			runes[0] = unicode.ToUpper(runes[0])
			words[i] = string(runes)
		}
	}

	// Join back into a single string
	return strings.Join(words, " ")
}
