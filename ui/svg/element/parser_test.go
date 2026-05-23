package element

import (
	"image/color"
	"testing"

	"gioui.org/f32"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const svgWrap = `<svg xmlns="http://www.w3.org/2000/svg" width="100" height="100" viewBox="0 0 100 100">`

func svgDoc(inner string) string {
	return svgWrap + inner + `</svg>`
}

func TestParse_SinglePath(t *testing.T) {
	t.Parallel()

	doc, err := Parse(svgDoc(`<path d="M 0 0 L 10 10" fill="#ff0000"/>`))

	require.NoError(t, err)
	require.Len(t, doc.Children, 1)

	cmd, ok := doc.Children[0].(*PathCmd)
	require.True(t, ok)
	assert.Len(t, cmd.Segments, 2)
	assert.Equal(t, color.NRGBA{R: 0xff, A: 0xff}, cmd.Style.Fill)
}

func TestParse_DimensionsAndViewbox(t *testing.T) {
	t.Parallel()

	doc, err := Parse(svgDoc(""))

	require.NoError(t, err)
	assert.InDelta(t, 100, doc.Width, 0.001)
	assert.InDelta(t, 100, doc.Height, 0.001)
	assert.Equal(t, [4]float64{0, 0, 100, 100}, doc.ViewBox)
}

func TestParse_GWithTransform(t *testing.T) {
	t.Parallel()

	doc, err := Parse(svgDoc(`<g transform="translate(10,20)"><rect x="0" y="0" width="50" height="50"/></g>`))

	require.NoError(t, err)
	require.Len(t, doc.Children, 1)

	g, ok := doc.Children[0].(*GroupCmd)
	require.True(t, ok)
	require.Len(t, g.Children, 1)

	// The group's transform should shift (0,0) to (10,20).
	pt := g.Transform.Transform(f32.Pt(0, 0))
	assert.InDelta(t, 10, pt.X, 0.001)
	assert.InDelta(t, 20, pt.Y, 0.001)
}

func TestParse_FillInherit(t *testing.T) {
	t.Parallel()

	// fill is set on outer <g>; inner <path> should inherit it.
	doc, err := Parse(svgDoc(`<g fill="#00ff00"><path d="M 0 0 L 5 5"/></g>`))

	require.NoError(t, err)
	g := doc.Children[0].(*GroupCmd)
	cmd := g.Children[0].(*PathCmd)
	assert.Equal(t, color.NRGBA{G: 0xff, A: 0xff}, cmd.Style.Fill)
}

func TestParse_FillInheritTwoLevels(t *testing.T) {
	t.Parallel()

	doc, err := Parse(svgDoc(`<g fill="#0000ff"><g><path d="M 0 0 L 5 5"/></g></g>`))

	require.NoError(t, err)
	outer := doc.Children[0].(*GroupCmd)
	inner := outer.Children[0].(*GroupCmd)
	cmd := inner.Children[0].(*PathCmd)
	assert.Equal(t, color.NRGBA{B: 0xff, A: 0xff}, cmd.Style.Fill)
}

func TestParse_Circle(t *testing.T) {
	t.Parallel()

	doc, err := Parse(svgDoc(`<circle cx="50" cy="50" r="30" fill="#ffffff"/>`))

	require.NoError(t, err)
	require.Len(t, doc.Children, 1)

	cmd, ok := doc.Children[0].(*CircleCmd)
	require.True(t, ok)
	assert.InDelta(t, 50, cmd.CX, 0.001)
	assert.InDelta(t, 50, cmd.CY, 0.001)
	assert.InDelta(t, 30, cmd.RX, 0.001)
	assert.InDelta(t, 30, cmd.RY, 0.001)
}

func TestParse_Ellipse(t *testing.T) {
	t.Parallel()

	doc, err := Parse(svgDoc(`<ellipse cx="50" cy="40" rx="30" ry="20"/>`))

	require.NoError(t, err)
	cmd, ok := doc.Children[0].(*CircleCmd)
	require.True(t, ok)
	assert.InDelta(t, 30, cmd.RX, 0.001)
	assert.InDelta(t, 20, cmd.RY, 0.001)
}

func TestParse_Rect(t *testing.T) {
	t.Parallel()

	doc, err := Parse(svgDoc(`<rect x="5" y="10" width="80" height="60" rx="4"/>`))

	require.NoError(t, err)
	cmd, ok := doc.Children[0].(*RectCmd)
	require.True(t, ok)
	assert.InDelta(t, 5, cmd.X, 0.001)
	assert.InDelta(t, 10, cmd.Y, 0.001)
	assert.InDelta(t, 80, cmd.W, 0.001)
	assert.InDelta(t, 60, cmd.H, 0.001)
	assert.InDelta(t, 4, cmd.RX, 0.001)
}

func TestParse_Line(t *testing.T) {
	t.Parallel()

	doc, err := Parse(svgDoc(`<line x1="0" y1="0" x2="10" y2="10" stroke="#000000"/>`))

	require.NoError(t, err)
	cmd, ok := doc.Children[0].(*PathCmd)
	require.True(t, ok)
	require.Len(t, cmd.Segments, 2)
	assert.Equal(t, byte('M'), cmd.Segments[0].Cmd)
	assert.Equal(t, byte('L'), cmd.Segments[1].Cmd)
}

func TestParse_Polyline(t *testing.T) {
	t.Parallel()

	doc, err := Parse(svgDoc(`<polyline points="0,0 10,10 20,0"/>`))

	require.NoError(t, err)
	cmd, ok := doc.Children[0].(*PathCmd)
	require.True(t, ok)
	require.Len(t, cmd.Segments, 3)
	assert.Equal(t, byte('M'), cmd.Segments[0].Cmd)
	assert.Equal(t, byte('L'), cmd.Segments[1].Cmd)
	assert.Equal(t, byte('L'), cmd.Segments[2].Cmd)
}

func TestParse_PolygonHasZ(t *testing.T) {
	t.Parallel()

	doc, err := Parse(svgDoc(`<polygon points="0,0 10,10 20,0"/>`))

	require.NoError(t, err)
	cmd := doc.Children[0].(*PathCmd)
	assert.Equal(t, byte('Z'), cmd.Segments[len(cmd.Segments)-1].Cmd)
}

func TestParse_DefsAndUse(t *testing.T) {
	t.Parallel()

	svg := svgDoc(`
        <defs>
            <path id="arrow" d="M 0 0 L 10 10"/>
        </defs>
        <use href="#arrow"/>
    `)

	doc, err := Parse(svg)

	require.NoError(t, err)
	require.Len(t, doc.Children, 1)

	g, ok := doc.Children[0].(*GroupCmd)
	require.True(t, ok)
	require.Len(t, g.Children, 1)
	_, ok = g.Children[0].(*PathCmd)
	assert.True(t, ok)
}

func TestParse_UseXlinkHref(t *testing.T) {
	t.Parallel()

	svg := svgDoc(`
        <defs><circle id="dot" cx="5" cy="5" r="3"/></defs>
        <use xlink:href="#dot"/>
    `)

	doc, err := Parse(svg)

	require.NoError(t, err)
	require.Len(t, doc.Children, 1)
}

func TestParse_Text(t *testing.T) {
	t.Parallel()

	doc, err := Parse(svgDoc(`<text x="10" y="20" font-size="14">hello</text>`))

	require.NoError(t, err)
	cmd, ok := doc.Children[0].(*TextCmd)
	require.True(t, ok)
	assert.Equal(t, "hello", cmd.Content)
	assert.InDelta(t, 10, cmd.X, 0.001)
	assert.InDelta(t, 20, cmd.Y, 0.001)
	assert.InDelta(t, 14, cmd.Style.FontSize, 0.001)
}

func TestParse_DisplayNoneExcluded(t *testing.T) {
	t.Parallel()

	doc, err := Parse(svgDoc(`<rect x="0" y="0" width="10" height="10" display="none"/>`))

	require.NoError(t, err)
	assert.Empty(t, doc.Children)
}

func TestParse_MalformedPathSkipped(t *testing.T) {
	t.Parallel()

	// A bad path should be skipped; the subsequent rect should still parse.
	doc, err := Parse(svgDoc(`
        <path d="M 0 0 X 1 1"/>
        <rect x="0" y="0" width="10" height="10"/>
    `))

	require.NoError(t, err)
	require.Len(t, doc.Children, 1)
	_, ok := doc.Children[0].(*RectCmd)
	assert.True(t, ok)
}

func TestParse_EmptyDoc(t *testing.T) {
	t.Parallel()

	doc, err := Parse(`<svg xmlns="http://www.w3.org/2000/svg"/>`)

	require.NoError(t, err)
	assert.Empty(t, doc.Children)
}

