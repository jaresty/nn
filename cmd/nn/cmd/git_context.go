package cmd

import (
	"os/exec"
	"path/filepath"
	"strings"
)

// gitContextQuery returns repo name and branch as extra BM25 query tokens
// when the cwd is inside a git repo with a remote. Returns "" otherwise.
func gitContextQuery() string {
	remoteURL, err := exec.Command("git", "remote", "get-url", "origin").Output()
	if err != nil {
		return ""
	}
	branch, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return ""
	}

	repoName := strings.TrimSuffix(filepath.Base(strings.TrimSpace(string(remoteURL))), ".git")
	branchName := strings.TrimSpace(string(branch))

	var parts []string
	if repoName != "" {
		parts = append(parts, repoName)
	}
	if branchName != "" && branchName != "HEAD" {
		parts = append(parts, branchName)
	}
	return strings.Join(parts, " ")
}
