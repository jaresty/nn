package cmd

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sort"
)

// writeCaptureJSON preserves json.Marshal(c)'s bytes without constructing a
// second, base64-expanded copy of every raw source in memory at once.
func writeCaptureJSON(w io.Writer, c *transcriptCapture) error {
	var err error
	text := func(s string) {
		if err == nil {
			_, err = io.WriteString(w, s)
		}
	}
	value := func(v any) {
		if err != nil {
			return
		}
		var b []byte
		b, err = json.Marshal(v)
		if err == nil {
			_, err = w.Write(b)
		}
	}
	text(`{"input_path":`)
	value(c.InputPath)
	text(`,"path":`)
	value(c.Path)
	text(`,"sources":`)
	if c.Sources == nil {
		text(`null`)
	} else {
		text(`{`)
		keys := make([]string, 0, len(c.Sources))
		for key := range c.Sources {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for i, key := range keys {
			if i > 0 {
				text(`,`)
			}
			value(key)
			text(`:{"prefix_bytes":`)
			source := c.Sources[key]
			value(source.PrefixBytes)
			text(`,"sha256":`)
			value(source.Digest)
			text(`,"bytes":`)
			if source.Raw == nil {
				text(`null`)
			} else {
				text(`"`)
				if err == nil {
					encoder := base64.NewEncoder(base64.StdEncoding, w)
					_, err = io.Copy(encoder, bytes.NewReader(source.Raw))
					if err == nil {
						err = encoder.Close()
					}
				}
				text(`"`)
			}
			text(`}`)
			if err != nil {
				return err
			}
		}
		text(`}`)
	}
	text(`,"authenticated_paths":`)
	value(c.AuthPaths)
	if len(c.AgentSelection) > 0 {
		text(`,"agent_selection":`)
		value(c.AgentSelection)
	}
	text(`}`)
	return err
}

func saveTranscriptCapture(c *transcriptCapture) error {
	dir, e := captureCacheDir()
	if e != nil {
		return e
	}
	f, e := os.CreateTemp(dir, ".capture-")
	if e != nil {
		return e
	}
	defer os.Remove(f.Name())
	h := sha256.New()
	buffered := bufio.NewWriterSize(f, 256*1024)
	if e = writeCaptureJSON(io.MultiWriter(buffered, h), c); e != nil {
		f.Close()
		return e
	}
	if e = buffered.Flush(); e != nil {
		f.Close()
		return e
	}
	if e = f.Close(); e != nil {
		return e
	}
	id := hex.EncodeToString(h.Sum(nil))
	if e = os.Rename(f.Name(), filepath.Join(dir, id+".json")); e != nil {
		return e
	}
	c.ID = id
	return nil
}
