package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/mattn/go-isatty"
)

// isTTYFn reports whether the process stdout is an interactive terminal.
// Overridden in tests.
var isTTYFn = func() bool {
	return isatty.IsTerminal(os.Stdout.Fd())
}

// openEditorFn opens the user's $EDITOR with the given initial content and
// returns the content after the editor exits. Overridden in tests.
var openEditorFn = func(initial string) (string, error) {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		return "", fmt.Errorf("$EDITOR is not set; use --content to supply note body or set $EDITOR")
	}

	f, err := os.CreateTemp("", "nn-edit-*.md")
	if err != nil {
		return "", fmt.Errorf("editor: create temp file: %w", err)
	}
	defer os.Remove(f.Name())

	if _, err := f.WriteString(initial); err != nil {
		f.Close()
		return "", fmt.Errorf("editor: write temp file: %w", err)
	}
	f.Close()

	cmd := exec.Command(editor, f.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("editor: %w", err)
	}

	data, err := os.ReadFile(f.Name())
	if err != nil {
		return "", fmt.Errorf("editor: read temp file: %w", err)
	}
	return string(data), nil
}
