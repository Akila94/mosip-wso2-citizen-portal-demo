// Package httpclient provides the CA-pinned HTTP client this service uses
// for OIDC discovery and JWKS retrieval against WSO2 IS. Copied verbatim
// from citizen-portal-bff's internal/oidcrp/httpclient.go (that package
// cannot be imported across module boundaries) — this project's invariant
// is "no InsecureSkipVerify anywhere," and this is the one approved way to
// trust IS's self-signed local certificate.
package httpclient

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
		// end-user input — the same justification citizen-portal-bff's
		// httpclient.go gives for its own #nosec G304 exclusion.
		pem, err := os.ReadFile(caFile) // #nosec G304 -- operator-supplied config path, not user input

		if err != nil {
			return nil, fmt.Errorf("httpclient: reading IS CA file %q: %w", caFile, err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("httpclient: %q contains no usable PEM certificate", caFile)
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
