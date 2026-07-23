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

// Palette is the drawing color swatch set (Tokyo Night-ish, readable on dark).
var Palette = []scene.Color{
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
