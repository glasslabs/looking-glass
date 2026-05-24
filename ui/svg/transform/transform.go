// Package transform parses SVG transform attribute strings into f32.Affine2D values.
package transform

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"gioui.org/f32"
)

// Parse parses an SVG transform attribute string and returns the composed
// f32.Affine2D. Multiple transforms are applied right-to-left per the SVG
// spec (the leftmost transform in the string is applied last). Returns the
// identity for an empty string and an error for any unrecognised function.
func Parse(s string) (f32.Affine2D, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return f32.AffineId(), nil
	}

	result := f32.AffineId()

	// Collect each function token and process left-to-right, composing with Mul.
	// SVG spec §7.6: "If a list of transforms is provided, then the net effect
	// is as if each transform had been specified separately in the order provided."
	// This means the first listed is the outermost (applied last to a point), so
	// Mul(result, next) accumulates in the conventional matrix-multiplication order.
	for {
		s = strings.TrimSpace(s)
		if s == "" {
			break
		}
		openIdx := strings.IndexByte(s, '(')
		if openIdx < 0 {
			return f32.AffineId(), fmt.Errorf("missing '(' in transform %q", s)
		}
		closeIdx := strings.IndexByte(s, ')')
		if closeIdx < 0 {
			return f32.AffineId(), fmt.Errorf("missing ')' in transform %q", s)
		}

		fn := strings.ToLower(strings.TrimSpace(s[:openIdx]))
		argsStr := s[openIdx+1 : closeIdx]
		s = strings.TrimSpace(s[closeIdx+1:])
		// Commas between function calls are optional.
		s = strings.TrimLeft(s, ", \t\n\r")

		xform, err := parseFunc(fn, argsStr)
		if err != nil {
			return f32.AffineId(), err
		}
		result = result.Mul(xform)
	}

	return result, nil
}

// parseFunc parses a single transform function with its argument string.
//
//nolint:cyclop // Parsing is by its nature a large switch.
func parseFunc(fn, argsStr string) (f32.Affine2D, error) {
	args, err := parseArgs(argsStr)
	if err != nil {
		return f32.AffineId(), fmt.Errorf("transform %s: %w", fn, err)
	}

	switch fn {
	case "translate":
		switch len(args) {
		case 1:
			return f32.AffineId().Offset(f32.Pt(args[0], 0)), nil
		case 2:
			return f32.AffineId().Offset(f32.Pt(args[0], args[1])), nil
		default:
			return f32.AffineId(), fmt.Errorf("translate requires 1 or 2 arguments, got %d", len(args))
		}
	case "scale":
		switch len(args) {
		case 1:
			return f32.AffineId().Scale(f32.Pt(0, 0), f32.Pt(args[0], args[0])), nil
		case 2:
			return f32.AffineId().Scale(f32.Pt(0, 0), f32.Pt(args[0], args[1])), nil
		default:
			return f32.AffineId(), fmt.Errorf("scale requires 1 or 2 arguments, got %d", len(args))
		}
	case "rotate":
		switch len(args) {
		case 1:
			rad := float64(args[0]) * math.Pi / 180
			cos, sin := float32(math.Cos(rad)), float32(math.Sin(rad))
			return f32.NewAffine2D(cos, -sin, 0, sin, cos, 0), nil
		case 3:
			cx, cy := args[1], args[2]
			rad := float64(args[0]) * math.Pi / 180
			cos, sin := float32(math.Cos(rad)), float32(math.Sin(rad))
			// SVG rotate(a,cx,cy) ≡ translate(cx,cy) rotate(a) translate(-cx,-cy):
			// x' = cos*(x-cx) - sin*(y-cy) + cx
			// y' = sin*(x-cx) + cos*(y-cy) + cy
			return f32.NewAffine2D(
				cos, -sin, cx*(1-cos)+cy*sin,
				sin, cos, cy*(1-cos)-cx*sin,
			), nil
		default:
			return f32.AffineId(), fmt.Errorf("rotate requires 1 or 3 arguments, got %d", len(args))
		}
	case "skewx":
		if len(args) != 1 {
			return f32.AffineId(), fmt.Errorf("skewX requires 1 argument, got %d", len(args))
		}
		rad := float32(args[0] * math.Pi / 180)
		// [1  tan 0]
		// [0  1   0]
		return f32.NewAffine2D(1, float32(math.Tan(float64(rad))), 0, 0, 1, 0), nil
	case "skewy":
		if len(args) != 1 {
			return f32.AffineId(), fmt.Errorf("skewY requires 1 argument, got %d", len(args))
		}
		rad := float32(args[0] * math.Pi / 180)
		// [1    0  0]
		// [tan  1  0]
		return f32.NewAffine2D(1, 0, 0, float32(math.Tan(float64(rad))), 1, 0), nil
	case "matrix":
		if len(args) != 6 {
			return f32.AffineId(), fmt.Errorf("matrix requires 6 arguments, got %d", len(args))
		}
		// SVG matrix(a,b,c,d,e,f) maps to:
		// x' = a*x + c*y + e
		// y' = b*x + d*y + f
		// f32.NewAffine2D(sx, hx, ox, hy, sy, oy):
		// x' = sx*x + hx*y + ox
		// y' = hy*x + sy*y + oy
		// Therefore: sx=a, hx=c, ox=e, hy=b, sy=d, oy=f
		return f32.NewAffine2D(args[0], args[2], args[4], args[1], args[3], args[5]), nil
	default:
		return f32.AffineId(), fmt.Errorf("unknown transform function %q", fn)
	}
}

// parseArgs splits a transform argument string on whitespace and commas.
func parseArgs(s string) ([]float32, error) {
	var args []float32
	for _, tok := range strings.FieldsFunc(s, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t' || r == '\n' || r == '\r'
	}) {
		v, err := strconv.ParseFloat(tok, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid number %q: %w", tok, err)
		}
		args = append(args, float32(v))
	}
	return args, nil
}
