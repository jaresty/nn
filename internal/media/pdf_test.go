package media

import (
	"context"
	"strings"
	"testing"
)

func TestIsPDFUsesExtensionOrMagic(t *testing.T) {
	if !IsPDF("guide.PDF", nil) || !IsPDF("guide.bin", []byte("\n\t%PDF-1.7")) || IsPDF("movie.mp4", []byte("video")) {
		t.Fatal("PDF detection did not honor extension and magic")
	}
}

func TestExtractPDFTextSelectsPlatformAdapterAndPreservesPages(t *testing.T) {
	for _, tc := range []struct {
		goos, executable, extractor string
	}{
		{goos: "darwin", executable: "swift", extractor: "pdfkit"},
		{goos: "linux", executable: "pdftotext", extractor: "pdftotext"},
		{goos: "windows", executable: "pdftotext", extractor: "pdftotext"},
	} {
		t.Run(tc.goos, func(t *testing.T) {
			runner := runnerFunc(func(_ context.Context, spec CommandSpec) (CommandResult, error) {
				if spec.Executable != tc.executable || !strings.Contains(strings.Join(spec.Args, " "), "guide.pdf") {
					t.Fatalf("adapter command: %#v", spec)
				}
				return CommandResult{Started: true, Stdout: []byte("first page\fsecond page\f")}, nil
			})
			got, err := ExtractPDFText(context.Background(), runner, "guide.pdf", tc.goos)
			if err != nil {
				t.Fatal(err)
			}
			if got.Extractor != tc.extractor || len(got.Pages) != 2 || got.Pages[0].Number != 1 || got.Pages[1].Text != "second page" {
				t.Fatalf("page-preserving extraction: %#v", got)
			}
		})
	}
}

func TestExtractPDFTextRejectsUnsupportedPlatform(t *testing.T) {
	_, err := ExtractPDFText(context.Background(), runnerFunc(func(context.Context, CommandSpec) (CommandResult, error) {
		t.Fatal("runner should not be called")
		return CommandResult{}, nil
	}), "guide.pdf", "plan9")
	if err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("unsupported platform error: %v", err)
	}
}
