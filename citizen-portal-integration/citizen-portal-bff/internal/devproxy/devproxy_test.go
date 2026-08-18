package devproxy

import (
	"bufio"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/akila94/mosip-wso2-citizen-portal-demo/citizen-portal-bff/internal/security"
)

const (
	indexBody  = "<!doctype html><title>citizen portal SPA</title>"
	assetBody  = "console.log('bundle');"
	secretBody = "TOP-SECRET-OUTSIDE-THE-STATIC-ROOT"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// newStaticTree writes a minimal built-SPA layout into a temp directory and
// a secret file *outside* it, and returns both paths. The secret file is
// what the path-traversal tests assert can never be reached.
func newStaticTree(t *testing.T) (staticDir, secretPath string) {
	t.Helper()
	root := t.TempDir()

	staticDir = filepath.Join(root, "dist")
	if err := os.MkdirAll(filepath.Join(staticDir, "assets"), 0o750); err != nil {
		t.Fatalf("creating static tree: %v", err)
	}
	writeFile(t, filepath.Join(staticDir, "index.html"), indexBody)
	writeFile(t, filepath.Join(staticDir, "assets", "app.js"), assetBody)

	secretPath = filepath.Join(root, "secret.txt")
	writeFile(t, secretPath, secretBody)
	return staticDir, secretPath
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}

func newStaticSPAForTest(t *testing.T) (http.Handler, string) {
	t.Helper()
	staticDir, secretPath := newStaticTree(t)
	handler, err := New(Config{StaticDir: staticDir, Logger: discardLogger()})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return handler, secretPath
}

func get(t *testing.T, handler http.Handler, target string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, target, nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

// --- Boot-time validation: a misconfigured SPA source must fail startup,
// not 404 mysteriously at demo time. ---

func TestNewFailsWhenStaticDirIsMissing(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "no-such-dist")
	_, err := New(Config{StaticDir: missing, Logger: discardLogger()})
	if err == nil {
		t.Fatal("expected New to fail for a missing STATIC_DIR")
	}
	if !strings.Contains(err.Error(), missing) {
		t.Errorf("error %q must name the offending path", err)
	}
	if !strings.Contains(err.Error(), "npm run build") {
		t.Errorf("error %q must tell the operator how to fix it", err)
	}
}

func TestNewFailsWhenIndexHTMLIsMissing(t *testing.T) {
	staticDir := t.TempDir()
	_, err := New(Config{StaticDir: staticDir, Logger: discardLogger()})
	if err == nil {
		t.Fatal("expected New to fail when the SPA entry point is absent")
	}
	if !strings.Contains(err.Error(), "index.html") {
		t.Errorf("error %q must name index.html", err)
	}
}

func TestNewFailsWhenStaticDirIsAFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dist")
	writeFile(t, path, "not a directory")
	if _, err := New(Config{StaticDir: path, Logger: discardLogger()}); err == nil {
		t.Fatal("expected New to fail when STATIC_DIR is not a directory")
	}
}

func TestNewFailsOnMalformedDevProxyTarget(t *testing.T) {
	for _, target := range []string{"localhost:5173", "/vite", "ftp://localhost:5173", "http://"} {
		if _, err := New(Config{DevProxyTarget: target, Logger: discardLogger()}); err == nil {
			t.Errorf("expected New to reject DevProxyTarget=%q", target)
		}
	}
}

// --- Static mode. ---

func TestStaticServesIndexAtRoot(t *testing.T) {
	handler, _ := newStaticSPAForTest(t)

	rec := get(t, handler, "/")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if rec.Body.String() != indexBody {
		t.Errorf("body = %q, want the SPA index", rec.Body.String())
	}
}

func TestStaticServesAnExistingAssetVerbatim(t *testing.T) {
	handler, _ := newStaticSPAForTest(t)

	rec := get(t, handler, "/assets/app.js")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if rec.Body.String() != assetBody {
		t.Errorf("body = %q, want the asset", rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "javascript") {
		t.Errorf("Content-Type = %q, want a JavaScript type", ct)
	}
}

func TestStaticFallsBackToIndexForReactRouterDeepLinks(t *testing.T) {
	handler, _ := newStaticSPAForTest(t)

	for _, deepLink := range []string{
		"/apps/driving-licence/step/2",
		"/timeline",
		"/services/svc-dl",
		"/wireframes/consent",
	} {
		rec := get(t, handler, deepLink)
		if rec.Code != http.StatusOK {
			t.Errorf("GET %s: status = %d, want 200", deepLink, rec.Code)
		}
		if rec.Body.String() != indexBody {
			t.Errorf("GET %s: body = %q, want the SPA index (hard refresh must survive)", deepLink, rec.Body.String())
		}
	}
}

func TestStaticNeverServesAFileOutsideStaticDir(t *testing.T) {
	handler, secretPath := newStaticSPAForTest(t)
	secretName := filepath.Base(secretPath)

	traversals := []string{
		"/../secret.txt",
		"/../../etc/passwd",
		"/assets/../../" + secretName,
		"/..%2f" + secretName,
		"/%2e%2e/" + secretName,
		"/%2e%2e%2f%2e%2e%2fetc/passwd",
		"//etc/passwd",
		"/assets/..%5c..%5c" + secretName,
		"/....//" + secretName,
	}
	for _, target := range traversals {
		rec := get(t, handler, target)
		if strings.Contains(rec.Body.String(), secretBody) {
			t.Errorf("GET %s leaked a file outside STATIC_DIR", target)
		}
		if strings.Contains(rec.Body.String(), "root:") {
			t.Errorf("GET %s leaked /etc/passwd", target)
		}
	}
}

func TestStaticDoesNotFollowASymlinkOutOfStaticDir(t *testing.T) {
	staticDir, secretPath := newStaticTree(t)
	if err := os.Symlink(secretPath, filepath.Join(staticDir, "leak.txt")); err != nil {
		t.Skipf("symlinks unavailable on this platform: %v", err)
	}
	handler, err := New(Config{StaticDir: staticDir, Logger: discardLogger()})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	rec := get(t, handler, "/leak.txt")
	if strings.Contains(rec.Body.String(), secretBody) {
		t.Fatal("a symlink pointing outside STATIC_DIR must not be served")
	}
}

// --- The /bff/ guard: an unknown API path must stay JSON, in both modes,
// so a typo'd API path is debuggable instead of silently returning HTML. ---

func TestUnknownBFFPathReturnsJSONNotFoundInStaticMode(t *testing.T) {
	handler, _ := newStaticSPAForTest(t)

	rec := get(t, handler, "/bff/portal/api/typo")
	assertJSONNotFound(t, rec)
}

func TestUnknownBFFPathReturnsJSONNotFoundInProxyMode(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("the dev server must never see %s — the /bff/ guard runs first", r.URL.Path)
	}))
	defer target.Close()

	handler, err := New(Config{DevProxyTarget: target.URL, Logger: discardLogger()})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	rec := get(t, handler, "/bff/driving-licence/api/typo")
	assertJSONNotFound(t, rec)
}

func assertJSONNotFound(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Errorf("Content-Type = %q, want application/json (never the SPA's HTML)", ct)
	}
	// An unmatched /bff path is still an API response, so it must carry the
	// strict API policy rather than the relaxed SPA one the router wraps
	// this handler in.
	if csp := rec.Header().Get("Content-Security-Policy"); csp != security.APIContentSecurityPolicy {
		t.Errorf("Content-Security-Policy = %q, want the strict API policy %q", csp, security.APIContentSecurityPolicy)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("body %q is not JSON: %v", rec.Body.String(), err)
	}
	if body["error"] == "" {
		t.Errorf("body = %v, want an error field", body)
	}
}

// --- Dev-proxy mode. ---

func TestProxyForwardsRequestsToTheDevServer(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		if _, err := io.WriteString(w, "vite:"+r.URL.Path); err != nil {
			t.Errorf("writing target response: %v", err)
		}
	}))
	defer target.Close()

	handler, err := New(Config{DevProxyTarget: target.URL, Logger: discardLogger()})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	rec := get(t, handler, "/apps/driving-licence/step/2")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if rec.Body.String() != "vite:/apps/driving-licence/step/2" {
		t.Errorf("body = %q, want the dev server's response for the same path", rec.Body.String())
	}
}

func TestProxyReturnsBadGatewayWhenTheDevServerIsDown(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	targetURL := target.URL
	target.Close() // nothing is listening any more

	handler, err := New(Config{DevProxyTarget: targetURL, Logger: discardLogger()})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	rec := get(t, handler, "/")
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502", rec.Code)
	}
}

// TestProxyPassesWebSocketUpgradesThrough drives a raw TCP connection
// through the proxy so the 101 Switching Protocols handshake and the
// bidirectional byte stream after it are both exercised — this is what Vite
// HMR needs, and it cannot be observed through httptest.ResponseRecorder.
func TestProxyPassesWebSocketUpgradesThrough(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
			http.Error(w, "expected an upgrade request", http.StatusBadRequest)
			return
		}
		hijacker, ok := w.(http.Hijacker)
		if !ok {
			t.Error("target ResponseWriter does not support hijacking")
			return
		}
		conn, buffered, err := hijacker.Hijack()
		if err != nil {
			t.Errorf("hijacking target connection: %v", err)
			return
		}
		defer conn.Close()

		if _, err := buffered.WriteString("HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n\r\n"); err != nil {
			t.Errorf("writing 101: %v", err)
			return
		}
		if err := buffered.Flush(); err != nil {
			t.Errorf("flushing 101: %v", err)
			return
		}
		line, err := buffered.ReadString('\n')
		if err != nil {
			t.Errorf("reading client frame: %v", err)
			return
		}
		if _, err := buffered.WriteString("echo:" + line); err != nil {
			t.Errorf("writing echo: %v", err)
			return
		}
		if err := buffered.Flush(); err != nil {
			t.Errorf("flushing echo: %v", err)
		}
	}))
	defer target.Close()

	handler, err := New(Config{DevProxyTarget: target.URL, Logger: discardLogger()})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	proxy := httptest.NewServer(handler)
	defer proxy.Close()

	conn, err := net.Dial("tcp", proxy.Listener.Addr().String())
	if err != nil {
		t.Fatalf("dialing the proxy: %v", err)
	}
	defer conn.Close()
	if err := conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatalf("setting deadline: %v", err)
	}

	request := "GET /@vite/client HTTP/1.1\r\n" +
		"Host: localhost:8090\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Version: 13\r\n" +
		"Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\n\r\n"
	if _, err := conn.Write([]byte(request)); err != nil {
		t.Fatalf("writing upgrade request: %v", err)
	}

	reader := bufio.NewReader(conn)
	statusLine, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("reading status line: %v", err)
	}
	if !strings.Contains(statusLine, "101") {
		t.Fatalf("status line = %q, want 101 Switching Protocols (Vite HMR needs the upgrade to pass through)", statusLine)
	}
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("reading upgrade response headers: %v", err)
		}
		if strings.TrimSpace(line) == "" {
			break
		}
	}

	if _, err := conn.Write([]byte("ping\n")); err != nil {
		t.Fatalf("writing to the upgraded connection: %v", err)
	}
	echoed, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("reading from the upgraded connection: %v", err)
	}
	if strings.TrimSpace(echoed) != "echo:ping" {
		t.Errorf("echoed = %q, want %q — bytes must flow both ways after the upgrade", echoed, "echo:ping")
	}
}
