package validation

import (
	"errors"
	"fmt"
	"jsmi-api/models"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
)

// ValidationError represents custom validation errors.
type ValidationError struct {
	Errors []string
}

func (e *ValidationError) Error() string {
	return "validation errors: " + strings.Join(e.Errors, ", ")
}

var validate *validator.Validate

func init() {
	validate = validator.New()
}

func ValidateUserData(user models.User) error {
	var validationErrors []string

	// Validate using validator package
	err := validate.Struct(user)
	if err != nil {
		for _, err := range err.(validator.ValidationErrors) {
			validationErrors = append(validationErrors, fmt.Sprintf("%s: %s", err.Field(), err.Tag()))
		}
		return &ValidationError{Errors: validationErrors}
	}

	// Additional custom validation
	if len(user.Password) < 8 {
		validationErrors = append(validationErrors, "password must be at least 8 characters long")
	}

	if !isComplexPassword(user.Password) {
		validationErrors = append(validationErrors, "password must include at least one uppercase letter, one lowercase letter, one digit, and one special character")
	}

	if len(validationErrors) > 0 {
		return &ValidationError{Errors: validationErrors}
	}

	return nil
}

// Define static errors at the package level
var (
	ErrPasswordTooShort   = errors.New("new password must be at least 8 characters long")
	ErrPasswordSameAsOld  = errors.New("new password must be different from the old password")
	ErrPasswordNotComplex = errors.New("new password must include at least one uppercase letter, one lowercase letter, one digit, and one special character")
)

// ValidatePasswordChange checks if the old and new passwords are valid
func ValidatePasswordChange(oldPassword, newPassword string) error {
	if len(newPassword) < 8 {
		return ErrPasswordTooShort
	}

	if oldPassword == newPassword {
		return ErrPasswordSameAsOld
	}

	if !isComplexPassword(newPassword) {
		return ErrPasswordNotComplex
	}

	return nil
}

// isComplexPassword checks password complexity
var (
	lowercaseRegex = regexp.MustCompile(`[a-z]`)
	uppercaseRegex = regexp.MustCompile(`[A-Z]`)
	digitRegex     = regexp.MustCompile(`\d`)
	specialRegex   = regexp.MustCompile(`[@$!%*?&]`)
)

func isComplexPassword(password string) bool {
	return lowercaseRegex.MatchString(password) &&
		uppercaseRegex.MatchString(password) &&
		digitRegex.MatchString(password) &&
		specialRegex.MatchString(password)
}
