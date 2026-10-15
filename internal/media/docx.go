package media

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

type DocumentBlock struct {
	Ordinal int        `json:"ordinal"`
	Kind    string     `json:"kind"`
	Text    string     `json:"text,omitempty"`
	Rows    [][]string `json:"rows,omitempty"`
}

func IsDOCX(path string) bool { return strings.EqualFold(filepath.Ext(path), ".docx") }

func ExtractDOCXText(path string) (DocumentText, error) {
	archive, err := zip.OpenReader(path)
	if err != nil {
		return DocumentText{}, fmt.Errorf("open DOCX package: %w", err)
	}
	defer archive.Close()
	var document *zip.File
	for _, file := range archive.File {
		if file.Name == "word/document.xml" {
			document = file
			break
		}
	}
	if document == nil {
		return DocumentText{}, fmt.Errorf("DOCX package has no word/document.xml")
	}
	reader, err := document.Open()
	if err != nil {
		return DocumentText{}, fmt.Errorf("open DOCX document XML: %w", err)
	}
	defer reader.Close()
	blocks, err := decodeDOCXBlocks(reader)
	if err != nil {
		return DocumentText{}, err
	}
	return DocumentText{MIME: "application/vnd.openxmlformats-officedocument.wordprocessingml.document", Extractor: "go-ooxml", Blocks: blocks}, nil
}

func decodeDOCXBlocks(reader io.Reader) ([]DocumentBlock, error) {
	decoder := xml.NewDecoder(reader)
	var blocks []DocumentBlock
	var paragraph strings.Builder
	var rows [][]string
	var row []string
	var cell []string
	inParagraph, inTable, inCell, inText := false, false, false, false
	emitParagraph := func() {
		text := strings.TrimSpace(paragraph.String())
		paragraph.Reset()
		if inCell {
			if text != "" {
				cell = append(cell, text)
			}
		} else if text != "" {
			blocks = append(blocks, DocumentBlock{Ordinal: len(blocks) + 1, Kind: "paragraph", Text: text})
		}
	}
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("decode DOCX document XML: %w", err)
		}
		switch value := token.(type) {
		case xml.StartElement:
			switch value.Name.Local {
			case "tbl":
				inTable, rows = true, nil
			case "tr":
				row = nil
			case "tc":
				inCell, cell = true, nil
			case "p":
				inParagraph = true
				paragraph.Reset()
			case "t":
				inText = true
			case "tab":
				if inParagraph {
					paragraph.WriteByte('\t')
				}
			case "br":
				if inParagraph {
					paragraph.WriteByte('\n')
				}
			}
		case xml.CharData:
			if inParagraph && inText {
				paragraph.Write([]byte(value))
			}
		case xml.EndElement:
			switch value.Name.Local {
			case "t":
				inText = false
			case "p":
				emitParagraph()
				inParagraph = false
			case "tc":
				row = append(row, strings.Join(cell, "\n"))
				inCell = false
			case "tr":
				rows = append(rows, row)
			case "tbl":
				blocks = append(blocks, DocumentBlock{Ordinal: len(blocks) + 1, Kind: "table", Rows: rows})
				inTable = false
			}
		}
	}
	if inTable {
		return nil, fmt.Errorf("decode DOCX document XML: unterminated table")
	}
	return blocks, nil
}
