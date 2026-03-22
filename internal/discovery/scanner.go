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
// ProjectName. When multiple distinct CWDs share the same basename (e.g.
// "/work/api" and "/personal/api"), the project names are expanded to
// "parent/basename" to prevent identity collisions. Unique basenames remain
// short. The returned slice is sorted alphabetically by project name.
func GroupByProject(sessions []Session) []Project {
	// Pass 1: group by basename, tracking distinct CWDs per basename.
	type basenameGroup struct {
		cwds     map[string]struct{} // distinct CWDs seen
		sessions []Session
	}
	byBase := make(map[string]*basenameGroup)

	for _, s := range sessions {
		name := s.ProjectName
		if name == "" {
			name = "(unknown)"
		}

		g, ok := byBase[name]
		if !ok {
			g = &basenameGroup{cwds: make(map[string]struct{})}
			byBase[name] = g
		}
		g.cwds[s.CWD] = struct{}{}
		g.sessions = append(g.sessions, s)
	}

	// Pass 2: detect collisions and build final projects.
	byName := make(map[string]*Project)

	for basename, g := range byBase {
		collides := len(g.cwds) > 1

		for i := range g.sessions {
			s := &g.sessions[i]
			name := basename

			if collides {
				expanded := expandedName(s.CWD)
				if expanded != "" {
					name = expanded
				}
				// else: CWD has no parent component, keep basename
			}

			// Update the session's ProjectName to match the resolved name.
			s.ProjectName = name

			p, ok := byName[name]
			if !ok {
				p = &Project{
					Name: name,
					Path: s.CWD,
				}
				byName[name] = p
			}
			p.Sessions = append(p.Sessions, *s)
		}
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

// expandedName returns "parent/basename" for a path, or empty string if the
// path has no parent directory (e.g. "/" or a single component).
func expandedName(cwd string) string {
	if cwd == "" {
		return ""
	}
	parent := filepath.Dir(cwd)
	base := filepath.Base(cwd)
	parentBase := filepath.Base(parent)

	// If Dir returned "/" or ".", there is no meaningful parent to prepend.
	if parentBase == "." || parentBase == "/" || parent == cwd {
		return ""
	}

	return parentBase + "/" + base
}
