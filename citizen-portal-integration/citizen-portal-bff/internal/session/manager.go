package session

import (
	"time"

	"github.com/akila94/mosip-wso2-citizen-portal-demo/citizen-portal-bff/internal/security"
)

// storeKeyBytes is the number of random bytes behind every login-transaction
// and session store key — 256 bits, matching the guideline's CSPRNG
// requirement for session identifiers (§1.25).
const storeKeyBytes = 32

// AssuranceLevel mirrors the SPA's existing `none | basic | substantial`
// model (citizen-portal-demo-app/src/services/types.go's AssuranceLevel),
// so the BFF's session projection slots into the UI with no shape change.
type AssuranceLevel string

const (
	AssuranceNone        AssuranceLevel = "none"
	AssuranceBasic       AssuranceLevel = "basic"
	AssuranceSubstantial AssuranceLevel = "substantial"
)

// esignetAMR is the `amr` value IS emits for the MOSIP eSignet federated
// authenticator (its Java class's simple name, per
// setup-without-bridge/.../EsignetOIDCAuthenticator.java). Any other amr
// value — including IS's built-in BasicAuthenticator — maps to "basic".
const esignetAMR = "EsignetOIDCAuthenticator"

// DeriveAssuranceLevel maps an ID token's `amr` claim to the SPA's
// assurance-level model. IS emits `acr` only when a value was actually
// resolved (verified against the shipped IS 7.3.0 source — see
// PORTAL-INTEGRATION-PLAN.md's appendix), so `amr` is the reliable signal
// and is used as the primary derivation with zero extra IS configuration.
func DeriveAssuranceLevel(amr []string) AssuranceLevel {
	for _, m := range amr {
		if m == esignetAMR {
			return AssuranceSubstantial
		}
	}
	return AssuranceBasic
}

// Config bounds and times out both stores the Manager owns.
type Config struct {
	MaxSessions  int
	MaxLoginTxns int // 0 -> reuse MaxSessions
	LoginTxnTTL  time.Duration
	IdleTimeout  time.Duration
}

// Manager owns the BFF's two server-side stores: short-lived login
// transactions and authenticated sessions. It is the only place that knows
// about store keys — callers deal in LoginTxn/AuthSession values and opaque
// key strings suitable for a cookie value.
type Manager struct {
	txns     *Store[LoginTxn]
	sessions *Store[AuthSession]
	cfg      Config
}

// NewManager constructs a Manager per cfg.
func NewManager(cfg Config) *Manager {
	maxTxns := cfg.MaxLoginTxns
	if maxTxns == 0 {
		maxTxns = cfg.MaxSessions
	}
	return &Manager{
		txns:     NewStore[LoginTxn](maxTxns),
		sessions: NewStore[AuthSession](cfg.MaxSessions),
		cfg:      cfg,
	}
}

// CreateLoginTxn stores txn under a fresh random key and returns that key,
// for use as an HttpOnly cookie value.
func (m *Manager) CreateLoginTxn(txn LoginTxn) (string, error) {
	key, err := security.RandomToken(storeKeyBytes)
	if err != nil {
		return "", err
	}
	if txn.CreatedAt.IsZero() {
		txn.CreatedAt = time.Now()
	}
	m.txns.Put(key, txn, m.cfg.LoginTxnTTL)
	return key, nil
}

// ConsumeLoginTxn retrieves and immediately deletes the transaction for key.
// A login transaction is single-use by construction: a replayed callback
// (the same "state" presented twice) always fails after the first use.
func (m *Manager) ConsumeLoginTxn(key string) (LoginTxn, bool) {
	txn, ok := m.txns.Get(key)
	if ok {
		m.txns.Delete(key)
	}
	return txn, ok
}

// CreateSession stores sess under a fresh random key and returns that key,
// for use as an HttpOnly cookie value.
func (m *Manager) CreateSession(sess AuthSession) (string, error) {
	key, err := security.RandomToken(storeKeyBytes)
	if err != nil {
		return "", err
	}
	m.sessions.Put(key, sess, m.cfg.IdleTimeout)
	return key, nil
}

// GetSession retrieves the session for key without consuming it.
func (m *Manager) GetSession(key string) (AuthSession, bool) {
	return m.sessions.Get(key)
}

// DestroySession removes exactly the session behind key (used by a
// same-app, cookie-driven logout).
func (m *Manager) DestroySession(key string) {
	m.sessions.Delete(key)
}

// DestroyBySid removes every session sharing the given IdP sid, across all
// three apps' entries in this single process. Called from the back-channel
// logout handler, this is what makes one WSO2 IS sign-out end every app's
// session — not just the app that received the logout token.
func (m *Manager) DestroyBySid(sid string) int {
	return m.sessions.DeleteWhere(func(s AuthSession) bool { return s.Sid == sid })
}

// Close stops both stores' background sweepers.
func (m *Manager) Close() {
	m.txns.Close()
	m.sessions.Close()
}
