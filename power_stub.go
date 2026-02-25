// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

//go:build !windows

package main

import (
	"fmt"
	"log"
)

func execPowerCommand(action string) error {
	switch action {
	case "shutdown", "restart", "hibernate", "sleep", "lock", "logoff", "screen-off":
		log.Printf("[stub] would execute power command: %s", action)
		return nil
	default:
		return fmt.Errorf("unknown action: %s", action)
	}
}
