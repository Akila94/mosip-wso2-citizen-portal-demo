package devproxy

import (
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/akila94/mosip-wso2-citizen-portal-demo/citizen-portal-bff/internal/security"
)

// indexFile is the SPA's entry point and its fallback: any path that is not
// a real file is answered with it, so React Router owns client-side routing
// and a hard refresh on /apps/driving-licence/step/2 still works.
const indexFile = "index.html"

// buildHint is appended to every start-up failure in static mode, because
// the overwhelmingly likely cause is simply that the SPA has not been built
// yet in a fresh clone.
const buildHint = "build the SPA first (cd citizen-portal-demo-app && npm run build), " +
	"or set DEV_PROXY_TARGET to serve it from the Vite dev server instead"

// staticHandler serves the built SPA out of a single directory.
//
// Every file access goes through an *os.Root opened on that directory. Root
// resolves each path component inside the root and refuses to traverse out
// of it — including via ".." and via symlinks that point outside — which is
// the guarantee this handler needs and the reason it does not simply join
// strings onto a directory name.
type staticHandler struct {
	root   *os.Root
	dir    string
	logger *slog.Logger
}

// newStaticHandler validates the built-SPA directory and returns a handler
// for it. It reports an error (rather than starting and 404ing later) when
// the directory is missing, is not a directory, or holds no index.html.
func newStaticHandler(dir string, logger *slog.Logger) (http.Handler, error) {
	if dir == "" {
		return nil, fmt.Errorf("devproxy: STATIC_DIR is empty: %s", buildHint)
	}

	info, err := os.Stat(dir)
	if err != nil {
		return nil, fmt.Errorf("devproxy: STATIC_DIR %q is not readable (%v): %s", dir, err, buildHint)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("devproxy: STATIC_DIR %q is not a directory: %s", dir, buildHint)
	}

	// The Root stays open for the process's lifetime — it is the handler's
	// only door onto the filesystem, exactly like the listening socket is
	// its only door onto the network.
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, fmt.Errorf("devproxy: cannot open STATIC_DIR %q: %w", dir, err)
	}
	if _, err := root.Stat(indexFile); err != nil {
		if closeErr := root.Close(); closeErr != nil {
			logger.Warn("closing STATIC_DIR after a failed start", "error", closeErr.Error())
		}
		return nil, fmt.Errorf("devproxy: STATIC_DIR %q has no %s (%v): %s", dir, indexFile, err, buildHint)
	}

	logger.Info("serving the SPA from static files", "staticDir", dir)
	return &staticHandler{root: root, dir: dir, logger: logger}, nil
}

func (h *staticHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// A built SPA is only ever read. Anything else that reached the
	// NotFound handler is an unrouted request, not a page.
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeJSONNotFound(w)
		return
	}

	name := resolveStaticName(r.URL.Path)
	file, info, ok := h.openFile(name)
	if !ok {
		// SPA fallback. This is also where every rejected traversal ends
		// up, which is deliberate: a caller learns nothing about what does
		// or does not exist outside the root.
		name = indexFile
		file, info, ok = h.openFile(name)
		if !ok {
			h.logger.Error("SPA index is unreadable", "staticDir", h.dir, "index", indexFile)
			http.Error(w, "SPA not available", http.StatusInternalServerError)
			return
		}
	}
	defer func() {
		if err := file.Close(); err != nil {
			h.logger.Warn("closing a served SPA file", "error", err.Error())
		}
	}()

	if name == indexFile {
		// The entry point must be revalidated on every load, or a browser
		// keeps serving an index.html that references replaced, hashed
		// bundles after a redeploy. The hashed assets themselves are
		// content-addressed and safe to cache under their default headers.
		w.Header().Set("Cache-Control", "no-cache")
	}
	http.ServeContent(w, r, name, info.ModTime(), file)
}

// openFile opens name inside the root, reporting failure for anything that
// is not a readable regular file. Directories deliberately count as failure:
// this handler never produces a directory listing.
func (h *staticHandler) openFile(name string) (*os.File, fs.FileInfo, bool) {
	file, err := h.root.Open(name)
	if err != nil {
		h.logger.Debug("SPA file not served, falling back to the index",
			"name", security.SanitizeForLog(name),
			"reason", security.SanitizeForLog(err.Error()))
		return nil, nil, false
	}
	info, err := file.Stat()
	if err != nil || info.IsDir() {
		if closeErr := file.Close(); closeErr != nil {
			h.logger.Warn("closing an unusable SPA file", "error", closeErr.Error())
		}
		return nil, nil, false
	}
	return file, info, true
}

// resolveStaticName turns a request path into a root-relative file name.
//
// path.Clean collapses ".", ".." and duplicate slashes *before* the name
// reaches the filesystem, so "/../../etc/passwd" becomes "etc/passwd" — a
// lookup inside the root that simply misses, rather than an escape attempt.
// os.Root enforces the same boundary independently; both are kept because
// the cost is one function call and the failure mode is serving arbitrary
// files off the host.
func resolveStaticName(urlPath string) string {
	// Reject Windows-style separators outright rather than reasoning about
	// how each platform's path resolution treats them.
	if strings.ContainsAny(urlPath, "\\\x00") {
		return indexFile
	}

	cleaned := strings.TrimPrefix(path.Clean("/"+urlPath), "/")
	if cleaned == "" || cleaned == "." {
		return indexFile
	}
	return cleaned
}
