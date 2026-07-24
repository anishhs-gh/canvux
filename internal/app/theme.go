package app

import "github.com/anishhs-gh/canvux/internal/scene"

// Theme collects every color the UI uses. Dark, low-contrast chrome so the
// canvas content pops.
type Theme struct {
	CanvasBG   scene.Color
	GridDot    scene.Color
	Axis       scene.Color
	BarBG      scene.Color
	BarFG      scene.Color
	BarDim     scene.Color
	Accent     scene.Color
	AccentText scene.Color
	Selection  scene.Color
	Handle     scene.Color
	Marquee    scene.Color
	OverlayBG  scene.Color
	OverlayFG  scene.Color
	OverlayDim scene.Color
	OverlaySel scene.Color
	Danger     scene.Color
	OK         scene.Color
}

var DefaultTheme = Theme{
	CanvasBG:   scene.Color{R: 0x12, G: 0x14, B: 0x1a},
	GridDot:    scene.Color{R: 0x2a, G: 0x2e, B: 0x3a},
	Axis:       scene.Color{R: 0x3a, G: 0x40, B: 0x52},
	BarBG:      scene.Color{R: 0x1c, G: 0x1f, B: 0x2a},
	BarFG:      scene.Color{R: 0xd8, G: 0xdc, B: 0xe8},
	BarDim:     scene.Color{R: 0x6a, G: 0x71, B: 0x85},
	Accent:     scene.Color{R: 0x7a, G: 0xa2, B: 0xf7},
	AccentText: scene.Color{R: 0x10, G: 0x12, B: 0x18},
	Selection:  scene.Color{R: 0x7a, G: 0xa2, B: 0xf7},
	Handle:     scene.Color{R: 0xe0, G: 0xaf, B: 0x68},
	Marquee:    scene.Color{R: 0x56, G: 0x8a, B: 0xf7},
	OverlayBG:  scene.Color{R: 0x20, G: 0x24, B: 0x30},
	OverlayFG:  scene.Color{R: 0xd8, G: 0xdc, B: 0xe8},
	OverlayDim: scene.Color{R: 0x84, G: 0x8b, B: 0xa0},
	OverlaySel: scene.Color{R: 0x33, G: 0x3a, B: 0x4d},
	Danger:     scene.Color{R: 0xf7, G: 0x76, B: 0x8e},
	OK:         scene.Color{R: 0x9e, G: 0xce, B: 0x6a},
}

// LightTheme is whiteboard mode: paper-light canvas, soft chrome.
var LightTheme = Theme{
	CanvasBG:   scene.Color{R: 0xf4, G: 0xf4, B: 0xef},
	GridDot:    scene.Color{R: 0xd8, G: 0xd8, B: 0xd0},
	Axis:       scene.Color{R: 0xc0, G: 0xc0, B: 0xb8},
	BarBG:      scene.Color{R: 0xe4, G: 0xe4, B: 0xdd},
	BarFG:      scene.Color{R: 0x2a, G: 0x2e, B: 0x3a},
	BarDim:     scene.Color{R: 0x8a, G: 0x8f, B: 0x9d},
	Accent:     scene.Color{R: 0x2e, G: 0x5c, B: 0xd6},
	AccentText: scene.Color{R: 0xff, G: 0xff, B: 0xff},
	Selection:  scene.Color{R: 0x2e, G: 0x5c, B: 0xd6},
	Handle:     scene.Color{R: 0xb0, G: 0x72, B: 0x10},
	Marquee:    scene.Color{R: 0x4a, G: 0x74, B: 0xdd},
	OverlayBG:  scene.Color{R: 0xea, G: 0xea, B: 0xe3},
	OverlayFG:  scene.Color{R: 0x2a, G: 0x2e, B: 0x3a},
	OverlayDim: scene.Color{R: 0x7a, G: 0x7f, B: 0x8d},
	OverlaySel: scene.Color{R: 0xd4, G: 0xd9, B: 0xe8},
	Danger:     scene.Color{R: 0xc4, G: 0x33, B: 0x4d},
	OK:         scene.Color{R: 0x3d, G: 0x84, B: 0x2f},
}

// HighContrastTheme maximizes chrome/content contrast for low-vision use:
// near-black canvas, pure-white text, a bright-yellow accent, and a magenta
// selection that never collides with the (blue/green/cyan) content colors.
var HighContrastTheme = Theme{
	CanvasBG:   scene.Color{R: 0x00, G: 0x00, B: 0x00},
	GridDot:    scene.Color{R: 0x50, G: 0x50, B: 0x50},
	Axis:       scene.Color{R: 0x80, G: 0x80, B: 0x80},
	BarBG:      scene.Color{R: 0x00, G: 0x00, B: 0x00},
	BarFG:      scene.Color{R: 0xff, G: 0xff, B: 0xff},
	BarDim:     scene.Color{R: 0xb0, G: 0xb0, B: 0xb0},
	Accent:     scene.Color{R: 0xff, G: 0xe4, B: 0x00},
	AccentText: scene.Color{R: 0x00, G: 0x00, B: 0x00},
	Selection:  scene.Color{R: 0xff, G: 0x2b, B: 0xff},
	Handle:     scene.Color{R: 0xff, G: 0xe4, B: 0x00},
	Marquee:    scene.Color{R: 0xff, G: 0x2b, B: 0xff},
	OverlayBG:  scene.Color{R: 0x00, G: 0x00, B: 0x00},
	OverlayFG:  scene.Color{R: 0xff, G: 0xff, B: 0xff},
	OverlayDim: scene.Color{R: 0xc8, G: 0xc8, B: 0xc8},
	OverlaySel: scene.Color{R: 0x00, G: 0x33, B: 0x66},
	Danger:     scene.Color{R: 0xff, G: 0x40, B: 0x40},
	OK:         scene.Color{R: 0x40, G: 0xff, B: 0x40},
}

// NamedTheme pairs a theme with a display name for cycling/config.
type NamedTheme struct {
	Name  string
	Theme Theme
}

// Themes are the selectable UI themes, in cycle order.
var Themes = []NamedTheme{
	{"dark", DefaultTheme},
	{"light", LightTheme},
	{"high-contrast", HighContrastTheme},
}

// ThemeByName returns the named theme, or the default and false if unknown.
func ThemeByName(name string) (Theme, bool) {
	for _, t := range Themes {
		if t.Name == name {
			return t.Theme, true
		}
	}
	return DefaultTheme, false
}

// Palette is the *active* drawing color swatch set. It is swapped in place by
// SetActivePalette so the many call sites that read Palette[i] stay valid.
var Palette = defaultPalette

var defaultPalette = []scene.Color{
	{R: 0xc0, G: 0xca, B: 0xf5}, // foreground
	{R: 0x7a, G: 0xa2, B: 0xf7}, // blue
	{R: 0x7d, G: 0xcf, B: 0xff}, // cyan
	{R: 0x9e, G: 0xce, B: 0x6a}, // green
	{R: 0xe0, G: 0xaf, B: 0x68}, // amber
	{R: 0xff, G: 0x9e, B: 0x64}, // orange
	{R: 0xf7, G: 0x76, B: 0x8e}, // red
	{R: 0xbb, G: 0x9a, B: 0xf7}, // purple
	{R: 0xff, G: 0xff, B: 0xff}, // white
	{R: 0x56, G: 0x5f, B: 0x89}, // slate
}

// colorblindPalette is the Okabe–Ito colorblind-safe qualitative set, padded
// to 10 with black and gray. Distinct under deuteranopia/protanopia/tritanopia.
var colorblindPalette = []scene.Color{
	{R: 0xff, G: 0xff, B: 0xff}, // white (foreground)
	{R: 0x00, G: 0x72, B: 0xb2}, // blue
	{R: 0x56, G: 0xb4, B: 0xe9}, // sky blue
	{R: 0x00, G: 0x9e, B: 0x73}, // bluish green
	{R: 0xf0, G: 0xe4, B: 0x42}, // yellow
	{R: 0xe6, G: 0x9f, B: 0x00}, // orange
	{R: 0xd5, G: 0x5e, B: 0x00}, // vermillion
	{R: 0xcc, G: 0x79, B: 0xa7}, // reddish purple
	{R: 0x00, G: 0x00, B: 0x00}, // black
	{R: 0x99, G: 0x99, B: 0x99}, // gray
}

// contrastPalette is a maximal-luminance, maximally-separated set for the
// high-contrast theme.
var contrastPalette = []scene.Color{
	{R: 0xff, G: 0xff, B: 0xff}, // white
	{R: 0x00, G: 0x9c, B: 0xff}, // bright blue
	{R: 0x00, G: 0xf0, B: 0xf0}, // bright cyan
	{R: 0x00, G: 0xff, B: 0x00}, // bright green
	{R: 0xff, G: 0xff, B: 0x00}, // bright yellow
	{R: 0xff, G: 0x9c, B: 0x00}, // bright orange
	{R: 0xff, G: 0x30, B: 0x30}, // bright red
	{R: 0xff, G: 0x2b, B: 0xff}, // bright magenta
	{R: 0xd0, G: 0xd0, B: 0xd0}, // light gray
	{R: 0x90, G: 0x90, B: 0x90}, // mid gray
}

// NamedPalette pairs a palette with a display name.
type NamedPalette struct {
	Name   string
	Colors []scene.Color
}

// Palettes are the selectable drawing palettes, in cycle order.
var Palettes = []NamedPalette{
	{"default", defaultPalette},
	{"colorblind", colorblindPalette},
	{"high-contrast", contrastPalette},
}

// SetActivePalette swaps the active drawing palette in place by name. Unknown
// names are ignored. Returns whether the name was recognized.
func SetActivePalette(name string) bool {
	for _, p := range Palettes {
		if p.Name == name {
			Palette = p.Colors
			return true
		}
	}
	return false
}
