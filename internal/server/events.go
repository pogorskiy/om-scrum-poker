package server

import (
	"encoding/json"
	"fmt"

	"om-scrum-poker/internal/domain"
)

// Envelope wraps all WebSocket messages with a type discriminator.
type Envelope struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// --- Client → Server payloads ---

type JoinPayload struct {
	SessionID string `json:"sessionId"`
	UserName  string `json:"userName"`
	RoomName  string `json:"roomName,omitempty"`
	Role      string `json:"role,omitempty"`
}

type VotePayload struct {
	Value string `json:"value"`
}

type UpdateNamePayload struct {
	UserName string `json:"userName"`
}

type PresencePayload struct {
	Status string `json:"status"`
}

// --- Server → Client payloads ---

type ParticipantInfo struct {
	SessionID string `json:"sessionId"`
	UserName  string `json:"userName"`
	Status    string `json:"status"`
	HasVoted  bool   `json:"hasVoted"`
	Role      string `json:"role"`
}

type RoomStatePayload struct {
	RoomID       string              `json:"roomId"`
	RoomName     string              `json:"roomName"`
	CreatedBy    string              `json:"createdBy"`
	Phase        domain.Phase        `json:"phase"`
	Participants []ParticipantInfo   `json:"participants"`
	Result       *domain.RoundResult `json:"result"`
	Timer        TimerStatePayload   `json:"timer"`
}

type ParticipantJoinedPayload struct {
	SessionID string `json:"sessionId"`
	UserName  string `json:"userName"`
	Status    string `json:"status"`
	Role      string `json:"role"`
}

type ParticipantLeftPayload struct {
	SessionID string `json:"sessionId"`
}

type VoteCastPayload struct {
	SessionID string `json:"sessionId"`
}

type VoteRetractedPayload struct {
	SessionID string `json:"sessionId"`
}

type PresenceChangedPayload struct {
	SessionID string `json:"sessionId"`
	Status    string `json:"status"`
}

type NameUpdatedPayload struct {
	SessionID string `json:"sessionId"`
	UserName  string `json:"userName"`
}

type UpdateRolePayload struct {
	Role string `json:"role"`
}

type RoleUpdatedPayload struct {
	SessionID string `json:"sessionId"`
	Role      string `json:"role"`
}

type TimerSetDurationPayload struct {
	Duration int `json:"duration"`
}

type TimerStatePayload struct {
	Duration  int    `json:"duration"`
	State     string `json:"state"`
	StartedAt *int64 `json:"startedAt"` // unix ms, null when idle
	Remaining int    `json:"remaining"` // seconds
}

type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// MakeEnvelope creates a JSON-encoded envelope.
func MakeEnvelope(eventType string, payload interface{}) ([]byte, error) {
	p, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}
	env := Envelope{
		Type:    eventType,
		Payload: p,
	}
	return json.Marshal(env)
}

// ParseEnvelope decodes an incoming message.
func ParseEnvelope(data []byte) (*Envelope, error) {
	var env Envelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, fmt.Errorf("unmarshal envelope: %w", err)
	}
	if env.Type == "" {
		return nil, fmt.Errorf("missing message type")
	}
	return &env, nil
}
