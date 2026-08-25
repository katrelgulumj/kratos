package session_test

import (
	"context"
	"errors"
	"testing"

	"katrelgulumj/kratos/session"
	"katrelgulumj/kratos/x"
)

type mockTokenizer struct {
	signFunc func(ctx context.Context, s *session.Session) (string, error)
}

func (m *mockTokenizer) SignToken(ctx context.Context, s *session.Session) (string, error) {
	if m.signFunc != nil {
		return m.signFunc(ctx, s)
	}
	return "mock_signed_token", nil
}

func (m *mockTokenizer) VerifyToken(ctx context.Context, token string) (*session.Session, error) {
	return nil, nil
}

func TestManager_Create_Success(t *testing.T) {
	persister := session.NewMemoryPersister()
	tokenizer := &mockTokenizer{}
	mgr := session.NewDefaultManager(persister, tokenizer, nil)

	sess, token, err := mgr.Create(context.Background(), "user-123", "flow-abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "mock_signed_token" {
		t.Errorf("expected token 'mock_signed_token', got %q", token)
	}
	if sess == nil || sess.IdentityID != "user-123" {
		t.Errorf("invalid session output: %+v", sess)
	}

	persisted, err := persister.GetSession(context.Background(), sess.ID)
	if err != nil || persisted == nil {
		t.Errorf("session should exist in persister: %v", err)
	}
}

func TestManager_Create_SigningFailure_RollbackNoOrphan(t *testing.T) {
	persister := session.NewMemoryPersister()
	tokenizer := &mockTokenizer{
		signFunc: func(ctx context.Context, s *session.Session) (string, error) {
			return "", session.ErrInvalidKey
		},
	}
	mgr := session.NewDefaultManager(persister, tokenizer, nil)

	sess, token, err := mgr.Create(context.Background(), "user-456", "flow-xyz")
	if err == nil {
		t.Fatal("expected signing error, got nil")
	}
	if sess != nil || token != "" {
		t.Errorf("expected nil session and empty token, got sess=%+v, token=%q", sess, token)
	}

	errDetail, ok := err.(*x.ErrorDetail)
	if !ok {
		t.Fatalf("expected *x.ErrorDetail, got %T (%v)", err, err)
	}
	if errDetail.Code != 500 {
		t.Errorf("expected code 500, got %d", errDetail.Code)
	}
	if errDetail.Reason != "token_signing_failed" {
		t.Errorf("expected reason 'token_signing_failed', got %q", errDetail.Reason)
	}

	sessions, err := persister.ListSessionsByIdentity(context.Background(), "user-456")
	if err != nil {
		t.Fatalf("unexpected persister error: %v", err)
	}
	if len(sessions) != 0 {
		t.Fatalf("expected 0 orphaned sessions after rollback, found %d", len(sessions))
	}
}
