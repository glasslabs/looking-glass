package svg

import (
	"image"
	"image/color"
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
	"github.com/glasslabs/looking-glass/ui/svg/element"
	svgpath "github.com/glasslabs/looking-glass/ui/svg/path"
)

// cache records and replays Gio op sequences for a single draw command.
// K is the cache key type; a key change triggers a re-record.
type cache[K comparable] struct {
	key   K
	buf   op.Ops
	call  op.CallOp
	valid bool
}

func (c *cache[K]) replay(key K, ops *op.Ops) bool {
	if !c.valid || c.key != key {
		return false
	}
	c.call.Add(ops)
	return true
}

func (c *cache[K]) record(key K, ops *op.Ops, fn func(*op.Ops)) {
	c.buf.Reset()
	macro := op.Record(&c.buf)
	fn(&c.buf)
	c.call = macro.Stop()
	c.key = key
	c.valid = true
	c.call.Add(ops)
}

// svgCacheKey discriminates one SVG element's rendered ops. It captures
// everything that affects pixel output: element content (via hash), the
// accumulated parent transform, the fit viewport, and the display metric.
type svgCacheKey struct {
	hash             uint64
	a0, a1, a2       float32 // accumulated transform rows
	a3, a4, a5       float32
	fitW, fitH       int32
	pxPerDp, pxPerSp float32
}

func makeSVGCacheKey(hash uint64, acc f32.Affine2D, fitSz image.Point, m unit.Metric) svgCacheKey {
	a0, a1, a2, a3, a4, a5 := acc.Elems()
	return svgCacheKey{
		hash: hash,
		a0:   a0, a1: a1, a2: a2,
		a3: a3, a4: a4, a5: a5,
		fitW: int32(fitSz.X), fitH: int32(fitSz.Y),
		pxPerDp: m.PxPerDp, pxPerSp: m.PxPerSp,
	}
}

// cacheNode holds the op cache for one position in the draw command tree.
type cacheNode struct {
	c        cache[svgCacheKey]
	children []*cacheNode
}

// Renderer caches Gio op sequences per SVG draw command, keyed by content
// hash, accumulated transform, viewport size, and display metric.
type Renderer struct {
	root []*cacheNode
}

// NewRenderer returns a new Renderer with an empty cache.
func NewRenderer() *Renderer {
	return &Renderer{}
}

// Render draws doc into gtx scaled to fit fitSz, returning the occupied
// dimensions. The shaper is used for TextCmd elements; pass nil to skip text.
func (r *Renderer) Render(
	gtx layout.Context, doc *element.Doc, fitSz image.Point, shaper *text.Shaper,
) layout.Dimensions {
	if doc == nil || fitSz.X == 0 || fitSz.Y == 0 {
		return layout.Dimensions{}
	}

	baseXform := viewBoxXform(doc, fitSz)

	defer clip.Rect{Max: fitSz}.Push(gtx.Ops).Pop()
	r.renderCmds(gtx, doc.Children, baseXform, fitSz, shaper, &r.root)
	return layout.Dimensions{Size: fitSz}
}

func (r *Renderer) renderCmds(
	gtx layout.Context, cmds []element.DrawCmd, accumulated f32.Affine2D,
	fitSz image.Point, shaper *text.Shaper, nodes *[]*cacheNode,
) {
	for len(*nodes) < len(cmds) {
		*nodes = append(*nodes, &cacheNode{})
	}
	for i, cmd := range cmds {
		r.renderCachedCmd(gtx, cmd, accumulated, fitSz, shaper, (*nodes)[i])
	}
}

func (r *Renderer) renderCachedCmd(
	gtx layout.Context, cmd element.DrawCmd, accumulated f32.Affine2D,
	fitSz image.Point, shaper *text.Shaper, node *cacheNode,
) {
	switch v := cmd.(type) {
	case *element.GroupCmd:
		key := makeSVGCacheKey(v.Hash, accumulated, fitSz, gtx.Metric)
		if node.c.replay(key, gtx.Ops) {
			return
		}
		next := accumulated.Mul(v.Transform)
		node.c.record(key, gtx.Ops, func(ops *op.Ops) {
			childGtx := gtx
			childGtx.Ops = ops
			r.renderCmds(childGtx, v.Children, next, fitSz, shaper, &node.children)
		})

	case *element.PathCmd:
		key := makeSVGCacheKey(v.Hash, accumulated, fitSz, gtx.Metric)
		if node.c.replay(key, gtx.Ops) {
			return
		}
		node.c.record(key, gtx.Ops, func(ops *op.Ops) {
			leafGtx := gtx
			leafGtx.Ops = ops
			renderPath(leafGtx, v, accumulated)
		})

	case *element.RectCmd:
		key := makeSVGCacheKey(v.Hash, accumulated, fitSz, gtx.Metric)
		if node.c.replay(key, gtx.Ops) {
			return
		}
		node.c.record(key, gtx.Ops, func(ops *op.Ops) {
			leafGtx := gtx
			leafGtx.Ops = ops
			renderRect(leafGtx, v, accumulated)
		})

	case *element.CircleCmd:
		key := makeSVGCacheKey(v.Hash, accumulated, fitSz, gtx.Metric)
		if node.c.replay(key, gtx.Ops) {
			return
		}
		node.c.record(key, gtx.Ops, func(ops *op.Ops) {
			leafGtx := gtx
			leafGtx.Ops = ops
			renderCircle(leafGtx, v, accumulated)
		})

	case *element.TextCmd:
		if shaper == nil {
			return
		}
		key := makeSVGCacheKey(v.Hash, accumulated, fitSz, gtx.Metric)
		if node.c.replay(key, gtx.Ops) {
			return
		}
		node.c.record(key, gtx.Ops, func(ops *op.Ops) {
			leafGtx := gtx
			leafGtx.Ops = ops
			renderText(leafGtx, v, accumulated, shaper)
		})
	}
}

// Render draws doc into gtx scaled to fit fitSz. It does not cache between
// calls; for repeated rendering of the same document use a Renderer.
func Render(gtx layout.Context, doc *element.Doc, fitSz image.Point, shaper *text.Shaper) layout.Dimensions {
	return NewRenderer().Render(gtx, doc, fitSz, shaper)
}

func viewBoxXform(doc *element.Doc, fitSz image.Point) f32.Affine2D {
	vbW := doc.ViewBox[2]
	vbH := doc.ViewBox[3]
	if vbW <= 0 {
		vbW = doc.Width
	}
	if vbH <= 0 {
		vbH = doc.Height
	}
	if vbW <= 0 || vbH <= 0 {
		return f32.AffineId()
	}

	scaleX := float32(fitSz.X) / float32(vbW)
	scaleY := float32(fitSz.Y) / float32(vbH)
	minX := float32(-doc.ViewBox[0]) * scaleX
	minY := float32(-doc.ViewBox[1]) * scaleY
	return f32.NewAffine2D(scaleX, 0, minX, 0, scaleY, minY)
}

func renderPath(gtx layout.Context, v *element.PathCmd, accumulated f32.Affine2D) {
	xform := accumulated.Mul(v.Transform)

	if !v.Style.FillNone {
		col := mulOpacity(v.Style.Fill, v.Style.FillOpacity*v.Style.Opacity)
		var p clip.Path
		p.Begin(gtx.Ops)
		svgpath.BuildClipPath(&p, v.Segments, xform)
		paint.FillShape(gtx.Ops, col, clip.Outline{Path: p.End()}.Op())
	}
	if !v.Style.StrokeNone && v.Style.StrokeWidth > 0 {
		col := mulOpacity(v.Style.Stroke, v.Style.StrokeOpacity*v.Style.Opacity)
		sx, _, _, _, sy, _ := xform.Elems()
		scaledWidth := v.Style.StrokeWidth * (sx + sy) / 2
		svgpath.StrokePath(gtx.Ops, v.Segments, xform, scaledWidth, col)
	}
}

func renderRect(gtx layout.Context, v *element.RectCmd, accumulated f32.Affine2D) {
	xform := accumulated.Mul(v.Transform)

	tl := xform.Transform(f32.Pt(float32(v.X), float32(v.Y)))
	br := xform.Transform(f32.Pt(float32(v.X+v.W), float32(v.Y+v.H)))
	bounds := image.Rect(int(tl.X), int(tl.Y), int(br.X), int(br.Y))
	if bounds.Empty() {
		return
	}

	sx, _, _, _, sy, _ := xform.Elems()
	cr := float32(v.RX) * (sx + sy) / 2
	if v.RY > 0 {
		cr = float32(v.RY) * (sx + sy) / 2
	}

	if !v.Style.FillNone {
		col := mulOpacity(v.Style.Fill, v.Style.FillOpacity*v.Style.Opacity)
		var shape clip.Op
		if cr > 0 {
			r := int(cr)
			shape = clip.RRect{Rect: bounds, SE: r, SW: r, NW: r, NE: r}.Op(gtx.Ops)
		} else {
			shape = clip.Rect(bounds).Op()
		}
		paint.FillShape(gtx.Ops, col, shape)
	}
	if !v.Style.StrokeNone && v.Style.StrokeWidth > 0 {
		col := mulOpacity(v.Style.Stroke, v.Style.StrokeOpacity*v.Style.Opacity)
		sw := v.Style.StrokeWidth * (sx + sy) / 2
		var p clip.Path
		p.Begin(gtx.Ops)
		if cr > 0 {
			svgpath.AddRRectPath(&p, bounds, cr)
		} else {
			svgpath.AddRectPath(&p, bounds)
		}
		paint.FillShape(gtx.Ops, col, clip.Stroke{Path: p.End(), Width: sw}.Op())
	}
}

func renderCircle(gtx layout.Context, v *element.CircleCmd, accumulated f32.Affine2D) {
	cx, cy, rx, ry := v.CX, v.CY, v.RX, v.RY
	right := cx + rx
	left := cx - rx

	topSegs := svgpath.ArcToCubics(right, cy, rx, ry, 0, false, false, left, cy)
	botSegs := svgpath.ArcToCubics(left, cy, rx, ry, 0, false, false, right, cy)

	segs := make([]svgpath.Segment, 0, 2+len(topSegs)+len(botSegs))
	segs = append(segs, svgpath.Segment{Cmd: 'M', Args: []float64{right, cy}})
	segs = append(segs, topSegs...)
	segs = append(segs, botSegs...)
	segs = append(segs, svgpath.Segment{Cmd: 'Z'})

	synth := &element.PathCmd{Segments: segs, Style: v.Style, Transform: v.Transform}
	renderPath(gtx, synth, accumulated)
}

func renderText(gtx layout.Context, v *element.TextCmd, accumulated f32.Affine2D, shaper *text.Shaper) {
	if v.Content == "" {
		return
	}
	xform := accumulated.Mul(v.Transform)

	pos := xform.Transform(f32.Pt(float32(v.X), float32(v.Y)))

	sx, _, _, hy, sy, _ := xform.Elems()
	effectiveScale := float32(math.Sqrt(float64(sx*sx+hy*hy+sy*sy) / 2))
	fontPx := max(int(v.Style.FontSize*effectiveScale), 1)
	sp := gtx.Metric.PxToSp(fontPx)
	if sp <= 0 {
		sp = unit.Sp(14)
	}

	col := mulOpacity(v.Style.Fill, v.Style.FillOpacity*v.Style.Opacity)

	var measureOps op.Ops
	mGtx := gtx
	mGtx.Ops = &measureOps
	mGtx.Constraints = layout.Constraints{Max: image.Pt(1<<24, 1<<24)}
	mrec := op.Record(&measureOps)
	paint.ColorOp{Color: col}.Add(&measureOps)
	mCall := mrec.Stop()
	dims := widget.Label{MaxLines: 1}.Layout(mGtx, shaper, font.Font{}, sp, v.Content, mCall)

	startX := int(pos.X)
	switch v.Style.TextAnchor {
	case "middle":
		startX -= dims.Size.X / 2
	case "end":
		startX -= dims.Size.X
	}
	startY := int(pos.Y) - dims.Size.Y/2

	stack := op.Offset(image.Pt(startX, startY)).Push(gtx.Ops)
	childGtx := gtx
	childGtx.Constraints = layout.Exact(dims.Size)
	rec := op.Record(gtx.Ops)
	paint.ColorOp{Color: col}.Add(gtx.Ops)
	call := rec.Stop()
	widget.Label{MaxLines: 1}.Layout(childGtx, shaper, font.Font{}, sp, v.Content, call)
	stack.Pop()
}

func mulOpacity(col color.NRGBA, opacity float32) color.NRGBA {
	if opacity >= 1 {
		return col
	}
	if opacity <= 0 {
		return color.NRGBA{}
	}
	col.A = uint8(float32(col.A) * opacity)
	return col
}
