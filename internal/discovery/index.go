package discovery

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// IndexEntry holds the subset of fields we extract from a sessions-index.json
// entry. Unknown fields are silently ignored.
type IndexEntry struct {
	SessionID    string `json:"sessionId"`
	Summary      string `json:"summary"`
	MessageCount int    `json:"messageCount"`
}

// sessionsIndex is the top-level structure of sessions-index.json.
type sessionsIndex struct {
	Version int          `json:"version"`
	Entries []IndexEntry `json:"entries"`
}

// ParseSessionIndex reads a sessions-index.json file and returns a map of
// session ID to IndexEntry. This is hints-only data used to enrich sessions
// with summaries and message counts.
//
// Defensive behavior:
//   - Returns empty map and no error if the file does not exist.
//   - Returns empty map and no error if the file is malformed JSON.
//   - Silently ignores unknown fields in the JSON.
//   - Never fails. If anything goes wrong, you just get less data.
func ParseSessionIndex(path string) (map[string]IndexEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return make(map[string]IndexEntry), nil
		}
		// For any other read error, still return empty rather than failing.
		return make(map[string]IndexEntry), nil
	}

	var idx sessionsIndex
	if err := json.Unmarshal(data, &idx); err != nil {
		// Malformed JSON -- return empty, don't fail.
		return make(map[string]IndexEntry), nil
	}

	result := make(map[string]IndexEntry, len(idx.Entries))
	for _, e := range idx.Entries {
		if e.SessionID == "" {
			continue
		}
		result[e.SessionID] = e
	}

	return result, nil
}

// FindSessionIndex locates the sessions-index.json file for a given project
// CWD under the projects directory.
//
// Returns an empty string if the file does not exist.
func FindSessionIndex(projectsDir, cwd string) string {
	encoded := EncodeCWD(cwd)
	path := filepath.Join(projectsDir, encoded, "sessions-index.json")

	if _, err := os.Stat(path); err != nil {
		return ""
	}
	return path
}
