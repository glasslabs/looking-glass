package svg

import (
	"encoding/xml"
	"image"
	"math"
	"strconv"
	"strings"
)

// FitSize returns the largest image.Point that fits vbW×vbH inside max
// while preserving the aspect ratio of vbW×vbH.
func FitSize(vbW, vbH float64, maxSize image.Point) image.Point {
	if vbW <= 0 || vbH <= 0 || maxSize.X <= 0 || maxSize.Y <= 0 {
		return maxSize
	}
	scale := math.Min(float64(maxSize.X)/vbW, float64(maxSize.Y)/vbH)
	return image.Pt(
		int(math.Round(vbW*scale)),
		int(math.Round(vbH*scale)),
	)
}

// Dimensions returns the intrinsic width and height from the root <svg> element's
// width and height attributes. Returns 0, 0 if the attributes are absent,
// unrecognized, or relative (e.g. percentages).
func Dimensions(content string) (width, height float64) {
	dec := xml.NewDecoder(strings.NewReader(content))
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		el, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		if el.Name.Local != "svg" {
			break // first element must be <svg>
		}
		return parseLength(xmlAttr(el.Attr, "width")), parseLength(xmlAttr(el.Attr, "height"))
	}
	return 0, 0
}

// parseLength parses an SVG length value (e.g. "24", "24px", "18pt") and
// returns the numeric value in SVG user units. Unsupported units return 0.
func parseLength(s string) float64 {
	s = strings.TrimSpace(s)
	switch {
	case strings.HasSuffix(s, "px"):
		s = s[:len(s)-2]
	case strings.HasSuffix(s, "pt"):
		// 1pt ≈ 1.333 user units (96dpi / 72ppi).
		s = s[:len(s)-2]
		v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
		if err != nil {
			return 0
		}
		return v * 4 / 3
	case strings.HasSuffix(s, "em"), strings.HasSuffix(s, "rem"),
		strings.HasSuffix(s, "%"):
		// Relative units require context; skip.
		return 0
	}
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0
	}
	return v
}

// xmlAttr returns the value of the XML attribute with the given local name,
// or an empty string if not found.
func xmlAttr(attrs []xml.Attr, name string) string {
	for _, a := range attrs {
		if a.Name.Local == name {
			return a.Value
		}
	}
	return ""
}
