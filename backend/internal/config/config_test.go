package config

import "testing"

func TestLoadUsesPrivateDefaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "host=localhost dbname=trend_graph_test")
	t.Setenv("ADMIN_PASSWORD", "test-password")
	t.Setenv("DEEPSEEK_MODEL", "")
	t.Setenv("DEEPSEEK_BASE_URL", "")
	t.Setenv("ADMIN_SESSION_HOURS", "")
	t.Setenv("SESSION_COOKIE_SECURE", "")
	t.Setenv("COLLECTOR_DIR", "")

	cfg := Load()
	if cfg.DeepSeekModel != "deepseek-v4-pro" {
		t.Fatalf("model = %q, want deepseek-v4-pro", cfg.DeepSeekModel)
	}
	if cfg.DeepSeekBaseURL != "https://api.deepseek.com" {
		t.Fatalf("base URL = %q", cfg.DeepSeekBaseURL)
	}
	if cfg.AdminSessionHours != 168 || !cfg.SessionCookieSecure {
		t.Fatalf("unexpected private defaults: %#v", cfg)
	}
	if cfg.CollectorDir != "../services/collector" {
		t.Fatalf("collector dir = %q", cfg.CollectorDir)
	}
	if !cfg.DigestEnabled || !cfg.MajorAlertsEnabled {
		t.Fatalf("delivery defaults disabled: %#v", cfg)
	}
}
