package path

import (
	"image/color"
	"testing"

	"gioui.org/f32"
	"gioui.org/op"
	"gioui.org/op/clip"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)


func TestParse_M(t *testing.T) {
	t.Parallel()

	segs, err := Parse("M 10 20")

	require.NoError(t, err)
	require.Len(t, segs, 1)
	assert.Equal(t, byte('M'), segs[0].Cmd)
	assert.InDeltaSlice(t, []float64{10, 20}, segs[0].Args, 0.001)
}

func TestParse_MRelative(t *testing.T) {
	t.Parallel()

	segs, err := Parse("M 5 5 m 3 4")

	require.NoError(t, err)
	require.Len(t, segs, 2)
	assert.Equal(t, byte('M'), segs[1].Cmd)
	assert.InDeltaSlice(t, []float64{8, 9}, segs[1].Args, 0.001)
}

func TestParse_MImplicitL(t *testing.T) {
	t.Parallel()

	// Extra coordinate pairs after M become implicit L commands.
	segs, err := Parse("M 0 0 10 10 20 20")

	require.NoError(t, err)
	require.Len(t, segs, 3)
	assert.Equal(t, byte('M'), segs[0].Cmd)
	assert.Equal(t, byte('L'), segs[1].Cmd)
	assert.Equal(t, byte('L'), segs[2].Cmd)
	assert.InDeltaSlice(t, []float64{20, 20}, segs[2].Args, 0.001)
}

func TestParse_L(t *testing.T) {
	t.Parallel()

	segs, err := Parse("M 0 0 L 10 20")

	require.NoError(t, err)
	require.Len(t, segs, 2)
	assert.Equal(t, byte('L'), segs[1].Cmd)
	assert.InDeltaSlice(t, []float64{10, 20}, segs[1].Args, 0.001)
}

func TestParse_LRelative(t *testing.T) {
	t.Parallel()

	segs, err := Parse("M 5 5 l 3 4")

	require.NoError(t, err)
	require.Len(t, segs, 2)
	assert.Equal(t, byte('L'), segs[1].Cmd)
	assert.InDeltaSlice(t, []float64{8, 9}, segs[1].Args, 0.001)
}

func TestParse_LRelativeImplicitContinuation(t *testing.T) {
	t.Parallel()

	// Implicit l continuations must each be relative to the previous endpoint,
	// not all relative to the original starting point.
	segs, err := Parse("M 0 0 l 1 1 2 2")

	require.NoError(t, err)
	require.Len(t, segs, 3)
	// Second segment: (0+1, 0+1) = (1, 1).
	assert.InDeltaSlice(t, []float64{1, 1}, segs[1].Args, 0.001)
	// Third segment: (1+2, 1+2) = (3, 3) — not (0+2, 0+2) = (2, 2).
	assert.InDeltaSlice(t, []float64{3, 3}, segs[2].Args, 0.001)
}

func TestParse_H(t *testing.T) {
	t.Parallel()

	segs, err := Parse("M 5 10 H 20")

	require.NoError(t, err)
	require.Len(t, segs, 2)
	assert.Equal(t, byte('L'), segs[1].Cmd)
	assert.InDeltaSlice(t, []float64{20, 10}, segs[1].Args, 0.001)
}

func TestParse_HRelative(t *testing.T) {
	t.Parallel()

	segs, err := Parse("M 5 10 h 3")

	require.NoError(t, err)
	assert.InDeltaSlice(t, []float64{8, 10}, segs[1].Args, 0.001)
}

func TestParse_HRelativeImplicitContinuation(t *testing.T) {
	t.Parallel()

	// Multiple values after h must each be relative to the previous position.
	segs, err := Parse("M 0 5 h 3 4")

	require.NoError(t, err)
	require.Len(t, segs, 3)
	assert.InDeltaSlice(t, []float64{3, 5}, segs[1].Args, 0.001)
	// Second step: 3+4=7, not 0+4=4.
	assert.InDeltaSlice(t, []float64{7, 5}, segs[2].Args, 0.001)
}

func TestParse_V(t *testing.T) {
	t.Parallel()

	segs, err := Parse("M 5 10 V 30")

	require.NoError(t, err)
	require.Len(t, segs, 2)
	assert.Equal(t, byte('L'), segs[1].Cmd)
	assert.InDeltaSlice(t, []float64{5, 30}, segs[1].Args, 0.001)
}

func TestParse_VRelative(t *testing.T) {
	t.Parallel()

	segs, err := Parse("M 5 10 v 4")

	require.NoError(t, err)
	assert.InDeltaSlice(t, []float64{5, 14}, segs[1].Args, 0.001)
}

func TestParse_VRelativeImplicitContinuation(t *testing.T) {
	t.Parallel()

	// Multiple values after v must each be relative to the previous position.
	segs, err := Parse("M 5 0 v 3 4")

	require.NoError(t, err)
	require.Len(t, segs, 3)
	assert.InDeltaSlice(t, []float64{5, 3}, segs[1].Args, 0.001)
	// Second step: 3+4=7, not 0+4=4.
	assert.InDeltaSlice(t, []float64{5, 7}, segs[2].Args, 0.001)
}

func TestParse_C(t *testing.T) {
	t.Parallel()

	segs, err := Parse("M 0 0 C 1 2 3 4 5 6")

	require.NoError(t, err)
	require.Len(t, segs, 2)
	assert.Equal(t, byte('C'), segs[1].Cmd)
	assert.InDeltaSlice(t, []float64{1, 2, 3, 4, 5, 6}, segs[1].Args, 0.001)
}

func TestParse_CRelative(t *testing.T) {
	t.Parallel()

	segs, err := Parse("M 10 10 c 1 2 3 4 5 6")

	require.NoError(t, err)
	assert.InDeltaSlice(t, []float64{11, 12, 13, 14, 15, 16}, segs[1].Args, 0.001)
}

func TestParse_CRelativeImplicitContinuation(t *testing.T) {
	t.Parallel()

	// Each group of 6 args must be relative to the endpoint of the previous cubic,
	// not all relative to the original starting point.
	// From (0,0): first cubic ends at (5,0). Second cubic's args should be
	// relative to (5,0), so end = (5+5, 0+0) = (10,0).
	segs, err := Parse("M 0 0 c 1 1 4 1 5 0 1 -1 4 -1 5 0")

	require.NoError(t, err)
	require.Len(t, segs, 3)
	// First cubic ends at (5, 0).
	assert.InDeltaSlice(t, []float64{1, 1, 4, 1, 5, 0}, segs[1].Args, 0.001)
	// Second cubic: relative to (5,0) → ctrl1=(6,-1), ctrl2=(9,-1), end=(10,0).
	assert.InDeltaSlice(t, []float64{6, -1, 9, -1, 10, 0}, segs[2].Args, 0.001)
}

func TestParse_SContinuationFromC(t *testing.T) {
	t.Parallel()

	// After a C, S reflects the second control point through the current pen.
	// C ends at (6,6) with ctrl2=(4,4); S reflection: (2*6-4, 2*6-4) = (8,8).
	segs, err := Parse("M 0 0 C 1 1 4 4 6 6 S 9 9 12 12")

	require.NoError(t, err)
	require.Len(t, segs, 3)
	assert.Equal(t, byte('C'), segs[2].Cmd)
	assert.InDeltaSlice(t, []float64{8, 8, 9, 9, 12, 12}, segs[2].Args, 0.001)
}

func TestParse_SNoPriorC(t *testing.T) {
	t.Parallel()

	// Without a preceding C/S, S uses the current pen as ctrl1.
	segs, err := Parse("M 5 5 S 8 8 10 10")

	require.NoError(t, err)
	require.Len(t, segs, 2)
	assert.Equal(t, byte('C'), segs[1].Cmd)
	// ctrl1 should be the pen position (5,5).
	assert.InDeltaSlice(t, []float64{5, 5, 8, 8, 10, 10}, segs[1].Args, 0.001)
}

func TestParse_Q(t *testing.T) {
	t.Parallel()

	segs, err := Parse("M 0 0 Q 5 10 10 0")

	require.NoError(t, err)
	require.Len(t, segs, 2)
	assert.Equal(t, byte('Q'), segs[1].Cmd)
	assert.InDeltaSlice(t, []float64{5, 10, 10, 0}, segs[1].Args, 0.001)
}

func TestParse_TContinuationFromQ(t *testing.T) {
	t.Parallel()

	// After Q with ctrl (5,10), T at pen (10,0) reflects: (2*10-5, 2*0-10) = (15,-10).
	segs, err := Parse("M 0 0 Q 5 10 10 0 T 20 0")

	require.NoError(t, err)
	require.Len(t, segs, 3)
	assert.Equal(t, byte('Q'), segs[2].Cmd)
	assert.InDeltaSlice(t, []float64{15, -10, 20, 0}, segs[2].Args, 0.001)
}

func TestParse_TContinuationFromT(t *testing.T) {
	t.Parallel()

	// Two T commands in sequence; each should reflect the control used by the previous.
	segs, err := Parse("M 0 0 Q 5 10 10 0 T 20 0 T 30 0")

	require.NoError(t, err)
	require.Len(t, segs, 4)
	assert.Equal(t, byte('Q'), segs[3].Cmd)
}

func TestParse_TNoPriorQ(t *testing.T) {
	t.Parallel()

	// Without a preceding Q/T, T uses the current pen as the control point.
	segs, err := Parse("M 5 5 T 10 10")

	require.NoError(t, err)
	require.Len(t, segs, 2)
	assert.Equal(t, byte('Q'), segs[1].Cmd)
	assert.InDeltaSlice(t, []float64{5, 5, 10, 10}, segs[1].Args, 0.001)
}

func TestParse_ASweepLargeArc(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		d        string
		wantSegs int
		wantErr  require.ErrorAssertionFunc
	}{
		{
			name:     "small arc no sweep",
			d:        "M 0 0 A 10 10 0 0 0 10 0",
			wantSegs: 2,
			wantErr:  require.NoError,
		},
		{
			name:     "large arc with sweep",
			d:        "M 0 0 A 10 10 0 1 1 10 0",
			wantSegs: 5,
			wantErr:  require.NoError,
		},
		{
			name:     "zero radius becomes line",
			d:        "M 0 0 A 0 0 0 0 0 10 0",
			wantSegs: 2,
			wantErr:  require.NoError,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			segs, err := Parse(test.d)

			test.wantErr(t, err)
			assert.Len(t, segs, test.wantSegs)
		})
	}
}

func TestParse_Z(t *testing.T) {
	t.Parallel()

	segs, err := Parse("M 0 0 L 10 10 Z")

	require.NoError(t, err)
	require.Len(t, segs, 3)
	assert.Equal(t, byte('Z'), segs[2].Cmd)
}

func TestParse_ZNoArgs(t *testing.T) {
	t.Parallel()

	// Z with no space and immediately followed by M is common in minified SVGs.
	segs, err := Parse("M 0 0 L 10 10 ZM 20 20")

	require.NoError(t, err)
	assert.Equal(t, byte('Z'), segs[2].Cmd)
	assert.Equal(t, byte('M'), segs[3].Cmd)
}

func TestParse_UnknownCommand(t *testing.T) {
	t.Parallel()

	_, err := Parse("M 0 0 X 1 1")

	assert.Error(t, err)
}

func TestParse_NegativeSignInNumberStream(t *testing.T) {
	t.Parallel()

	// "1-2" should parse as two numbers: 1 and -2.
	segs, err := Parse("M 0 0 L 1-2")

	require.NoError(t, err)
	assert.InDeltaSlice(t, []float64{1, -2}, segs[1].Args, 0.001)
}

func TestBuildClipPath_Identity(t *testing.T) {
	t.Parallel()

	segs, err := Parse("M 0 0 L 10 10 C 1 2 3 4 5 6 Q 7 8 9 10 Z")
	require.NoError(t, err)

	var ops op.Ops
	var p clip.Path
	p.Begin(&ops)

	assert.NotPanics(t, func() {
		BuildClipPath(&p, segs, f32.AffineId())
		p.End()
	})
}

func TestBuildClipPath_ScaleTranslate(t *testing.T) {
	t.Parallel()

	segs, err := Parse("M 0 0 L 10 10 Z")
	require.NoError(t, err)

	// Scale by 2, translate by (5,5).
	xform := f32.NewAffine2D(2, 0, 5, 0, 2, 5)

	var ops op.Ops
	var p clip.Path
	p.Begin(&ops)

	assert.NotPanics(t, func() {
		BuildClipPath(&p, segs, xform)
		p.End()
	})
}

func TestParse_ImplicitSepBySecondDecimalPoint(t *testing.T) {
	t.Parallel()

	// SVG path data uses a second '.' as an implicit separator:
	// "-.7.6" means -0.7 followed by 0.6, not a single invalid number.
	segs, err := Parse("M0 0c-.7.6-1.3.6-1.3 0z")

	require.NoError(t, err)
	require.Len(t, segs, 3) // M, C, Z
	assert.Equal(t, byte('C'), segs[1].Cmd)
	assert.InDelta(t, -0.7, segs[1].Args[0], 0.001)
	assert.InDelta(t, 0.6, segs[1].Args[1], 0.001)
	assert.InDelta(t, -1.3, segs[1].Args[2], 0.001)
	assert.InDelta(t, 0.6, segs[1].Args[3], 0.001)
}

func TestStrokePath(t *testing.T) {
	t.Parallel()

	segs, err := Parse("M 0 0 L 10 10 Z")
	require.NoError(t, err)

	var ops op.Ops

	assert.NotPanics(t, func() {
		StrokePath(&ops, segs, f32.AffineId(), 2, color.NRGBA{R: 255, A: 255})
	})
}
