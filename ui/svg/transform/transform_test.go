package transform

import (
	"math"
	"testing"

	"gioui.org/f32"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse_EmptyIsIdentity(t *testing.T) {
	t.Parallel()

	xform, err := Parse("")

	require.NoError(t, err)
	assert.Equal(t, f32.AffineId(), xform)
}

func TestParse_TranslateOneArg(t *testing.T) {
	t.Parallel()

	xform, err := Parse("translate(10)")

	require.NoError(t, err)
	got := xform.Transform(f32.Pt(0, 0))
	assert.InDelta(t, 10, got.X, 0.001)
	assert.InDelta(t, 0, got.Y, 0.001)
}

func TestParse_TranslateTwoArgs(t *testing.T) {
	t.Parallel()

	xform, err := Parse("translate(10, 20)")

	require.NoError(t, err)
	got := xform.Transform(f32.Pt(0, 0))
	assert.InDelta(t, 10, got.X, 0.001)
	assert.InDelta(t, 20, got.Y, 0.001)
}

func TestParse_ScaleUniform(t *testing.T) {
	t.Parallel()

	xform, err := Parse("scale(2)")

	require.NoError(t, err)
	got := xform.Transform(f32.Pt(3, 5))
	assert.InDelta(t, 6, got.X, 0.001)
	assert.InDelta(t, 10, got.Y, 0.001)
}

func TestParse_ScaleNonUniform(t *testing.T) {
	t.Parallel()

	xform, err := Parse("scale(2, 3)")

	require.NoError(t, err)
	got := xform.Transform(f32.Pt(4, 5))
	assert.InDelta(t, 8, got.X, 0.001)
	assert.InDelta(t, 15, got.Y, 0.001)
}

func TestParse_RotateAboutOrigin(t *testing.T) {
	t.Parallel()

	// 90° rotation: (1,0) → (0,1).
	xform, err := Parse("rotate(90)")

	require.NoError(t, err)
	got := xform.Transform(f32.Pt(1, 0))
	assert.InDelta(t, 0, got.X, 0.001)
	assert.InDelta(t, 1, got.Y, 0.001)
}

func TestParse_RotateWithCentre(t *testing.T) {
	t.Parallel()

	// 180° rotation around (5,5): (5,0) → (5,10).
	xform, err := Parse("rotate(180, 5, 5)")

	require.NoError(t, err)
	got := xform.Transform(f32.Pt(5, 0))
	assert.InDelta(t, 5, got.X, 0.001)
	assert.InDelta(t, 10, got.Y, 0.001)
}

func TestParse_SkewX(t *testing.T) {
	t.Parallel()

	xform, err := Parse("skewX(45)")

	require.NoError(t, err)
	// skewX(45): x' = x + y*tan(45°), y' = y. Point (0,1) → (1,1).
	got := xform.Transform(f32.Pt(0, 1))
	assert.InDelta(t, math.Tan(math.Pi/4), float64(got.X), 0.001)
	assert.InDelta(t, 1, got.Y, 0.001)
}

func TestParse_SkewY(t *testing.T) {
	t.Parallel()

	xform, err := Parse("skewY(45)")

	require.NoError(t, err)
	// skewY(45): y' = y + x*tan(45°), x' = x. Point (1,0) → (1,1).
	got := xform.Transform(f32.Pt(1, 0))
	assert.InDelta(t, 1, got.X, 0.001)
	assert.InDelta(t, math.Tan(math.Pi/4), float64(got.Y), 0.001)
}

func TestParse_Matrix(t *testing.T) {
	t.Parallel()

	// Identity matrix: matrix(1,0,0,1,0,0).
	xform, err := Parse("matrix(1,0,0,1,0,0)")

	require.NoError(t, err)
	got := xform.Transform(f32.Pt(3, 7))
	assert.InDelta(t, 3, got.X, 0.001)
	assert.InDelta(t, 7, got.Y, 0.001)
}

func TestParse_MatrixTranslate(t *testing.T) {
	t.Parallel()

	// matrix(1,0,0,1,10,20) is a pure translation by (10,20).
	xform, err := Parse("matrix(1,0,0,1,10,20)")

	require.NoError(t, err)
	got := xform.Transform(f32.Pt(0, 0))
	assert.InDelta(t, 10, got.X, 0.001)
	assert.InDelta(t, 20, got.Y, 0.001)
}

func TestParse_ChainedTranslateScale(t *testing.T) {
	t.Parallel()

	// translate(10,0) scale(2): scale first, then translate.
	xform, err := Parse("translate(10, 0) scale(2)")

	require.NoError(t, err)
	// Point (3,0): scale → (6,0), translate → (16,0).
	got := xform.Transform(f32.Pt(3, 0))
	assert.InDelta(t, 16, got.X, 0.001)
	assert.InDelta(t, 0, got.Y, 0.001)
}

func TestParse_UnknownFunctionErrors(t *testing.T) {
	t.Parallel()

	_, err := Parse("zap(1,2)")

	assert.Error(t, err)
}

func TestParse_MissingParenErrors(t *testing.T) {
	t.Parallel()

	_, err := Parse("translate 10 20")

	assert.Error(t, err)
}

