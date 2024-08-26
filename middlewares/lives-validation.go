package middlewares

import (
	"errors"
	"fmt"
	"jsmi-api/models"
	"net/url"
	"regexp"
)

// IsValidURL checks if a URL is valid
func IsValidURL(toTest string) bool {
	u, err := url.ParseRequestURI(toTest)
	if err != nil {
		return false
	}

	// Check if the scheme is valid (http, https)
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}

	// Check if the host is valid using a regex
	hostRegex := regexp.MustCompile(`^[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

	return hostRegex.MatchString(u.Host)
}

// ValidateLives validates a live post's content.
func ValidateLives(live models.Live) error {
	// Sanitize inputs
	live.Title = SanitizeInput(live.Title)

	if live.Title == "" {
		return errors.New("title is required")
	}

	if err := ValidateWordCount(live.Title, 15); err != nil {
		return fmt.Errorf("title %w", err)
	}

	if !IsValidURL(live.Link) {
		return errors.New("invalid URL")
	}

	return nil
}
