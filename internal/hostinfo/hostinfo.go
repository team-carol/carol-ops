// Package hostinfo reads basic load/memory/disk figures for the box
// carol-ops runs on. Docker containers share the host's /proc by default
// (no cgroup-aware masking unless something explicitly sets it up), so
// /proc/meminfo and /proc/loadavg here reflect the real host, not just this
// container's own cgroup — good enough for an at-a-glance panel without
// pulling in a metrics library.
package hostinfo

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"syscall"
)

type Info struct {
	LoadAvg1        float64 `json:"loadAvg1"`
	MemTotalBytes   uint64  `json:"memTotalBytes"`
	MemUsedBytes    uint64  `json:"memUsedBytes"`
	DiskTotalBytes  uint64  `json:"diskTotalBytes"`
	DiskUsedBytes   uint64  `json:"diskUsedBytes"`
}

func Collect() (*Info, error) {
	info := &Info{}

	if load, err := readLoadAvg1(); err == nil {
		info.LoadAvg1 = load
	}
	if total, used, err := readMemInfo(); err == nil {
		info.MemTotalBytes, info.MemUsedBytes = total, used
	}
	if total, used, err := readDiskUsage("/"); err == nil {
		info.DiskTotalBytes, info.DiskUsedBytes = total, used
	}
	return info, nil
}

func readLoadAvg1() (float64, error) {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return 0, err
	}
	fields := strings.Fields(string(data))
	if len(fields) == 0 {
		return 0, nil
	}
	return strconv.ParseFloat(fields[0], 64)
}

// readMemInfo parses MemTotal/MemAvailable (kB) from /proc/meminfo. Used is
// derived as total-available rather than total-free, since "available"
// accounts for reclaimable cache/buffers the way `free -h` does — free
// alone would make a healthy box look far more full than it is.
func readMemInfo() (totalBytes, usedBytes uint64, err error) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()

	var totalKB, availableKB uint64
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "MemTotal:"):
			totalKB = parseMemInfoValue(line)
		case strings.HasPrefix(line, "MemAvailable:"):
			availableKB = parseMemInfoValue(line)
		}
	}
	totalBytes = totalKB * 1024
	if availableKB <= totalKB {
		usedBytes = (totalKB - availableKB) * 1024
	}
	return totalBytes, usedBytes, scanner.Err()
}

func parseMemInfoValue(line string) uint64 {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return 0
	}
	n, _ := strconv.ParseUint(fields[1], 10, 64)
	return n
}

func readDiskUsage(path string) (totalBytes, usedBytes uint64, err error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, 0, err
	}
	blockSize := uint64(stat.Bsize)
	totalBytes = stat.Blocks * blockSize
	freeBytes := stat.Bfree * blockSize
	usedBytes = totalBytes - freeBytes
	return totalBytes, usedBytes, nil
}
