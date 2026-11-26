package config

import (
	"strings"
	"testing"
)

// The fail-fast contract from docs/06-deployment.md: report every missing variable at
// once. Nothing is required at T-05, so this exercises the mechanism directly rather
// than through Load — T-06 and T-08 populate the real list.
func TestCollectMissing(t *testing.T) {
	tests := []struct {
		name     string
		required map[string]string
		want     []string
	}{
		{"nothing required", map[string]string{}, nil},
		{"all present", map[string]string{"A": "x", "B": "y"}, nil},
		{"one missing", map[string]string{"A": "x", "B": ""}, []string{"B"}},
		{
			// The important case. Failing on the first would mean three deploys to
			// discover three problems.
			name:     "several missing, reported together and sorted",
			required: map[string]string{"C": "", "A": "", "B": "set", "D": ""},
			want:     []string{"A", "C", "D"},
		},
		{"whitespace counts as missing", map[string]string{"A": "   "}, []string{"A"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := collectMissing(tt.required)
			if len(got) != len(tt.want) {
				t.Fatalf("collectMissing() = %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("collectMissing()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestLoadDefaults(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("ENV", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Port != "8080" {
		t.Errorf("Port = %q, want 8080", cfg.Port)
	}
	if !cfg.IsDevelopment() {
		t.Errorf("Env = %q, want development by default", cfg.Env)
	}
}

func TestLoadReadsPort(t *testing.T) {
	// Fly injects PORT at runtime, so this must be read rather than hardcoded.
	t.Setenv("PORT", "9999")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Port != "9999" {
		t.Errorf("Port = %q, want 9999", cfg.Port)
	}
}

func TestLoadRejectsUnknownEnv(t *testing.T) {
	// A typo'd ENV must not silently fall through to development behaviour in
	// production — that would enable dev CORS on a live deploy.
	t.Setenv("ENV", "prod")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want an error for ENV=prod")
	}
	if !strings.Contains(err.Error(), "ENV") {
		t.Errorf("error %q does not name the offending variable", err)
	}
}

func TestIsDevelopment(t *testing.T) {
	if (Config{Env: EnvProduction}).IsDevelopment() {
		t.Error("production reported as development — dev CORS would ship to prod")
	}
	if !(Config{Env: EnvDevelopment}).IsDevelopment() {
		t.Error("development not reported as development")
	}
}
