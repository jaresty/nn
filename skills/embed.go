// Package skills exposes the embedded skill directories for nn install-skills.
package skills

import "embed"

// Embedding each skill directory recursively includes SKILL.md and references/.
//
//go:embed nn-*
var FS embed.FS
