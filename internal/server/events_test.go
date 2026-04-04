package server

import (
	"encoding/json"
	"testing"
)

func TestMakeEnvelope(t *testing.T) {
	tests := []struct {
		name      string
		eventType string
		payload   interface{}
		wantType  string
		wantErr   bool
	}{
		{
			name:      "simple payload",
			eventType: "vote_cast",
			payload:   VoteCastPayload{SessionID: "abc123"},
			wantType:  "vote_cast",
		},
		{
			name:      "empty struct payload",
			eventType: "round_reset",
			payload:   struct{}{},
			wantType:  "round_reset",
		},
		{
			name:      "error payload",
			eventType: "error",
			payload:   ErrorPayload{Code: "invalid_vote", Message: "bad value"},
			wantType:  "error",
		},
		{
			name:      "nil payload",
			eventType: "test",
			payload:   nil,
			wantType:  "test",
		},
		{
			name:      "unmarshalable payload",
			eventType: "bad",
			payload:   make(chan int),
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := MakeEnvelope(tt.eventType, tt.payload)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			var env Envelope
			if err := json.Unmarshal(data, &env); err != nil {
				t.Fatalf("result is not valid JSON: %v", err)
			}
			if env.Type != tt.wantType {
				t.Errorf("type = %q, want %q", env.Type, tt.wantType)
			}
			if env.Payload == nil {
				t.Error("payload is nil")
			}
		})
	}
}

func TestMakeEnvelope_PayloadContent(t *testing.T) {
	data, err := MakeEnvelope("vote_cast", VoteCastPayload{SessionID: "sess-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var env Envelope
	if err := json.Unmarshal(data, &env); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}

	var payload VoteCastPayload
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.SessionID != "sess-1" {
		t.Errorf("sessionID = %q, want %q", payload.SessionID, "sess-1")
	}
}

func TestParseEnvelope(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantType string
		wantErr  bool
	}{
		{
			name:     "valid envelope",
			input:    `{"type":"vote","payload":{"value":"5"}}`,
			wantType: "vote",
		},
		{
			name:     "type with null payload",
			input:    `{"type":"reveal","payload":null}`,
			wantType: "reveal",
		},
		{
			name:     "type with empty object payload",
			input:    `{"type":"new_round","payload":{}}`,
			wantType: "new_round",
		},
		{
			name:    "missing type field",
			input:   `{"payload":{"value":"5"}}`,
			wantErr: true,
		},
		{
			name:    "empty type string",
			input:   `{"type":"","payload":{}}`,
			wantErr: true,
		},
		{
			name:    "invalid JSON",
			input:   `{not json}`,
			wantErr: true,
		},
		{
			name:    "empty input",
			input:   ``,
			wantErr: true,
		},
		{
			name:    "just a string",
			input:   `"hello"`,
			wantErr: true,
		},
		{
			name:    "array instead of object",
			input:   `[1,2,3]`,
			wantErr: true,
		},
		{
			name:     "extra fields are ignored",
			input:    `{"type":"vote","payload":{},"extra":"field"}`,
			wantType: "vote",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env, err := ParseEnvelope([]byte(tt.input))
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if env.Type != tt.wantType {
				t.Errorf("type = %q, want %q", env.Type, tt.wantType)
			}
		})
	}
}

func TestRoundTrip(t *testing.T) {
	// MakeEnvelope then ParseEnvelope should preserve data.
	original := ErrorPayload{Code: "test_code", Message: "test message"}
	data, err := MakeEnvelope("error", original)
	if err != nil {
		t.Fatalf("MakeEnvelope: %v", err)
	}

	env, err := ParseEnvelope(data)
	if err != nil {
		t.Fatalf("ParseEnvelope: %v", err)
	}

	if env.Type != "error" {
		t.Errorf("type = %q, want %q", env.Type, "error")
	}

	var decoded ErrorPayload
	if err := json.Unmarshal(env.Payload, &decoded); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if decoded.Code != original.Code || decoded.Message != original.Message {
		t.Errorf("payload mismatch: got %+v, want %+v", decoded, original)
	}
}
