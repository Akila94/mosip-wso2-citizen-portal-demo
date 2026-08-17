// Command govapi runs gov-services-api — the resource server behind the
// citizen-portal-bff. It validates every request's JWT OAuth2 access token
// itself (signature via WSO2 IS's live JWKS, issuer, expiry, then a
// per-router required audience and scope) and never talks to a browser or
// presents credentials of its own.
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

	"github.com/akila94/mosip-wso2-citizen-portal-demo/gov-services-api/internal/authmw"
	"github.com/akila94/mosip-wso2-citizen-portal-demo/gov-services-api/internal/config"
	"github.com/akila94/mosip-wso2-citizen-portal-demo/gov-services-api/internal/httpapi"
	"github.com/akila94/mosip-wso2-citizen-portal-demo/gov-services-api/internal/httpclient"
	"github.com/akila94/mosip-wso2-citizen-portal-demo/gov-services-api/internal/registry"
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
		slog.Error("govapi exited", "error", err)
		os.Exit(1)
	}
}

func run() error {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	cfg, err := config.Load(os.LookupEnv)
	if err != nil {
		return err
	}

	client, err := httpclient.NewHTTPClient(cfg.ISCAFile)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	keySet, err := authmw.NewKeySet(ctx, cfg.ISIssuer, client)
	if err != nil {
		return err
	}
	verifier := authmw.NewVerifier(keySet, cfg.ISIssuer)

	reg := registry.New()

	router := httpapi.NewRouter(verifier, cfg.PortalClientID, cfg.DrivingLicenceClientID, cfg.RevenueLicenceClientID, reg, logger)

	httpServer := &http.Server{
		Addr:              ":" + cfg.ServicesPort,
		Handler:           router,
		ReadTimeout:       readTimeout,
		ReadHeaderTimeout: readHeaderTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}

	serveErr := make(chan error, 1)
	go func() {
		logger.Info("gov-services-api listening", "addr", httpServer.Addr, "issuer", cfg.ISIssuer)
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
