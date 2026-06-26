package config

import (
	"os"
	"testing"
)

func setEnv(key, val string) func() {
	old := os.Getenv(key)
	os.Setenv(key, val)
	return func() { os.Setenv(key, old) }
}

func TestLoadDefaults(t *testing.T) {
	cleanups := []func(){
		setEnv("DATABASE_URL", "postgres://localhost/test"),
		setEnv("JWT_SECRET", "this-is-a-32-char-secret-key-for-test!"),
		setEnv("ALLOWED_ORIGINS", "http://localhost:5173"),
		setEnv("FRONTEND_URL", "http://localhost:5173"),
		setEnv("APP_ENV", ""),
		setEnv("PORT", ""),
	}
	defer func() {
		for _, c := range cleanups {
			c()
		}
	}()

	cfg := Load()
	if cfg.Port != "8080" {
		t.Fatalf("default port should be 8080, got: %s", cfg.Port)
	}
	if cfg.AppEnv != "development" {
		t.Fatalf("expected development, got: %s", cfg.AppEnv)
	}
}

func TestLoadCustomPort(t *testing.T) {
	cleanups := []func(){
		setEnv("DATABASE_URL", "postgres://localhost/test"),
		setEnv("JWT_SECRET", "this-is-a-32-char-secret-key-for-test!"),
		setEnv("ALLOWED_ORIGINS", "http://localhost:5173"),
		setEnv("FRONTEND_URL", "http://localhost:5173"),
		setEnv("PORT", "9090"),
	}
	defer func() {
		for _, c := range cleanups {
			c()
		}
	}()

	cfg := Load()
	if cfg.Port != "9090" {
		t.Fatalf("expected 9090, got: %s", cfg.Port)
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name: "valid",
			cfg: &Config{
				JWTSecret:      "this-is-a-32-char-secret-key-for-test!",
				AllowedOrigins: []string{"http://localhost:5173"},
				FrontendURL:    "http://localhost:5173",
			},
			wantErr: false,
		},
		{
			name: "short jwt",
			cfg: &Config{
				JWTSecret:      "short",
				AllowedOrigins: []string{"http://localhost:5173"},
			},
			wantErr: true,
		},
		{
			name: "wildcard origins",
			cfg: &Config{
				JWTSecret:      "this-is-a-32-char-secret-key-for-test!",
				AllowedOrigins: []string{"*"},
			},
			wantErr: true,
		},
		{
			name: "empty origins",
			cfg: &Config{
				JWTSecret:      "this-is-a-32-char-secret-key-for-test!",
				AllowedOrigins: []string{},
			},
			wantErr: true,
		},
		{
			name: "s3 without keys",
			cfg: &Config{
				JWTSecret:      "this-is-a-32-char-secret-key-for-test!",
				AllowedOrigins: []string{"http://localhost:5173"},
				S3Endpoint:     "http://minio:9000",
				S3AccessKey:    "",
				S3SecretKey:    "",
			},
			wantErr: true,
		},
		{
			name: "smtp without from",
			cfg: &Config{
				JWTSecret:      "this-is-a-32-char-secret-key-for-test!",
				AllowedOrigins: []string{"http://localhost:5173"},
				SMTPHost:       "smtp.example.com",
				SMTPFrom:       "",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
		})
	}
}

func TestAllowedOriginsTrim(t *testing.T) {
	cleanups := []func(){
		setEnv("DATABASE_URL", "postgres://localhost/test"),
		setEnv("JWT_SECRET", "this-is-a-32-char-secret-key-for-test!"),
		setEnv("ALLOWED_ORIGINS", "http://a.com, http://b.com,http://c.com"),
		setEnv("FRONTEND_URL", "http://localhost:5173"),
	}
	defer func() {
		for _, c := range cleanups {
			c()
		}
	}()

	cfg := Load()
	if len(cfg.AllowedOrigins) != 3 {
		t.Fatalf("expected 3 origins, got: %d", len(cfg.AllowedOrigins))
	}
	if cfg.AllowedOrigins[1] != "http://b.com" {
		t.Fatalf("expected 'http://b.com', got: '%s'", cfg.AllowedOrigins[1])
	}
}

func TestLoadIntEnv(t *testing.T) {
	tests := []struct {
		name     string
		val      string
		def      int
		expected int
	}{
		{"set", "42", 10, 42},
		{"empty", "", 10, 10},
		{"invalid", "abc", 10, 10},
		{"negative", "-5", 10, 10},
		{"zero", "0", 10, 10},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := "TEST_INT_ENV"
			if tt.val != "" {
				os.Setenv(key, tt.val)
				defer os.Unsetenv(key)
			}
			got := loadIntEnv(key, tt.def)
			if got != tt.expected {
				t.Fatalf("expected %d, got: %d", tt.expected, got)
			}
		})
	}
}
