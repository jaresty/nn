package note

import "testing"

const metaSample = `---
id: 20260101000000-0001
title: 'A note with a body and links'
type: observation
status: draft
tags:
    - alpha
    - beta
expires: 2026-02-01T00:00:00Z
---

Body paragraph one.

Body paragraph two.

## Links

- [[20260101000000-0002|Other]] [refines] {reviewed} — sharpens it
`

// property [19a]: ParseMeta yields the same metadata fields and links as Parse.
// property [19b]: ParseMeta discards the body (Body == "").
func TestParseMetaMatchesParseMetadataButDropsBody(t *testing.T) {
	full, err := Parse([]byte(metaSample))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	meta, err := ParseMeta([]byte(metaSample))
	if err != nil {
		t.Fatalf("ParseMeta: %v", err)
	}

	if meta.ID != full.ID || meta.Title != full.Title || meta.Type != full.Type || meta.Status != full.Status {
		t.Fatalf("meta core fields = %+v, want %+v", meta, full)
	}
	if len(meta.Tags) != len(full.Tags) {
		t.Fatalf("meta tags = %v, want %v", meta.Tags, full.Tags)
	}
	if (meta.Expires == nil) != (full.Expires == nil) {
		t.Fatalf("meta expires = %v, want %v", meta.Expires, full.Expires)
	}
	if len(meta.Links) != len(full.Links) {
		t.Fatalf("meta links = %v, want %v", meta.Links, full.Links)
	}
	if len(full.Links) > 0 && (meta.Links[0].TargetID != full.Links[0].TargetID || meta.Links[0].Type != full.Links[0].Type) {
		t.Fatalf("meta link[0] = %+v, want %+v", meta.Links[0], full.Links[0])
	}
	// property [19b]: body dropped.
	if meta.Body != "" {
		t.Fatalf("meta body = %q, want empty", meta.Body)
	}
	// full parse still retains the body (sanity: they differ only in body).
	if full.Body == "" {
		t.Fatalf("full parse unexpectedly dropped body")
	}
}
