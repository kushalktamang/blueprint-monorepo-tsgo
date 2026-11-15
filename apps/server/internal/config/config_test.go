package config

import "testing"

func TestNormalizeEnvironmentKey(t *testing.T) {
	tests := map[string]string{
		"BLUEPRINT_PRIMARY_ENV":                           "primary.env",
		"BLUEPRINT_SERVER_CORS_ALLOWED_ORIGINS":           "server.cors_allowed_origins",
		"BLUEPRINT_OBSERVABILITY_NEW_RELIC_LICENSE_KEY":   "observability.new_relic.license_key",
		"BLUEPRINT_OBSERVABILITY.NEW_RELIC.DEBUG_LOGGING": "observability.new_relic.debug_logging",
		"BLUEPRINT_OBSERVABILITY_HEALTH_CHECKS_TIMEOUT":   "observability.health_checks.timeout",
		"BLUEPRINT_DATABASE_CONN_MAX_IDLE_TIME":           "database.conn_max_idle_time",
	}

	for input, want := range tests {
		if got := normalizeEnvironmentKey(input); got != want {
			t.Errorf("normalizeEnvironmentKey(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestParseList(t *testing.T) {
	got := parseList("database, redis health")
	want := []string{"database", "redis", "health"}

	if len(got) != len(want) {
		t.Fatalf("parseList() returned %d values, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("parseList()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
