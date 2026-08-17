// Package oidcrp is the BFF's OpenID Connect relying-party layer: discovery
// against WSO2 IS, the authorization-code + PKCE round trip, ID-token
// verification, and back-channel logout-token verification. It is the only
// package that imports go-oidc/oauth2 — everything above it deals in plain
// Go values (session.AuthSession, claims), never library types.
package oidcrp

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"time"
)

// NewHTTPClient returns an http.Client that trusts only the certificate in
// caFile (PEM), in addition to nothing else — not the system root store —
// for talking to WSO2 IS's self-signed local certificate. This is the only
// place TLS verification is ever relaxed in this codebase, and it is never
// relaxed: caFile is loaded as an *additional* explicit trust anchor, never
// as InsecureSkipVerify. If caFile is empty, the system root store is used
// unmodified (for a deployment behind a certificate issued by a public CA).
func NewHTTPClient(caFile string) (*http.Client, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()

	if caFile != "" {
		// caFile is deployment-time configuration (IS_CA_FILE), never
		// end-user input — the same justification setup-without-bridge's
		// spotbugs-exclude.xml gives its one PATH_TRAVERSAL_IN exclusion,
		// for the operator-run JwkExporter CLI's keystore path.
		pem, err := os.ReadFile(caFile) // #nosec G304 -- operator-supplied config path, not user input

		if err != nil {
			return nil, fmt.Errorf("oidcrp: reading IS CA file %q: %w", caFile, err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("oidcrp: %q contains no usable PEM certificate", caFile)
		}
		transport.TLSClientConfig = &tls.Config{
			RootCAs:    pool,
			MinVersion: tls.VersionTLS12,
		}
	}

	return &http.Client{
		Transport: transport,
		Timeout:   15 * time.Second,
	}, nil
}
