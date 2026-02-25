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
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

type serverConfig struct {
	Addr       string
	CertFile   string
	KeyFile    string
	CAFile     string
	AllowCIDRs string
	DryRun     bool
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [command] [options]\n\n", os.Args[0])
		fmt.Fprintln(os.Stderr, "Commands:")
		fmt.Fprintln(os.Stderr, "  install    Install as a Windows service (flags are stored as service args)")
		fmt.Fprintln(os.Stderr, "  remove     Remove the Windows service")
		fmt.Fprintln(os.Stderr, "\nOptions:")
		flag.PrintDefaults()
	}

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

	addr := flag.String("addr", "127.0.0.1:9090", "listen address")
	certFile := flag.String("cert", "", "TLS certificate file (required)")
	keyFile := flag.String("key", "", "TLS private key file (required)")
	caFile := flag.String("ca", "", "CA certificate for mTLS client verification (required)")
	allowCIDRs := flag.String("allow", "", "allowed client CIDRs, comma-separated (e.g. 192.168.1.0/24,10.0.0.0/8)")
	dryRun := flag.Bool("dry-run", false, "log commands without executing")
	flag.Parse()

	// Catch subcommands placed after flags (e.g. winshut --cert ... install)
	if arg := flag.Arg(0); arg == "install" || arg == "remove" {
		fmt.Fprintf(os.Stderr, "error: %q must be the first argument\n", arg)
		fmt.Fprintf(os.Stderr, "usage: %s %s [options]\n", os.Args[0], arg)
		os.Exit(1)
	}

	if *certFile == "" || *keyFile == "" || *caFile == "" {
		fmt.Fprintln(os.Stderr, "error: --cert, --key, and --ca are required")
		flag.Usage()
		os.Exit(1)
	}

	cfg := serverConfig{
		Addr:       *addr,
		CertFile:   *certFile,
		KeyFile:    *keyFile,
		CAFile:     *caFile,
		AllowCIDRs: *allowCIDRs,
		DryRun:     *dryRun,
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

	// Parse IP allowlist
	var cidrs []*net.IPNet
	if cfg.AllowCIDRs != "" {
		for _, s := range strings.Split(cfg.AllowCIDRs, ",") {
			_, cidr, err := net.ParseCIDR(strings.TrimSpace(s))
			if err != nil {
				return nil, fmt.Errorf("invalid CIDR %q: %w", strings.TrimSpace(s), err)
			}
			cidrs = append(cidrs, cidr)
		}
	}

	rl := newPowerRateLimiter(0.5, 2) // 1 action per 2s, burst of 2

	mux := http.NewServeMux()
	mux.Handle("/health", http.HandlerFunc(healthHandler))
	mux.Handle("/stats", authMiddleware(http.HandlerFunc(statsHandler)))
	for _, action := range []string{"shutdown", "restart", "hibernate", "sleep", "lock", "logoff", "screen-off"} {
		mux.Handle("/"+action, authMiddleware(rl.middleware(powerHandler(action, cfg.DryRun))))
	}

	var handler http.Handler = mux
	if len(cidrs) > 0 {
		handler = allowlistMiddleware(cidrs, mux)
	}

	return &http.Server{
		Addr:              cfg.Addr,
		Handler:           handler,
		TLSConfig:         tlsConfig,
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    4096,
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
