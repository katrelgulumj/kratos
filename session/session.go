package session

import (
	"time"
)

type Session struct {
	ID              string    `json:"id"`
	Active          bool      `json:"active"`
	IdentityID      string    `json:"identity_id"`
	Token           string    `json:"token,omitempty"`
	IssuedAt        time.Time `json:"issued_at"`
	ExpiresAt       time.Time `json:"expires_at"`
	AuthenticatedAt time.Time `json:"authenticated_at"`
}
