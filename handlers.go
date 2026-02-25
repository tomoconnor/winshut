// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

type response struct {
	Status  string `json:"status"`
	Action  string `json:"action,omitempty"`
	Message string `json:"message,omitempty"`
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, response{Status: "error", Message: "method not allowed"})
		return
	}
	writeJSON(w, http.StatusOK, response{Status: "ok"})
}

func powerHandler(action string, dryRun bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, response{Status: "error", Message: "method not allowed"})
			return
		}

		if dryRun {
			log.Printf("[dry-run] would execute: %s", action)
			writeJSON(w, http.StatusOK, response{Status: "ok", Action: action, Message: "dry-run"})
			return
		}

		// Send response before executing power command
		writeJSON(w, http.StatusOK, response{Status: "ok", Action: action, Message: "executing"})

		// Flush the response
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}

		// Execute power command after a delay so the HTTP response has time to reach the client
		go func() {
			time.Sleep(500 * time.Millisecond)
			if err := execPowerCommand(action); err != nil {
				log.Printf("failed to execute %s: %v", action, err)
			}
		}()
	}
}

func statsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, response{Status: "error", Message: "method not allowed"})
		return
	}

	stats, err := getSystemStats()
	if err != nil {
		log.Printf("failed to get system stats: %v", err)
		writeJSON(w, http.StatusInternalServerError, response{Status: "error", Message: "failed to get stats"})
		return
	}

	writeJSON(w, http.StatusOK, stats)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("failed to write response: %v", err)
	}
}
