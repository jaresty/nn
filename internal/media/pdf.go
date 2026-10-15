package media

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"strings"
)

const pdfKitScript = `
import Foundation
import PDFKit

let path = CommandLine.arguments[1]
guard let document = PDFDocument(url: URL(fileURLWithPath: path)) else {
    fputs("unable to open PDF\n", stderr)
    exit(1)
}
for index in 0..<document.pageCount {
    if index > 0 { print("\u{000C}", terminator: "") }
    print(document.page(at: index)?.string ?? "", terminator: "")
}
`

type DocumentPage struct {
	Number int    `json:"number"`
	Text   string `json:"text"`
}

type DocumentText struct {
	MIME      string          `json:"mime"`
	Extractor string          `json:"extractor"`
	Pages     []DocumentPage  `json:"pages,omitempty"`
	Blocks    []DocumentBlock `json:"blocks,omitempty"`
}

func IsPDF(path string, header []byte) bool {
	return strings.EqualFold(filepath.Ext(path), ".pdf") || bytes.HasPrefix(bytes.TrimSpace(header), []byte("%PDF-"))
}

func ExtractPDFText(ctx context.Context, runner ProcessRunner, path, goos string) (DocumentText, error) {
	var spec CommandSpec
	var extractor string
	switch goos {
	case "darwin":
		extractor = "pdfkit"
		spec = CommandSpec{Executable: "swift", Args: []string{"-e", pdfKitScript, path}, Streams: StreamPolicy{CaptureStdout: true, CaptureStderr: true}}
	case "linux", "windows":
		extractor = "pdftotext"
		spec = CommandSpec{Executable: "pdftotext", Args: []string{"-layout", path, "-"}, Streams: StreamPolicy{CaptureStdout: true, CaptureStderr: true}}
	default:
		return DocumentText{}, fmt.Errorf("PDF text extraction is unsupported on %s", goos)
	}
	result, err := runner.Run(ctx, spec)
	if err != nil {
		return DocumentText{}, fmt.Errorf("extract PDF text with %s: %w", extractor, err)
	}
	parts := bytes.Split(result.Stdout, []byte{0x0c})
	pages := make([]DocumentPage, 0, len(parts))
	for i, part := range parts {
		// pdftotext commonly emits one trailing form feed after the final page.
		if i == len(parts)-1 && len(bytes.TrimSpace(part)) == 0 && len(parts) > 1 {
			continue
		}
		pages = append(pages, DocumentPage{Number: i + 1, Text: strings.TrimRight(string(part), "\r\n")})
	}
	if len(pages) == 0 {
		pages = append(pages, DocumentPage{Number: 1})
	}
	return DocumentText{MIME: "application/pdf", Extractor: extractor, Pages: pages}, nil
}

func PDFExtractorDependency(goos string) (string, bool) {
	switch goos {
	case "darwin":
		return "swift", true
	case "linux", "windows":
		return "pdftotext", true
	default:
		return "", false
	}
}
