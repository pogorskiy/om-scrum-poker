package main

import (
	"testing"
)

func TestParseAllowedOrigins_Empty(t *testing.T) {
	result := parseAllowedOrigins("")
	if len(result) != 0 {
		t.Errorf("expected empty slice, got %v", result)
	}
}

func TestParseAllowedOrigins_Single(t *testing.T) {
	result := parseAllowedOrigins("https://example.com")
	if len(result) != 1 {
		t.Fatalf("expected 1 origin, got %d", len(result))
	}
	if result[0] != "https://example.com" {
		t.Errorf("expected https://example.com, got %s", result[0])
	}
}

func TestParseAllowedOrigins_Multiple(t *testing.T) {
	result := parseAllowedOrigins("http://localhost:5173, https://example.com")
	if len(result) != 2 {
		t.Fatalf("expected 2 origins, got %d", len(result))
	}
	if result[0] != "http://localhost:5173" {
		t.Errorf("expected http://localhost:5173, got %s", result[0])
	}
	if result[1] != "https://example.com" {
		t.Errorf("expected https://example.com, got %s", result[1])
	}
}

func TestParseAllowedOrigins_WhitespaceHandling(t *testing.T) {
	result := parseAllowedOrigins("  http://a.com  ,  http://b.com  ")
	if len(result) != 2 {
		t.Fatalf("expected 2 origins, got %d", len(result))
	}
	if result[0] != "http://a.com" {
		t.Errorf("expected http://a.com, got %s", result[0])
	}
	if result[1] != "http://b.com" {
		t.Errorf("expected http://b.com, got %s", result[1])
	}
}

func TestParseAllowedOrigins_TrailingComma(t *testing.T) {
	result := parseAllowedOrigins("http://example.com,")
	if len(result) != 1 {
		t.Fatalf("expected 1 origin, got %d", len(result))
	}
	if result[0] != "http://example.com" {
		t.Errorf("expected http://example.com, got %s", result[0])
	}
}
