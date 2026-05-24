// Package path parses SVG path d-strings and emits Gio clip operations.
package path

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"strconv"
	"strings"

	"gioui.org/f32"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
)

// Segment is a single normalised command in an SVG path d-string.
// All coordinates are absolute; relative commands are resolved during parsing.
type Segment struct {
	// Cmd is the upper-case SVG path command letter (M, L, C, Q, Z, …).
	Cmd byte
	// Args holds the numeric arguments for the command.
	Args []float64
}

// pathState carries mutable pen state across commands to resolve smooth
// bezier continuations (S/s uses prevCtrl2, T/t uses prevCtrl1).
type pathState struct {
	prevCmd    byte
	prevCtrl2X float64
	prevCtrl2Y float64
	prevCtrl1X float64
	prevCtrl1Y float64
}

// Parse parses an SVG path d-string into a slice of Segments.
// Relative commands (lowercase) are converted to absolute coordinates.
// Smooth bezier commands are resolved against the preceding control point.
// Returns an error if the d-string contains unknown commands.
func Parse(d string) ([]Segment, error) {
	var segments []Segment
	var curX, curY float64
	var st pathState

	tokens := tokenise(d)
	i := 0
	for i < len(tokens) {
		if len(tokens[i]) == 0 {
			i++
			continue
		}
		cmd := tokens[i][0]
		if (cmd < 'A' || cmd > 'Z') && (cmd < 'a' || cmd > 'z') {
			return nil, fmt.Errorf("unexpected token %q", tokens[i])
		}
		if !isKnownCmd(cmd) {
			return nil, fmt.Errorf("unknown command %q", cmd)
		}
		i++

		var rawArgs []float64
		for i < len(tokens) {
			c := tokens[i][0]
			if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') {
				break
			}
			v, err := strconv.ParseFloat(tokens[i], 64)
			if err != nil {
				return nil, fmt.Errorf("invalid number %q: %w", tokens[i], err)
			}
			rawArgs = append(rawArgs, v)
			i++
		}

		segs, nx, ny := expandCmd(cmd, rawArgs, curX, curY, &st)
		segments = append(segments, segs...)
		curX, curY = nx, ny

		absCmd := cmd
		if absCmd >= 'a' && absCmd <= 'z' {
			absCmd -= 32
		}
		st.prevCmd = absCmd
	}
	return segments, nil
}

// BuildClipPath appends the given segments to p, transforming all coordinates
// through xform. The zero value of f32.Affine2D is the identity transform.
func BuildClipPath(p *clip.Path, segments []Segment, xform f32.Affine2D) {
	pt := func(x, y float64) f32.Point {
		return xform.Transform(f32.Pt(float32(x), float32(y)))
	}
	for _, seg := range segments {
		switch seg.Cmd {
		case 'M':
			p.MoveTo(pt(seg.Args[0], seg.Args[1]))
		case 'L':
			p.LineTo(pt(seg.Args[0], seg.Args[1]))
		case 'C':
			p.CubeTo(
				pt(seg.Args[0], seg.Args[1]),
				pt(seg.Args[2], seg.Args[3]),
				pt(seg.Args[4], seg.Args[5]),
			)
		case 'Q':
			p.QuadTo(
				pt(seg.Args[0], seg.Args[1]),
				pt(seg.Args[2], seg.Args[3]),
			)
		case 'Z':
			p.Close()
		}
	}
}

// StrokePath strokes the given segments with the specified width and colour,
// transforming all coordinates through xform before drawing.
func StrokePath(ops *op.Ops, segments []Segment, xform f32.Affine2D, width float32, col color.NRGBA) {
	var p clip.Path
	p.Begin(ops)
	BuildClipPath(&p, segments, xform)
	spec := clip.Stroke{Path: p.End(), Width: width}.Op()
	paint.FillShape(ops, col, spec)
}

// AddRectPath traces a rectangle outline into p.
func AddRectPath(p *clip.Path, r image.Rectangle) {
	p.MoveTo(f32.Pt(float32(r.Min.X), float32(r.Min.Y)))
	p.LineTo(f32.Pt(float32(r.Max.X), float32(r.Min.Y)))
	p.LineTo(f32.Pt(float32(r.Max.X), float32(r.Max.Y)))
	p.LineTo(f32.Pt(float32(r.Min.X), float32(r.Max.Y)))
	p.Close()
}

// AddRRectPath traces a rounded rectangle outline into p using quadratic beziers for corners.
func AddRRectPath(p *clip.Path, r image.Rectangle, cr float32) {
	minX := float32(r.Min.X)
	minY := float32(r.Min.Y)
	maxX := float32(r.Max.X)
	maxY := float32(r.Max.Y)

	p.MoveTo(f32.Pt(minX+cr, minY))
	p.LineTo(f32.Pt(maxX-cr, minY))
	p.QuadTo(f32.Pt(maxX, minY), f32.Pt(maxX, minY+cr))
	p.LineTo(f32.Pt(maxX, maxY-cr))
	p.QuadTo(f32.Pt(maxX, maxY), f32.Pt(maxX-cr, maxY))
	p.LineTo(f32.Pt(minX+cr, maxY))
	p.QuadTo(f32.Pt(minX, maxY), f32.Pt(minX, maxY-cr))
	p.LineTo(f32.Pt(minX, minY+cr))
	p.QuadTo(f32.Pt(minX, minY), f32.Pt(minX+cr, minY))
	p.Close()
}

// ArcToCubics converts an SVG elliptical arc from (x1,y1) to (x2,y2) with
// radii rx,ry and x-axis rotation xRot (radians) into cubic Bezier Segments.
func ArcToCubics(x1, y1, rx, ry, xRot float64, largeArc, sweep bool, x2, y2 float64) []Segment {
	return arcToCubics(x1, y1, rx, ry, xRot, largeArc, sweep, x2, y2)
}

//nolint:cyclop,gocognit // Path command expansion is an inherently large switch.
func expandCmd(cmd byte, args []float64, cx, cy float64, st *pathState) (segs []Segment, nx, ny float64) {
	nx, ny = cx, cy
	rel := cmd >= 'a' && cmd <= 'z'
	abs := cmd
	if rel {
		abs = cmd - 32
	}

	dx, dy := 0.0, 0.0
	if rel {
		dx, dy = cx, cy
	}

	switch abs {
	case 'M':
		for i := 0; i+1 < len(args); i += 2 {
			x, y := args[i]+dx, args[i+1]+dy
			segCmd := byte('M')
			if i > 0 {
				segCmd = 'L'
			}
			segs = append(segs, Segment{Cmd: segCmd, Args: []float64{x, y}})
			nx, ny = x, y
			// Rebase dx/dy after each point so subsequent relative pairs are cumulative.
			if rel {
				dx, dy = x, y
			}
		}
	case 'L':
		for i := 0; i+1 < len(args); i += 2 {
			x, y := args[i]+dx, args[i+1]+dy
			segs = append(segs, Segment{Cmd: 'L', Args: []float64{x, y}})
			nx, ny = x, y
			if rel {
				dx, dy = nx, ny
			}
		}
	case 'H':
		for _, v := range args {
			x := v + dx
			segs = append(segs, Segment{Cmd: 'L', Args: []float64{x, ny}})
			nx = x
			if rel {
				dx = nx
			}
		}
	case 'V':
		for _, v := range args {
			y := v + dy
			segs = append(segs, Segment{Cmd: 'L', Args: []float64{nx, y}})
			ny = y
			if rel {
				dy = ny
			}
		}
	case 'C':
		for i := 0; i+5 < len(args); i += 6 {
			x1, y1 := args[i]+dx, args[i+1]+dy
			x2, y2 := args[i+2]+dx, args[i+3]+dy
			x, y := args[i+4]+dx, args[i+5]+dy
			segs = append(segs, Segment{Cmd: 'C', Args: []float64{x1, y1, x2, y2, x, y}})
			st.prevCtrl2X, st.prevCtrl2Y = x2, y2
			nx, ny = x, y
			if rel {
				dx, dy = nx, ny
			}
		}
	case 'S':
		for i := 0; i+3 < len(args); i += 4 {
			var x1, y1 float64
			if st.prevCmd == 'C' || st.prevCmd == 'S' {
				x1 = 2*cx - st.prevCtrl2X
				y1 = 2*cy - st.prevCtrl2Y
			} else {
				x1, y1 = cx, cy
			}
			x2, y2 := args[i]+dx, args[i+1]+dy
			x, y := args[i+2]+dx, args[i+3]+dy
			segs = append(segs, Segment{Cmd: 'C', Args: []float64{x1, y1, x2, y2, x, y}})
			st.prevCtrl2X, st.prevCtrl2Y = x2, y2
			nx, ny = x, y
			cx, cy = x, y
			if rel {
				dx, dy = nx, ny
			}
		}
	case 'Q':
		for i := 0; i+3 < len(args); i += 4 {
			x1, y1 := args[i]+dx, args[i+1]+dy
			x, y := args[i+2]+dx, args[i+3]+dy
			segs = append(segs, Segment{Cmd: 'Q', Args: []float64{x1, y1, x, y}})
			st.prevCtrl1X, st.prevCtrl1Y = x1, y1
			nx, ny = x, y
			if rel {
				dx, dy = nx, ny
			}
		}
	case 'T':
		for i := 0; i+1 < len(args); i += 2 {
			var x1, y1 float64
			if st.prevCmd == 'Q' || st.prevCmd == 'T' {
				x1 = 2*cx - st.prevCtrl1X
				y1 = 2*cy - st.prevCtrl1Y
			} else {
				x1, y1 = cx, cy
			}
			x, y := args[i]+dx, args[i+1]+dy
			segs = append(segs, Segment{Cmd: 'Q', Args: []float64{x1, y1, x, y}})
			st.prevCtrl1X, st.prevCtrl1Y = x1, y1
			nx, ny = x, y
			cx, cy = x, y
			if rel {
				dx, dy = nx, ny
			}
		}
	case 'A':
		for i := 0; i+6 < len(args); i += 7 {
			rx, ry := args[i], args[i+1]
			xRot := args[i+2] * math.Pi / 180
			largeArc := args[i+3] != 0
			sweep := args[i+4] != 0
			ex, ey := args[i+5]+dx, args[i+6]+dy
			cubics := arcToCubics(nx, ny, rx, ry, xRot, largeArc, sweep, ex, ey)
			segs = append(segs, cubics...)
			nx, ny = ex, ey
			if rel {
				dx, dy = nx, ny
			}
		}
	case 'Z':
		segs = append(segs, Segment{Cmd: 'Z'})
	}
	return segs, nx, ny
}

// arcToCubics converts an SVG elliptical arc to cubic bezier segments.
// Implements the endpoint-to-centre parameterisation from the SVG spec appendix.
func arcToCubics(x1, y1, rx, ry, xRot float64, largeArc, sweep bool, x2, y2 float64) []Segment {
	if x1 == x2 && y1 == y2 {
		return nil
	}
	if rx == 0 || ry == 0 {
		return []Segment{{Cmd: 'L', Args: []float64{x2, y2}}}
	}

	cosRot, sinRot := math.Cos(xRot), math.Sin(xRot)

	dx := (x1 - x2) / 2
	dy := (y1 - y2) / 2
	x1p := cosRot*dx + sinRot*dy
	y1p := -sinRot*dx + cosRot*dy

	rx = math.Abs(rx)
	ry = math.Abs(ry)

	lambda := (x1p*x1p)/(rx*rx) + (y1p*y1p)/(ry*ry)
	if lambda > 1 {
		s := math.Sqrt(lambda)
		rx *= s
		ry *= s
	}

	num := rx*rx*ry*ry - rx*rx*y1p*y1p - ry*ry*x1p*x1p
	den := rx*rx*y1p*y1p + ry*ry*x1p*x1p
	sq := 0.0
	if den != 0 && num/den > 0 {
		sq = math.Sqrt(num / den)
	}
	if largeArc == sweep {
		sq = -sq
	}
	cxp := sq * rx * y1p / ry
	cyp := -sq * ry * x1p / rx

	cx := cosRot*cxp - sinRot*cyp + (x1+x2)/2
	cy := sinRot*cxp + cosRot*cyp + (y1+y2)/2

	startAngle := vecAngle(1, 0, (x1p-cxp)/rx, (y1p-cyp)/ry)
	deltaAngle := vecAngle((x1p-cxp)/rx, (y1p-cyp)/ry, (-x1p-cxp)/rx, (-y1p-cyp)/ry)

	if !sweep && deltaAngle > 0 {
		deltaAngle -= 2 * math.Pi
	} else if sweep && deltaAngle < 0 {
		deltaAngle += 2 * math.Pi
	}

	n := max(int(math.Ceil(math.Abs(deltaAngle)/(math.Pi/2))), 1)
	step := deltaAngle / float64(n)

	var out []Segment
	for i := range n {
		a1 := startAngle + float64(i)*step
		a2 := a1 + step
		out = append(out, arcSegmentToCubic(cx, cy, rx, ry, xRot, a1, a2)...)
	}
	return out
}

func arcSegmentToCubic(cx, cy, rx, ry, xRot, a1, a2 float64) []Segment {
	cosRot, sinRot := math.Cos(xRot), math.Sin(xRot)
	alpha := math.Sin(a2-a1) * (math.Sqrt(4+3*math.Pow(math.Tan((a2-a1)/2), 2)) - 1) / 3

	x1 := cx + cosRot*rx*math.Cos(a1) - sinRot*ry*math.Sin(a1)
	y1 := cy + sinRot*rx*math.Cos(a1) + cosRot*ry*math.Sin(a1)
	dx1 := -cosRot*rx*math.Sin(a1) - sinRot*ry*math.Cos(a1)
	dy1 := -sinRot*rx*math.Sin(a1) + cosRot*ry*math.Cos(a1)
	x2 := cx + cosRot*rx*math.Cos(a2) - sinRot*ry*math.Sin(a2)
	y2 := cy + sinRot*rx*math.Cos(a2) + cosRot*ry*math.Sin(a2)
	dx2 := -cosRot*rx*math.Sin(a2) - sinRot*ry*math.Cos(a2)
	dy2 := -sinRot*rx*math.Sin(a2) + cosRot*ry*math.Cos(a2)

	return []Segment{{
		Cmd: 'C',
		Args: []float64{
			x1 + alpha*dx1, y1 + alpha*dy1,
			x2 - alpha*dx2, y2 - alpha*dy2,
			x2, y2,
		},
	}}
}

func vecAngle(ux, uy, vx, vy float64) float64 {
	n := math.Sqrt(ux*ux+uy*uy) * math.Sqrt(vx*vx+vy*vy)
	if n == 0 {
		return 0
	}
	// Clamp to guard against floating-point drift past ±1.
	cos := math.Max(-1, math.Min(1, (ux*vx+uy*vy)/n))
	angle := math.Acos(cos)
	if ux*vy-uy*vx < 0 {
		angle = -angle
	}
	return angle
}

func isKnownCmd(b byte) bool {
	const known = "MmLlHhVvCcSsQqTtAaZz"
	for i := range len(known) {
		if known[i] == b {
			return true
		}
	}
	return false
}

//nolint:cyclop,gocognit // Tokenization is a large switch over character classes.
func tokenise(d string) []string {
	d = strings.TrimSpace(d)
	var tokens []string
	start := -1
	for i, ch := range d {
		switch {
		case (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z'):
			if start >= 0 {
				tokens = append(tokens, strings.TrimSpace(d[start:i]))
				start = -1
			}
			tokens = append(tokens, string(ch))
		case ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' || ch == ',':
			if start >= 0 {
				tokens = append(tokens, strings.TrimSpace(d[start:i]))
				start = -1
			}
		case (ch >= '0' && ch <= '9') || ch == '.' || ch == '-' || ch == '+':
			// A bare '-' or '+' immediately after a digit that is not preceded by 'e'/'E'
			// starts a new signed number (e.g. "1-2" → ["1", "-2"]).
			if (ch == '-' || ch == '+') && start >= 0 {
				prev := d[start:i]
				if len(prev) > 0 && prev[len(prev)-1] != 'e' && prev[len(prev)-1] != 'E' {
					tokens = append(tokens, strings.TrimSpace(prev))
					start = i
					continue
				}
			}
			// A '.' inside a number that already has a decimal point starts a new number
			// (e.g. "-.7.6" → ["-0.7", "0.6"]), per the SVG path grammar.
			if ch == '.' && start >= 0 && strings.ContainsRune(d[start:i], '.') {
				tokens = append(tokens, strings.TrimSpace(d[start:i]))
				start = i
				continue
			}
			if start < 0 {
				start = i
			}
		}
	}
	if start >= 0 {
		tokens = append(tokens, strings.TrimSpace(d[start:]))
	}
	return tokens
}
