package config

import "testing"

func TestApplyDatabaseDefaults(t *testing.T) {
	cfg := DatabaseConfig{}
	applyDatabaseDefaults(&cfg)

	if cfg.MaxOpenConns != 25 || cfg.MaxIdleConns != 25 || cfg.ConnMaxLifetime != 300 || cfg.ConnMaxIdleTime != 300 {
		t.Fatalf("applyDatabaseDefaults() = %+v, want the standard pool defaults", cfg)
	}
}

func TestValidateDatabaseURL(t *testing.T) {
	if err := validateDatabaseURL("postgres://user:password@localhost:5432/blueprint?sslmode=disable"); err != nil {
		t.Fatalf("validateDatabaseURL() rejected a valid URL: %v", err)
	}
	if err := validateDatabaseURL("postgres://"); err == nil {
		t.Fatal("validateDatabaseURL() accepted an incomplete URL")
	}
}
