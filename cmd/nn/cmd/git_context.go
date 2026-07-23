package cmd

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// gitContextQuery returns repo name and branch as extra BM25 query tokens
// when the cwd is inside a git repo with a remote. Returns "" otherwise.
func gitContextQuery() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return gitContextQueryFromFiles(cwd)
}

// gitContextQueryFromFiles reads .git/HEAD and .git/config directly (no subprocess)
// starting from dir and walking up to find the git root.
func gitContextQueryFromFiles(dir string) string {
	gitDir := findGitDir(dir)
	if gitDir == "" {
		return ""
	}

	branch := readBranch(gitDir)
	repoName := readRemoteName(gitDir)

	if repoName == "" {
		return ""
	}

	var parts []string
	parts = append(parts, repoName)
	if branch != "" {
		parts = append(parts, branch)
	}
	return strings.Join(parts, " ")
}

// findGitDir walks up from dir until it finds a .git directory, returning its path.
func findGitDir(dir string) string {
	for {
		candidate := filepath.Join(dir, ".git")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// readBranch parses .git/HEAD and returns the branch name, or "" for detached HEAD.
func readBranch(gitDir string) string {
	data, err := os.ReadFile(filepath.Join(gitDir, "HEAD"))
	if err != nil {
		return ""
	}
	line := strings.TrimSpace(string(data))
	const prefix = "ref: refs/heads/"
	if !strings.HasPrefix(line, prefix) {
		return "" // detached HEAD
	}
	return strings.TrimPrefix(line, prefix)
}

// readRemoteName parses .git/config and returns the base name of the origin remote URL.
func readRemoteName(gitDir string) string {
	f, err := os.Open(filepath.Join(gitDir, "config"))
	if err != nil {
		return ""
	}
	defer f.Close()

	inOrigin := false
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == `[remote "origin"]` {
			inOrigin = true
			continue
		}
		if strings.HasPrefix(line, "[") {
			inOrigin = false
			continue
		}
		if inOrigin && strings.HasPrefix(line, "url") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				url := strings.TrimSpace(parts[1])
				base := filepath.Base(url)
				return strings.TrimSuffix(base, ".git")
			}
		}
	}
	return ""
}
