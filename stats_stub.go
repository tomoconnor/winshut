// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

//go:build !windows

package main

import (
	"runtime"
	"time"
)

var startTime = time.Now()

type systemStats struct {
	CPUUsage      float64 `json:"cpu_usage_percent"`
	MemoryTotal   uint64  `json:"memory_total_bytes"`
	MemoryFree    uint64  `json:"memory_free_bytes"`
	MemoryUsed    uint64  `json:"memory_used_bytes"`
	UptimeSeconds int64   `json:"uptime_seconds"`
}

func getSystemStats() (*systemStats, error) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return &systemStats{
		CPUUsage:      0,
		MemoryTotal:   m.Sys,
		MemoryFree:    m.Sys - m.Alloc,
		MemoryUsed:    m.Alloc,
		UptimeSeconds: int64(time.Since(startTime).Seconds()),
	}, nil
}
