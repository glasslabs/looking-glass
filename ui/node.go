package ui

import (
	"fmt"
	"image"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"github.com/glasslabs/client-go"
	"github.com/glasslabs/looking-glass/ui/svg"
	"github.com/glasslabs/looking-glass/ui/svg/element"
	svgpath "github.com/glasslabs/looking-glass/ui/svg/path"
	"github.com/glasslabs/looking-glass/ui/svg/style"
)

// node is the host-side representation of a client.Widget with cached layout state.
type node interface {
	size(gtx layout.Context, shaper *text.Shaper) image.Point
	layout(gtx layout.Context, shaper *text.Shaper) layout.Dimensions
}

// newNode creates a node from a client.Widget, reusing prev's cached state
// where the content is unchanged. Pass prev == nil on first wrap.
//
//nolint:cyclop,gocognit // Splitting this will not make it simpler.
func newNode(w client.Widget, prev node) (node, error) {
	if w == nil {
		return nil, fmt.Errorf("nil Widget")
	}
	switch v := w.(type) {
	case *client.Text:
		if p, ok := prev.(*textNode); ok && p.w.Equals(v) {
			return p, nil
		}
		return newTextNode(v), nil
	case *client.SVG:
		prevSVG, ok := prev.(*svgNode)
		if ok && prevSVG.w.Content == v.Content {
			return prevSVG, nil
		}
		return newSVGNode(v, prevSVG)
	case *client.Canvas:
		prevCanvas, ok := prev.(*canvasNode)
		if ok && prevCanvas.w.Equals(v) {
			return prevCanvas, nil
		}
		return newCanvasNode(v, prevCanvas), nil
	case *client.VStack:
		prevNode, _ := prev.(*vstackNode)
		var prevChildren []node
		if prevNode != nil {
			prevChildren = prevNode.children
		}
		allSame := prevNode != nil && len(v.Children) == len(prevChildren)
		children := make([]node, len(v.Children))
		for i, c := range v.Children {
			var prevChild node
			if i < len(prevChildren) {
				prevChild = prevChildren[i]
			}
			n, err := newNode(c, prevChild)
			if err != nil {
				return nil, err
			}
			children[i] = n
			if allSame && children[i] != prevChild {
				allSame = false
			}
		}
		if allSame {
			return prevNode, nil
		}
		return newVStackNode(v, children), nil
	case *client.HStack:
		prevNode, _ := prev.(*hstackNode)
		var prevChildren []node
		if prevNode != nil {
			prevChildren = prevNode.children
		}
		allSame := prevNode != nil && len(v.Children) == len(prevChildren)
		children := make([]node, len(v.Children))
		for i, c := range v.Children {
			var prevChild node
			if i < len(prevChildren) {
				prevChild = prevChildren[i]
			}
			n, err := newNode(c, prevChild)
			if err != nil {
				return nil, err
			}
			children[i] = n
			if allSame && children[i] != prevChild {
				allSame = false
			}
		}
		if allSame {
			return prevNode, nil
		}
		return newHStackNode(v, children), nil
	case *client.Table:
		prevNode, _ := prev.(*tableNode)
		var prevCells [][]node
		if prevNode != nil {
			prevCells = prevNode.cells
		}
		allSame := prevNode != nil && len(v.Rows) == len(prevCells)
		cells := make([][]node, len(v.Rows))
		for r, row := range v.Rows {
			var prevRow []node
			if r < len(prevCells) {
				prevRow = prevCells[r]
			}
			if allSame && len(row.Columns) != len(prevRow) {
				allSame = false
			}
			cells[r] = make([]node, len(row.Columns))
			for c, col := range row.Columns {
				var prevChild node
				if c < len(prevRow) {
					prevChild = prevRow[c]
				}
				var n node
				if col.Child != nil {
					var err error
					n, err = newNode(col.Child, prevChild)
					if err != nil {
						return nil, err
					}
				} else {
					n = &spacerNode{}
				}
				cells[r][c] = n
				if allSame && cells[r][c] != prevChild {
					allSame = false
				}
			}
		}
		if allSame {
			return prevNode, nil
		}
		return newTableNode(v, cells), nil
	case *client.Spacer:
		return &spacerNode{min: v.Min}, nil
	default:
		return nil, fmt.Errorf("unknown node type: %T", v)
	}
}

type textNode struct {
	sizeCache
	layoutCache

	w *client.Text
}

func newTextNode(w *client.Text) *textNode {
	n := &textNode{
		w: w,
	}
	n.sizeFn = n.doSize
	n.layoutFn = n.doLayout
	return n
}

func (n *textNode) doSize(gtx layout.Context, shaper *text.Shaper) image.Point {
	if n.w.Content == "" {
		return image.Point{}
	}

	f, sz := textFontAndSize(n.w)

	var measureOps op.Ops
	measureGtx := gtx
	measureGtx.Ops = &measureOps
	measureGtx.Constraints = layout.Constraints{Max: image.Pt(1<<24, 1<<24)}

	return widget.Label{Alignment: text.Start}.Layout(measureGtx, shaper, f, sz, n.w.Content, op.CallOp{}).Size
}

func (n *textNode) doLayout(gtx layout.Context, shaper *text.Shaper) layout.Dimensions {
	col := defaultTextColor
	if n.w.Color != "" {
		col, _ = style.ParseColor(n.w.Color)
	}

	f, sz := textFontAndSize(n.w)

	align := text.Start
	switch n.w.Align {
	case "center":
		align = text.Middle
	case "right":
		align = text.End
	}

	colorMacro := op.Record(gtx.Ops)
	paint.ColorOp{Color: col}.Add(gtx.Ops)
	colorCall := colorMacro.Stop()

	return widget.Label{Alignment: align}.Layout(gtx, shaper, f, sz, n.w.Content, colorCall)
}

func textFontAndSize(w *client.Text) (font.Font, unit.Sp) {
	f := font.Font{Typeface: robotoTypeface}
	sz := defaultFontSizeSp
	if w.FontSize > 0 {
		sz = unit.Sp(w.FontSize)
	}
	if w.Condensed {
		f.Typeface = robotoCondensedTypeface
	}
	switch {
	case w.Bold:
		f.Weight = font.Bold
	case w.Light:
		f.Weight = font.Light
	}
	if w.Italic {
		f.Style = font.Italic
	}
	return f, sz
}

type svgNode struct {
	sizeCache
	layoutCache

	w        *client.SVG
	renderer *svg.Renderer

	doc  *element.Doc
	svgW float64
	svgH float64
}

// newSVGNode creates an svgNode, eagerly parsing the SVG content.
// When prev is provided its renderer is reused so warm caches survive a content change.
func newSVGNode(w *client.SVG, prev *svgNode) (*svgNode, error) {
	var r *svg.Renderer
	if prev != nil {
		r = prev.renderer
	} else {
		r = svg.NewRenderer()
	}
	n := &svgNode{w: w, renderer: r}
	n.sizeFn = n.doSize
	n.layoutFn = n.doLayout
	if w.Content == "" {
		return n, nil
	}

	doc, err := element.Parse(w.Content)
	if err != nil {
		return nil, fmt.Errorf("parsing element %q: %w", w.Content, err)
	}
	n.doc = doc
	n.svgW, n.svgH = doc.Width, doc.Height
	if n.svgW <= 0 || n.svgH <= 0 {
		n.svgW, n.svgH = doc.ViewBox[2], doc.ViewBox[3]
	}
	return n, nil
}

func (n *svgNode) doSize(gtx layout.Context, _ *text.Shaper) image.Point {
	if n.doc == nil || n.svgW <= 0 || n.svgH <= 0 {
		return image.Point{}
	}
	return image.Pt(
		gtx.Metric.Dp(unit.Dp(float32(n.svgW))),
		gtx.Metric.Dp(unit.Dp(float32(n.svgH))),
	)
}

func (n *svgNode) doLayout(gtx layout.Context, shaper *text.Shaper) layout.Dimensions {
	if n.doc == nil {
		return layout.Dimensions{}
	}
	constMax := gtx.Constraints.Max
	if constMax.X == 0 || constMax.Y == 0 {
		return layout.Dimensions{}
	}

	svgW, svgH := n.svgW, n.svgH
	if svgW <= 0 || svgH <= 0 {
		svgW, svgH = float64(constMax.X), float64(constMax.Y)
	}

	naturalW := gtx.Metric.Dp(unit.Dp(float32(svgW)))
	naturalH := gtx.Metric.Dp(unit.Dp(float32(svgH)))
	fitMax := image.Pt(min(naturalW, constMax.X), min(naturalH, constMax.Y))

	fitSz := svg.FitSize(svgW, svgH, fitMax)
	if fitSz.X == 0 || fitSz.Y == 0 {
		return layout.Dimensions{}
	}

	return n.renderer.Render(gtx, n.doc, fitSz, shaper)
}

type canvasNode struct {
	sizeCache
	layoutCache

	w *client.Canvas
	r *canvasRenderer
}

func newCanvasNode(w *client.Canvas, prev *canvasNode) *canvasNode {
	var prevPaths map[string][]svgpath.Segment
	if prev != nil {
		prevPaths = prev.r.paths
	}
	r := newCanvasRenderer(w.Ops, prevPaths)

	n := &canvasNode{
		w: w,
		r: r,
	}
	n.sizeFn = n.doSize
	n.layoutFn = n.doLayout
	return n
}

func (n *canvasNode) doSize(gtx layout.Context, _ *text.Shaper) image.Point {
	if n.w.Width <= 0 || n.w.Height <= 0 {
		return image.Point{}
	}
	return image.Pt(
		gtx.Metric.Dp(unit.Dp(n.w.Width)),
		gtx.Metric.Dp(unit.Dp(n.w.Height)),
	)
}

func (n *canvasNode) doLayout(gtx layout.Context, shaper *text.Shaper) layout.Dimensions {
	c := n.w
	if c.Width <= 0 || c.Height <= 0 {
		return layout.Dimensions{}
	}

	fitSz := svg.FitSize(float64(c.Width), float64(c.Height), gtx.Constraints.Max)
	if fitSz.X == 0 || fitSz.Y == 0 {
		return layout.Dimensions{}
	}

	scaleX := float32(fitSz.X) / c.Width
	scaleY := float32(fitSz.Y) / c.Height

	defer clip.Rect{Max: fitSz}.Push(gtx.Ops).Pop()

	n.r.layout(gtx, scaleX, scaleY, shaper)

	return layout.Dimensions{Size: fitSz}
}

type vstackNode struct {
	sizeCache
	layoutCache

	w        *client.VStack
	children []node
}

func newVStackNode(w *client.VStack, children []node) *vstackNode {
	n := &vstackNode{
		w:        w,
		children: children,
	}
	n.sizeFn = n.doSize
	n.layoutFn = n.doLayout
	return n
}

func (n *vstackNode) doSize(gtx layout.Context, shaper *text.Shaper) image.Point {
	return stackNaturalSize(gtx, n.children, layout.Vertical, shaper)
}

func (n *vstackNode) doLayout(gtx layout.Context, shaper *text.Shaper) layout.Dimensions {
	// Cap Max.X to the natural width so that a VStack inside an HStack does not
	// greedily consume all remaining horizontal space. The pin Min.X = Max.X inside
	// layoutStack then uses the natural width as the text alignment box, which is
	// the correct behaviour regardless of whether the parent is a VStack or HStack.
	if natW := n.doSize(gtx, shaper).X; natW < gtx.Constraints.Max.X {
		gtx.Constraints.Max.X = natW
	}
	return layoutStack(gtx, n.children, layout.Vertical, shaper)
}

type hstackNode struct {
	sizeCache
	layoutCache

	w        *client.HStack
	children []node
}

func newHStackNode(w *client.HStack, children []node) *hstackNode {
	n := &hstackNode{
		w:        w,
		children: children,
	}
	n.sizeFn = n.doSize
	n.layoutFn = n.doLayout
	return n
}

func (n *hstackNode) doSize(gtx layout.Context, shaper *text.Shaper) image.Point {
	return stackNaturalSize(gtx, n.children, layout.Horizontal, shaper)
}

func (n *hstackNode) doLayout(gtx layout.Context, shaper *text.Shaper) layout.Dimensions {
	// Cap Max.Y to the natural height so that an HStack inside a VStack does not
	// greedily consume all remaining vertical space (the symmetric case of the
	// VStack-in-HStack fix above).
	if natH := n.doSize(gtx, shaper).Y; natH < gtx.Constraints.Max.Y {
		gtx.Constraints.Max.Y = natH
	}
	return layoutStack(gtx, n.children, layout.Horizontal, shaper)
}

type tableNode struct {
	sizeCache
	layoutCache

	w     *client.Table
	cells [][]node
}

func newTableNode(w *client.Table, cells [][]node) *tableNode {
	n := &tableNode{w: w, cells: cells}
	n.sizeFn = n.doSize
	n.layoutFn = n.doLayout
	return n
}

// colWidths returns the effective pixel width for each column: the maximum of all cells'
// natural widths and their MinWidth values across every row.
func (n *tableNode) colWidths(gtx layout.Context, shaper *text.Shaper) []int {
	ncols := 0
	for _, row := range n.cells {
		if len(row) > ncols {
			ncols = len(row)
		}
	}
	widths := make([]int, ncols)
	for r, row := range n.cells {
		for c, cell := range row {
			w := cell.size(gtx, shaper).X
			if r < len(n.w.Rows) && c < len(n.w.Rows[r].Columns) {
				if minW := gtx.Metric.Dp(unit.Dp(n.w.Rows[r].Columns[c].MinWidth)); minW > w {
					w = minW
				}
			}
			if w > widths[c] {
				widths[c] = w
			}
		}
	}
	return widths
}

func (n *tableNode) doSize(gtx layout.Context, shaper *text.Shaper) image.Point {
	colW := n.colWidths(gtx, shaper)
	totalW := 0
	for _, w := range colW {
		totalW += w
	}
	spacingPx := gtx.Metric.Dp(unit.Dp(n.w.RowSpacing))
	totalH := 0
	for i, row := range n.cells {
		rowH := 0
		for _, cell := range row {
			if h := cell.size(gtx, shaper).Y; h > rowH {
				rowH = h
			}
		}
		if i > 0 {
			totalH += spacingPx
		}
		totalH += rowH
	}
	return image.Pt(totalW, totalH)
}

func (n *tableNode) doLayout(gtx layout.Context, shaper *text.Shaper) layout.Dimensions {
	colW := n.colWidths(gtx, shaper)
	spacingPx := gtx.Metric.Dp(unit.Dp(n.w.RowSpacing))
	y := 0
	for i, row := range n.cells {
		rowH := 0
		for _, cell := range row {
			if h := cell.size(gtx, shaper).Y; h > rowH {
				rowH = h
			}
		}
		if i > 0 {
			y += spacingPx
		}
		x := 0
		for c, cell := range row {
			if c >= len(colW) {
				break
			}
			w := colW[c]
			cellGtx := gtx
			cellGtx.Constraints = layout.Constraints{
				Min: image.Pt(w, 0),
				Max: image.Pt(w, rowH),
			}
			stack := op.Offset(image.Pt(x, y)).Push(gtx.Ops)
			cell.layout(cellGtx, shaper)
			stack.Pop()
			x += w
		}
		y += rowH
	}
	totalW := 0
	for _, w := range colW {
		totalW += w
	}
	return layout.Dimensions{Size: image.Pt(totalW, y)}
}

type spacerNode struct {
	min float32
}

func (n *spacerNode) size(_ layout.Context, _ *text.Shaper) image.Point {
	return image.Point{}
}

func (n *spacerNode) layout(gtx layout.Context, _ *text.Shaper) layout.Dimensions {
	return layout.Dimensions{Size: gtx.Constraints.Min}
}

// stackNaturalSize returns the natural pixel size of a sequence of nodes along axis.
func stackNaturalSize(gtx layout.Context, children []node, axis layout.Axis, shaper *text.Shaper) image.Point {
	var result image.Point
	for _, child := range children {
		if s, isSpacer := child.(*spacerNode); isSpacer {
			if s.min > 0 {
				minPx := gtx.Metric.Dp(unit.Dp(s.min))
				if axis == layout.Vertical {
					result.Y += minPx
				} else {
					result.X += minPx
				}
			}
			continue
		}
		sz := child.size(gtx, shaper)
		if axis == layout.Vertical {
			if sz.X > result.X {
				result.X = sz.X
			}
			result.Y += sz.Y
		} else {
			result.X += sz.X
			if sz.Y > result.Y {
				result.Y = sz.Y
			}
		}
	}
	return result
}

// layoutStack lays out children along axis using a Gio Flex.
// For vertical stacks, Min.X is pinned to Max.X so text alignment has a consistent fixed-width box.
// Spacers with a minimum size are guaranteed that floor via a Rigid reservation; the remaining
// free space is distributed among spacers weighted by their minimum (larger minimum → larger share).
func layoutStack(gtx layout.Context, children []node, axis layout.Axis, shaper *text.Shaper) layout.Dimensions {
	// Each spacer with min>0 emits two FlexChild entries (Rigid floor + Flexed remainder),
	// so allocate up to twice the child count.
	flexChildren := make([]layout.FlexChild, 0, 2*len(children))

	for _, child := range children {
		c := child
		s, isSpacer := c.(*spacerNode)
		if !isSpacer {
			flexChildren = append(flexChildren, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if axis == layout.Vertical {
					gtx.Constraints.Min.X = gtx.Constraints.Max.X
				}
				return c.layout(gtx, shaper)
			}))
			continue
		}

		// Reserve the minimum as a Rigid child so the floor is always honoured,
		// even when the remaining free space is exhausted.
		if s.min > 0 {
			minPx := gtx.Metric.Dp(unit.Dp(s.min))
			flexChildren = append(flexChildren, layout.Rigid(func(_ layout.Context) layout.Dimensions {
				// Fill only the primary axis so the spacer does not inflate the
				// cross size of the containing row or column.
				if axis == layout.Horizontal {
					return layout.Dimensions{Size: image.Pt(minPx, 0)}
				}
				return layout.Dimensions{Size: image.Pt(0, minPx)}
			}))
		}
		// Weight the flexible portion by the minimum so that a spacer with a
		// larger minimum claims a proportionally larger share of the remaining
		// free space. Fall back to weight 1 for zero-min spacers.
		weight := float32(1)
		if s.min > 0 {
			weight = s.min
		}
		flexChildren = append(flexChildren, layout.Flexed(weight, func(gtx layout.Context) layout.Dimensions {
			// Fill only the primary axis; cross-axis is zero so the spacer
			// does not inflate the containing row/column's reported cross size.
			if axis == layout.Horizontal {
				return layout.Dimensions{Size: image.Pt(gtx.Constraints.Max.X, 0)}
			}
			return layout.Dimensions{Size: image.Pt(0, gtx.Constraints.Max.Y)}
		}))
	}

	return layout.Flex{Axis: axis}.Layout(gtx, flexChildren...)
}

type metricKey struct {
	pxPerDp float32
	pxPerSp float32
}

func metricOf(m unit.Metric) metricKey {
	return metricKey{pxPerDp: m.PxPerDp, pxPerSp: m.PxPerSp}
}

type layoutKey struct {
	min    image.Point
	max    image.Point
	metric metricKey
}

func layoutKeyOf(gtx layout.Context) layoutKey {
	return layoutKey{
		min:    gtx.Constraints.Min,
		max:    gtx.Constraints.Max,
		metric: metricOf(gtx.Metric),
	}
}

// layoutCache holds a recorded Gio op sequence and the key under which it was
// recorded. The buf field must outlive any op.CallOp that references it.
type layoutCache struct {
	layoutFn func(gtx layout.Context, shaper *text.Shaper) layout.Dimensions

	key  layoutKey
	buf  op.Ops
	call op.CallOp
	dims layout.Dimensions
}

func (c *layoutCache) layout(gtx layout.Context, shaper *text.Shaper) layout.Dimensions {
	key := layoutKeyOf(gtx)
	if key == c.key {
		c.call.Add(gtx.Ops)
		return c.dims
	}

	c.buf.Reset()

	recordGtx := gtx
	recordGtx.Ops = &c.buf
	macro := op.Record(&c.buf)

	dims := c.layoutFn(recordGtx, shaper)

	c.call = macro.Stop()
	c.key = key
	c.dims = dims

	c.call.Add(gtx.Ops)
	return dims
}

type sizeCache struct {
	sizeFn func(gtx layout.Context, shaper *text.Shaper) image.Point

	key      metricKey
	nodeSize image.Point
}

func (c *sizeCache) size(gtx layout.Context, shaper *text.Shaper) image.Point {
	key := metricOf(gtx.Metric)
	if key == c.key {
		return c.nodeSize
	}

	nodeSize := c.sizeFn(gtx, shaper)

	c.nodeSize = nodeSize
	c.key = key
	return nodeSize
}
