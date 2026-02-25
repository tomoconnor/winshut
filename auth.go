// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package main

import (
	"log"
	"net/http"
)

func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.TLS != nil && len(r.TLS.VerifiedChains) > 0 {
			next.ServeHTTP(w, r)
			return
		}

		log.Printf("auth failed from %s", r.RemoteAddr)
		writeJSON(w, http.StatusUnauthorized, response{Status: "error", Message: "unauthorized"})
	})
}
