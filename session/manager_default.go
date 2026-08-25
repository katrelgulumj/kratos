package session

import (
	"context"
	"fmt"
	"log"
	"time"

	"katrelgulumj/kratos/x"
)

type Logger interface {
	Printf(format string, v ...interface{})
}

type Manager interface {
	Create(ctx context.Context, identityID string, flowID string) (*Session, string, error)
	IssueToken(ctx context.Context, s *Session, flowID string) (string, error)
}

type DefaultManager struct {
	persister Persister
	tokenizer Tokenizer
	logger    Logger
}

func NewDefaultManager(persister Persister, tokenizer Tokenizer, logger Logger) *DefaultManager {
	if logger == nil {
		logger = log.Default()
	}
	return &DefaultManager{
		persister: persister,
		tokenizer: tokenizer,
		logger:    logger,
	}
}

func (m *DefaultManager) Create(ctx context.Context, identityID string, flowID string) (*Session, string, error) {
	sessionID := fmt.Sprintf("sess_%d", time.Now().UnixNano())
	s := &Session{
		ID:              sessionID,
		IdentityID:      identityID,
		Active:          true,
		IssuedAt:        time.Now(),
		ExpiresAt:       time.Now().Add(24 * time.Hour),
		AuthenticatedAt: time.Now(),
	}

	if err := m.persister.CreateSession(ctx, s); err != nil {
		m.logger.Printf("[ERROR] Failed to persist session for identity=%s flow_id=%s: %v", identityID, flowID, err)
		return nil, "", x.ErrInternalServerError("session_persistence_failed", "Unable to persist session record.", map[string]interface{}{
			"flow_id":     flowID,
			"identity_id": identityID,
		})
	}

	token, err := m.IssueToken(ctx, s, flowID)
	if err != nil {
		if delErr := m.persister.DeleteSession(ctx, s.ID); delErr != nil {
			m.logger.Printf("[CRITICAL] Failed to rollback orphaned session id=%s for identity=%s flow_id=%s: %v", s.ID, identityID, flowID, delErr)
		} else {
			m.logger.Printf("[INFO] Successfully rolled back session id=%s after token signing failure for flow_id=%s", s.ID, flowID)
		}
		return nil, "", err
	}

	s.Token = token
	return s, token, nil
}

func (m *DefaultManager) IssueToken(ctx context.Context, s *Session, flowID string) (string, error) {
	if m.tokenizer == nil {
		m.logger.Printf("[ERROR] Session tokenizer is uninitialized for flow_id=%s identity_id=%s", flowID, s.IdentityID)
		return "", x.ErrInternalServerError("tokenizer_not_configured", "Session token signer is not configured.", map[string]interface{}{
			"flow_id":     flowID,
			"identity_id": s.IdentityID,
		})
	}

	token, err := m.tokenizer.SignToken(ctx, s)
	if err != nil {
		m.logger.Printf("[ERROR] Session token signing failure: flow_id=%s identity_id=%s session_id=%s root_cause=%q",
			flowID, s.IdentityID, s.ID, err.Error())

		return "", x.ErrInternalServerError("token_signing_failed", "An error occurred while signing the session token.", map[string]interface{}{
			"flow_id":     flowID,
			"identity_id": s.IdentityID,
			"error_type":  "cryptographic_signing_error",
		})
	}

	return token, nil
}
