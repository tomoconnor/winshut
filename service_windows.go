// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

//go:build windows

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/eventlog"
	"golang.org/x/sys/windows/svc/mgr"
)

const serviceName = "WinShut"

func signalNotify(c chan<- os.Signal) {
	signal.Notify(c, os.Interrupt)
}

func runService(cfg serverConfig, server *http.Server) error {
	isService, err := svc.IsWindowsService()
	if err != nil {
		return fmt.Errorf("failed to detect service mode: %w", err)
	}
	if !isService {
		return runInteractive(cfg, server)
	}

	elog, err := eventlog.Open(serviceName)
	if err != nil {
		return fmt.Errorf("failed to open event log: %w", err)
	}
	defer elog.Close()

	log.SetOutput(&eventLogWriter{elog: elog})

	return svc.Run(serviceName, &winshutService{cfg: cfg, server: server})
}

type winshutService struct {
	cfg    serverConfig
	server *http.Server
}

func (s *winshutService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (bool, uint32) {
	changes <- svc.Status{State: svc.StartPending}

	errCh := make(chan error, 1)
	go func() {
		if err := s.server.ListenAndServeTLS(s.cfg.CertFile, s.cfg.KeyFile); err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	changes <- svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown}
	log.Printf("service started on %s (dry-run=%v)", s.cfg.Addr, s.cfg.DryRun)

	for {
		select {
		case err := <-errCh:
			if err != nil {
				log.Printf("server error: %v", err)
				return false, 1
			}
			return false, 0
		case c := <-r:
			switch c.Cmd {
			case svc.Stop, svc.Shutdown:
				changes <- svc.Status{State: svc.StopPending}
				log.Println("service stopping...")
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				if err := s.server.Shutdown(ctx); err != nil {
					log.Printf("shutdown error: %v", err)
				}
				cancel()
				return false, 0
			case svc.Interrogate:
				changes <- c.CurrentStatus
			}
		}
	}
}

type eventLogWriter struct {
	elog *eventlog.Log
}

func (w *eventLogWriter) Write(p []byte) (int, error) {
	err := w.elog.Info(1, string(p))
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

func serviceInstall(args []string) {
	m, err := mgr.Connect()
	if err != nil {
		log.Fatalf("failed to connect to service manager: %v", err)
	}
	defer m.Disconnect()

	exePath, err := os.Executable()
	if err != nil {
		log.Fatalf("failed to get executable path: %v", err)
	}

	s, err := m.CreateService(serviceName, exePath, mgr.Config{
		DisplayName: "WinShut",
		Description: "Remote Windows power management over HTTPS",
		StartType:   mgr.StartAutomatic,
	}, args...)
	if err != nil {
		log.Fatalf("failed to create service: %v", err)
	}
	defer s.Close()

	err = eventlog.InstallAsEventCreate(serviceName, eventlog.Info|eventlog.Warning|eventlog.Error)
	if err != nil {
		s.Delete()
		log.Fatalf("failed to install event log source: %v", err)
	}

	fmt.Printf("service %q installed\n", serviceName)

	if err := s.Start(); err != nil {
		log.Fatalf("failed to start service: %v", err)
	}
	fmt.Printf("service %q started\n", serviceName)
}

func serviceRemove() {
	m, err := mgr.Connect()
	if err != nil {
		log.Fatalf("failed to connect to service manager: %v", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(serviceName)
	if err != nil {
		log.Fatalf("failed to open service: %v", err)
	}
	defer s.Close()

	err = s.Delete()
	if err != nil {
		log.Fatalf("failed to delete service: %v", err)
	}

	_ = eventlog.Remove(serviceName)

	fmt.Printf("service %q removed\n", serviceName)
}
