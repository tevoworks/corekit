package iam

import (
	"testing"
)

func TestGeneratePKCEPair(t *testing.T) {
	verifier, challenge, err := generatePKCEPair()
	if err != nil {
		t.Fatalf("generatePKCEPair: %v", err)
	}
	if verifier == "" {
		t.Fatal("expected non-empty code verifier")
	}
	if challenge == "" {
		t.Fatal("expected non-empty code challenge")
	}
	if len(verifier) < 43 {
		t.Fatalf("expected code verifier length >= 43, got: %d", len(verifier))
	}
	if len(challenge) < 43 {
		t.Fatalf("expected code challenge length >= 43, got: %d", len(challenge))
	}

	verifier2, challenge2, err := generatePKCEPair()
	if err != nil {
		t.Fatalf("generatePKCEPair: %v", err)
	}
	if verifier == verifier2 {
		t.Fatal("expected different code verifier each call")
	}
	if challenge == challenge2 {
		t.Fatal("expected different code challenge each call")
	}
}

func TestIsValidEmailURL(t *testing.T) {
	tests := []struct {
		url   string
		valid bool
	}{
		{"", true},
		{"https://example.com/verify?token=abc", true},
		{"http://example.com/verify", true},
		{"javascript:alert(1)", false},
		{"data:text/html,<script>alert(1)</script>", false},
		{"ftp://files.example.com", false},
		{"file:///etc/passwd", false},
		{" //evil.com", false},
	}
	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got := isValidEmailURL(tt.url)
			if got != tt.valid {
				t.Errorf("isValidEmailURL(%q) = %v, want %v", tt.url, got, tt.valid)
			}
		})
	}
}

func TestExtractURL(t *testing.T) {
	tests := []struct {
		body string
		want string
	}{
		{"no url here", ""},
		{"http://example.com", "http://example.com"},
		{"visit https://example.com/path now", "https://example.com/path"},
		{"before http://short.link after", "http://short.link"},
		{"https://example.com/path?q=1&r=2 end", "https://example.com/path?q=1&r=2"},
	}
	for _, tt := range tests {
		t.Run(tt.body[:min(len(tt.body), 20)], func(t *testing.T) {
			got := extractURL(tt.body)
			if got != tt.want {
				t.Errorf("extractURL(%q) = %q, want %q", tt.body, got, tt.want)
			}
		})
	}
}

func TestIsSessionRecent(t *testing.T) {
	if isSessionRecent(nil, nil, "") {
		t.Fatal("expected false for empty tokenID")
	}
}
