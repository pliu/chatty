package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
)

// SecretKey should ideally be loaded from environment variables
var SecretKey = []byte("super-secret-key-change-me-in-production")

// SignCookie creates a signed cookie value in the format "value|signature"
func SignCookie(value string) string {
	mac := hmac.New(sha256.New, SecretKey)
	mac.Write([]byte(value))
	signature := mac.Sum(nil)
	return fmt.Sprintf("%s|%s", base64.URLEncoding.EncodeToString([]byte(value)), base64.URLEncoding.EncodeToString(signature))
}

// VerifyCookie verifies the signed cookie and returns the original value
func VerifyCookie(signedValue string) (string, error) {
	parts := strings.Split(signedValue, "|")
	if len(parts) != 2 {
		return "", errors.New("invalid cookie format")
	}

	valueBase64 := parts[0]
	signatureBase64 := parts[1]

	valueBytes, err := base64.URLEncoding.DecodeString(valueBase64)
	if err != nil {
		return "", errors.New("invalid value encoding")
	}
	value := string(valueBytes)

	signature, err := base64.URLEncoding.DecodeString(signatureBase64)
	if err != nil {
		return "", errors.New("invalid signature encoding")
	}

	mac := hmac.New(sha256.New, SecretKey)
	mac.Write([]byte(value))
	expectedSignature := mac.Sum(nil)

	if !hmac.Equal(signature, expectedSignature) {
		return "", errors.New("invalid signature")
	}

	return value, nil
}
