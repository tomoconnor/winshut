// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

type serverConfig struct {
	Addr     string
	CertFile string
	KeyFile  string
	CAFile   string
	DryRun   bool
}

func main() {
	// Subcommand dispatch (before flag parsing)
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "install":
			serviceInstall(os.Args[2:])
			return
		case "remove":
			serviceRemove()
			return
		}
	}

	addr := flag.String("addr", ":9090", "listen address")
	certFile := flag.String("cert", "", "TLS certificate file (required)")
	keyFile := flag.String("key", "", "TLS private key file (required)")
	caFile := flag.String("ca", "", "CA certificate for mTLS client verification (required)")
	dryRun := flag.Bool("dry-run", false, "log commands without executing")
	flag.Parse()

	if *certFile == "" || *keyFile == "" || *caFile == "" {
		fmt.Fprintln(os.Stderr, "error: --cert, --key, and --ca are required")
		flag.Usage()
		os.Exit(1)
	}

	cfg := serverConfig{
		Addr:     *addr,
		CertFile: *certFile,
		KeyFile:  *keyFile,
		CAFile:   *caFile,
		DryRun:   *dryRun,
	}

	server, err := buildServer(cfg)
	if err != nil {
		log.Fatalf("failed to build server: %v", err)
	}

	if err := runService(cfg, server); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func buildServer(cfg serverConfig) (*http.Server, error) {
	caCert, err := os.ReadFile(cfg.CAFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA file: %w", err)
	}
	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}

	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS13,
		ClientCAs:  caPool,
		ClientAuth: tls.RequireAndVerifyClientCert,
	}

	mux := http.NewServeMux()
	mux.Handle("/health", http.HandlerFunc(healthHandler))
	mux.Handle("/stats", http.HandlerFunc(statsHandler))
	mux.Handle("/shutdown", authMiddleware(powerHandler("shutdown", cfg.DryRun)))
	mux.Handle("/restart", authMiddleware(powerHandler("restart", cfg.DryRun)))
	mux.Handle("/hibernate", authMiddleware(powerHandler("hibernate", cfg.DryRun)))
	mux.Handle("/sleep", authMiddleware(powerHandler("sleep", cfg.DryRun)))
	mux.Handle("/lock", authMiddleware(powerHandler("lock", cfg.DryRun)))
	mux.Handle("/logoff", authMiddleware(powerHandler("logoff", cfg.DryRun)))
	mux.Handle("/screen-off", authMiddleware(powerHandler("screen-off", cfg.DryRun)))

	return &http.Server{
		Addr:           cfg.Addr,
		Handler:        mux,
		TLSConfig:      tlsConfig,
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   15 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 4096,
	}, nil
}

func runInteractive(cfg serverConfig, server *http.Server) error {
	done := make(chan os.Signal, 1)
	signalNotify(done)

	go func() {
		log.Printf("starting winshut on %s (dry-run=%v)", cfg.Addr, cfg.DryRun)
		if err := server.ListenAndServeTLS(cfg.CertFile, cfg.KeyFile); err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-done
	log.Println("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown error: %w", err)
	}
	log.Println("stopped")
	return nil
}
