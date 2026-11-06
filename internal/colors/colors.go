// Package colors provides utilities for resolving and computing colors
// used in the ssx interface styles.
package colors

import (
	"fmt"
	"image/color"
	"math"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/lucasb-eyer/go-colorful"
)

// Resolve normalizes a color string to a format lipgloss v2 understands:
//   - hex (#RGB / #RRGGBB) → as is
//   - CSS/ANSI name        → converted to #RRGGBB using the internal map
//   - ANSI number ("86")   → as is (lipgloss accepts it directly)
func Resolve(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "#") {
		return s
	}
	if rgb, ok := namedColorRGB[strings.ToLower(s)]; ok {
		return fmt.Sprintf("#%02X%02X%02X", rgb[0], rgb[1], rgb[2])
	}
	return s
}

// TintedForeground generates a text color that harmonizes with the background:
// it keeps the same hue with amplified saturation and inverted
// lightness (light on dark, dark on light).
// This replicates the effect of the default style (#312E81 → #C4B5FD).
// hexBg must be a hex color previously resolved with Resolve.
func TintedForeground(hexBg string) color.Color {
	col, err := colorful.Hex(hexBg)
	if err != nil {
		// Fallback for ANSI indexes or other non-hex formats.
		r, g, b, ok := parseRGB(hexBg)
		if ok && relativeLuminance(r, g, b) > 0.179 {
			return lipgloss.Color("#1A1A1A")
		}
		return lipgloss.Color("#FAFAFA")
	}

	h, s, l := col.Hsl()

	var fgL, fgS float64
	if l < 0.5 {
		// Dark background → light tinted text
		fgL = 0.85
		fgS = math.Min(1.0, s*1.8)
	} else {
		// Light background → dark tinted text
		fgL = 0.18
		fgS = math.Min(1.0, s*1.5)
	}

	return colorful.Hsl(h, fgS, fgL)
}

// Hex formats c as "#RRGGBB".
func Hex(c color.Color) string {
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("#%02X%02X%02X", r>>8, g>>8, b>>8)
}

// relativeLuminance computes the relative luminance per WCAG 2.1.
func relativeLuminance(r, g, b uint8) float64 {
	lin := func(c uint8) float64 {
		s := float64(c) / 255.0
		if s <= 0.04045 {
			return s / 12.92
		}
		return math.Pow((s+0.055)/1.055, 2.4)
	}
	return 0.2126*lin(r) + 0.7152*lin(g) + 0.0722*lin(b)
}

// parseRGB extracts the RGB components from a color string.
// Supports #RGB, #RRGGBB and common ANSI color names.
func parseRGB(s string) (r, g, b uint8, ok bool) {
	s = strings.TrimSpace(strings.ToLower(s))
	if strings.HasPrefix(s, "#") {
		return parseHex(s[1:])
	}
	if rgb, found := namedColorRGB[s]; found {
		return rgb[0], rgb[1], rgb[2], true
	}
	return 0, 0, 0, false
}

func parseHex(h string) (r, g, b uint8, ok bool) {
	switch len(h) {
	case 3:
		rv, e1 := strconv.ParseUint(string(h[0]), 16, 8)
		gv, e2 := strconv.ParseUint(string(h[1]), 16, 8)
		bv, e3 := strconv.ParseUint(string(h[2]), 16, 8)
		if e1 != nil || e2 != nil || e3 != nil {
			return 0, 0, 0, false
		}
		return uint8(rv * 17), uint8(gv * 17), uint8(bv * 17), true
	case 6:
		v, err := strconv.ParseUint(h, 16, 32)
		if err != nil {
			return 0, 0, 0, false
		}
		return uint8(v >> 16), uint8(v >> 8), uint8(v), true
	}
	return 0, 0, 0, false
}

// namedColorRGB maps basic ANSI/CSS color names to their approximate RGB values.
var namedColorRGB = map[string][3]uint8{
	"black":         {0, 0, 0},
	"red":           {170, 0, 0},
	"green":         {0, 170, 0},
	"yellow":        {170, 170, 0},
	"blue":          {0, 0, 170},
	"magenta":       {170, 0, 170},
	"cyan":          {0, 170, 170},
	"white":         {170, 170, 170},
	"brightblack":   {85, 85, 85},
	"brightred":     {255, 85, 85},
	"brightgreen":   {85, 255, 85},
	"brightyellow":  {255, 255, 85},
	"brightblue":    {85, 85, 255},
	"brightmagenta": {255, 85, 255},
	"brightcyan":    {85, 255, 255},
	"brightwhite":   {255, 255, 255},
	"lime":          {0, 255, 0},
	"aqua":          {0, 255, 255},
	"fuchsia":       {255, 0, 255},
	"silver":        {192, 192, 192},
	"gray":          {128, 128, 128},
	"grey":          {128, 128, 128},
	"maroon":        {128, 0, 0},
	"olive":         {128, 128, 0},
	"navy":          {0, 0, 128},
	"purple":        {128, 0, 128},
	"teal":          {0, 128, 128},
	"orange":        {255, 165, 0},
	"pink":          {255, 192, 203},
}
