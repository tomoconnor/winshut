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
	"os/signal"
	"syscall"
	"time"
)

func main() {
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

	// Build TLS config
	caCert, err := os.ReadFile(*caFile)
	if err != nil {
		log.Fatalf("failed to read CA file: %v", err)
	}
	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caCert) {
		log.Fatal("failed to parse CA certificate")
	}

	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS13,
		ClientCAs:  caPool,
		ClientAuth: tls.VerifyClientCertIfGiven,
	}

	// Set up routes
	mux := http.NewServeMux()

	mux.Handle("/health", http.HandlerFunc(healthHandler))
	mux.Handle("/stats", http.HandlerFunc(statsHandler))
	mux.Handle("/shutdown", authMiddleware(powerHandler("shutdown", *dryRun)))
	mux.Handle("/restart", authMiddleware(powerHandler("restart", *dryRun)))
	mux.Handle("/hibernate", authMiddleware(powerHandler("hibernate", *dryRun)))
	mux.Handle("/sleep", authMiddleware(powerHandler("sleep", *dryRun)))
	mux.Handle("/lock", authMiddleware(powerHandler("lock", *dryRun)))
	mux.Handle("/logoff", authMiddleware(powerHandler("logoff", *dryRun)))
	mux.Handle("/screen-off", authMiddleware(powerHandler("screen-off", *dryRun)))

	server := &http.Server{
		Addr:         *addr,
		Handler:      mux,
		TLSConfig:    tlsConfig,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("starting winshut on %s (dry-run=%v)", *addr, *dryRun)
		if err := server.ListenAndServeTLS(*certFile, *keyFile); err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-done
	log.Println("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("shutdown error: %v", err)
	}
	log.Println("stopped")
}
