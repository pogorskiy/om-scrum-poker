package server

import (
	"bytes"
	"encoding/json"
	"log"
	"os"
	"strings"
	"testing"
)

func TestMakeEnvelopeOrLog_Success(t *testing.T) {
	payload := VoteCastPayload{SessionID: "abc123"}
	msg := makeEnvelopeOrLog("vote_cast", payload)
	if msg == nil {
		t.Fatal("expected non-nil message")
	}

	var env Envelope
	if err := json.Unmarshal(msg, &env); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if env.Type != "vote_cast" {
		t.Errorf("expected type vote_cast, got %q", env.Type)
	}

	var p VoteCastPayload
	if err := json.Unmarshal(env.Payload, &p); err != nil {
		t.Fatalf("failed to unmarshal payload: %v", err)
	}
	if p.SessionID != "abc123" {
		t.Errorf("expected sessionId abc123, got %q", p.SessionID)
	}
}

func TestMakeEnvelopeOrLog_AllEventTypes(t *testing.T) {
	tests := []struct {
		name    string
		event   string
		payload interface{}
	}{
		{"room_state", "room_state", RoomStatePayload{RoomID: "r1", RoomName: "Test"}},
		{"participant_joined", "participant_joined", ParticipantJoinedPayload{SessionID: "s1", UserName: "Alice", Status: "active", Role: "voter"}},
		{"participant_left", "participant_left", ParticipantLeftPayload{SessionID: "s1"}},
		{"vote_cast", "vote_cast", VoteCastPayload{SessionID: "s1"}},
		{"vote_retracted", "vote_retracted", VoteRetractedPayload{SessionID: "s1"}},
		{"presence_changed", "presence_changed", PresenceChangedPayload{SessionID: "s1", Status: "active"}},
		{"name_updated", "name_updated", NameUpdatedPayload{SessionID: "s1", UserName: "Bob"}},
		{"role_updated", "role_updated", RoleUpdatedPayload{SessionID: "s1", Role: "observer"}},
		{"round_reset", "round_reset", struct{}{}},
		{"room_cleared", "room_cleared", struct{}{}},
		{"timer_updated", "timer_updated", TimerStatePayload{Duration: 300, State: "idle", Remaining: 300}},
		{"error", "error", ErrorPayload{Code: "test", Message: "test error"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := makeEnvelopeOrLog(tt.event, tt.payload)
			if msg == nil {
				t.Fatal("expected non-nil message")
			}
			var env Envelope
			if err := json.Unmarshal(msg, &env); err != nil {
				t.Fatalf("unmarshal failed: %v", err)
			}
			if env.Type != tt.event {
				t.Errorf("expected type %q, got %q", tt.event, env.Type)
			}
		})
	}
}

// unmarshallable is a type that causes json.Marshal to fail.
type unmarshallable struct{}

func (u unmarshallable) MarshalJSON() ([]byte, error) {
	return nil, &json.MarshalerError{Type: nil, Err: nil}
}

func TestMakeEnvelopeOrLog_ErrorLogged(t *testing.T) {
	// Capture log output.
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	msg := makeEnvelopeOrLog("bad_event", unmarshallable{})
	if msg != nil {
		t.Fatal("expected nil message on marshal failure")
	}

	logOutput := buf.String()
	if !strings.Contains(logOutput, "MakeEnvelope(bad_event)") {
		t.Errorf("expected log to contain event type, got: %s", logOutput)
	}
}

func TestMakeEnvelopeOrLog_NilPayload(t *testing.T) {
	// nil payload should serialize to JSON null — valid.
	msg := makeEnvelopeOrLog("test_event", nil)
	if msg == nil {
		t.Fatal("expected non-nil message for nil payload")
	}

	var env Envelope
	if err := json.Unmarshal(msg, &env); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if env.Type != "test_event" {
		t.Errorf("expected type test_event, got %q", env.Type)
	}
}
