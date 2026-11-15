package database

import (
	"net"
	"net/url"
	"strconv"

	"github.com/kushalktamang/blueprint-monorepo-tsgo/internal/config"
)

// buildDSN returns cfg.URL if set, otherwise assembles a postgres URL
// from the individual fields. net/url handles escaping.
func buildDSN(cfg *config.DatabaseConfig) string {
	if cfg.URL != "" {
		return cfg.URL
	}
	url := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(cfg.User, cfg.Password),
		Host:     net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port)),
		Path:     "/" + cfg.Name,
		RawQuery: url.Values{"sslmode": {cfg.SSLMode}}.Encode(),
	}
	return url.String()
}
