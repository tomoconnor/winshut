// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package main

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

func authMiddleware(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check mTLS: if client presented a verified certificate, allow
			if r.TLS != nil && len(r.TLS.VerifiedChains) > 0 {
				next.ServeHTTP(w, r)
				return
			}

			// Check bearer token
			if token != "" {
				auth := r.Header.Get("Authorization")
				if strings.HasPrefix(auth, "Bearer ") {
					provided := auth[len("Bearer "):]
					if subtle.ConstantTimeCompare([]byte(provided), []byte(token)) == 1 {
						next.ServeHTTP(w, r)
						return
					}
				}
			}

			writeJSON(w, http.StatusUnauthorized, response{Status: "error", Message: "unauthorized"})
		})
	}
}
