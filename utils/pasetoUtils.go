package utils

import (
	"errors"
	"os"
	"time"

	"github.com/o1egl/paseto"
	"golang.org/x/crypto/chacha20poly1305"
)

// CustomClaims represents the custom claims in the PASETO token
type CustomClaims struct {
	UserID int64     `json:"user_id"`
	Expiry time.Time `json:"expiry"`
}

// GetPasetoSecret retrieves the PASETO secret from the environment variables
// and ensures it is the correct length.
func GetPasetoSecret() ([]byte, error) {
	pasetoSecret := os.Getenv("PASETO_SECRET")
	if pasetoSecret == "" {
		return nil, errors.New("server configuration error: PASETO_SECRET is not set")
	}

	// Ensure the secret key is 32 bytes long
	symmetricKey := []byte(pasetoSecret)
	if len(symmetricKey) < chacha20poly1305.KeySize {
		return nil, errors.New("secret key is too short")
	}
	if len(symmetricKey) > chacha20poly1305.KeySize {
		symmetricKey = symmetricKey[:chacha20poly1305.KeySize]
	}

	return symmetricKey, nil
}

// GeneratePASETO generates a PASETO token with an expiration time
func GeneratePASETO(userID int64, expiration time.Duration) (string, error) {
	symmetricKey, err := GetPasetoSecret()
	if err != nil {
		return "", err
	}

	now := time.Now()
	expiry := now.Add(expiration)

	claims := CustomClaims{
		UserID: userID,
		Expiry: expiry,
	}

	v2 := paseto.NewV2()
	token, err := v2.Encrypt(symmetricKey, claims, nil)
	if err != nil {
		return "", err
	}

	return token, nil
}

// ValidatePASETO validates a PASETO token and returns the claims
func ValidatePASETO(tokenString string) (*CustomClaims, error) {
	symmetricKey, err := GetPasetoSecret()
	if err != nil {
		return nil, err
	}

	var claims CustomClaims
	v2 := paseto.NewV2()
	err = v2.Decrypt(tokenString, symmetricKey, &claims, nil)
	if err != nil {
		return nil, err
	}

	// Check for token expiration
	if time.Now().After(claims.Expiry) {
		return nil, errors.New("token has expired")
	}

	return &claims, nil
}
