package discovery

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// maxTailBytes is the maximum number of bytes to read from the end of a
// transcript file. Reading the full file is wasteful for large transcripts.
const maxTailBytes = 32 * 1024

// transcriptEntry holds the subset of fields we care about from a JSONL
// transcript line.
type transcriptEntry struct {
	Timestamp string `json:"timestamp"`
}

// ReadLastActivity reads the tail of a JSONL transcript file and returns the
// timestamp of the most recent entry that has one.
//
// If the file does not exist or is empty, it returns the zero time and no
// error. Malformed lines are silently skipped.
func ReadLastActivity(path string) (time.Time, error) {
	data, err := readTail(path, maxTailBytes)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return time.Time{}, nil
		}
		return time.Time{}, err
	}

	if len(data) == 0 {
		return time.Time{}, nil
	}

	// Scan all lines in the tail and keep the latest timestamp.
	var latest time.Time
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var entry transcriptEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			// Skip malformed lines.
			continue
		}

		if entry.Timestamp == "" {
			continue
		}

		t, err := time.Parse(time.RFC3339Nano, entry.Timestamp)
		if err != nil {
			continue
		}

		if t.After(latest) {
			latest = t
		}
	}

	return latest, nil
}

// readTail reads the last n bytes of a file. If the file is smaller than n
// bytes, it returns the entire contents.
func readTail(path string, n int64) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, err
	}

	size := info.Size()
	if size == 0 {
		return nil, nil
	}

	offset := int64(0)
	readSize := size
	if size > n {
		offset = size - n
		readSize = n
	}

	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return nil, err
	}

	buf := make([]byte, readSize)
	nRead, err := io.ReadFull(f, buf)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) {
		return nil, err
	}

	return buf[:nRead], nil
}

// EncodeCWD converts a filesystem path to the encoded directory name used by
// Claude Code under ~/.claude/projects/. Forward slashes are replaced with
// hyphens, and non-ASCII characters are also replaced with hyphens to match
// Claude Code's actual encoding behavior.
//
// Example: "/Users/jason/project" -> "-Users-jason-project"
// Example: "/Users/jason/Documents/实习/理想实习/FusionSQL" -> "-Users-jason-Documents---------FusionSQL"
func EncodeCWD(cwd string) string {
	var b strings.Builder
	for _, r := range cwd {
		if r == '/' || r > 127 {
			b.WriteByte('-')
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// FindTranscriptPath locates the transcript JSONL file for a given session.
// It tries two strategies:
//  1. Direct path: projectsDir/<encodedCWD>/<sessionID>.jsonl
//  2. Fallback scan: search all project dirs for the sessionID.jsonl file
//
// Returns an empty string if the transcript file does not exist.
func FindTranscriptPath(projectsDir, cwd, sessionID string) string {
	// Strategy 1: direct encoded path.
	encoded := EncodeCWD(cwd)
	path := filepath.Join(projectsDir, encoded, sessionID+".jsonl")
	if _, err := os.Stat(path); err == nil {
		return path
	}

	// Strategy 2: scan project directories for the session file.
	// This handles cases where our encoding doesn't match Claude Code's exactly.
	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		return ""
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		candidate := filepath.Join(projectsDir, entry.Name(), sessionID+".jsonl")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	return ""
}
