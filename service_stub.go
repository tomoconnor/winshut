// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

//go:build !windows

package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func signalNotify(c chan<- os.Signal) {
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
}

func runService(cfg serverConfig, server *http.Server) error {
	return runInteractive(cfg, server)
}

func serviceInstall(_ []string) {
	fmt.Fprintln(os.Stderr, "error: service install is only supported on Windows")
	os.Exit(1)
}

func serviceRemove() {
	fmt.Fprintln(os.Stderr, "error: service remove is only supported on Windows")
	os.Exit(1)
}
