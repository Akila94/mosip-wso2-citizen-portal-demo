// Command bff runs the citizen-portal-bff — the OIDC relying party that
// stands between the browser and WSO2 Identity Server. M1 scope: the
// "Citizen Portal" app only. Application A/B and the SPA dev-proxy/static
// serving are added in later milestones (see PORTAL-INTEGRATION-PLAN.md).
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
	"github.com/akila94/mosip-wso2-citizen-portal-demo/citizen-portal-bff/internal/httpapi"
	"github.com/akila94/mosip-wso2-citizen-portal-demo/citizen-portal-bff/internal/oidcrp"
	"github.com/akila94/mosip-wso2-citizen-portal-demo/citizen-portal-bff/internal/session"
)

const (
	readTimeout       = 10 * time.Second
	writeTimeout      = 15 * time.Second
	idleTimeout       = 60 * time.Second
	readHeaderTimeout = 5 * time.Second
	shutdownTimeout   = 10 * time.Second
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
	}

	sessions := session.NewManager(session.Config{
		MaxSessions: cfg.SessionMaxEntries,
		LoginTxnTTL: cfg.LoginTxnTTL,
		IdleTimeout: cfg.SessionIdleTimeout,
	})
	defer sessions.Close()

	apiServer := httpapi.NewServer(
		map[string]*httpapi.AppRoute{"portal": portalApp},
		sessions,
		cfg.CookieSecure,
		cfg.SessionIdleTimeout,
		logger,
	)

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
