package login

import (
	"encoding/json"
	"net/http"
	"time"

	"katrelgulumj/kratos/session"
	"katrelgulumj/kratos/x"
)

type FlowState string

const (
	FlowStatePending FlowState = "pending"
	FlowStateSuccess FlowState = "success"
	FlowStateFailed  FlowState = "failed"
)

type Flow struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	ExpiresAt time.Time `json:"expires_at"`
	State     FlowState `json:"state"`
}

type IdentityValidator interface {
	Validate(credentials map[string]string) (string, error)
}

type Handler struct {
	sessionManager    session.Manager
	identityValidator IdentityValidator
	flows             map[string]*Flow
}

func NewHandler(sm session.Manager, validator IdentityValidator) *Handler {
	return &Handler{
		sessionManager:    sm,
		identityValidator: validator,
		flows:             make(map[string]*Flow),
	}
}

func (h *Handler) RegisterFlow(f *Flow) {
	h.flows[f.ID] = f
}

func (h *Handler) GetFlow(id string) *Flow {
	return h.flows[id]
}

type LoginSubmission struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

type LoginSuccessResponse struct {
	Session *session.Session `json:"session"`
	Token   string           `json:"session_token"`
}

func (h *Handler) HandleLoginAPI(w http.ResponseWriter, r *http.Request) {
	flowID := r.URL.Query().Get("flow")
	if flowID == "" {
		x.WriteJSONError(w, r, x.ErrInternalServerError("missing_flow", "Flow ID is required in query parameter.", nil))
		return
	}

	flow, exists := h.flows[flowID]
	if !exists {
		flow = &Flow{
			ID:        flowID,
			Type:      "api",
			ExpiresAt: time.Now().Add(10 * time.Minute),
			State:     FlowStatePending,
		}
		h.flows[flowID] = flow
	}

	var sub LoginSubmission
	if err := json.NewDecoder(r.Body).Decode(&sub); err != nil {
		flow.State = FlowStateFailed
		x.WriteJSONError(w, r, x.ErrInternalServerError("invalid_payload", "Failed to parse request payload.", map[string]interface{}{"flow_id": flowID}))
		return
	}

	identityID, err := h.identityValidator.Validate(map[string]string{
		"identifier": sub.Identifier,
		"password":   sub.Password,
	})
	if err != nil {
		flow.State = FlowStateFailed
		x.WriteJSONError(w, r, x.ErrInternalServerError("invalid_credentials", "Invalid login credentials.", map[string]interface{}{"flow_id": flowID}))
		return
	}

	sess, token, err := h.sessionManager.Create(r.Context(), identityID, flowID)
	if err != nil {
		flow.State = FlowStateFailed
		if errDetail, ok := err.(*x.ErrorDetail); ok {
			x.WriteJSONError(w, r, errDetail)
			return
		}
		x.WriteJSONError(w, r, x.ErrInternalServerError("session_issuance_failed", err.Error(), map[string]interface{}{"flow_id": flowID}))
		return
	}

	flow.State = FlowStateSuccess
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(&LoginSuccessResponse{
		Session: sess,
		Token:   token,
	})
}
