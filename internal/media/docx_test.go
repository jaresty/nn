package media

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func writeDOCXFixture(t *testing.T, documentXML string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "guide.docx")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	archive := zip.NewWriter(file)
	entry, err := archive.Create("word/document.xml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write([]byte(documentXML)); err != nil {
		t.Fatal(err)
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestExtractDOCXTextPreservesParagraphAndTableOrder(t *testing.T) {
	path := writeDOCXFixture(t, `<?xml version="1.0"?><w:document xmlns:w="urn:w"><w:body>
<w:p><w:r><w:t>Introduction</w:t></w:r></w:p>
<w:tbl><w:tr><w:tc><w:p><w:r><w:t>Name</w:t></w:r></w:p></w:tc><w:tc><w:p><w:r><w:t>Value</w:t></w:r></w:p></w:tc></w:tr><w:tr><w:tc><w:p><w:r><w:t>Alpha</w:t></w:r></w:p></w:tc><w:tc><w:p><w:r><w:t>42</w:t></w:r></w:p></w:tc></w:tr></w:tbl>
<w:p><w:r><w:t>Conclusion</w:t></w:r></w:p>
</w:body></w:document>`)
	got, err := ExtractDOCXText(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Extractor != "go-ooxml" || len(got.Blocks) != 3 {
		t.Fatalf("DOCX extraction: %#v", got)
	}
	if got.Blocks[0].Ordinal != 1 || got.Blocks[0].Text != "Introduction" || got.Blocks[1].Kind != "table" || got.Blocks[1].Rows[1][1] != "42" || got.Blocks[2].Text != "Conclusion" {
		t.Fatalf("ordered blocks: %#v", got.Blocks)
	}
}

func TestExtractDOCXTextRejectsNonDOCXPackage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "broken.docx")
	if err := os.WriteFile(path, []byte("not a zip"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ExtractDOCXText(path); err == nil {
		t.Fatal("invalid DOCX package succeeded")
	}
}
