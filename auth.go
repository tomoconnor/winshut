// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package main

import (
	"crypto/sha256"
	"fmt"
	"log"
	"net"
	"net/http"
)

func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.TLS != nil && len(r.TLS.VerifiedChains) > 0 {
			cert := r.TLS.VerifiedChains[0][0]
			fp := sha256.Sum256(cert.Raw)
			log.Printf("auth cn=%s fp=%x from %s", cert.Subject.CommonName, fp[:8], r.RemoteAddr)
			next.ServeHTTP(w, r)
			return
		}

		log.Printf("auth failed from %s", r.RemoteAddr)
		writeJSON(w, http.StatusUnauthorized, response{Status: "error", Message: "unauthorized"})
	})
}

func allowlistMiddleware(cidrs []*net.IPNet, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			writeJSON(w, http.StatusForbidden, response{Status: "error", Message: "forbidden"})
			return
		}
		ip := net.ParseIP(host)
		for _, cidr := range cidrs {
			if cidr.Contains(ip) {
				next.ServeHTTP(w, r)
				return
			}
		}
		log.Printf("blocked %s (not in allowlist)", r.RemoteAddr)
		writeJSON(w, http.StatusForbidden, response{Status: "error", Message: fmt.Sprintf("forbidden: %s not in allowlist", host)})
	})
}
