// Package plugins exposes the embedded Claude Code plugin for nn install-hooks.
package plugins

import "embed"

//go:embed .claude-plugin nn-hooks
var FS embed.FS
