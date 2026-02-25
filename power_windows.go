// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

//go:build windows

package main

import (
	"fmt"
	"os/exec"
)

func execPowerCommand(action string) error {
	var cmd *exec.Cmd

	switch action {
	case "shutdown":
		cmd = exec.Command("shutdown", "/s", "/t", "0")
	case "restart":
		cmd = exec.Command("shutdown", "/r", "/t", "0")
	case "hibernate":
		cmd = exec.Command("shutdown", "/h")
	case "sleep":
		cmd = exec.Command("rundll32.exe", "powrprof.dll,SetSuspendState", "0,1,0")
	case "lock":
		cmd = exec.Command("rundll32.exe", "user32.dll,LockWorkStation")
	case "logoff":
		cmd = exec.Command("shutdown", "/l")
	case "screen-off":
		cmd = exec.Command("powershell", "-NoProfile", "-Command",
			`Add-Type -TypeDefinition 'using System;using System.Runtime.InteropServices;public class Screen{[DllImport("user32.dll")]static extern IntPtr SendMessage(IntPtr h,uint m,IntPtr w,IntPtr l);public static void Off(){SendMessage((IntPtr)0xFFFF,0x0112,(IntPtr)0xF170,(IntPtr)2);}}'; [Screen]::Off()`)
	default:
		return fmt.Errorf("unknown action: %s", action)
	}

	return cmd.Run()
}
