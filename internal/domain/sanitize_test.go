package domain

import (
	"testing"
)

func TestSanitizeName(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		want   string
	}{
		{"normal name", "Alice", "Alice"},
		{"name with spaces", "John Doe", "John Doe"},
		{"leading trailing spaces", "  Alice  ", "Alice"},
		{"tab characters", "Al\tice", "Alice"},
		{"newline", "Al\nice", "Alice"},
		{"carriage return", "Al\rice", "Alice"},
		{"null byte", "Al\x00ice", "Alice"},
		{"DEL character", "Al\x7Fice", "Alice"},
		{"all control chars", "\x01\x02\x03", ""},
		{"only spaces", "   ", ""},
		{"only tabs", "\t\t\t", ""},
		{"only newlines", "\n\n\n", ""},
		{"zero-width space U+200B", "Al\u200Bice", "Alice"},
		{"zero-width non-joiner U+200C", "Al\u200Cice", "Alice"},
		{"zero-width joiner U+200D", "Al\u200Dice", "Alice"},
		{"left-to-right mark U+200E", "Al\u200Eice", "Alice"},
		{"right-to-left mark U+200F", "Al\u200Fice", "Alice"},
		{"line separator U+2028", "Al\u2028ice", "Alice"},
		{"paragraph separator U+2029", "Al\u2029ice", "Alice"},
		{"left-to-right embedding U+202A", "Al\u202Aice", "Alice"},
		{"right-to-left override U+202E", "Al\u202Eice", "Alice"},
		{"word joiner U+2060", "Al\u2060ice", "Alice"},
		{"invisible separator U+2063", "Al\u2063ice", "Alice"},
		{"BOM U+FEFF", "\uFEFFAlice", "Alice"},
		{"soft hyphen U+00AD (Cf)", "Al\u00ADice", "Alice"},
		{"mixed valid and invalid", "\x00Al\u200Bice\x7F Bo\u200Eb\n", "Alice Bob"},
		{"emoji preserved", "Alice 🎉", "Alice 🎉"},
		{"cyrillic preserved", "Алиса", "Алиса"},
		{"chinese preserved", "李明", "李明"},
		{"arabic preserved", "أحمد", "أحمد"},
		{"internal spaces preserved", "A  B  C", "A  B  C"},
		{"RTL override wrapping attack", "\u202EmaliciousText\u202C", "maliciousText"},
		{"invisible plus valid", "\u200B\u200CAlice\u200D\u200E", "Alice"},
		{"only zero-width chars", "\u200B\u200C\u200D", ""},
		{"only BOM", "\uFEFF", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeName(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestJoin_SanitizesName(t *testing.T) {
	r, _ := NewRoom("room1", "Test", "")

	// Name with control characters should be sanitized.
	p, _, err := r.Join("s1", "Al\x00ice", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Name != "Alice" {
		t.Errorf("expected sanitized name %q, got %q", "Alice", p.Name)
	}
}

func TestJoin_OnlySpacesName(t *testing.T) {
	r, _ := NewRoom("room1", "Test", "")

	_, _, err := r.Join("s1", "   ", "")
	if err == nil {
		t.Fatal("expected error for name consisting only of spaces")
	}
}

func TestJoin_OnlyControlCharsName(t *testing.T) {
	r, _ := NewRoom("room1", "Test", "")

	_, _, err := r.Join("s1", "\x01\x02\x03", "")
	if err == nil {
		t.Fatal("expected error for name consisting only of control characters")
	}
}

func TestJoin_OnlyZeroWidthCharsName(t *testing.T) {
	r, _ := NewRoom("room1", "Test", "")

	_, _, err := r.Join("s1", "\u200B\u200C\u200D", "")
	if err == nil {
		t.Fatal("expected error for name consisting only of zero-width characters")
	}
}

func TestUpdateName_SanitizesName(t *testing.T) {
	r, _ := NewRoom("room1", "Test", "")
	r.Join("s1", "Alice", "")

	err := r.UpdateName("s1", "Bo\u200Bb")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Participants["s1"].Name != "Bob" {
		t.Errorf("expected sanitized name %q, got %q", "Bob", r.Participants["s1"].Name)
	}
}

func TestUpdateName_OnlySpaces(t *testing.T) {
	r, _ := NewRoom("room1", "Test", "")
	r.Join("s1", "Alice", "")

	err := r.UpdateName("s1", "   ")
	if err == nil {
		t.Fatal("expected error for name consisting only of spaces")
	}
	// Name should remain unchanged.
	if r.Participants["s1"].Name != "Alice" {
		t.Errorf("expected name to remain %q, got %q", "Alice", r.Participants["s1"].Name)
	}
}

func TestUpdateName_RTLMarkers(t *testing.T) {
	r, _ := NewRoom("room1", "Test", "")
	r.Join("s1", "Alice", "")

	err := r.UpdateName("s1", "\u202EevE\u202C")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Participants["s1"].Name != "evE" {
		t.Errorf("expected sanitized name %q, got %q", "evE", r.Participants["s1"].Name)
	}
}
