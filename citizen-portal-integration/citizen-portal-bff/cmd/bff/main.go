// Command bff runs the citizen-portal-bff — the OIDC relying party that
// stands between the browser and WSO2 Identity Server. M1 wired up the
// "Citizen Portal" app only; M2 added Application A (Driving Licence
// Service) and Application B (Vehicle Revenue Licence) alongside it; M3
// adds the upstream.Client that calls gov-services-api on the citizen's
// behalf, using the OAuth2 access token captured at login; M5 makes this
// process the browser's only origin by serving the SPA itself — from the
// Vite dev server when DEV_PROXY_TARGET is set, from STATIC_DIR otherwise
// (see internal/devproxy).
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/akila94/mosip-wso2-citizen-portal-demo/citizen-portal-bff/internal/config"
	"github.com/akila94/mosip-wso2-citizen-portal-demo/citizen-portal-bff/internal/devproxy"
	"github.com/akila94/mosip-wso2-citizen-portal-demo/citizen-portal-bff/internal/httpapi"
	"github.com/akila94/mosip-wso2-citizen-portal-demo/citizen-portal-bff/internal/oidcrp"
	"github.com/akila94/mosip-wso2-citizen-portal-demo/citizen-portal-bff/internal/session"
	"github.com/akila94/mosip-wso2-citizen-portal-demo/citizen-portal-bff/internal/upstream"
)

const (
	readTimeout       = 10 * time.Second
	writeTimeout      = 15 * time.Second
	idleTimeout       = 60 * time.Second
	readHeaderTimeout = 5 * time.Second
	shutdownTimeout   = 10 * time.Second

	// upstreamClientTimeout matches oidcrp.NewHTTPClient's timeout for the
	// IS client, for consistency across this process's two outbound HTTP
	// clients. gov-services-api runs on plain HTTP, so no CA pinning is
	// needed here — only IS's self-signed certificate requires that.
	upstreamClientTimeout = 15 * time.Second
)

func main() {
	if err := run(); err != nil {
		slog.Error("bff exited", "error", err)
		os.Exit(1)
	}
}

func run() error {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	cfg, err := config.Load(os.LookupEnv)
	if err != nil {
		return err
	}

	// The SPA is served from this same origin, so the browser never learns
	// about a second host. This is built before any network work: a missing
	// STATIC_DIR is a local mistake and should be reported immediately,
	// rather than after a 30-second discovery attempt against IS.
	spa, err := devproxy.New(devproxy.Config{
		DevProxyTarget: cfg.DevProxyTarget,
		StaticDir:      cfg.StaticDir,
		Logger:         logger,
	})
	if err != nil {
		return err
	}

	httpClient, err := oidcrp.NewHTTPClient(cfg.ISCAFile)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	provider, err := oidcrp.NewProvider(ctx, cfg.ISIssuer, httpClient)
	if err != nil {
		return err
	}

	portalRedirectURL := cfg.BFFPublicURL + "/bff/portal/callback"
	portalRP := oidcrp.NewRP(provider, httpClient, cfg.Portal.ClientID, cfg.Portal.ClientSecret, portalRedirectURL, cfg.Portal.Scopes)

	portalApp := &httpapi.AppRoute{
		Key:                   "portal",
		RoutePrefix:           "/bff/portal",
		ReturnToPrefix:        "/",
		Client:                portalRP,
		SessionCookieName:     "cp_sid",
		LoginTxnCookieName:    "cp_txn",
		CSRFCookieName:        "cp_csrf",
		PostLogoutRedirectURI: cfg.BFFPublicURL + "/",
		ClientID:              cfg.Portal.ClientID,
		AppName:               "Citizen Portal",
	}

	dlRedirectURL := cfg.BFFPublicURL + "/bff/driving-licence/callback"
	dlRP := oidcrp.NewRP(provider, httpClient, cfg.DrivingLicence.ClientID, cfg.DrivingLicence.ClientSecret, dlRedirectURL, cfg.DrivingLicence.Scopes)

	dlApp := &httpapi.AppRoute{
		Key:                   "driving-licence",
		RoutePrefix:           "/bff/driving-licence",
		ReturnToPrefix:        "/apps/driving-licence",
		Client:                dlRP,
		SessionCookieName:     "dl_sid",
		LoginTxnCookieName:    "dl_txn",
		CSRFCookieName:        "dl_csrf",
		PostLogoutRedirectURI: cfg.BFFPublicURL + "/apps/driving-licence",
		ClientID:              cfg.DrivingLicence.ClientID,
		AppName:               "Driving Licence Service",
	}

	vrlRedirectURL := cfg.BFFPublicURL + "/bff/revenue-licence/callback"
	vrlRP := oidcrp.NewRP(provider, httpClient, cfg.RevenueLicence.ClientID, cfg.RevenueLicence.ClientSecret, vrlRedirectURL, cfg.RevenueLicence.Scopes)

	vrlApp := &httpapi.AppRoute{
		Key:                   "revenue-licence",
		RoutePrefix:           "/bff/revenue-licence",
		ReturnToPrefix:        "/apps/revenue-licence",
		Client:                vrlRP,
		SessionCookieName:     "vrl_sid",
		LoginTxnCookieName:    "vrl_txn",
		CSRFCookieName:        "vrl_csrf",
		PostLogoutRedirectURI: cfg.BFFPublicURL + "/apps/revenue-licence",
		ClientID:              cfg.RevenueLicence.ClientID,
		AppName:               "Vehicle Revenue Licence",
	}

	sessions := session.NewManager(session.Config{
		MaxSessions: cfg.SessionMaxEntries,
		LoginTxnTTL: cfg.LoginTxnTTL,
		IdleTimeout: cfg.SessionIdleTimeout,
	})
	defer sessions.Close()

	upstreamClient := upstream.New(cfg.ServicesAPIURL, &http.Client{Timeout: upstreamClientTimeout})

	apiServer := httpapi.NewServer(
		map[string]*httpapi.AppRoute{
			"portal":          portalApp,
			"driving-licence": dlApp,
			"revenue-licence": vrlApp,
		},
		sessions,
		cfg.CookieSecure,
		cfg.SessionIdleTimeout,
		logger,
		upstreamClient,
	)

	apiServer.SPA = spa
	// SPADevMode must track exactly the same condition devproxy.New branches
	// on, since it selects the looser development CSP.
	apiServer.SPADevMode = cfg.DevProxyTarget != ""

	httpServer := &http.Server{
		Addr:              ":" + cfg.BFFPort,
		Handler:           apiServer.Router(),
		ReadTimeout:       readTimeout,
		ReadHeaderTimeout: readHeaderTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}

	serveErr := make(chan error, 1)
	go func() {
		logger.Info("citizen-portal-bff listening", "addr", httpServer.Addr, "publicURL", cfg.BFFPublicURL)
		serveErr <- httpServer.ListenAndServe()
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serveErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-stop:
		logger.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			return err
		}
	}
	return nil
}
