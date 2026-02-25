// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

//go:build windows

package main

import (
	"fmt"
	"os/exec"

	"golang.org/x/sys/windows"
)

var (
	user32          = windows.NewLazySystemDLL("user32.dll")
	procSendMessage = user32.NewProc("SendMessageW")
)

func execPowerCommand(action string) error {
	switch action {
	case "shutdown":
		return exec.Command("shutdown", "/s", "/t", "0").Run()
	case "restart":
		return exec.Command("shutdown", "/r", "/t", "0").Run()
	case "hibernate":
		return exec.Command("shutdown", "/h").Run()
	case "sleep":
		return exec.Command("rundll32.exe", "powrprof.dll,SetSuspendState", "0,1,0").Run()
	case "lock":
		return exec.Command("rundll32.exe", "user32.dll,LockWorkStation").Run()
	case "logoff":
		return exec.Command("shutdown", "/l").Run()
	case "screen-off":
		return screenOff()
	default:
		return fmt.Errorf("unknown action: %s", action)
	}
}

func screenOff() error {
	procSendMessage.Call(
		0xFFFF,  // HWND_BROADCAST
		0x0112,  // WM_SYSCOMMAND
		0xF170,  // SC_MONITORPOWER
		2,       // MONITOR_OFF
	)
	return nil
}
