package ui

import (
	"image"
	"math"

	"gioui.org/f32"
	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"github.com/glasslabs/client-go"
	svgpath "github.com/glasslabs/looking-glass/ui/svg/path"
	"github.com/glasslabs/looking-glass/ui/svg/style"
)

type canvasRenderer struct {
	ops []client.DrawOp

	paths map[string][]svgpath.Segment
}

func newCanvasRenderer(ops []client.DrawOp, prevPaths map[string][]svgpath.Segment) *canvasRenderer {
	r := &canvasRenderer{
		ops:   ops,
		paths: make(map[string][]svgpath.Segment),
	}

	for _, op := range ops {
		p, ok := op.(*client.Path)
		if !ok || p.D == "" {
			continue
		}
		if _, exists := r.paths[p.D]; exists {
			continue
		}
		if s, exists := prevPaths[p.D]; exists {
			r.paths[p.D] = s
			continue
		}
		segs, err := svgpath.Parse(p.D)
		if err == nil {
			r.paths[p.D] = segs
		}
	}
	return r
}

func (r *canvasRenderer) layout(gtx layout.Context, scaleX, scaleY float32, shaper *text.Shaper) {
	for _, drawOp := range r.ops {
		if drawOp == nil {
			continue
		}
		r.layoutDrawOp(gtx, drawOp, scaleX, scaleY, shaper)
	}
}

func (r *canvasRenderer) layoutDrawOp(gtx layout.Context, op client.DrawOp, scaleX, scaleY float32, shaper *text.Shaper) {
	switch v := op.(type) {
	case *client.Arc:
		layoutArc(gtx, v, scaleX, scaleY)
	case *client.Rect:
		layoutRect(gtx, v, scaleX, scaleY)
	case *client.Label:
		layoutLabel(gtx, v, scaleX, scaleY, shaper)
	case *client.Path:
		r.layoutPath(gtx, v, scaleX, scaleY)
	}
}

func layoutArc(gtx layout.Context, a *client.Arc, scaleX, scaleY float32) {
	col, ok := style.ParseColor(a.Color)
	if !ok {
		return
	}

	scale := (scaleX + scaleY) / 2
	cx := a.Cx * scaleX
	cy := a.Cy * scaleY
	r := a.Radius * scale
	sw := a.StrokeWidth * scale

	startRad := float64(a.StartAngle) * math.Pi / 180
	sweepRad := float64(a.SweepAngle) * math.Pi / 180
	if sweepRad == 0 {
		return
	}

	var p clip.Path
	p.Begin(gtx.Ops)

	const maxSegments = 128
	steps := int(math.Ceil(math.Abs(float64(maxSegments) * sweepRad / (2 * math.Pi))))
	if steps < 1 {
		steps = 1
	} else if steps > maxSegments {
		steps = maxSegments
	}

	x0 := cx + r*float32(math.Cos(startRad))
	y0 := cy + r*float32(math.Sin(startRad))
	p.MoveTo(f32.Pt(x0, y0))

	stepAngle := sweepRad / float64(steps)
	for i := 1; i <= steps; i++ {
		angle := startRad + float64(i)*stepAngle
		p.LineTo(f32.Pt(
			cx+r*float32(math.Cos(angle)),
			cy+r*float32(math.Sin(angle)),
		))
	}

	spec := clip.Stroke{Path: p.End(), Width: sw}.Op()
	paint.FillShape(gtx.Ops, col, spec)
}

func layoutRect(gtx layout.Context, r *client.Rect, scaleX, scaleY float32) {
	x := int(r.X * scaleX)
	y := int(r.Y * scaleY)
	w := int(r.W * scaleX)
	h := int(r.H * scaleY)
	cr := r.CornerRadius * (scaleX + scaleY) / 2

	bounds := image.Rect(x, y, x+w, y+h)

	if r.Fill != "" {
		col, _ := style.ParseColor(r.Fill)
		var shape clip.Op
		if cr > 0 {
			shape = clip.RRect{Rect: bounds, SE: int(cr), SW: int(cr), NW: int(cr), NE: int(cr)}.Op(gtx.Ops)
		} else {
			shape = clip.Rect(bounds).Op()
		}
		paint.FillShape(gtx.Ops, col, shape)
	}

	if r.Stroke != "" && r.StrokeWidth > 0 {
		col, _ := style.ParseColor(r.Stroke)
		sw := r.StrokeWidth * (scaleX + scaleY) / 2
		var p clip.Path
		p.Begin(gtx.Ops)
		if cr > 0 {
			svgpath.AddRRectPath(&p, bounds, cr)
		} else {
			svgpath.AddRectPath(&p, bounds)
		}
		spec := clip.Stroke{Path: p.End(), Width: sw}.Op()
		paint.FillShape(gtx.Ops, col, spec)
	}
}

func layoutLabel(gtx layout.Context, l *client.Label, scaleX, scaleY float32, shaper *text.Shaper) {
	if len(l.Runs) == 0 {
		return
	}

	px := int(l.X * scaleX)
	py := int(l.Y * scaleY)

	type measure struct{ w, h int }
	measures := make([]measure, len(l.Runs))
	totalW := 0
	maxH := 0
	for i, run := range l.Runs {
		fontPx := max(int(run.FontSize*(scaleX+scaleY)/2), 1)
		sp := gtx.Metric.PxToSp(fontPx)
		if sp <= 0 {
			sp = defaultFontSizeSp
		}
		sz := measureLabelRun(gtx, shaper, font.Font{Typeface: robotoTypeface}, sp, run.Content)
		measures[i] = measure{w: sz.X, h: sz.Y}
		totalW += sz.X
		if sz.Y > maxH {
			maxH = sz.Y
		}
	}

	startX := px
	switch l.Align {
	case "middle":
		startX = px - totalW/2
	case "end":
		startX = px - totalW
	}

	startY := py - maxH/2

	curX := startX
	for i, run := range l.Runs {
		col := defaultTextColor
		if run.Color != "" {
			col, _ = style.ParseColor(run.Color)
		}

		fontPx := max(int(run.FontSize*(scaleX+scaleY)/2), 1)
		sp := gtx.Metric.PxToSp(fontPx)
		if sp <= 0 {
			sp = defaultFontSizeSp
		}

		shiftPx := int(run.BaselineShift * (scaleX + scaleY) / 2)
		runY := startY + shiftPx

		stack := op.Offset(image.Pt(curX, runY)).Push(gtx.Ops)
		childGtx := gtx
		childGtx.Constraints = layout.Exact(image.Pt(measures[i].w, measures[i].h))

		m := op.Record(gtx.Ops)
		paint.ColorOp{Color: col}.Add(gtx.Ops)
		call := m.Stop()

		widget.Label{Alignment: text.Start, MaxLines: 1}.Layout(
			childGtx, shaper, font.Font{Typeface: robotoTypeface}, sp, run.Content, call)
		stack.Pop()

		curX += measures[i].w
	}
}

func measureLabelRun(gtx layout.Context, shaper *text.Shaper, f font.Font, size unit.Sp, content string) image.Point {
	var ops op.Ops
	measureGtx := gtx
	measureGtx.Ops = &ops
	measureGtx.Constraints = layout.Constraints{Max: image.Pt(1<<24, 1<<24)}

	dims := widget.Label{Alignment: text.Start, MaxLines: 1}.Layout(measureGtx, shaper, f, size, content, op.CallOp{})
	return dims.Size
}

func (r *canvasRenderer) layoutPath(gtx layout.Context, p *client.Path, scaleX, scaleY float32) {
	col := defaultTextColor
	if p.Fill != "" {
		col, _ = style.ParseColor(p.Fill)
	}

	pathScale := p.Scale
	if pathScale == 0 {
		pathScale = 1
	}

	segments, ok := r.paths[p.D]
	if !ok || len(segments) == 0 {
		return
	}

	var cp clip.Path
	cp.Begin(gtx.Ops)
	xform := f32.NewAffine2D(scaleX*pathScale, 0, p.X*scaleX, 0, scaleY*pathScale, p.Y*scaleY)
	svgpath.BuildClipPath(&cp, segments, xform)
	paint.FillShape(gtx.Ops, col, clip.Outline{Path: cp.End()}.Op())
}
