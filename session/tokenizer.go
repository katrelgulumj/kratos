package session

import (
	"context"
	"errors"
)

var (
	ErrInvalidKey        = errors.New("invalid or corrupted cryptographic key")
	ErrSigningFailed     = errors.New("failed to sign session token")
	ErrKeyNotFound       = errors.New("token signing key not found in keystore")
	ErrUnsupportedKeyAlg = errors.New("unsupported key signing algorithm")
)

type Tokenizer interface {
	SignToken(ctx context.Context, s *Session) (string, error)
	VerifyToken(ctx context.Context, token string) (*Session, error)
}
