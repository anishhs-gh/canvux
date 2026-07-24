package clipboard

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestOSC52Encoding(t *testing.T) {
	text := "<svg>hello & <world></svg>"
	seq := OSC52(text)
	if !strings.HasPrefix(seq, "\x1b]52;c;") || !strings.HasSuffix(seq, "\x07") {
		t.Fatalf("OSC52 framing wrong: %q", seq)
	}
	payload := strings.TrimSuffix(strings.TrimPrefix(seq, "\x1b]52;c;"), "\x07")
	dec, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		t.Fatalf("payload not valid base64: %v", err)
	}
	if string(dec) != text {
		t.Errorf("round-trip mismatch: %q != %q", dec, text)
	}
}

func TestOSC52Empty(t *testing.T) {
	seq := OSC52("")
	if seq != "\x1b]52;c;\x07" {
		t.Errorf("empty OSC52 = %q", seq)
	}
}
