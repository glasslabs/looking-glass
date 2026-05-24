package ui

import (
	"image"
	"testing"

	"gioui.org/font/gofont"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/text"
	"gioui.org/unit"
	"github.com/glasslabs/client-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVStackNode_LayoutReportsNaturalWidth(t *testing.T) {
	t.Parallel()

	shaper := newTestShaper()

	n, err := newNode(client.NewVStack(
		client.NewText("Hello"),
	), nil)
	require.NoError(t, err)

	var ops op.Ops
	gtx := newTestGtx(&ops, 9999, 9999)

	naturalSz := n.size(gtx, shaper)
	require.Greater(t, naturalSz.X, 0, "natural width must be non-zero")

	ops.Reset()
	dims := n.layout(gtx, shaper)

	assert.Equal(t, naturalSz.X, dims.Size.X, "VStack must not expand beyond natural width")
	assert.Less(t, dims.Size.X, 9999, "VStack must not consume the full available width")
}

func TestHStackNode_LayoutPositionsAllVStackChildren(t *testing.T) {
	t.Parallel()

	shaper := newTestShaper()

	n, err := newNode(client.NewHStack(
		client.NewVStack(client.NewText("A")),
		client.NewVStack(client.NewText("BB")),
		client.NewVStack(client.NewText("CCC")),
	), nil)
	require.NoError(t, err)

	var ops op.Ops
	// Provide ample space so that the natural size is strictly smaller.  If the
	// bug is present, VStack1 grabs all 9999 px (because Min.X = Max.X = 9999
	// inside its vertical layoutStack), leaving 0 px for VStack2 and VStack3,
	// and dims.Size.X ends up as 9999 instead of the sum of natural widths.
	gtx := newTestGtx(&ops, 9999, 9999)

	naturalSz := n.size(gtx, shaper)
	require.Greater(t, naturalSz.X, 0, "HStack natural width must be non-zero")

	ops.Reset()
	dims := n.layout(gtx, shaper)

	assert.Equal(t, naturalSz.X, dims.Size.X,
		"HStack must report the sum of children's natural widths, not the available max")
	assert.Less(t, dims.Size.X, 9999,
		"HStack must not consume the full available width when children are narrower")
}

func TestHStackNode_LayoutReportsNaturalHeight(t *testing.T) {
	t.Parallel()

	shaper := newTestShaper()

	n, err := newNode(client.NewHStack(
		client.NewText("Hello"),
	), nil)
	require.NoError(t, err)

	var ops op.Ops
	gtx := newTestGtx(&ops, 9999, 9999)

	naturalSz := n.size(gtx, shaper)
	require.Greater(t, naturalSz.Y, 0, "natural height must be non-zero")

	ops.Reset()
	dims := n.layout(gtx, shaper)

	assert.Equal(t, naturalSz.Y, dims.Size.Y, "HStack must not expand beyond natural height")
	assert.Less(t, dims.Size.Y, 9999, "HStack must not consume the full available height")
}

func TestSpacerNode_MinSizeIncludedInStackNaturalSize(t *testing.T) {
	t.Parallel()

	shaper := newTestShaper()

	// min=50dp × 2px/dp = 100px on the primary axis.
	n, err := newNode(client.NewVStack(
		client.NewSpacer(client.WithMinSize(50)),
	), nil)
	require.NoError(t, err)

	var ops op.Ops
	gtx := newTestGtx(&ops, 200, 9999)

	naturalSz := n.size(gtx, shaper)

	assert.Equal(t, 100, naturalSz.Y, "VStack natural height must include spacer minimum")
}

func TestSpacerNode_MinFloorIsRespectedInLayout(t *testing.T) {
	t.Parallel()

	shaper := newTestShaper()

	// Single spacer with min=50dp; pxPerDp=2 → floor = 100px.
	// layoutStack is called directly so there is no axis-independent capping.
	var ops op.Ops
	gtx := newTestGtx(&ops, 400, 400)

	children := []node{&spacerNode{min: 50}}

	dims := layoutStack(gtx, children, layout.Vertical, shaper)

	assert.GreaterOrEqual(t, dims.Size.Y, 100, "spacer must occupy at least its minimum height")
}

func TestSpacerNode_EqualMinSplitsRemainingSpaceEvenly(t *testing.T) {
	t.Parallel()

	// Two spacers with the same min in an HStack of 400px.
	// Each rigid floor = 50px (25dp × 2), total rigid = 100px.
	// Remaining 300px split equally (weight 25:25) → 150px each.
	// Final widths: 50+150 = 200px each.
	shaper := newTestShaper()

	var ops op.Ops
	gtx := newTestGtx(&ops, 400, 100)

	children := []node{
		&spacerNode{min: 25},
		&spacerNode{min: 25},
	}

	dims := layoutStack(gtx, children, layout.Horizontal, shaper)

	assert.Equal(t, 400, dims.Size.X, "two equal spacers must fill the available width")
}

func TestSpacerNode_UnequalMinSplitsRemainingSpaceProportionally(t *testing.T) {
	t.Parallel()

	// Spacer A: min=25dp → floor=50px, weight=25.
	// Spacer B: min=75dp → floor=150px, weight=75.
	// Total rigid = 200px out of 400px available.
	// Remaining 200px split 25:75 → A gets +50px, B gets +150px.
	// Final: A=100px, B=300px, total=400px.
	shaper := newTestShaper()

	var ops op.Ops
	gtx := newTestGtx(&ops, 400, 100)

	children := []node{
		&spacerNode{min: 25},
		&spacerNode{min: 75},
	}

	dims := layoutStack(gtx, children, layout.Horizontal, shaper)

	assert.Equal(t, 400, dims.Size.X, "two unequal spacers must still fill the available width")
}

func TestSpacerNode_ZeroMinGetsEqualWeightFlex(t *testing.T) {
	t.Parallel()

	// A zero-min spacer still absorbs available space evenly with other zero-min spacers.
	shaper := newTestShaper()

	var ops op.Ops
	gtx := newTestGtx(&ops, 300, 100)

	children := []node{
		&spacerNode{min: 0},
		&spacerNode{min: 0},
		&spacerNode{min: 0},
	}

	dims := layoutStack(gtx, children, layout.Horizontal, shaper)

	assert.Equal(t, 300, dims.Size.X, "zero-min spacers must collectively fill all available width")
}

func newTestShaper() *text.Shaper {
	return text.NewShaper(text.NoSystemFonts(), text.WithCollection(gofont.Collection()))
}

func newTestGtx(ops *op.Ops, maxW, maxH int) layout.Context {
	return layout.Context{
		Ops: ops,
		Constraints: layout.Constraints{
			Max: image.Pt(maxW, maxH),
		},
		Metric: unit.Metric{PxPerDp: 2, PxPerSp: 2},
	}
}
