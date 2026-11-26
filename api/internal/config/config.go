// Package config loads the service's configuration from the environment.
package config

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

// Config is the fully-resolved configuration. Every other package receives what it
// needs as parameters; only cmd/api/main.go sees this whole struct.
type Config struct {
	Port string
	Env  string

	// DatabaseURL is empty until T-06. Required from then on.
	DatabaseURL string

	// Supabase JWT verification. Required from T-08.
	SupabaseJWKSURL string
	SupabaseJWTIss  string
	SupabaseJWTAud  string
}

const (
	EnvDevelopment = "development"
	EnvProduction  = "production"
)

// IsDevelopment reports whether development-only behaviour should be enabled —
// currently CORS for the Expo web dev server, and human-readable logs.
func (c Config) IsDevelopment() bool { return c.Env == EnvDevelopment }

// Load reads configuration from the environment.
//
// It collects every problem before returning, rather than failing on the first one. A
// deploy that is missing three variables should say so once, not three times across
// three restarts.
func Load() (Config, error) {
	var problems []string

	cfg := Config{
		Port:            envOr("PORT", "8080"),
		Env:             envOr("ENV", EnvDevelopment),
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		SupabaseJWKSURL: os.Getenv("SUPABASE_JWKS_URL"),
		SupabaseJWTIss:  os.Getenv("SUPABASE_JWT_ISS"),
		SupabaseJWTAud:  os.Getenv("SUPABASE_JWT_AUD"),
	}

	// Nothing is required yet. T-06 adds {"DATABASE_URL": cfg.DatabaseURL} and T-08 the
	// three Supabase values — one line each, and the fail-fast behaviour comes for free.
	problems = collectMissing(map[string]string{})

	if cfg.Env != EnvDevelopment && cfg.Env != EnvProduction {
		return Config{}, fmt.Errorf(
			"config: ENV must be %q or %q, got %q", EnvDevelopment, EnvProduction, cfg.Env)
	}

	if len(problems) > 0 {
		return Config{}, fmt.Errorf(
			"config: missing required environment variables: %s", strings.Join(problems, ", "))
	}

	return cfg, nil
}

// collectMissing returns the names of every blank value, sorted.
//
// Every one of them, not just the first: a deploy missing three variables should say so
// once rather than across three restarts. Sorted because map iteration order in Go is
// deliberately randomised, and an error message that reorders itself between runs is
// hard to trust.
func collectMissing(required map[string]string) []string {
	var missing []string
	for name, value := range required {
		if strings.TrimSpace(value) == "" {
			missing = append(missing, name)
		}
	}
	sort.Strings(missing)
	return missing
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
