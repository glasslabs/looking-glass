// Package style parses and resolves SVG presentation attributes and inline CSS.
package style

import (
	"image/color"
	"strconv"
	"strings"
)

const fillNone = "none"

// Style holds the resolved presentation attributes for a single SVG element.
// Boolean None fields take precedence over the colour: when FillNone is true
// the element is not filled regardless of Fill's value.
type Style struct {
	Fill        color.NRGBA
	FillNone    bool
	FillOpacity float32
	// FillRule is "nonzero" or "evenodd"; empty string means nonzero (the SVG default).
	FillRule string

	Stroke        color.NRGBA
	StrokeNone    bool
	StrokeWidth   float32
	StrokeOpacity float32

	Opacity float32

	FontSize   float32
	FontWeight string
	TextAnchor string

	Display string
}

// isSet tracks which fields were explicitly provided, enabling correct
// inheritance (an unset field inherits; an explicitly-set field does not).
type isSet struct {
	fill          bool
	fillOpacity   bool
	fillRule      bool
	stroke        bool
	strokeWidth   bool
	strokeOpacity bool
	opacity       bool
	fontSize      bool
	fontWeight    bool
	textAnchor    bool
	display       bool
}

// entry pairs a resolved Style with its isSet mask.
type entry struct {
	s Style
	m isSet
}

// Default returns the SVG initial values: black fill, no stroke, full opacity.
func Default() Style {
	return Style{
		FillOpacity:   1,
		StrokeNone:    true,
		StrokeWidth:   1,
		StrokeOpacity: 1,
		Opacity:       1,
		FontSize:      16,
		FontWeight:    "normal",
		TextAnchor:    "start",
		Display:       "inline",
	}
}

// Parse builds a Style from a map of XML attribute names to values.
// Individual presentation attributes (fill, stroke, …) are applied first;
// then the style="" attribute is parsed and its declarations take priority,
// matching the SVG specificity rules.
func Parse(attrs map[string]string) Style {
	e := entry{}
	applyPresentation(&e, attrs)
	if css, ok := attrs["style"]; ok {
		applyCSS(&e, css)
	}
	return e.s
}

// Inherit resolves child against parent, returning a Style where every
// unset inheritable field in child takes its value from parent.
// Non-inheritable fields (display) always come from child when set,
// otherwise from the SVG default.
func Inherit(parent, child Style, childAttrs map[string]string) Style {
	ce := entry{}
	applyPresentation(&ce, childAttrs)
	if css, ok := childAttrs["style"]; ok {
		applyCSS(&ce, css)
	}

	result := child

	if !ce.m.fill {
		result.Fill = parent.Fill
		result.FillNone = parent.FillNone
	}
	if !ce.m.fillOpacity {
		result.FillOpacity = parent.FillOpacity
	}
	if !ce.m.fillRule {
		result.FillRule = parent.FillRule
	}
	if !ce.m.stroke {
		result.Stroke = parent.Stroke
		result.StrokeNone = parent.StrokeNone
	}
	if !ce.m.strokeWidth {
		result.StrokeWidth = parent.StrokeWidth
	}
	if !ce.m.strokeOpacity {
		result.StrokeOpacity = parent.StrokeOpacity
	}
	if !ce.m.opacity {
		result.Opacity = parent.Opacity
	}
	if !ce.m.fontSize {
		result.FontSize = parent.FontSize
	}
	if !ce.m.fontWeight {
		result.FontWeight = parent.FontWeight
	}
	if !ce.m.textAnchor {
		result.TextAnchor = parent.TextAnchor
	}
	// display is not inherited; if unset, keep the SVG default.
	if !ce.m.display {
		result.Display = "inline"
	}

	return result
}

// ParseColor parses an SVG colour value into a color.NRGBA.
// Recognised forms: #rgb, #rrggbb, #rgba, #rrggbbaa, rgb(r,g,b),
// rgba(r,g,b,a), and the 16 CSS Level 1 named colours.
// Returns (zero, false) for "none" or any unrecognised value.
func ParseColor(s string) (color.NRGBA, bool) {
	s = strings.TrimSpace(s)
	if s == "" || s == fillNone {
		return color.NRGBA{}, false
	}

	if strings.HasPrefix(s, "#") {
		return parseHex(s[1:])
	}
	if strings.HasPrefix(s, "rgba(") {
		return parseRGBA(s[5 : len(s)-1])
	}
	if strings.HasPrefix(s, "rgb(") {
		return parseRGB(s[4 : len(s)-1])
	}

	if c, ok := namedColors[strings.ToLower(s)]; ok {
		return c, true
	}
	return color.NRGBA{}, false
}

// applyPresentation reads individual SVG presentation attributes into e.
func applyPresentation(e *entry, attrs map[string]string) {
	if v, ok := attrs["fill"]; ok {
		applyFill(e, v)
	}
	if v, ok := attrs["fill-opacity"]; ok {
		if f, err := strconv.ParseFloat(strings.TrimSpace(v), 32); err == nil {
			e.s.FillOpacity = float32(f)
			e.m.fillOpacity = true
		}
	}
	if v, ok := attrs["fill-rule"]; ok {
		v = strings.TrimSpace(v)
		if v == "evenodd" || v == "nonzero" {
			e.s.FillRule = v
			e.m.fillRule = true
		}
	}
	if v, ok := attrs["stroke"]; ok {
		applyStroke(e, v)
	}
	if v, ok := attrs["stroke-width"]; ok {
		if f, err := strconv.ParseFloat(strings.TrimSpace(v), 32); err == nil {
			e.s.StrokeWidth = float32(f)
			e.m.strokeWidth = true
		}
	}
	if v, ok := attrs["stroke-opacity"]; ok {
		if f, err := strconv.ParseFloat(strings.TrimSpace(v), 32); err == nil {
			e.s.StrokeOpacity = float32(f)
			e.m.strokeOpacity = true
		}
	}
	if v, ok := attrs["opacity"]; ok {
		if f, err := strconv.ParseFloat(strings.TrimSpace(v), 32); err == nil {
			e.s.Opacity = float32(f)
			e.m.opacity = true
		}
	}
	if v, ok := attrs["font-size"]; ok {
		if f, err := strconv.ParseFloat(strings.TrimSpace(strings.TrimSuffix(v, "px")), 32); err == nil {
			e.s.FontSize = float32(f)
			e.m.fontSize = true
		}
	}
	if v, ok := attrs["font-weight"]; ok {
		e.s.FontWeight = strings.TrimSpace(v)
		e.m.fontWeight = true
	}
	if v, ok := attrs["text-anchor"]; ok {
		e.s.TextAnchor = strings.TrimSpace(v)
		e.m.textAnchor = true
	}
	if v, ok := attrs["display"]; ok {
		e.s.Display = strings.TrimSpace(v)
		e.m.display = true
	}
}

// applyCSS parses the value of a style="" attribute and overlays it onto e,
// giving CSS declarations priority over presentation attributes.
func applyCSS(e *entry, css string) {
	for decl := range strings.SplitSeq(css, ";") {
		decl = strings.TrimSpace(decl)
		if decl == "" {
			continue
		}
		before, after, ok := strings.Cut(decl, ":")
		if !ok {
			continue
		}
		prop := strings.TrimSpace(before)
		val := strings.TrimSpace(after)
		// Re-use presentation attribute logic by constructing a one-entry map.
		applyPresentation(e, map[string]string{prop: val})
	}
}

func applyFill(e *entry, v string) {
	v = strings.TrimSpace(v)
	e.m.fill = true
	if v == fillNone {
		e.s.FillNone = true
		return
	}
	if c, ok := ParseColor(v); ok {
		e.s.Fill = c
		e.s.FillNone = false
	}
}

func applyStroke(e *entry, v string) {
	v = strings.TrimSpace(v)
	e.m.stroke = true
	if v == fillNone {
		e.s.StrokeNone = true
		return
	}
	if c, ok := ParseColor(v); ok {
		e.s.Stroke = c
		e.s.StrokeNone = false
	}
}

func parseHex(s string) (color.NRGBA, bool) {
	switch len(s) {
	case 3: // #rgb → #rrggbb
		r := hexNibble(s[0])
		g := hexNibble(s[1])
		b := hexNibble(s[2])
		return color.NRGBA{R: r | r<<4, G: g | g<<4, B: b | b<<4, A: 0xff}, true
	case 4: // #rgba → #rrggbbaa
		r := hexNibble(s[0])
		g := hexNibble(s[1])
		b := hexNibble(s[2])
		a := hexNibble(s[3])
		return color.NRGBA{R: r | r<<4, G: g | g<<4, B: b | b<<4, A: a | a<<4}, true
	case 6: // #rrggbb
		r, ok1 := hexByte(s[0], s[1])
		g, ok2 := hexByte(s[2], s[3])
		b, ok3 := hexByte(s[4], s[5])
		if !ok1 || !ok2 || !ok3 {
			return color.NRGBA{}, false
		}
		return color.NRGBA{R: r, G: g, B: b, A: 0xff}, true
	case 8: // #rrggbbaa
		r, ok1 := hexByte(s[0], s[1])
		g, ok2 := hexByte(s[2], s[3])
		b, ok3 := hexByte(s[4], s[5])
		a, ok4 := hexByte(s[6], s[7])
		if !ok1 || !ok2 || !ok3 || !ok4 {
			return color.NRGBA{}, false
		}
		return color.NRGBA{R: r, G: g, B: b, A: a}, true
	}
	return color.NRGBA{}, false
}

func parseRGB(s string) (color.NRGBA, bool) {
	parts := strings.SplitN(s, ",", 3)
	if len(parts) != 3 {
		return color.NRGBA{}, false
	}
	r, ok1 := parseChannel(parts[0])
	g, ok2 := parseChannel(parts[1])
	b, ok3 := parseChannel(parts[2])
	if !ok1 || !ok2 || !ok3 {
		return color.NRGBA{}, false
	}
	return color.NRGBA{R: r, G: g, B: b, A: 0xff}, true
}

func parseRGBA(s string) (color.NRGBA, bool) {
	parts := strings.SplitN(s, ",", 4)
	if len(parts) != 4 {
		return color.NRGBA{}, false
	}
	r, ok1 := parseChannel(parts[0])
	g, ok2 := parseChannel(parts[1])
	b, ok3 := parseChannel(parts[2])
	if !ok1 || !ok2 || !ok3 {
		return color.NRGBA{}, false
	}
	// Alpha in rgba() is 0–1 float.
	af, err := strconv.ParseFloat(strings.TrimSpace(parts[3]), 64)
	if err != nil {
		return color.NRGBA{}, false
	}
	a := uint8(af * 255)
	return color.NRGBA{R: r, G: g, B: b, A: a}, true
}

func parseChannel(s string) (uint8, bool) {
	v, err := strconv.ParseUint(strings.TrimSpace(s), 10, 8)
	if err != nil {
		return 0, false
	}
	return uint8(v), true
}

func hexNibble(c byte) byte {
	switch {
	case c >= '0' && c <= '9':
		return c - '0'
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10
	}
	return 0
}

func hexByte(hi, lo byte) (uint8, bool) {
	h := hexNibble(hi)
	l := hexNibble(lo)
	// Invalid if the nibble mapped to 0 but the input wasn't '0'.
	if h == 0 && hi != '0' && hi != 'A' && hi != 'a' {
		if (hi < '0' || hi > '9') && (hi < 'a' || hi > 'f') && (hi < 'A' || hi > 'F') {
			return 0, false
		}
	}
	return h<<4 | l, true
}

// namedColors contains the 16 CSS Level 1 named colours plus common aliases.
var namedColors = map[string]color.NRGBA{
	"black":   {R: 0x00, G: 0x00, B: 0x00, A: 0xff},
	"white":   {R: 0xff, G: 0xff, B: 0xff, A: 0xff},
	"red":     {R: 0xff, G: 0x00, B: 0x00, A: 0xff},
	"lime":    {R: 0x00, G: 0xff, B: 0x00, A: 0xff},
	"green":   {R: 0x00, G: 0x80, B: 0x00, A: 0xff},
	"blue":    {R: 0x00, G: 0x00, B: 0xff, A: 0xff},
	"yellow":  {R: 0xff, G: 0xff, B: 0x00, A: 0xff},
	"cyan":    {R: 0x00, G: 0xff, B: 0xff, A: 0xff},
	"aqua":    {R: 0x00, G: 0xff, B: 0xff, A: 0xff},
	"magenta": {R: 0xff, G: 0x00, B: 0xff, A: 0xff},
	"fuchsia": {R: 0xff, G: 0x00, B: 0xff, A: 0xff},
	"silver":  {R: 0xc0, G: 0xc0, B: 0xc0, A: 0xff},
	"gray":    {R: 0x80, G: 0x80, B: 0x80, A: 0xff},
	"grey":    {R: 0x80, G: 0x80, B: 0x80, A: 0xff},
	"maroon":  {R: 0x80, G: 0x00, B: 0x00, A: 0xff},
	"olive":   {R: 0x80, G: 0x80, B: 0x00, A: 0xff},
	"navy":    {R: 0x00, G: 0x00, B: 0x80, A: 0xff},
	"teal":    {R: 0x00, G: 0x80, B: 0x80, A: 0xff},
	"purple":  {R: 0x80, G: 0x00, B: 0x80, A: 0xff},
	"orange":  {R: 0xff, G: 0xa5, B: 0x00, A: 0xff},
}
