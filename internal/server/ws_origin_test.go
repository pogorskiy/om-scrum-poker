package server

import (
	"testing"
)

func TestBuildAcceptOptions_Empty(t *testing.T) {
	opts := buildAcceptOptions(nil)
	if opts.InsecureSkipVerify {
		t.Error("expected InsecureSkipVerify to be false for empty origins")
	}
	if len(opts.OriginPatterns) != 0 {
		t.Errorf("expected no OriginPatterns, got %v", opts.OriginPatterns)
	}
}

func TestBuildAcceptOptions_Wildcard(t *testing.T) {
	opts := buildAcceptOptions([]string{"*"})
	if !opts.InsecureSkipVerify {
		t.Error("expected InsecureSkipVerify to be true for wildcard origin")
	}
}

func TestBuildAcceptOptions_SpecificOrigins(t *testing.T) {
	origins := []string{"http://localhost:5173", "https://example.com"}
	opts := buildAcceptOptions(origins)
	if opts.InsecureSkipVerify {
		t.Error("expected InsecureSkipVerify to be false for specific origins")
	}
	if len(opts.OriginPatterns) != 2 {
		t.Fatalf("expected 2 OriginPatterns, got %d", len(opts.OriginPatterns))
	}
	if opts.OriginPatterns[0] != "http://localhost:5173" {
		t.Errorf("expected first pattern http://localhost:5173, got %s", opts.OriginPatterns[0])
	}
	if opts.OriginPatterns[1] != "https://example.com" {
		t.Errorf("expected second pattern https://example.com, got %s", opts.OriginPatterns[1])
	}
}

func TestBuildAcceptOptions_WildcardAmongOthers(t *testing.T) {
	opts := buildAcceptOptions([]string{"http://example.com", "*"})
	if !opts.InsecureSkipVerify {
		t.Error("expected InsecureSkipVerify to be true when wildcard is among origins")
	}
}
