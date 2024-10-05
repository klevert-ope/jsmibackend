package validation

import (
	"errors"
	"fmt"
	"jsmi-api/models"
	"regexp"
	"strings"
	"unicode"
)

// ValidatePost validates a blog post's content.
func ValidatePost(post models.Post) error {
	// Sanitize inputs
	post.Title = SanitizeInput(post.Title)
	post.Excerpt = SanitizeInput(post.Excerpt)
	post.Body = SanitizeInput(post.Body)

	if post.Title == "" {
		return errors.New("title is required")
	}
	if post.Excerpt == "" {
		return errors.New("excerpt is required")
	}
	if post.Body == "" {
		return errors.New("body is required")
	}

	if err := ValidateWordCount(post.Title, 15); err != nil {
		return fmt.Errorf("title %w", err)
	}
	if err := ValidateWordCount(post.Excerpt, 60); err != nil {
		return fmt.Errorf("excerpt %w", err)
	}
	if err := ValidateWordCount(post.Body, 10000); err != nil {
		return fmt.Errorf("body %w", err)
	}

	return nil
}

// WordCount returns the number of words in the input string using a regular expression.
func WordCount(input string) int {
	// Define a regular expression to match words
	re := regexp.MustCompile(`\b\w+\b`)
	// Find all matches
	matches := re.FindAllString(input, -1)
	return len(matches)
}

// ValidateWordCount checks if the word count of the input string exceeds the limit.
func ValidateWordCount(input string, limit int) error {
	wordCount := WordCount(input)
	if wordCount > limit {
		return errors.New("exceeds word count limit")
	}
	return nil
}

// SanitizeInput sanitizes user input by removing potentially harmful characters.
func SanitizeInput(input string) string {
	// Remove potentially harmful characters.
	return removeUnsafeCharacters(input)
}

// removeUnsafeCharacters removes potentially harmful characters from the input.
func removeUnsafeCharacters(input string) string {
	var safeRunes []rune
	for _, r := range input {
		if isSafeCharacter(r) {
			safeRunes = append(safeRunes, r)
		}
	}
	return string(safeRunes)
}

// isSafeCharacter checks if a rune is a safe character (letters, digits, or safe symbols).
func isSafeCharacter(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || isSafeSymbol(r)
}

// isSafeSymbol checks if a rune is a safe symbol.
func isSafeSymbol(r rune) bool {
	safeSymbols := " .,~-!/@#%*&$+÷€£¥×=;:?<>[]{}|\\\"'()"
	return strings.ContainsRune(safeSymbols, r)
}
