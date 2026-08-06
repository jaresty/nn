package cmd

import "strings"

var mermaidLabelEscaper = strings.NewReplacer(
	"#", "#35;",
	"\"", "#34;",
	"&", "#38;",
	"<", "#60;",
	">", "#62;",
	"\n", "#10;",
	"\r", "#13;",
	"\\", "#92;",
	"|", "#124;",
)

func escapeMermaidLabel(value string) string {
	return mermaidLabelEscaper.Replace(value)
}
