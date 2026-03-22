//go:build darwin

package discovery

import (
	"encoding/binary"
	"math"
	"syscall"
	"time"
	"unsafe"
)

// IsProcessAlive checks whether a process with the given PID exists.
// On macOS, this uses kill(pid, 0) which returns an error if the process
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

// getProcessStartTime returns the start time of a process on macOS using
// sysctl. Returns zero time if the process doesn't exist or the call fails.
func getProcessStartTime(pid int) time.Time {
	if pid <= 0 {
		return time.Time{}
	}

	// CTL_KERN, KERN_PROC, KERN_PROC_PID
	mib := [4]int32{1, 14, 1, int32(pid)}

	// First call to get the size.
	var size uintptr
	_, _, errno := syscall.Syscall6(
		syscall.SYS___SYSCTL,
		uintptr(unsafe.Pointer(&mib[0])),
		4,
		0,
		uintptr(unsafe.Pointer(&size)),
		0,
		0,
	)
	if errno != 0 || size == 0 {
		return time.Time{}
	}

	buf := make([]byte, size)
	_, _, errno = syscall.Syscall6(
		syscall.SYS___SYSCTL,
		uintptr(unsafe.Pointer(&mib[0])),
		4,
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&size)),
		0,
		0,
	)
	if errno != 0 {
		return time.Time{}
	}

	// The kinfo_proc struct on arm64 macOS has p_starttime (a timeval) at
	// offset 128. On amd64 it may differ, but arm64 is what we target.
	// The timeval struct is { int64 tv_sec; int64 tv_usec } on 64-bit.
	//
	// We'll use a safer approach: search for the timeval containing the
	// start time by using the sysctl KERN_PROC_PID result. The struct
	// layout is stable per-architecture.
	return parseKinfoStartTime(buf)
}

// parseKinfoStartTime extracts the process start time from a kinfo_proc
// buffer. On darwin/arm64 (and amd64), the extern_proc's p_starttime field
// is at a known offset within the kinfo_proc structure.
//
// kinfo_proc layout (simplified):
//   - kp_proc (extern_proc): starts at offset 0
//     - p_starttime (struct timeval): offset 128 on arm64, 136 on amd64
//
// We try both known offsets to be robust.
func parseKinfoStartTime(buf []byte) time.Time {
	offsets := []int{128, 136}
	for _, off := range offsets {
		if off+16 > len(buf) {
			continue
		}
		tvSec := int64(binary.LittleEndian.Uint64(buf[off : off+8]))
		tvUsec := int64(binary.LittleEndian.Uint64(buf[off+8 : off+16]))

		// Sanity check: tv_sec should be a reasonable Unix timestamp
		// (after year 2000, before year 2100).
		if tvSec > 946684800 && tvSec < 4102444800 && tvUsec >= 0 && tvUsec < 1e6 {
			return time.Unix(tvSec, tvUsec*1000)
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
		// exists. This is a safe fallback -- worst case we show a dead
		// session as alive until the next scan.
		return true
	}

	// Allow up to 5 seconds of drift between the session file timestamp
	// and the actual process start time, since they're recorded at
	// different points during startup.
	diff := procStart.Sub(startedAt)
	if diff < 0 {
		diff = -diff
	}
	return diff < 5*time.Second || isReasonableStartTime(procStart, startedAt)
}

// isReasonableStartTime provides a fallback for second-granularity rounding
// differences between process start time and session startedAt.
func isReasonableStartTime(procStart, sessionStart time.Time) bool {
	diff := math.Abs(float64(procStart.Unix() - sessionStart.Unix()))
	return diff < 2 // only compensate for rounding to whole seconds
}
