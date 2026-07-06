// Package pi exposes embedded Pi package resources for nn install-pi.
package pi

import "embed"

//go:embed extensions
var FS embed.FS
