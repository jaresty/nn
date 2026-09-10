package cmd

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestTranscriptRecordReaderByteEquivalence(t *testing.T) {
	input := "\u2003\n malformed\nnull\r\n" +
		" {\"type\":\"message\",\"id\":\"one\",\"message\":{\"role\":\"user\",\"content\":\"こんにちは\"}} \r\n" +
		"42\n{\"type\":\"message\",\"id\":\"two\",\"message\":{\"role\":\"assistant\",\"content\":\"retained without newline\"}}"
	path := filepath.Join(t.TempDir(), "records.jsonl")
	if err := os.WriteFile(path, []byte(input), 0600); err != nil {
		t.Fatal(err)
	}
	// Original string-based procedure is the compatibility oracle, including its
	// final-line behavior (distinct from capture's complete-prefix contract).
	var expected []rawRecord
	scanner := bufio.NewScanner(strings.NewReader(input))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var r rawRecord
		if json.Unmarshal([]byte(line), &r) == nil {
			r.RecordOrdinal = len(expected) + 1
			expected = append(expected, r)
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	actual, err := readRecords(path)
	if err != nil || !reflect.DeepEqual(actual, expected) {
		t.Fatalf("BYTE_READER_EQUIVALENCE FAIL: %v\ngot=%+v\nwant=%+v", err, actual, expected)
	}
	if len(actual) != 3 || actual[1].ID != "one" || actual[2].ID != "two" {
		t.Fatal("BYTE_READER_EQUIVALENCE FAIL: invalid fixture")
	}
	t.Log("BYTE_READER_EQUIVALENCE PASS")
}
