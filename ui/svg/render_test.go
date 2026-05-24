package svg

import (
	"image"
	"testing"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"github.com/glasslabs/looking-glass/ui/svg/element"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestGtx(w, h int) layout.Context {
	return layout.Context{
		Ops:         new(op.Ops),
		Constraints: layout.Exact(image.Pt(w, h)),
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
	}
}

func TestRender_PathWithEvenOddFillRule(t *testing.T) {
	t.Parallel()

	doc, err := element.Parse(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100">
		<polygon points="10,10 90,10 90,90 10,90" fill="#ffffff" fill-rule="evenodd"/>
	</svg>`)
	require.NoError(t, err)

	assert.NotPanics(t, func() {
		Render(newTestGtx(100, 100), doc, image.Pt(100, 100), nil)
	})
}

func TestRender_ScalesStrokeWidth(t *testing.T) {
	t.Parallel()

	doc, err := element.Parse(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 50 50">
		<path d="M 10 10 L 40 40" fill="none" stroke="#ff0000" stroke-width="1"/>
	</svg>`)
	require.NoError(t, err)

	assert.NotPanics(t, func() {
		Render(newTestGtx(100, 100), doc, image.Pt(100, 100), nil)
	})
}

func TestRender_ReturnsZeroForNilDoc(t *testing.T) {
	t.Parallel()

	dims := Render(newTestGtx(300, 300), nil, image.Pt(300, 300), nil)

	assert.Equal(t, 0, dims.Size.X)
}

func TestRender_ReturnsZeroForZeroFitSize(t *testing.T) {
	t.Parallel()

	doc, err := element.Parse(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"/>`)
	require.NoError(t, err)

	dims := Render(newTestGtx(0, 0), doc, image.Pt(0, 0), nil)

	assert.Equal(t, 0, dims.Size.X)
}

func TestRender_RendersCircle(t *testing.T) {
	t.Parallel()

	doc, err := element.Parse(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100">
		<circle cx="50" cy="50" r="40" fill="#333333"/>
	</svg>`)
	require.NoError(t, err)

	assert.NotPanics(t, func() {
		Render(newTestGtx(300, 300), doc, image.Pt(300, 300), nil)
	})
}

func TestRender_ReturnsFitSizeDimensions(t *testing.T) {
	t.Parallel()

	doc, err := element.Parse(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100">
		<rect x="0" y="0" width="100" height="100" fill="#ffffff"/>
	</svg>`)
	require.NoError(t, err)

	dims := Render(newTestGtx(200, 150), doc, image.Pt(200, 150), nil)

	assert.Equal(t, 200, dims.Size.X)
	assert.Equal(t, 150, dims.Size.Y)
}

func TestRender_RendersPathWithFillAndStroke(t *testing.T) {
	t.Parallel()

	doc, err := element.Parse(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100">
		<path d="M 10 10 L 90 90 Z" fill="#ff0000" stroke="#0000ff" stroke-width="2"/>
	</svg>`)
	require.NoError(t, err)

	assert.NotPanics(t, func() {
		Render(newTestGtx(100, 100), doc, image.Pt(100, 100), nil)
	})
}

func TestRender_RendersGroupWithTransform(t *testing.T) {
	t.Parallel()

	doc, err := element.Parse(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100">
		<g transform="translate(10,10)">
			<rect x="0" y="0" width="50" height="50" fill="#ffffff"/>
		</g>
	</svg>`)
	require.NoError(t, err)

	assert.NotPanics(t, func() {
		Render(newTestGtx(100, 100), doc, image.Pt(100, 100), nil)
	})
}

func TestRender_RendersEmptyDoc(t *testing.T) {
	t.Parallel()

	doc, err := element.Parse(`<svg xmlns="http://www.w3.org/2000/svg"/>`)
	require.NoError(t, err)

	dims := Render(newTestGtx(100, 100), doc, image.Pt(100, 100), nil)

	assert.Equal(t, 100, dims.Size.X)
}

func TestRenderer_RenderReusesCachedOps(t *testing.T) {
	t.Parallel()

	doc, err := element.Parse(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100">
		<rect x="0" y="0" width="100" height="100" fill="#ffffff"/>
		<circle cx="50" cy="50" r="30" fill="#ff0000"/>
	</svg>`)
	require.NoError(t, err)

	r := NewRenderer()
	gtx := newTestGtx(100, 100)
	fitSz := image.Pt(100, 100)

	// Warm the cache.
	dims1 := r.Render(gtx, doc, fitSz, nil)

	gtx2 := newTestGtx(100, 100)
	dims2 := r.Render(gtx2, doc, fitSz, nil)

	assert.Equal(t, dims1, dims2)
}

func TestRenderer_RenderInvalidatesCacheOnChange(t *testing.T) {
	t.Parallel()

	doc1, err := element.Parse(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100">
		<circle cx="50" cy="50" r="30" fill="#ff0000"/>
	</svg>`)
	require.NoError(t, err)

	doc2, err := element.Parse(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100">
		<circle cx="50" cy="50" r="30" fill="#00ff00"/>
	</svg>`)
	require.NoError(t, err)

	r := NewRenderer()
	fitSz := image.Pt(100, 100)

	r.Render(newTestGtx(100, 100), doc1, fitSz, nil)

	assert.NotPanics(t, func() {
		dims := r.Render(newTestGtx(100, 100), doc2, fitSz, nil)
		assert.Equal(t, 100, dims.Size.X)
	})
}

func TestRender_HandlesViewboxOffset(t *testing.T) {
	t.Parallel()

	doc, err := element.Parse(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="10 10 80 80">
		<circle cx="50" cy="50" r="30" fill="#aaaaaa"/>
	</svg>`)
	require.NoError(t, err)

	assert.NotPanics(t, func() {
		Render(newTestGtx(200, 200), doc, image.Pt(200, 200), nil)
	})
}

