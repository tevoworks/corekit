package validation

import (
	"testing"
)

type nohtmlStruct struct {
	Name string `validate:"nohtml"`
}

type passwordStruct struct {
	Password string `validate:"password"`
}

type emailStruct struct {
	Email string `validate:"emailfmt"`
}

type urlStrictStruct struct {
	URL string `validate:"urlstrict"`
}

func TestNoHTML(t *testing.T) {
	v := NewEchoValidator()
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"plain text", "hello world", false},
		{"with <", "<script>", true},
		{"with >", "a > b", true},
		{"empty", "", false},
		{"safe chars", "foo_bar-baz", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := nohtmlStruct{Name: tt.value}
			err := v.Validate(&s)
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
		})
	}
}

func TestPassword(t *testing.T) {
	v := NewEchoValidator()
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"strong", "Abcd1234!", false},
		{"too short", "Ab1!", true},
		{"no upper", "abcd1234!", true},
		{"no lower", "ABCD1234!", true},
		{"no digit", "Abcdefgh!", true},
		{"no special", "Abcd1234", true},
		{"complex", "Str0ng!Pass", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := passwordStruct{Password: tt.value}
			err := v.Validate(&s)
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
		})
	}
}

func TestEmailFmt(t *testing.T) {
	v := NewEchoValidator()
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid", "test@example.com", false},
		{"valid plus", "test+tag@example.com", false},
		{"no domain", "test", true},
		{"no at", "testexample.com", true},
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := emailStruct{Email: tt.value}
			err := v.Validate(&s)
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
		})
	}
}

func TestURLStrict(t *testing.T) {
	v := NewEchoValidator()
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"valid https", "https://example.com", false},
		{"valid http", "http://example.com/path", false},
		{"ftp invalid", "ftp://example.com", true},
		{"no scheme", "example.com", true},
		{"no host", "http://", true},
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := urlStrictStruct{URL: tt.value}
			err := v.Validate(&s)
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
		})
	}
}

type multiField struct {
	Name  string `validate:"required,nohtml"`
	Email string `validate:"required,emailfmt"`
	Age   int    `validate:"min=0"`
}

func TestMultiField(t *testing.T) {
	v := NewEchoValidator()
	t.Run("valid", func(t *testing.T) {
		m := multiField{Name: "John", Email: "john@example.com", Age: 25}
		if err := v.Validate(&m); err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
	})
	t.Run("missing name", func(t *testing.T) {
		m := multiField{Name: "", Email: "john@example.com", Age: 25}
		if err := v.Validate(&m); err == nil {
			t.Fatal("expected error, got nil")
		}
	})
	t.Run("invalid email", func(t *testing.T) {
		m := multiField{Name: "John", Email: "bad", Age: 25}
		if err := v.Validate(&m); err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
