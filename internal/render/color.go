package render

import (
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/anishhs-gh/canvux/internal/scene"
)

// Profile is a terminal's color capability. ANSI output degrades to match it
// so Canvux renders correctly outside truecolor terminals.
type Profile int

const (
	// TrueColor emits 24-bit \x1b[38;2;R;G;Bm sequences.
	TrueColor Profile = iota
	// ANSI256 maps to the xterm 256-color cube (\x1b[38;5;Nm).
	ANSI256
	// ANSI16 maps to the 16 basic colors (\x1b[3Nm / \x1b[9Nm).
	ANSI16
	// Mono emits no color; shapes still read via the block/braille glyphs.
	Mono
)

func (p Profile) String() string {
	switch p {
	case ANSI256:
		return "256"
	case ANSI16:
		return "16"
	case Mono:
		return "mono"
	default:
		return "truecolor"
	}
}

// ParseProfile resolves a --color flag value; "auto" defers to DetectProfile.
func ParseProfile(s string) (Profile, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "auto":
		return DetectProfile(), true
	case "truecolor", "24bit", "24":
		return TrueColor, true
	case "256", "8bit":
		return ANSI256, true
	case "16", "ansi", "4bit":
		return ANSI16, true
	case "off", "none", "mono", "no":
		return Mono, true
	default:
		return TrueColor, false
	}
}

// DetectProfile inspects the environment: NO_COLOR forces mono; COLORTERM
// truecolor/24bit means truecolor; a 256-capable TERM means ANSI256; a bare
// TERM means 16; no TERM (piped/dumb) means mono.
func DetectProfile() Profile {
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		return Mono
	}
	ct := strings.ToLower(os.Getenv("COLORTERM"))
	if strings.Contains(ct, "truecolor") || strings.Contains(ct, "24bit") {
		return TrueColor
	}
	term := strings.ToLower(os.Getenv("TERM"))
	switch {
	case term == "" || term == "dumb":
		return Mono
	case strings.Contains(term, "256color"):
		return ANSI256
	case strings.Contains(term, "truecolor") || strings.Contains(term, "direct"):
		return TrueColor
	default:
		return ANSI16
	}
}

// writeFg appends the foreground color escape for c under profile p.
func (p Profile) writeFg(b *strings.Builder, c scene.Color) {
	switch p {
	case Mono:
		return
	case ANSI256:
		fmt.Fprintf(b, "\x1b[38;5;%dm", to256(c))
	case ANSI16:
		fmt.Fprintf(b, "\x1b[%dm", to16(c, false))
	default:
		fmt.Fprintf(b, "\x1b[38;2;%d;%d;%dm", c.R, c.G, c.B)
	}
}

// writeBg appends the background color escape for c under profile p.
func (p Profile) writeBg(b *strings.Builder, c scene.Color) {
	switch p {
	case Mono:
		return
	case ANSI256:
		fmt.Fprintf(b, "\x1b[48;5;%dm", to256(c))
	case ANSI16:
		fmt.Fprintf(b, "\x1b[%dm", to16(c, true))
	default:
		fmt.Fprintf(b, "\x1b[48;2;%d;%d;%dm", c.R, c.G, c.B)
	}
}

// to256 maps an RGB color to the nearest xterm-256 index (6x6x6 cube plus the
// grayscale ramp, whichever is closer).
func to256(c scene.Color) int {
	// Grayscale ramp (232..255) when the channels are close.
	if absU8(c.R, c.G) < 8 && absU8(c.G, c.B) < 8 && absU8(c.R, c.B) < 8 {
		gray := (int(c.R) + int(c.G) + int(c.B)) / 3
		if gray < 8 {
			return 16
		}
		if gray > 248 {
			return 231
		}
		return 232 + (gray-8)*24/247
	}
	q := func(v uint8) int { // 0,95,135,175,215,255 cube levels
		switch {
		case v < 48:
			return 0
		case v < 115:
			return 1
		default:
			return (int(v) - 35) / 40
		}
	}
	return 16 + 36*q(c.R) + 6*q(c.G) + q(c.B)
}

// to16 maps an RGB color to a basic 16-color SGR code. bg selects 4x/10x range.
func to16(c scene.Color, bg bool) int {
	// Choose the nearest of the 16 standard colors by simple distance.
	best, bestD := 0, math.MaxFloat64
	for i, pal := range ansi16Palette {
		dr := float64(c.R) - float64(pal.R)
		dg := float64(c.G) - float64(pal.G)
		db := float64(c.B) - float64(pal.B)
		if d := dr*dr + dg*dg + db*db; d < bestD {
			best, bestD = i, d
		}
	}
	// 0..7 -> 30..37 / 40..47; 8..15 -> 90..97 / 100..107.
	base := 30
	if bg {
		base = 40
	}
	if best >= 8 {
		base += 60
		best -= 8
	}
	return base + best
}

func absU8(a, b uint8) int {
	if a > b {
		return int(a - b)
	}
	return int(b - a)
}

// ansi16Palette is the conventional xterm 16-color palette (0..15).
var ansi16Palette = [16]scene.Color{
	{R: 0x00, G: 0x00, B: 0x00}, {R: 0xcd, G: 0x00, B: 0x00}, {R: 0x00, G: 0xcd, B: 0x00}, {R: 0xcd, G: 0xcd, B: 0x00},
	{R: 0x00, G: 0x00, B: 0xee}, {R: 0xcd, G: 0x00, B: 0xcd}, {R: 0x00, G: 0xcd, B: 0xcd}, {R: 0xe5, G: 0xe5, B: 0xe5},
	{R: 0x7f, G: 0x7f, B: 0x7f}, {R: 0xff, G: 0x00, B: 0x00}, {R: 0x00, G: 0xff, B: 0x00}, {R: 0xff, G: 0xff, B: 0x00},
	{R: 0x5c, G: 0x5c, B: 0xff}, {R: 0xff, G: 0x00, B: 0xff}, {R: 0x00, G: 0xff, B: 0xff}, {R: 0xff, G: 0xff, B: 0xff},
}
