// Package skills exposes the embedded skill directories for nn install-skills.
package skills

import "embed"

//go:embed nn-workflow nn-guide nn-capture-discipline nn-link-suggester nn-refine nn-session-debrief nn-refine-workflow
var FS embed.FS
