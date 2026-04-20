package cmd

import "embed"

//go:embed templates/graph.html templates/d3.min.js
var graphTemplateFS embed.FS
