package storage

import (
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/lan-dot-party/flowgauge/internal/config"
)

func TestBuildDSNRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		cfg  config.PostgresConfig
	}{
		{"plain", config.PostgresConfig{Host: "db.example", Port: 5432, Database: "flowgauge", User: "fg", Password: "secret", SSLMode: "require"}},
		{"special chars", config.PostgresConfig{Host: "10.0.0.5", Port: 6543, Database: "flow gauge", User: "fg user", Password: "p@ss w=rd sslmode=disable '\"/?#", SSLMode: "verify-full"}},
		{"ipv6", config.PostgresConfig{Host: "::1", Port: 5432, Database: "fg", User: "fg", SSLMode: "disable"}},
		{"socket", config.PostgresConfig{Host: "/var/run/postgresql", Port: 5432, Database: "fg", User: "fg", SSLMode: "disable"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &PostgresStorage{cfg: tt.cfg}
			pc, err := pgconn.ParseConfig(s.buildDSN())
			if err != nil {
				t.Fatalf("ParseConfig: %v", err)
			}
			if pc.Host != tt.cfg.Host || int(pc.Port) != tt.cfg.Port || pc.Database != tt.cfg.Database ||
				pc.User != tt.cfg.User || pc.Password != tt.cfg.Password {
				t.Errorf("round trip mismatch: got host=%q port=%d db=%q user=%q pwOK=%v",
					pc.Host, pc.Port, pc.Database, pc.User, pc.Password == tt.cfg.Password)
			}
			wantTLS := tt.cfg.SSLMode != "disable"
			if (pc.TLSConfig != nil) != wantTLS {
				t.Errorf("TLS config present = %v, want %v (sslmode %q not applied)", pc.TLSConfig != nil, wantTLS, tt.cfg.SSLMode)
			}
		})
	}
}
