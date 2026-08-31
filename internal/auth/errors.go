package auth

import "errors"

var (
	ErrInvalidCredentials = errors.New("invalid_credentials")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrInvalidToken       = errors.New("invalid_token")
	ErrSeedIncomplete     = errors.New("seed_incomplete")
)
