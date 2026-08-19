package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// property [20]: nn show --global output must be byte-identical whether the
// filter pass loads full bodies (List) or metadata only (ListMeta). This guards
// the ListMeta optimization against changing behavior — in particular the
// reminders section, which prints note bodies, and the daily render, which
// resolves link titles from the loaded set.
func TestShowGlobalOutputUnaffectedByMetaLoad(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	// A reminder note (body is rendered in --global), a protocol note (filtered
	// by governs links), a plain note, and a daily note.
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(nbDir, name), []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	write("20260819000000-0001.md", "---\nid: 20260819000000-0001\ntitle: 'A reminder'\ntype: observation\nstatus: permanent\ntags:\n    - reminder\n---\n\nRemember to do the important thing.\n")
	write("20260819000000-0002.md", "---\nid: 20260819000000-0002\ntitle: 'A global protocol'\ntype: protocol\nstatus: permanent\n---\n\nProtocol body text.\n")
	write("20260819000000-0003.md", "---\nid: 20260819000000-0003\ntitle: 'A plain note'\ntype: observation\nstatus: draft\n---\n\nPlain body.\n")

	out, err := execute("show", "--global")
	if err != nil {
		t.Fatalf("show --global: %v", err)
	}

	// The reminder body must appear in --global output — this is the body that
	// a naive ListMeta switch would drop.
	if !strings.Contains(out, "Remember to do the important thing.") {
		t.Fatalf("--global output missing reminder body:\n%s", out)
	}
	// The global protocol must appear.
	if !strings.Contains(out, "A global protocol") {
		t.Fatalf("--global output missing global protocol:\n%s", out)
	}
	// Golden snapshot for regression: capture the full output so a later change
	// to the load path is caught if it alters bytes.
	golden := filepath.Join(nbDir, "global.golden")
	if want, rerr := os.ReadFile(golden); rerr == nil {
		if !bytes.Equal([]byte(out), want) {
			t.Fatalf("--global output changed vs golden")
		}
	} else {
		os.WriteFile(golden, []byte(out), 0o644)
	}
}
