package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

type transcriptRoot struct {
	Source string
	Path   string
}

func transcriptDefaultRoots(home string, getenv func(string) string) []transcriptRoot {
	codexHome := getenv("CODEX_HOME")
	if codexHome == "" {
		codexHome = filepath.Join(home, ".codex")
	}
	return []transcriptRoot{
		{Source: "claude", Path: filepath.Join(home, ".claude", "projects")},
		{Source: "codex", Path: filepath.Join(codexHome, "sessions")},
		{Source: "codex", Path: filepath.Join(codexHome, "archived_sessions")},
		{Source: "pi", Path: filepath.Join(home, ".pi", "agent", "sessions")},
	}
}

func transcriptAvailableDefaultRoots(home string, getenv func(string) string) (available []string, unavailable []transcriptRoot, err error) {
	seen := map[string]bool{}
	for _, root := range transcriptDefaultRoots(home, getenv) {
		info, statErr := os.Stat(root.Path)
		if statErr != nil {
			if os.IsNotExist(statErr) {
				unavailable = append(unavailable, root)
				continue
			}
			return nil, unavailable, statErr
		}
		if !info.IsDir() {
			unavailable = append(unavailable, root)
			continue
		}
		canonical, evalErr := filepath.EvalSymlinks(root.Path)
		if evalErr != nil {
			return nil, unavailable, evalErr
		}
		canonical, evalErr = filepath.Abs(canonical)
		if evalErr != nil {
			return nil, unavailable, evalErr
		}
		if !seen[canonical] {
			seen[canonical] = true
			available = append(available, canonical)
		}
	}
	return available, unavailable, nil
}

func defaultTranscriptRoots() ([]string, []transcriptRoot, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, nil, err
	}
	return transcriptAvailableDefaultRoots(home, os.Getenv)
}

func reportUnavailableTranscriptRoots(w io.Writer, roots []transcriptRoot) {
	for _, root := range roots {
		fmt.Fprintf(w, "transcript %s root unavailable: %s\n", root.Source, root.Path)
	}
}

func resolveTranscript(session string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return resolveTranscriptSession(session, home, os.Getenv)
}

func transcriptSessionArgs(validate cobra.PositionalArgs) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if err := validate(cmd, args); err != nil {
			return err
		}
		resolved, err := resolveTranscript(args[0])
		if err != nil {
			return err
		}
		args[0] = resolved
		return nil
	}
}

func transcriptMetadataID(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		return ""
	}
	var header struct {
		Type    string `json:"type"`
		ID      string `json:"id"`
		Payload struct {
			SessionID string `json:"session_id"`
			ID        string `json:"id"`
		} `json:"payload"`
	}
	if json.Unmarshal(scanner.Bytes(), &header) != nil {
		return ""
	}
	if header.Type == "session_meta" {
		if header.Payload.SessionID != "" {
			return header.Payload.SessionID
		}
		return header.Payload.ID
	}
	if header.Type == "session" {
		return header.ID
	}
	return ""
}

func resolveTranscriptSession(session, home string, getenv func(string) string) (string, error) {
	if info, err := os.Stat(session); err == nil && !info.IsDir() {
		return session, nil
	}
	if filepath.IsAbs(session) || strings.ContainsRune(session, filepath.Separator) || strings.HasSuffix(session, ".jsonl") || strings.HasSuffix(session, ".output") {
		// Explicit path-like operands remain byte-preserved so retained snapshots,
		// replay checks, and command test doubles keep their existing identity.
		return session, nil
	}

	roots, unavailable, err := transcriptAvailableDefaultRoots(home, getenv)
	if err != nil {
		return "", err
	}
	matches := make([]string, 0, 1)
	seen := map[string]bool{}
	for _, root := range roots {
		err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() || (!strings.HasSuffix(path, ".jsonl") && !strings.HasSuffix(path, ".output")) {
				return nil
			}
			base := filepath.Base(path)
			id := strings.TrimSuffix(strings.TrimSuffix(base, ".jsonl"), ".output")
			if id != session && base != session && transcriptMetadataID(path) != session {
				return nil
			}
			canonical, err := filepath.EvalSymlinks(path)
			if err != nil {
				return err
			}
			canonical, err = filepath.Abs(canonical)
			if err != nil {
				return err
			}
			if !seen[canonical] {
				seen[canonical] = true
				matches = append(matches, canonical)
			}
			return nil
		})
		if err != nil {
			return "", err
		}
	}
	sort.Strings(matches)
	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		searched := strings.Join(roots, ", ")
		if searched == "" {
			searched = "none"
		}
		return "", fmt.Errorf("transcript session %q not found (searched: %s; unavailable roots: %d)", session, searched, len(unavailable))
	default:
		return "", fmt.Errorf("transcript session %q is ambiguous; candidates: %s", session, strings.Join(matches, ", "))
	}
}
