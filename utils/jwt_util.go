package utils

import (
	"errors"
	"time"

	"github.com/o1egl/paseto"
	"golang.org/x/crypto/chacha20poly1305"
)

type CustomClaims struct {
	UserID int64     `json:"user_id"`
	Expiry time.Time `json:"expiry"`
}

// GeneratePASETO generates a PASETO token with an expiration time
func GeneratePASETO(userID int64, secret string, expiration time.Duration) (string, error) {
	now := time.Now()
	expiry := now.Add(expiration)

	claims := CustomClaims{
		UserID: userID,
		Expiry: expiry,
	}

	// Ensure the secret key is 32 bytes long
	symmetricKey := []byte(secret)
	if len(symmetricKey) < chacha20poly1305.KeySize {
		return "", errors.New("secret key is too short")
	}
	if len(symmetricKey) > chacha20poly1305.KeySize {
		symmetricKey = symmetricKey[:chacha20poly1305.KeySize]
	}

	v2 := paseto.NewV2()
	token, err := v2.Encrypt(symmetricKey, claims, nil)
	if err != nil {
		return "", err
	}

	return token, nil
}

// ValidatePASETO validates a PASETO token and returns the claims
func ValidatePASETO(tokenString, secret string) (*CustomClaims, error) {
	var claims CustomClaims

	// Ensure the secret key is 32 bytes long
	symmetricKey := []byte(secret)
	if len(symmetricKey) < chacha20poly1305.KeySize {
		return nil, errors.New("secret key is too short")
	}
	if len(symmetricKey) > chacha20poly1305.KeySize {
		symmetricKey = symmetricKey[:chacha20poly1305.KeySize]
	}

	v2 := paseto.NewV2()
	err := v2.Decrypt(tokenString, symmetricKey, &claims, nil)
	if err != nil {
		return nil, err
	}

	// Check for token expiration
	if time.Now().After(claims.Expiry) {
		return nil, errors.New("token has expired")
	}

	return &claims, nil
}
