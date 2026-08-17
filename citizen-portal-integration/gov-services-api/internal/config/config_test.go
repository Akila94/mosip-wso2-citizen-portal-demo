package config

import (
	"strings"
	"testing"
)

func lookupFrom(m map[string]string) LookupFunc {
	return func(key string) (string, bool) {
		v, ok := m[key]
		return v, ok
	}
}

func validEnv() map[string]string {
	return map[string]string{
		"PORTAL_CLIENT_ID": "portal-client-id",
		"DL_CLIENT_ID":     "dl-client-id",
		"VRL_CLIENT_ID":    "vrl-client-id",
	}
}

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load(lookupFrom(validEnv()))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.ServicesPort != "8091" {
		t.Errorf("ServicesPort = %q, want 8091", cfg.ServicesPort)
	}
	if cfg.ISIssuer != "https://localhost:9443/oauth2/token" {
		t.Errorf("ISIssuer = %q", cfg.ISIssuer)
	}
	if cfg.ISCAFile != "../certs/wso2is-local.pem" {
		t.Errorf("ISCAFile = %q", cfg.ISCAFile)
	}
	if cfg.PortalClientID != "portal-client-id" {
		t.Errorf("PortalClientID = %q", cfg.PortalClientID)
	}
	if cfg.DrivingLicenceClientID != "dl-client-id" {
		t.Errorf("DrivingLicenceClientID = %q", cfg.DrivingLicenceClientID)
	}
	if cfg.RevenueLicenceClientID != "vrl-client-id" {
		t.Errorf("RevenueLicenceClientID = %q", cfg.RevenueLicenceClientID)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want info", cfg.LogLevel)
	}
}

func TestLoadMissingPortalClientIDFailsFast(t *testing.T) {
	env := validEnv()
	delete(env, "PORTAL_CLIENT_ID")
	if _, err := Load(lookupFrom(env)); err == nil {
		t.Fatal("expected Load to fail without PORTAL_CLIENT_ID")
	}
}

func TestLoadMissingDLClientIDFailsFast(t *testing.T) {
	env := validEnv()
	delete(env, "DL_CLIENT_ID")
	if _, err := Load(lookupFrom(env)); err == nil {
		t.Fatal("expected Load to fail without DL_CLIENT_ID")
	}
}

func TestLoadMissingVRLClientIDFailsFast(t *testing.T) {
	env := validEnv()
	delete(env, "VRL_CLIENT_ID")
	if _, err := Load(lookupFrom(env)); err == nil {
		t.Fatal("expected Load to fail without VRL_CLIENT_ID")
	}
}

func TestLoadEmptyClientIDFailsFast(t *testing.T) {
	env := validEnv()
	env["PORTAL_CLIENT_ID"] = ""
	if _, err := Load(lookupFrom(env)); err == nil {
		t.Fatal("expected Load to fail with an empty PORTAL_CLIENT_ID")
	}
}

func TestLoadAggregatesAllMissingClientIDs(t *testing.T) {
	_, err := Load(lookupFrom(map[string]string{}))
	if err == nil {
		t.Fatal("expected Load to fail with no env set")
	}
	msg := err.Error()
	for _, want := range []string{"PORTAL_CLIENT_ID", "DL_CLIENT_ID", "VRL_CLIENT_ID"} {
		if !strings.Contains(msg, want) {
			t.Errorf("aggregated error %q does not mention %s", msg, want)
		}
	}
}

func TestLoadOverridesEverythingFromEnv(t *testing.T) {
	env := validEnv()
	env["SERVICES_PORT"] = "9091"
	env["IS_ISSUER"] = "https://is.example.gov/oauth2/token"
	env["IS_CA_FILE"] = "/etc/certs/is.pem"
	env["LOG_LEVEL"] = "debug"

	cfg, err := Load(lookupFrom(env))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.ServicesPort != "9091" {
		t.Errorf("ServicesPort = %q", cfg.ServicesPort)
	}
	if cfg.ISIssuer != "https://is.example.gov/oauth2/token" {
		t.Errorf("ISIssuer = %q", cfg.ISIssuer)
	}
	if cfg.ISCAFile != "/etc/certs/is.pem" {
		t.Errorf("ISCAFile = %q", cfg.ISCAFile)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel = %q", cfg.LogLevel)
	}
}
