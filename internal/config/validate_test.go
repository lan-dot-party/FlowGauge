package config

import (
	"reflect"
	"testing"
)

func TestParseTrustedProxies(t *testing.T) {
	got, err := ParseTrustedProxies([]string{"127.0.0.1", " ::1 ", "10.0.0.0/8", "172.18.5.1/16", "::ffff:192.0.2.1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"127.0.0.1/32", "::1/128", "10.0.0.0/8", "172.18.0.0/16", "192.0.2.1/32"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	for _, bad := range []string{"", "localhost", "10.0.0.0/33", "1.2.3"} {
		if _, err := ParseTrustedProxies([]string{bad}); err == nil {
			t.Errorf("entry %q: expected error", bad)
		}
	}
}

func TestValidateRejectsBadTrustedProxy(t *testing.T) {
	cfg := NewDefault()
	cfg.Connections = []ConnectionConfig{{Name: "WAN1", Enabled: true}}
	if err := Validate(cfg); err != nil {
		t.Fatalf("baseline config invalid: %v", err)
	}
	cfg.Webserver.TrustedProxies = []string{"nginx"}
	if err := Validate(cfg); err == nil {
		t.Error("expected error for hostname in trusted_proxies")
	}
}

func TestValidateAuthNeedsBothFields(t *testing.T) {
	for _, tc := range []struct {
		auth    *AuthConfig
		wantErr bool
	}{
		{nil, false},
		{&AuthConfig{}, false},
		{&AuthConfig{Username: "admin", Password: "secret"}, false},
		{&AuthConfig{Username: "admin"}, true},
		{&AuthConfig{Password: "secret"}, true},
	} {
		cfg := NewDefault()
		cfg.Connections = []ConnectionConfig{{Name: "WAN1", Enabled: true}}
		cfg.Webserver.Auth = tc.auth
		if err := Validate(cfg); (err != nil) != tc.wantErr {
			t.Errorf("auth %+v: err = %v, wantErr %v", tc.auth, err, tc.wantErr)
		}
	}
}

func TestValidatePostgresSSLMode(t *testing.T) {
	for mode, wantErr := range map[string]bool{"disable": false, "verify-full": false, "": false, "off": true, "require ": true} {
		cfg := NewDefault()
		cfg.Connections = []ConnectionConfig{{Name: "WAN1", Enabled: true}}
		cfg.Storage.Type = "postgres"
		cfg.Storage.Postgres.Host = "db"
		cfg.Storage.Postgres.Database = "fg"
		cfg.Storage.Postgres.SSLMode = mode
		if err := Validate(cfg); (err != nil) != wantErr {
			t.Errorf("ssl_mode %q: err = %v, wantErr %v", mode, err, wantErr)
		}
	}
}
