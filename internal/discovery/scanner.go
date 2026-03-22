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
// CWD. When multiple distinct CWDs share the same basename, their display names
// are expanded to include up to 3 path components (e.g. "work/api" instead of
// "api") until all names are unique.
//
// The returned slice is sorted alphabetically by project name.
func GroupByProject(sessions []Session) []Project {
	// First pass: group sessions by CWD so each unique working directory gets
	// exactly one Project entry.
	byCWD := make(map[string]*Project)
	for _, s := range sessions {
		cwd := s.CWD
		if cwd == "" {
			cwd = "(unknown)"
		}
		p, ok := byCWD[cwd]
		if !ok {
			p = &Project{
				Path: cwd,
			}
			byCWD[cwd] = p
		}
		p.Sessions = append(p.Sessions, s)
	}

	// Collect the distinct CWDs so we can run disambiguation.
	cwds := make([]string, 0, len(byCWD))
	for cwd := range byCWD {
		cwds = append(cwds, cwd)
	}

	const maxLevels = 3
	names := disambiguateNames(cwds, maxLevels)

	projects := make([]Project, 0, len(byCWD))
	for cwd, p := range byCWD {
		p.Name = names[cwd]
		projects = append(projects, *p)
	}

	sort.Slice(projects, func(i, j int) bool {
		return projects[i].Name < projects[j].Name
	})

	return projects
}

// disambiguateNames assigns a short display name to each CWD path. It begins
// with the basename of each path and progressively includes more parent
// components (up to maxLevels) for any CWDs that still share a name.
func disambiguateNames(cwds []string, maxLevels int) map[string]string {
	names := make(map[string]string, len(cwds))

	nameAt := func(path string, n int) string {
		if path == "" || path == "(unknown)" {
			return "(unknown)"
		}
		parts := strings.Split(filepath.ToSlash(filepath.Clean(path)), "/")
		if n >= len(parts) {
			return strings.Join(parts, "/")
		}
		return strings.Join(parts[len(parts)-n:], "/")
	}

	depth := make(map[string]int, len(cwds))
	for _, cwd := range cwds {
		depth[cwd] = 1
		names[cwd] = nameAt(cwd, 1)
	}

	for level := 1; level < maxLevels; level++ {
		nameCount := make(map[string]int, len(cwds))
		for _, cwd := range cwds {
			nameCount[names[cwd]]++
		}

		anyExpanded := false
		for _, cwd := range cwds {
			if nameCount[names[cwd]] > 1 {
				depth[cwd]++
				names[cwd] = nameAt(cwd, depth[cwd])
				anyExpanded = true
			}
		}
		if !anyExpanded {
			break
		}
	}

	return names
}
