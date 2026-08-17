package oidcrp

import (
	"fmt"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
)

// Claims is the subset of ID-token claims the BFF acts on, decoded from a
// verified token. Field names follow the OIDC standard claim names (OIDC
// Core 5.1) except AuthTime, which is `auth_time` (a Unix timestamp on the
// wire) converted to time.Time for convenience.
type Claims struct {
	Sub         string
	Name        string
	GivenName   string
	FamilyName  string
	Email       string
	PhoneNumber string
	Birthdate   string
	Picture     string
	Sid         string
	Acr         string
	Amr         []string
	AuthTime    time.Time
	Expiry      time.Time
}

// rawClaims mirrors the wire representation of the claims Claims cares
// about. Every field is optional per OIDC Core — WSO2 IS omits `acr`
// entirely when no value was resolved for the flow (verified against the
// shipped 7.3.0 source), so none of these may be assumed present.
type rawClaims struct {
	Sub         string   `json:"sub"`
	Name        string   `json:"name"`
	GivenName   string   `json:"given_name"`
	FamilyName  string   `json:"family_name"`
	Email       string   `json:"email"`
	PhoneNumber string   `json:"phone_number"`
	Birthdate   string   `json:"birthdate"`
	Picture     string   `json:"picture"`
	Sid         string   `json:"sid"`
	Acr         string   `json:"acr"`
	Amr         []string `json:"amr"`
	AuthTime    int64    `json:"auth_time"`
}

func claimsFromIDToken(idToken *oidc.IDToken) (*Claims, error) {
	var raw rawClaims
	if err := idToken.Claims(&raw); err != nil {
		return nil, fmt.Errorf("oidcrp: decoding ID token claims: %w", err)
	}
	c := &Claims{
		Sub:         idToken.Subject,
		Name:        raw.Name,
		GivenName:   raw.GivenName,
		FamilyName:  raw.FamilyName,
		Email:       raw.Email,
		PhoneNumber: raw.PhoneNumber,
		Birthdate:   raw.Birthdate,
		Picture:     raw.Picture,
		Sid:         raw.Sid,
		Acr:         raw.Acr,
		Amr:         raw.Amr,
		Expiry:      idToken.Expiry,
	}
	if raw.AuthTime > 0 {
		c.AuthTime = time.Unix(raw.AuthTime, 0).UTC()
	}
	return c, nil
}
