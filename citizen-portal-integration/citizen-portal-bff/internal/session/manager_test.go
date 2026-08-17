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
