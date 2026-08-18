package session

import (
	"testing"
	"time"
)

func TestManagerLoginTxnRoundTrip(t *testing.T) {
	m := NewManager(Config{MaxSessions: 10, LoginTxnTTL: time.Minute, IdleTimeout: time.Minute})
	defer m.Close()

	key, err := m.CreateLoginTxn(LoginTxn{AppKey: "portal", State: "s1", Nonce: "n1", CodeVerifier: "v1", ReturnTo: "/"})
	if err != nil {
		t.Fatalf("CreateLoginTxn: %v", err)
	}
	if key == "" {
		t.Fatal("expected a non-empty transaction key")
	}

	txn, ok := m.ConsumeLoginTxn(key)
	if !ok {
		t.Fatal("expected the transaction to be found")
	}
	if txn.State != "s1" || txn.Nonce != "n1" || txn.CodeVerifier != "v1" {
		t.Errorf("unexpected txn contents: %+v", txn)
	}

	// A transaction must be single-use: consumed once, gone after.
	if _, ok := m.ConsumeLoginTxn(key); ok {
		t.Fatal("expected the transaction to be single-use")
	}
}

func TestManagerConsumeUnknownTxn(t *testing.T) {
	m := NewManager(Config{MaxSessions: 10, LoginTxnTTL: time.Minute, IdleTimeout: time.Minute})
	defer m.Close()

	if _, ok := m.ConsumeLoginTxn("does-not-exist"); ok {
		t.Fatal("expected ok=false for an unknown transaction key")
	}
}

func TestManagerSessionRoundTrip(t *testing.T) {
	m := NewManager(Config{MaxSessions: 10, LoginTxnTTL: time.Minute, IdleTimeout: time.Minute})
	defer m.Close()

	sess := AuthSession{
		AppKey: "portal",
		User:   User{Sub: "abc-123", Name: "Jane Citizen"},
		Sid:    "sid-1",
	}
	key, err := m.CreateSession(sess)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	got, ok := m.GetSession(key)
	if !ok {
		t.Fatal("expected the session to be found")
	}
	if got.User.Sub != "abc-123" {
		t.Errorf("unexpected session contents: %+v", got)
	}

	m.DestroySession(key)
	if _, ok := m.GetSession(key); ok {
		t.Fatal("expected the session to be gone after DestroySession")
	}
}

func TestManagerDestroyBySidDestroysEveryMatchingSession(t *testing.T) {
	m := NewManager(Config{MaxSessions: 10, LoginTxnTTL: time.Minute, IdleTimeout: time.Minute})
	defer m.Close()

	kPortal, _ := m.CreateSession(AuthSession{AppKey: "portal", Sid: "shared-sid"})
	kDL, _ := m.CreateSession(AuthSession{AppKey: "driving-licence", Sid: "shared-sid"})
	kOther, _ := m.CreateSession(AuthSession{AppKey: "revenue-licence", Sid: "different-sid"})

	n := m.DestroyBySid("shared-sid")
	if n != 2 {
		t.Fatalf("DestroyBySid destroyed %d sessions, want 2", n)
	}
	if _, ok := m.GetSession(kPortal); ok {
		t.Error("portal session should be destroyed")
	}
	if _, ok := m.GetSession(kDL); ok {
		t.Error("driving-licence session should be destroyed")
	}
	if _, ok := m.GetSession(kOther); !ok {
		t.Error("session with a different sid should survive")
	}
}

func TestManagerFindBySidReturnsEveryMatchingSessionWithoutDestroyingThem(t *testing.T) {
	m := NewManager(Config{MaxSessions: 10, LoginTxnTTL: time.Minute, IdleTimeout: time.Minute})
	defer m.Close()

	kPortal, _ := m.CreateSession(AuthSession{AppKey: "portal", Sid: "shared-sid"})
	kDL, _ := m.CreateSession(AuthSession{AppKey: "driving-licence", Sid: "shared-sid"})
	kOther, _ := m.CreateSession(AuthSession{AppKey: "revenue-licence", Sid: "different-sid"})

	found := m.FindBySid("shared-sid")
	if len(found) != 2 {
		t.Fatalf("FindBySid returned %d sessions, want 2: %+v", len(found), found)
	}
	appKeys := map[string]bool{}
	for _, sess := range found {
		appKeys[sess.AppKey] = true
	}
	if !appKeys["portal"] || !appKeys["driving-licence"] {
		t.Errorf("FindBySid returned app keys %v, want portal and driving-licence", appKeys)
	}

	// FindBySid must be a query, never a mutation — the session inspector
	// calls it on every poll.
	for _, key := range []string{kPortal, kDL, kOther} {
		if _, ok := m.GetSession(key); !ok {
			t.Errorf("session %q must survive FindBySid", key)
		}
	}
}

func TestManagerFindBySidReturnsNothingForAnUnknownSid(t *testing.T) {
	m := NewManager(Config{MaxSessions: 10, LoginTxnTTL: time.Minute, IdleTimeout: time.Minute})
	defer m.Close()

	m.CreateSession(AuthSession{AppKey: "portal", Sid: "shared-sid"})
	if found := m.FindBySid("no-such-sid"); len(found) != 0 {
		t.Fatalf("FindBySid returned %d sessions, want 0", len(found))
	}
}

func TestManagerFindBySidNeverMatchesAnEmptySid(t *testing.T) {
	m := NewManager(Config{MaxSessions: 10, LoginTxnTTL: time.Minute, IdleTimeout: time.Minute})
	defer m.Close()

	// Two unrelated sessions that simply have no sid claim must never be
	// reported as sharing one IdP session.
	m.CreateSession(AuthSession{AppKey: "portal"})
	m.CreateSession(AuthSession{AppKey: "driving-licence"})

	if found := m.FindBySid(""); len(found) != 0 {
		t.Fatalf("FindBySid(\"\") returned %d sessions, want 0", len(found))
	}
}

func TestDeriveAssuranceLevel(t *testing.T) {
	cases := []struct {
		name string
		amr  []string
		want AssuranceLevel
	}{
		{"empty amr defaults to basic", nil, AssuranceBasic},
		{"local username/password is basic", []string{"BasicAuthenticator"}, AssuranceBasic},
		{"eSignet federated authenticator is substantial", []string{"EsignetOIDCAuthenticator"}, AssuranceSubstantial},
		{"mixed favors the strongest present", []string{"BasicAuthenticator", "EsignetOIDCAuthenticator"}, AssuranceSubstantial},
		{"unrecognized authenticator defaults to basic", []string{"SomeOtherAuthenticator"}, AssuranceBasic},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := DeriveAssuranceLevel(tc.amr); got != tc.want {
				t.Errorf("DeriveAssuranceLevel(%v) = %v, want %v", tc.amr, got, tc.want)
			}
		})
	}
}

func TestDeriveIdentityProvider(t *testing.T) {
	cases := []struct {
		name string
		amr  []string
		want string
	}{
		{"empty amr means the local directory", nil, IdentityProviderLocal},
		{"local username/password", []string{"BasicAuthenticator"}, IdentityProviderLocal},
		{"eSignet federated authenticator", []string{"EsignetOIDCAuthenticator"}, IdentityProviderESignet},
		{"mixed reports the federated IdP", []string{"BasicAuthenticator", "EsignetOIDCAuthenticator"}, IdentityProviderESignet},
		{"unrecognized authenticator falls back to local", []string{"SomeOtherAuthenticator"}, IdentityProviderLocal},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := DeriveIdentityProvider(tc.amr); got != tc.want {
				t.Errorf("DeriveIdentityProvider(%v) = %q, want %q", tc.amr, got, tc.want)
			}
		})
	}
}
