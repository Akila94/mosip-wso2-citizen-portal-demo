// Package config loads gov-services-api's configuration from environment
// variables only — there is no config file, matching the convention set by
// citizen-portal-bff's internal/config. This service holds no secrets: it
// never authenticates to anything else, it only validates tokens presented
// to it, so the only configuration it truly cannot start without is the
// three OAuth2 client IDs it treats as per-router required audiences — a
// missing one must fail loudly at boot, not obscurely reject every request
// to that router at demo time.
package config

import (
	"fmt"
	"strings"
)

// Config is gov-services-api's fully validated configuration.
type Config struct {
	ServicesPort string

	ISIssuer string
	ISCAFile string

	// PortalClientID is the expected audience for the /portal/* router.
	PortalClientID string
	// DrivingLicenceClientID is the expected audience for the
	// /driving-licence/* router.
	DrivingLicenceClientID string
	// RevenueLicenceClientID is the expected audience for the
	// /vehicle-registry/* router.
	RevenueLicenceClientID string

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

// Load builds a Config from lookup, applying defaults and failing with a
// single aggregated error listing every problem found, so a misconfigured
// deployment reports all of its mistakes at once rather than one per
// restart.
func Load(lookup LookupFunc) (Config, error) {
	l := &loader{lookup: lookup}

	cfg := Config{
		ServicesPort: l.str("SERVICES_PORT", "8091"),

		ISIssuer: l.str("IS_ISSUER", "https://localhost:9443/oauth2/token"),
		ISCAFile: l.str("IS_CA_FILE", "../certs/wso2is-local.pem"),

		PortalClientID:         l.required("PORTAL_CLIENT_ID"),
		DrivingLicenceClientID: l.required("DL_CLIENT_ID"),
		RevenueLicenceClientID: l.required("VRL_CLIENT_ID"),

		LogLevel: l.str("LOG_LEVEL", "info"),
	}

	if len(l.errs) > 0 {
		return Config{}, fmt.Errorf("invalid configuration:\n  - %s", strings.Join(l.errs, "\n  - "))
	}
	return cfg, nil
}
