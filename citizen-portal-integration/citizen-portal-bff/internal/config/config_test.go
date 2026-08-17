package config

import (
	"testing"
	"time"
)

func lookupFrom(m map[string]string) LookupFunc {
	return func(key string) (string, bool) {
		v, ok := m[key]
		return v, ok
	}
}

func validEnv() map[string]string {
	return map[string]string{
		"PORTAL_CLIENT_ID":     "portal-client-id",
		"PORTAL_CLIENT_SECRET": "portal-client-secret",
		"DL_CLIENT_ID":         "dl-client-id",
		"DL_CLIENT_SECRET":     "dl-client-secret",
		"VRL_CLIENT_ID":        "vrl-client-id",
		"VRL_CLIENT_SECRET":    "vrl-client-secret",
	}
}

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load(lookupFrom(validEnv()))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.BFFPort != "8090" {
		t.Errorf("BFFPort = %q, want 8090", cfg.BFFPort)
	}
	if cfg.BFFPublicURL != "http://localhost:8090" {
		t.Errorf("BFFPublicURL = %q", cfg.BFFPublicURL)
	}
	if cfg.ISIssuer != "https://localhost:9443/oauth2/token" {
		t.Errorf("ISIssuer = %q", cfg.ISIssuer)
	}
	if cfg.Portal.ClientID != "portal-client-id" || cfg.Portal.ClientSecret != "portal-client-secret" {
		t.Errorf("Portal client = %+v", cfg.Portal)
	}
	if cfg.Portal.Scopes != "openid profile email" {
		t.Errorf("Portal.Scopes = %q", cfg.Portal.Scopes)
	}
	if cfg.DrivingLicence.ClientID != "dl-client-id" || cfg.DrivingLicence.ClientSecret != "dl-client-secret" {
		t.Errorf("DrivingLicence client = %+v", cfg.DrivingLicence)
	}
	if cfg.DrivingLicence.Scopes != "openid profile email address" {
		t.Errorf("DrivingLicence.Scopes = %q", cfg.DrivingLicence.Scopes)
	}
	if cfg.RevenueLicence.ClientID != "vrl-client-id" || cfg.RevenueLicence.ClientSecret != "vrl-client-secret" {
		t.Errorf("RevenueLicence client = %+v", cfg.RevenueLicence)
	}
	if cfg.RevenueLicence.Scopes != "openid profile email" {
		t.Errorf("RevenueLicence.Scopes = %q", cfg.RevenueLicence.Scopes)
	}
	if cfg.SessionIdleTimeout != 60*time.Minute {
		t.Errorf("SessionIdleTimeout = %v, want 60m", cfg.SessionIdleTimeout)
	}
	if cfg.LoginTxnTTL != 5*time.Minute {
		t.Errorf("LoginTxnTTL = %v, want 5m", cfg.LoginTxnTTL)
	}
	if cfg.SessionMaxEntries != 5000 {
		t.Errorf("SessionMaxEntries = %d, want 5000", cfg.SessionMaxEntries)
	}
	if cfg.CookieSecure {
		t.Error("CookieSecure should default to false for an http:// public URL")
	}
	if !cfg.LogSanitize {
		t.Error("LogSanitize should default to true")
	}
}

func TestLoadMissingClientIDFailsFast(t *testing.T) {
	env := validEnv()
	delete(env, "PORTAL_CLIENT_ID")
	if _, err := Load(lookupFrom(env)); err == nil {
		t.Fatal("expected Load to fail without PORTAL_CLIENT_ID")
	}
}

func TestLoadMissingClientSecretFailsFast(t *testing.T) {
	env := validEnv()
	delete(env, "PORTAL_CLIENT_SECRET")
	if _, err := Load(lookupFrom(env)); err == nil {
		t.Fatal("expected Load to fail without PORTAL_CLIENT_SECRET")
	}
}

func TestLoadMissingDLClientIDFailsFast(t *testing.T) {
	env := validEnv()
	delete(env, "DL_CLIENT_ID")
	if _, err := Load(lookupFrom(env)); err == nil {
		t.Fatal("expected Load to fail without DL_CLIENT_ID")
	}
}

func TestLoadMissingDLClientSecretFailsFast(t *testing.T) {
	env := validEnv()
	delete(env, "DL_CLIENT_SECRET")
	if _, err := Load(lookupFrom(env)); err == nil {
		t.Fatal("expected Load to fail without DL_CLIENT_SECRET")
	}
}

func TestLoadMissingVRLClientIDFailsFast(t *testing.T) {
	env := validEnv()
	delete(env, "VRL_CLIENT_ID")
	if _, err := Load(lookupFrom(env)); err == nil {
		t.Fatal("expected Load to fail without VRL_CLIENT_ID")
	}
}

func TestLoadMissingVRLClientSecretFailsFast(t *testing.T) {
	env := validEnv()
	delete(env, "VRL_CLIENT_SECRET")
	if _, err := Load(lookupFrom(env)); err == nil {
		t.Fatal("expected Load to fail without VRL_CLIENT_SECRET")
	}
}

func TestLoadEmptyClientSecretFailsFast(t *testing.T) {
	env := validEnv()
	env["PORTAL_CLIENT_SECRET"] = ""
	if _, err := Load(lookupFrom(env)); err == nil {
		t.Fatal("expected Load to fail with an empty PORTAL_CLIENT_SECRET")
	}
}

func TestLoadCookieSecureInferredFromHTTPS(t *testing.T) {
	env := validEnv()
	env["BFF_PUBLIC_URL"] = "https://portal.example.gov"
	cfg, err := Load(lookupFrom(env))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.CookieSecure {
		t.Error("CookieSecure should default to true for an https:// public URL")
	}
}

func TestLoadCookieSecureExplicitOverride(t *testing.T) {
	env := validEnv()
	env["BFF_PUBLIC_URL"] = "https://portal.example.gov"
	env["COOKIE_SECURE"] = "false"
	cfg, err := Load(lookupFrom(env))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.CookieSecure {
		t.Error("explicit COOKIE_SECURE=false should override the https inference")
	}
}

func TestLoadRejectsInvalidPublicURL(t *testing.T) {
	env := validEnv()
	env["BFF_PUBLIC_URL"] = "not a url"
	if _, err := Load(lookupFrom(env)); err == nil {
		t.Fatal("expected Load to reject a malformed BFF_PUBLIC_URL")
	}
}

func TestLoadRejectsInvalidDuration(t *testing.T) {
	env := validEnv()
	env["SESSION_IDLE_TIMEOUT"] = "not-a-duration"
	if _, err := Load(lookupFrom(env)); err == nil {
		t.Fatal("expected Load to reject a malformed duration")
	}
}

func TestLoadRejectsNonPositiveSessionMaxEntries(t *testing.T) {
	env := validEnv()
	env["SESSION_MAX_ENTRIES"] = "0"
	if _, err := Load(lookupFrom(env)); err == nil {
		t.Fatal("expected Load to reject SESSION_MAX_ENTRIES=0")
	}
}

func TestLoadOverridesEverythingFromEnv(t *testing.T) {
	env := validEnv()
	env["BFF_PORT"] = "9090"
	env["BFF_PUBLIC_URL"] = "http://localhost:9090"
	env["IS_ISSUER"] = "https://is.example.gov/oauth2/token"
	env["PORTAL_CLIENT_SCOPES"] = "openid profile"
	env["DL_CLIENT_SCOPES"] = "openid extra_test_scope_a"
	env["VRL_CLIENT_SCOPES"] = "openid extra_test_scope_b"
	env["SESSION_IDLE_TIMEOUT"] = "30m"
	env["LOGIN_TXN_TTL"] = "2m"
	env["SESSION_MAX_ENTRIES"] = "1000"
	env["DEV_PROXY_TARGET"] = "http://localhost:5173"

	cfg, err := Load(lookupFrom(env))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.BFFPort != "9090" || cfg.ISIssuer != "https://is.example.gov/oauth2/token" {
		t.Errorf("unexpected cfg: %+v", cfg)
	}
	if cfg.Portal.Scopes != "openid profile" {
		t.Errorf("Portal.Scopes = %q", cfg.Portal.Scopes)
	}
	if cfg.DrivingLicence.Scopes != "openid extra_test_scope_a" {
		t.Errorf("DrivingLicence.Scopes = %q", cfg.DrivingLicence.Scopes)
	}
	if cfg.RevenueLicence.Scopes != "openid extra_test_scope_b" {
		t.Errorf("RevenueLicence.Scopes = %q", cfg.RevenueLicence.Scopes)
	}
	if cfg.SessionIdleTimeout != 30*time.Minute || cfg.LoginTxnTTL != 2*time.Minute {
		t.Errorf("unexpected timeouts: %+v", cfg)
	}
	if cfg.SessionMaxEntries != 1000 {
		t.Errorf("SessionMaxEntries = %d", cfg.SessionMaxEntries)
	}
	if cfg.DevProxyTarget != "http://localhost:5173" {
		t.Errorf("DevProxyTarget = %q", cfg.DevProxyTarget)
	}
}
