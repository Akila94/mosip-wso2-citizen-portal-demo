// Package upstream calls gov-services-api on the citizen's behalf, using the
// OAuth2 access token the BFF captured at login (internal/session's
// AuthSession.AccessToken). It is a thin, typed HTTP client — one named
// method per gov-services-api route — that does response passthrough
// rather than typed unmarshal/remarshal: the BFF's job is token injection
// and routing, not re-modeling gov-services-api's JSON shapes into a second
// set of Go structs across a module boundary. See
// PORTAL-INTEGRATION-PLAN.md's Component 1: "there is deliberately no
// generic token-injecting proxy ... every upstream call is a named
// handler" — this package is that named-handler set's HTTP leg.
package upstream

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

// maxUpstreamResponseBytes caps how much of gov-services-api's response
// body this client will read, as defense in depth (this project's standing
// discipline of bounding every body it reads, mirroring the BFF's own
// maxRequestBodyBytes and gov-services-api's own request-body cap) even
// though gov-services-api is not an attacker-controlled boundary in this
// deployment.
const maxUpstreamResponseBytes = 1 << 20 // 1 MiB

// Response is one named call's raw HTTP result: status code and body
// bytes, exactly as gov-services-api returned them. A non-2xx status is
// NOT a Go error here — the caller (an httpapi data handler) decides how
// to translate gov-services-api's status into its own response. Only a
// genuine transport failure (connection refused, timeout, context
// cancellation) is a Go error.
type Response struct {
	StatusCode  int
	ContentType string
	Body        []byte
}

// Client calls gov-services-api. httpClient should be a plain
// *http.Client with a sane timeout — gov-services-api runs on plain HTTP,
// so no CA pinning is needed here (contrast internal/oidcrp's IS client).
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// New constructs a Client that sends every request to baseURL
// (gov-services-api's SERVICES_API_URL, e.g. "http://localhost:8091") using
// httpClient.
func New(baseURL string, httpClient *http.Client) *Client {
	return &Client{baseURL: baseURL, httpClient: httpClient}
}

// do is the request path for every call made on a citizen's behalf: it
// builds an "Authorization: Bearer <accessToken>" request and sends it. It
// is the only place in this package that sets an Authorization header.
func (c *Client) do(ctx context.Context, method, path, accessToken string, body io.Reader) (Response, error) {
	req, err := c.newRequest(ctx, method, path, body)
	if err != nil {
		return Response{}, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	return c.send(req, method, path)
}

// doPublic is the request path for gov-services-api's unauthenticated
// /public/* routes. It sets **no** Authorization header at all — deliberately
// not an empty "Bearer " value, which is a different request on the wire and
// would misrepresent an anonymous call as a failed authenticated one (and,
// on a stricter upstream, would be answered 400 for a malformed header
// rather than served).
func (c *Client) doPublic(ctx context.Context, method, path string) (Response, error) {
	req, err := c.newRequest(ctx, method, path, nil)
	if err != nil {
		return Response{}, err
	}
	return c.send(req, method, path)
}

// newRequest builds the request both paths share, without any credential.
func (c *Client) newRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return nil, fmt.Errorf("upstream: building request for %s %s: %w", method, path, err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

// send performs req and reads the response body up to
// maxUpstreamResponseBytes via io.LimitReader. method and path are carried
// only for error messages, since req's own URL includes the base URL.
func (c *Client) send(req *http.Request, method, path string) (Response, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return Response{}, fmt.Errorf("upstream: calling %s %s: %w", method, path, err)
	}
	defer func() { _ = resp.Body.Close() }()

	limited := io.LimitReader(resp.Body, maxUpstreamResponseBytes)
	respBody, err := io.ReadAll(limited)
	if err != nil {
		return Response{}, fmt.Errorf("upstream: reading response body for %s %s: %w", method, path, err)
	}

	return Response{
		StatusCode:  resp.StatusCode,
		ContentType: resp.Header.Get("Content-Type"),
		Body:        respBody,
	}, nil
}

// PublicPortalCatalogue calls GET /public/catalogue — gov-services-api's
// unauthenticated service catalogue, used by the portal's landing page for
// signed-out visitors.
//
// It takes no access token (there is no session behind this call) and sends
// no assuranceLevel (there is no session to derive one from, and
// gov-services-api ignores the parameter on this route anyway). The absence
// of both parameters is the point: a public call has structurally nothing to
// leak.
func (c *Client) PublicPortalCatalogue(ctx context.Context) (Response, error) {
	return c.doPublic(ctx, http.MethodGet, "/public/catalogue")
}

// PortalCatalogue calls GET /portal/catalogue with assuranceLevel as the
// query parameter gov-services-api expects. Callers must derive
// assuranceLevel server-side from the verified session — never take it
// from the incoming request — see internal/httpapi/portal_data.go.
func (c *Client) PortalCatalogue(ctx context.Context, accessToken, assuranceLevel string) (Response, error) {
	q := url.Values{}
	if assuranceLevel != "" {
		q.Set("assuranceLevel", assuranceLevel)
	}
	path := "/portal/catalogue"
	if encoded := q.Encode(); encoded != "" {
		path += "?" + encoded
	}
	return c.do(ctx, http.MethodGet, path, accessToken, nil)
}

// PortalTimeline calls GET /portal/timeline.
func (c *Client) PortalTimeline(ctx context.Context, accessToken string) (Response, error) {
	return c.do(ctx, http.MethodGet, "/portal/timeline", accessToken, nil)
}

// PortalAttributes calls GET /portal/attributes.
func (c *Client) PortalAttributes(ctx context.Context, accessToken string) (Response, error) {
	return c.do(ctx, http.MethodGet, "/portal/attributes", accessToken, nil)
}

// PortalConsents calls GET /portal/consents.
func (c *Client) PortalConsents(ctx context.Context, accessToken string) (Response, error) {
	return c.do(ctx, http.MethodGet, "/portal/consents", accessToken, nil)
}

// PortalDocuments calls GET /portal/documents.
func (c *Client) PortalDocuments(ctx context.Context, accessToken string) (Response, error) {
	return c.do(ctx, http.MethodGet, "/portal/documents", accessToken, nil)
}

// PortalDepartmentRecords calls GET /portal/department-records.
func (c *Client) PortalDepartmentRecords(ctx context.Context, accessToken string) (Response, error) {
	return c.do(ctx, http.MethodGet, "/portal/department-records", accessToken, nil)
}

// DrivingLicenceConfig calls GET /driving-licence/config.
func (c *Client) DrivingLicenceConfig(ctx context.Context, accessToken string) (Response, error) {
	return c.do(ctx, http.MethodGet, "/driving-licence/config", accessToken, nil)
}

// DrivingLicenceTestSlots calls GET /driving-licence/test-slots with week as
// the query parameter gov-services-api expects.
func (c *Client) DrivingLicenceTestSlots(ctx context.Context, accessToken string, week int) (Response, error) {
	q := url.Values{}
	q.Set("week", strconv.Itoa(week))
	return c.do(ctx, http.MethodGet, "/driving-licence/test-slots?"+q.Encode(), accessToken, nil)
}

// DrivingLicenceSubmitApplication calls POST /driving-licence/applications
// with body forwarded verbatim as the request body.
func (c *Client) DrivingLicenceSubmitApplication(ctx context.Context, accessToken string, body io.Reader) (Response, error) {
	return c.do(ctx, http.MethodPost, "/driving-licence/applications", accessToken, body)
}

// VehicleRegistryVehicles calls GET /vehicle-registry/vehicles.
func (c *Client) VehicleRegistryVehicles(ctx context.Context, accessToken string) (Response, error) {
	return c.do(ctx, http.MethodGet, "/vehicle-registry/vehicles", accessToken, nil)
}

// VehicleRegistryRenew calls POST /vehicle-registry/vehicles/{id}/renew for
// vehicleID.
func (c *Client) VehicleRegistryRenew(ctx context.Context, accessToken, vehicleID string) (Response, error) {
	path := "/vehicle-registry/vehicles/" + url.PathEscape(vehicleID) + "/renew"
	return c.do(ctx, http.MethodPost, path, accessToken, nil)
}

// CitizenProfile calls GET /citizen/profile.
func (c *Client) CitizenProfile(ctx context.Context, accessToken string) (Response, error) {
	return c.do(ctx, http.MethodGet, "/citizen/profile", accessToken, nil)
}
