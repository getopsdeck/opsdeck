//go:build linux

package discovery

import (
	"math"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// IsProcessAlive checks whether a process with the given PID exists.
// On Linux, this uses kill(pid, 0) which returns an error if the process
// does not exist. EPERM (permission denied) still means the process exists.
func IsProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	if err == nil {
		return true
	}
	// EPERM means we can't signal it but it exists.
	return err == syscall.EPERM
}

// getProcessStartTime returns the start time of a process on Linux by
// reading /proc/<pid>/stat. Returns zero time if the process doesn't exist
// or the file can't be read.
func getProcessStartTime(pid int) time.Time {
	if pid <= 0 {
		return time.Time{}
	}

	data, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
	if err != nil {
		return time.Time{}
	}

	// /proc/<pid>/stat format: <pid> (<comm>) <state> ... field 22 is starttime
	// The comm field can contain spaces and parentheses, so we find the last ')'.
	content := string(data)
	idx := strings.LastIndex(content, ")")
	if idx < 0 || idx+2 >= len(content) {
		return time.Time{}
	}

	fields := strings.Fields(content[idx+2:])
	// Field 22 (starttime) is at index 19 in the fields after the comm.
	// (fields[0] = state, fields[1] = ppid, ... fields[19] = starttime)
	if len(fields) < 20 {
		return time.Time{}
	}

	startTicks, err := strconv.ParseInt(fields[19], 10, 64)
	if err != nil {
		return time.Time{}
	}

	// Convert from clock ticks to time. SC_CLK_TCK is typically 100 on Linux.
	clkTck := int64(100) // sysconf(_SC_CLK_TCK) default
	bootTime := getBootTime()
	if bootTime.IsZero() {
		return time.Time{}
	}

	startSec := startTicks / clkTck
	startNsec := (startTicks % clkTck) * (1e9 / clkTck)
	return bootTime.Add(time.Duration(startSec)*time.Second + time.Duration(startNsec))
}

// getBootTime reads the system boot time from /proc/stat.
func getBootTime() time.Time {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return time.Time{}
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "btime ") {
			fields := strings.Fields(line)
			if len(fields) < 2 {
				return time.Time{}
			}
			sec, err := strconv.ParseInt(fields[1], 10, 64)
			if err != nil {
				return time.Time{}
			}
			return time.Unix(sec, 0)
		}
	}
	return time.Time{}
}

// CheckSession checks whether the process for a session is still the same
// process that started the session. It verifies both that the PID is alive
// and that the process start time is close to the session's startedAt.
//
// This detects PID reuse: if a process dies and a new process gets the same
// PID, the start times won't match.
func CheckSession(pid int, startedAt time.Time) bool {
	if !IsProcessAlive(pid) {
		return false
	}

	procStart := getProcessStartTime(pid)
	if procStart.IsZero() {
		// Couldn't determine start time; assume alive if the process
		// exists.
		return true
	}

	diff := procStart.Sub(startedAt)
	if diff < 0 {
		diff = -diff
	}
	return diff < 5*time.Second || isReasonableStartTime(procStart, startedAt)
}

// isReasonableStartTime provides a fallback check for cases where the
// session startedAt is recorded in a different epoch or precision.
func isReasonableStartTime(procStart, sessionStart time.Time) bool {
	diff := math.Abs(float64(procStart.Unix() - sessionStart.Unix()))
	return diff < 600 // 10 minutes
}
