// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"

	"gopkg.in/yaml.v3"
)

type config struct {
	Server string `yaml:"server"`
	CA     string `yaml:"ca"`
	Cert   string `yaml:"cert"`
	Key    string `yaml:"key"`
}

var commands = map[string]struct {
	method string
	path   string
}{
	"health":     {http.MethodGet, "/health"},
	"stats":      {http.MethodGet, "/stats"},
	"shutdown":   {http.MethodPost, "/shutdown"},
	"restart":    {http.MethodPost, "/restart"},
	"hibernate":  {http.MethodPost, "/hibernate"},
	"sleep":      {http.MethodPost, "/sleep"},
	"lock":       {http.MethodPost, "/lock"},
	"logoff":     {http.MethodPost, "/logoff"},
	"screen-off": {http.MethodPost, "/screen-off"},
}

func main() {
	configPath := flag.String("config", "winshut-client.yml", "path to config file")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: winshut-client [--config path] <command>\n\nCommands: health, stats, shutdown, restart, hibernate, sleep, lock, logoff, screen-off\n\nFlags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}

	cmdName := flag.Arg(0)
	cmd, ok := commands[cmdName]
	if !ok {
		fmt.Fprintf(os.Stderr, "error: unknown command %q\n", cmdName)
		flag.Usage()
		os.Exit(1)
	}

	// Load config
	info, err := os.Stat(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: cannot stat config %s: %v\n", *configPath, err)
		os.Exit(1)
	}
	if info.Mode().Perm()&0o077 != 0 {
		fmt.Fprintf(os.Stderr, "warning: config file %s has loose permissions %o, consider chmod 600\n", *configPath, info.Mode().Perm())
	}

	data, err := os.ReadFile(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: cannot read config %s: %v\n", *configPath, err)
		os.Exit(1)
	}

	var cfg config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		fmt.Fprintf(os.Stderr, "error: invalid config: %v\n", err)
		os.Exit(1)
	}

	if cfg.Server == "" {
		fmt.Fprintln(os.Stderr, "error: 'server' is required in config")
		os.Exit(1)
	}

	if cfg.Cert == "" || cfg.Key == "" {
		fmt.Fprintln(os.Stderr, "error: 'cert' and 'key' are required in config for mTLS")
		os.Exit(1)
	}

	// Build TLS config
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS13,
	}

	if cfg.CA != "" {
		caCert, err := os.ReadFile(cfg.CA)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: cannot read CA file: %v\n", err)
			os.Exit(1)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(caCert) {
			fmt.Fprintln(os.Stderr, "error: failed to parse CA certificate")
			os.Exit(1)
		}
		tlsConfig.RootCAs = pool
	}

	cert, err := tls.LoadX509KeyPair(cfg.Cert, cfg.Key)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: cannot load client cert/key: %v\n", err)
		os.Exit(1)
	}
	tlsConfig.Certificates = []tls.Certificate{cert}

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	// Build request
	url := cfg.Server + cmd.path
	req, err := http.NewRequest(cmd.method, url, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	// Execute
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: reading response: %v\n", err)
		os.Exit(1)
	}

	// Pretty-print JSON if valid, otherwise print raw
	var parsed any
	if err := json.Unmarshal(body, &parsed); err == nil {
		pretty, _ := json.MarshalIndent(parsed, "", "  ")
		fmt.Println(string(pretty))
	} else {
		fmt.Print(string(body))
	}

	if resp.StatusCode >= 300 {
		os.Exit(1)
	}
}
