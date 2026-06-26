package config

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

func init() {
	appEnv := os.Getenv("APP_ENV")
	if strings.EqualFold(appEnv, "production") {
		return
	}
	if _, err := os.Stat(".env"); err == nil {
		loadEnvFile(".env")
	}
}

func loadEnvFile(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		val = strings.Trim(val, `"'`)
		if os.Getenv(key) == "" {
			os.Setenv(key, val)
		}
	}
}

type Config struct {
	Port                   string
	DatabaseURL            string
	JWTSecret              string
	AllowedOrigins         []string
	RedisURL               string
	IntrospectionRateLimit int
	S3Endpoint             string
	S3Region               string
	S3Bucket               string
	S3AccessKey            string
	S3SecretKey            string
	AppEnv                 string
	GoogleClientID         string
	GoogleClientSecret     string
	GoogleRedirectURL      string
	FrontendURL            string
	SMTPHost               string
	SMTPPort               int
	SMTPUsername           string
	SMTPPassword           string
	SMTPFrom               string
	SMTPSkipVerify         bool
	CSRFEnabled            bool
	WebhookEncryptionKey   string
}

func Load() *Config {
	appEnv := os.Getenv("APP_ENV")
	if appEnv == "" {
		appEnv = "development"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL must be set")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET must be set")
	}

	originsEnv := os.Getenv("ALLOWED_ORIGINS")
	var origins []string
	if originsEnv == "" {
		log.Fatal("ALLOWED_ORIGINS must be set (comma-separated)")
	}
	rawOrigins := strings.Split(originsEnv, ",")
	for _, o := range rawOrigins {
		trimmed := strings.TrimSpace(o)
		if trimmed != "" {
			origins = append(origins, trimmed)
		}
	}

	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		log.Fatal("FRONTEND_URL must be set")
	}

	smtpPort := loadIntEnv("SMTP_PORT", 587)
	smtpSkipVerify := os.Getenv("SMTP_SKIP_VERIFY") == "true"

	csrfEnabled := os.Getenv("CSRF_ENABLED") != "false"
	webhookEncryptionKey := os.Getenv("WEBHOOK_ENCRYPTION_KEY")

	return &Config{
		Port:                   port,
		DatabaseURL:            dbURL,
		JWTSecret:              jwtSecret,
		AllowedOrigins:         origins,
		RedisURL:               os.Getenv("REDIS_URL"),
		IntrospectionRateLimit: loadIntEnv("INTROSPECTION_RATE_LIMIT", 100),
		S3Endpoint:             os.Getenv("S3_ENDPOINT"),
		S3Region:               os.Getenv("S3_REGION"),
		S3Bucket:               os.Getenv("S3_BUCKET"),
		S3AccessKey:            os.Getenv("S3_ACCESS_KEY"),
		S3SecretKey:            os.Getenv("S3_SECRET_KEY"),
		AppEnv:                 appEnv,
		GoogleClientID:         os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret:     os.Getenv("GOOGLE_CLIENT_SECRET"),
		GoogleRedirectURL:      os.Getenv("GOOGLE_REDIRECT_URL"),
		FrontendURL:            frontendURL,
		SMTPHost:               os.Getenv("SMTP_HOST"),
		SMTPPort:               smtpPort,
		SMTPUsername:           os.Getenv("SMTP_USERNAME"),
		SMTPPassword:           os.Getenv("SMTP_PASSWORD"),
		SMTPFrom:               os.Getenv("SMTP_FROM"),
		SMTPSkipVerify:         smtpSkipVerify,
		CSRFEnabled:            csrfEnabled,
		WebhookEncryptionKey:   webhookEncryptionKey,
	}
}

func (c *Config) Validate() error {
	var errs []string

	if len(c.JWTSecret) < 32 {
		errs = append(errs, "JWT_SECRET must be at least 32 characters")
	}

	if len(c.AllowedOrigins) == 0 || (len(c.AllowedOrigins) == 1 && c.AllowedOrigins[0] == "*") {
		errs = append(errs, "ALLOWED_ORIGINS must be explicitly set (cannot be '*')")
	}

	if c.S3Endpoint != "" && (c.S3AccessKey == "" || c.S3SecretKey == "") {
		errs = append(errs, "S3_ACCESS_KEY and S3_SECRET_KEY must be set when S3_ENDPOINT is provided")
	}

	if c.SMTPHost != "" && c.SMTPFrom == "" {
		errs = append(errs, "SMTP_FROM must be set when SMTP_HOST is provided")
	}

	if c.SMTPSkipVerify && c.AppEnv == "production" {
		errs = append(errs, "SMTP_SKIP_VERIFY must not be true in production")
	}

	if c.AppEnv == "production" && !strings.Contains(c.DatabaseURL, "sslmode=require") && !strings.Contains(c.DatabaseURL, "sslmode=verify-full") {
		errs = append(errs, "DATABASE_URL must use sslmode=require or sslmode=verify-full in production")
	}

	if len(errs) > 0 {
		return fmt.Errorf("config validation failed:\n  - %s", strings.Join(errs, "\n  - "))
	}
	return nil
}

func loadIntEnv(key string, defaultVal int) int {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return defaultVal
	}
	return n
}
