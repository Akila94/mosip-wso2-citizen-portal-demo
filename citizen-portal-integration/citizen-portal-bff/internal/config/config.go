// Package config loads the BFF's configuration from environment variables
// only — there is no config file, matching the convention already set by
// esignet-bridge/server.js. Every value has a working localhost default
// except the three OIDC client credentials, which the process refuses to
// start without: a missing client secret must fail loudly at boot, not
// obscurely inside a redirect handler at demo time.
package config

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// ClientConfig is one app's registered OIDC client.
type ClientConfig struct {
	ClientID     string
	ClientSecret string
	Scopes       string
}

// Config is the BFF's fully validated configuration. M1 registered only the
// Citizen Portal app (Config.Portal); M2 adds Application A and B —
// DrivingLicence and RevenueLicence — as their own ClientConfig fields,
// loaded with their own env-var prefixes, following the same pattern.
type Config struct {
	BFFPort      string
	BFFPublicURL string

	ISIssuer string
	ISCAFile string

	Portal         ClientConfig
	DrivingLicence ClientConfig
	RevenueLicence ClientConfig

	ServicesAPIURL string

	SessionIdleTimeout time.Duration
	SessionMaxEntries  int
	LoginTxnTTL        time.Duration

	DevProxyTarget string
	StaticDir      string

	CookieSecure bool
	LogSanitize  bool

	LogLevel string
}

// LookupFunc mirrors os.LookupEnv's signature, so tests can substitute a
// fake environment without process-global state.
type LookupFunc func(key string) (string, bool)

type loader struct {
	lookup LookupFunc
	errs   []string
}

func (l *loader) str(key, def string) string {
	if v, ok := l.lookup(key); ok {
		return v
	}
	return def
}

func (l *loader) required(key string) string {
	v, ok := l.lookup(key)
	if !ok || v == "" {
		l.errs = append(l.errs, fmt.Sprintf("%s is required and must not be empty", key))
		return ""
	}
	return v
}

func (l *loader) durationOrDefault(key string, def time.Duration) time.Duration {
	v, ok := l.lookup(key)
	if !ok || v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		l.errs = append(l.errs, fmt.Sprintf("%s: invalid duration %q: %v", key, v, err))
		return def
	}
	return d
}

func (l *loader) intOrDefault(key string, def int) int {
	v, ok := l.lookup(key)
	if !ok || v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		l.errs = append(l.errs, fmt.Sprintf("%s: invalid integer %q: %v", key, v, err))
		return def
	}
	return n
}

func (l *loader) boolPtr(key string) *bool {
	v, ok := l.lookup(key)
	if !ok || v == "" {
		return nil
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		l.errs = append(l.errs, fmt.Sprintf("%s: invalid boolean %q: %v", key, v, err))
		return nil
	}
	return &b
}

// Load builds a Config from lookup, applying defaults and failing with a
// single aggregated error listing every problem found, so a misconfigured
// deployment reports all of its mistakes at once rather than one per
// restart.
func Load(lookup LookupFunc) (Config, error) {
	l := &loader{lookup: lookup}

	cfg := Config{
		BFFPort:      l.str("BFF_PORT", "8090"),
		BFFPublicURL: l.str("BFF_PUBLIC_URL", "http://localhost:8090"),

		ISIssuer: l.str("IS_ISSUER", "https://localhost:9443/oauth2/token"),
		ISCAFile: l.str("IS_CA_FILE", "../certs/wso2is-local.pem"),

		Portal: ClientConfig{
			ClientID:     l.required("PORTAL_CLIENT_ID"),
			ClientSecret: l.required("PORTAL_CLIENT_SECRET"),
			Scopes:       l.str("PORTAL_CLIENT_SCOPES", "openid profile email"),
		},
		DrivingLicence: ClientConfig{
			ClientID:     l.required("DL_CLIENT_ID"),
			ClientSecret: l.required("DL_CLIENT_SECRET"),
			Scopes:       l.str("DL_CLIENT_SCOPES", "openid profile email address driving_licence.write"),
		},
		RevenueLicence: ClientConfig{
			ClientID:     l.required("VRL_CLIENT_ID"),
			ClientSecret: l.required("VRL_CLIENT_SECRET"),
			Scopes:       l.str("VRL_CLIENT_SCOPES", "openid profile email vehicle_registry.read"),
		},

		ServicesAPIURL: l.str("SERVICES_API_URL", "http://localhost:8091"),

		SessionIdleTimeout: l.durationOrDefault("SESSION_IDLE_TIMEOUT", 60*time.Minute),
		SessionMaxEntries:  l.intOrDefault("SESSION_MAX_ENTRIES", 5000),
		LoginTxnTTL:        l.durationOrDefault("LOGIN_TXN_TTL", 5*time.Minute),

		DevProxyTarget: l.str("DEV_PROXY_TARGET", ""),
		StaticDir:      l.str("STATIC_DIR", "../../citizen-portal-demo-app/dist"),

		LogLevel: l.str("LOG_LEVEL", "info"),
	}

	publicURL, err := url.Parse(cfg.BFFPublicURL)
	if err != nil || publicURL.Scheme == "" || publicURL.Host == "" {
		l.errs = append(l.errs, fmt.Sprintf("BFF_PUBLIC_URL: invalid URL %q", cfg.BFFPublicURL))
	}

	if cfg.SessionMaxEntries <= 0 {
		l.errs = append(l.errs, "SESSION_MAX_ENTRIES must be a positive integer")
	}

	// COOKIE_SECURE defaults to whether the public URL is https, but an
	// explicit value always wins — needed for a TLS-terminating reverse
	// proxy in front of a plain-http origin.
	cfg.CookieSecure = publicURL != nil && publicURL.Scheme == "https"
	if override := l.boolPtr("COOKIE_SECURE"); override != nil {
		cfg.CookieSecure = *override
	}

	cfg.LogSanitize = true
	if override := l.boolPtr("LOG_SANITIZE"); override != nil {
		cfg.LogSanitize = *override
	}

	if len(l.errs) > 0 {
		return Config{}, fmt.Errorf("invalid configuration:\n  - %s", strings.Join(l.errs, "\n  - "))
	}
	return cfg, nil
}
