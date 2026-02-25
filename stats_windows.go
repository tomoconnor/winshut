// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

//go:build windows

package main

import (
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type systemStats struct {
	CPUUsage      float64 `json:"cpu_usage_percent"`
	MemoryTotal   uint64  `json:"memory_total_bytes"`
	MemoryFree    uint64  `json:"memory_free_bytes"`
	MemoryUsed    uint64  `json:"memory_used_bytes"`
	UptimeSeconds int64   `json:"uptime_seconds"`
}

func getSystemStats() (*systemStats, error) {
	cpu, err := getCPUUsage()
	if err != nil {
		return nil, err
	}

	total, free, err := getMemory()
	if err != nil {
		return nil, err
	}

	uptime, err := getUptime()
	if err != nil {
		return nil, err
	}

	return &systemStats{
		CPUUsage:      cpu,
		MemoryTotal:   total,
		MemoryFree:    free,
		MemoryUsed:    total - free,
		UptimeSeconds: uptime,
	}, nil
}

func getCPUUsage() (float64, error) {
	out, err := exec.Command("wmic", "cpu", "get", "LoadPercentage", "/value").Output()
	if err != nil {
		return 0, err
	}
	// Output: "LoadPercentage=XX\r\n"
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "LoadPercentage=") {
			val := strings.TrimPrefix(line, "LoadPercentage=")
			return strconv.ParseFloat(strings.TrimSpace(val), 64)
		}
	}
	return 0, nil
}

func getMemory() (total, free uint64, err error) {
	out, err := exec.Command("wmic", "OS", "get", "TotalVisibleMemorySize,FreePhysicalMemory", "/value").Output()
	if err != nil {
		return 0, 0, err
	}
	// Output contains "FreePhysicalMemory=XXXX\r\nTotalVisibleMemorySize=XXXX\r\n" (in KB)
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "TotalVisibleMemorySize=") {
			val := strings.TrimPrefix(line, "TotalVisibleMemorySize=")
			kb, _ := strconv.ParseUint(strings.TrimSpace(val), 10, 64)
			total = kb * 1024
		}
		if strings.HasPrefix(line, "FreePhysicalMemory=") {
			val := strings.TrimPrefix(line, "FreePhysicalMemory=")
			kb, _ := strconv.ParseUint(strings.TrimSpace(val), 10, 64)
			free = kb * 1024
		}
	}
	return total, free, nil
}

func getUptime() (int64, error) {
	out, err := exec.Command("wmic", "os", "get", "LastBootUpTime", "/value").Output()
	if err != nil {
		return 0, err
	}
	// Output: "LastBootUpTime=20250101120000.000000-060\r\n"
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "LastBootUpTime=") {
			val := strings.TrimPrefix(line, "LastBootUpTime=")
			val = strings.TrimSpace(val)
			if len(val) < 14 {
				continue
			}
			// Parse WMI datetime: YYYYMMDDHHmmss
			t, err := time.Parse("20060102150405", val[:14])
			if err != nil {
				return 0, err
			}
			return int64(time.Since(t).Seconds()), nil
		}
	}
	return 0, nil
}
