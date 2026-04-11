// Package skills exposes the embedded skill directories for nn install-skills.
package skills

import "embed"

//go:embed nn-workflow nn-guide
var FS embed.FS
