package discovery

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// rawSession is the JSON structure stored in ~/.claude/sessions/<pid>.json.
type rawSession struct {
	PID       int    `json:"pid"`
	SessionID string `json:"sessionId"`
	CWD       string `json:"cwd"`
	StartedAt int64  `json:"startedAt"` // Unix milliseconds
}

// ScanSessions reads all session files from the given directory and returns
// parsed Session values. It silently skips files that are not valid JSON, are
// not .json files, or are missing required fields.
//
// If the directory does not exist, ScanSessions returns an empty slice and no
// error.
func ScanSessions(sessionsDir string) ([]Session, error) {
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading sessions dir: %w", err)
	}

	var sessions []Session
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		path := filepath.Join(sessionsDir, entry.Name())
		s, err := parseSessionFile(path)
		if err != nil {
			// Skip malformed or incomplete files.
			continue
		}
		sessions = append(sessions, s)
	}

	return sessions, nil
}

// parseSessionFile reads and validates a single session JSON file.
func parseSessionFile(path string) (Session, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Session{}, err
	}

	var raw rawSession
	if err := json.Unmarshal(data, &raw); err != nil {
		return Session{}, fmt.Errorf("parsing %s: %w", path, err)
	}

	// Require at minimum a session ID and PID.
	if raw.SessionID == "" {
		return Session{}, fmt.Errorf("missing sessionId in %s", path)
	}
	if raw.PID == 0 {
		return Session{}, fmt.Errorf("missing pid in %s", path)
	}

	startedAt := time.UnixMilli(raw.StartedAt)

	return Session{
		ID:          raw.SessionID,
		PID:         raw.PID,
		CWD:         raw.CWD,
		ProjectName: projectName(raw.CWD),
		StartedAt:   startedAt,
	}, nil
}

// projectName extracts a human-readable project name from a working directory
// path. It uses the last path component (basename).
func projectName(cwd string) string {
	if cwd == "" {
		return ""
	}
	return filepath.Base(cwd)
}

// GroupByProject groups a flat list of sessions into Project values, keyed by
// ProjectName. The returned slice is sorted alphabetically by project name.
func GroupByProject(sessions []Session) []Project {
	byName := make(map[string]*Project)

	for _, s := range sessions {
		name := s.ProjectName
		if name == "" {
			name = "(unknown)"
		}

		p, ok := byName[name]
		if !ok {
			p = &Project{
				Name: name,
				Path: s.CWD,
			}
			byName[name] = p
		}
		p.Sessions = append(p.Sessions, s)
	}

	projects := make([]Project, 0, len(byName))
	for _, p := range byName {
		projects = append(projects, *p)
	}

	sort.Slice(projects, func(i, j int) bool {
		return projects[i].Name < projects[j].Name
	})

	return projects
}
