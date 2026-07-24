package render

import (
	"os"
	"strings"
	"testing"

	"github.com/anishhs-gh/canvux/internal/scene"
)

func TestParseProfile(t *testing.T) {
	cases := map[string]Profile{
		"truecolor": TrueColor, "24bit": TrueColor,
		"256": ANSI256, "8bit": ANSI256,
		"16": ANSI16, "ansi": ANSI16,
		"off": Mono, "none": Mono, "mono": Mono,
	}
	for in, want := range cases {
		if got, ok := ParseProfile(in); !ok || got != want {
			t.Errorf("ParseProfile(%q) = %v, %v; want %v", in, got, ok, want)
		}
	}
	if _, ok := ParseProfile("banana"); ok {
		t.Error("ParseProfile(banana) should be invalid")
	}
}

func TestDetectProfileNoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	if DetectProfile() != Mono {
		t.Error("NO_COLOR should force Mono")
	}
}

func TestDetectProfileColorterm(t *testing.T) {
	// NO_COLOR is presence-based and cannot be unset via t.Setenv; skip if the
	// host environment has it set.
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		t.Skip("NO_COLOR set in environment")
	}
	t.Setenv("COLORTERM", "truecolor")
	if DetectProfile() != TrueColor {
		t.Error("COLORTERM=truecolor should detect TrueColor")
	}
	t.Setenv("COLORTERM", "")
	t.Setenv("TERM", "xterm-256color")
	if DetectProfile() != ANSI256 {
		t.Error("TERM=xterm-256color should detect ANSI256")
	}
	t.Setenv("TERM", "xterm")
	if DetectProfile() != ANSI16 {
		t.Error("TERM=xterm should detect ANSI16")
	}
	t.Setenv("TERM", "dumb")
	if DetectProfile() != Mono {
		t.Error("TERM=dumb should detect Mono")
	}
}

func TestANSIProfiles(t *testing.T) {
	mk := func(p Profile) string {
		g := NewCellGrid(2, 1, scene.Color{})
		g.Profile = p
		g.Set(0, 0, Cell{Ch: 'X', Fg: scene.Color{R: 0xff, G: 0x40, B: 0x10}, Bg: scene.Color{}})
		return g.ANSI()
	}
	if out := mk(TrueColor); !strings.Contains(out, "\x1b[38;2;255;64;16m") {
		t.Errorf("truecolor: missing 24-bit fg seq: %q", out)
	}
	if out := mk(ANSI256); !strings.Contains(out, "\x1b[38;5;") {
		t.Errorf("256: missing indexed fg seq: %q", out)
	}
	if out := mk(ANSI16); !strings.Contains(out, "\x1b[9") && !strings.Contains(out, "\x1b[3") {
		t.Errorf("16: missing basic fg seq: %q", out)
	}
	mono := mk(Mono)
	if strings.Contains(mono, "\x1b[") {
		t.Errorf("mono: should contain no escapes: %q", mono)
	}
	if !strings.Contains(mono, "X") {
		t.Errorf("mono: glyph lost: %q", mono)
	}
}

func TestTo256GrayscaleAndCube(t *testing.T) {
	// Pure black and white land on the cube corners; mid-gray on the ramp.
	if idx := to256(scene.Color{}); idx != 16 {
		t.Errorf("black -> %d, want 16", idx)
	}
	if idx := to256(scene.Color{R: 0xff, G: 0xff, B: 0xff}); idx != 231 {
		t.Errorf("white -> %d, want 231", idx)
	}
	if idx := to256(scene.Color{R: 0x80, G: 0x80, B: 0x80}); idx < 232 || idx > 255 {
		t.Errorf("mid gray -> %d, want grayscale ramp (232..255)", idx)
	}
	// A saturated color must map into the 16..231 color cube.
	if idx := to256(scene.Color{R: 0xff, G: 0x00, B: 0x00}); idx < 16 || idx > 231 {
		t.Errorf("red -> %d, want color cube", idx)
	}
}
