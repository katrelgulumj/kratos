package login_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"katrelgulumj/kratos/selfservice/flow/login"
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
	return "valid_signed_jwt", nil
}

func (m *mockTokenizer) VerifyToken(ctx context.Context, token string) (*session.Session, error) {
	return nil, nil
}

type mockValidator struct {
	validFunc func(credentials map[string]string) (string, error)
}

func (v *mockValidator) Validate(credentials map[string]string) (string, error) {
	if v.validFunc != nil {
		return v.validFunc(credentials)
	}
	return "identity_1234", nil
}

func TestHandleLoginAPI_TokenSigningFailure_Structured500AndNoOrphanSession(t *testing.T) {
	persister := session.NewMemoryPersister()
	tokenizer := &mockTokenizer{
		signFunc: func(ctx context.Context, s *session.Session) (string, error) {
			return "", session.ErrInvalidKey
		},
	}
	sm := session.NewDefaultManager(persister, tokenizer, nil)
	val := &mockValidator{}
	handler := login.NewHandler(sm, val)

	flowID := "flow_test_123"
	handler.RegisterFlow(&login.Flow{
		ID:    flowID,
		Type:  "api",
		State: login.FlowStatePending,
	})

	reqBody := []byte(`{"identifier":"user@example.com","password":"correct_password"}`)
	req := httptest.NewRequest(http.MethodPost, "/self-service/login?flow="+flowID, bytes.NewBuffer(reqBody))
	rec := httptest.NewRecorder()

	handler.HandleLoginAPI(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}

	var resp x.ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response body is not a valid JSON ErrorResponse: %v, raw: %s", err, rec.Body.String())
	}

	if resp.Error == nil {
		t.Fatalf("expected non-nil error payload")
	}
	if resp.Error.Code != 500 {
		t.Errorf("expected error.code 500, got %d", resp.Error.Code)
	}
	if resp.Error.Reason != "token_signing_failed" {
		t.Errorf("expected reason 'token_signing_failed', got %q", resp.Error.Reason)
	}

	// Check flow state integrity
	flow := handler.GetFlow(flowID)
	if flow.State != login.FlowStateFailed {
		t.Errorf("expected flow state %s, got %s", login.FlowStateFailed, flow.State)
	}

	// Verify no orphaned session in database
	sessions, err := persister.ListSessionsByIdentity(context.Background(), "identity_1234")
	if err != nil {
		t.Fatalf("unexpected error listing sessions: %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("expected 0 orphaned sessions, found %d", len(sessions))
	}
}
