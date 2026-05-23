package style

import (
	"image/color"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefault(t *testing.T) {
	t.Parallel()

	s := Default()

	assert.Equal(t, color.NRGBA{}, s.Fill)
	assert.False(t, s.FillNone)
	assert.Equal(t, float32(1), s.FillOpacity)
	assert.True(t, s.StrokeNone)
	assert.Equal(t, float32(1), s.StrokeWidth)
	assert.Equal(t, float32(1), s.Opacity)
	assert.Equal(t, float32(16), s.FontSize)
	assert.Equal(t, "normal", s.FontWeight)
	assert.Equal(t, "start", s.TextAnchor)
	assert.Equal(t, "inline", s.Display)
}

func TestParse_FillHex(t *testing.T) {
	t.Parallel()

	s := Parse(map[string]string{"fill": "#ff0000"})

	assert.Equal(t, color.NRGBA{R: 0xff, A: 0xff}, s.Fill)
	assert.False(t, s.FillNone)
}

func TestParse_FillNone(t *testing.T) {
	t.Parallel()

	s := Parse(map[string]string{"fill": "none"})

	assert.True(t, s.FillNone)
}

func TestParse_Stroke(t *testing.T) {
	t.Parallel()

	s := Parse(map[string]string{"stroke": "#0000ff", "stroke-width": "3"})

	assert.Equal(t, color.NRGBA{B: 0xff, A: 0xff}, s.Stroke)
	assert.False(t, s.StrokeNone)
	assert.Equal(t, float32(3), s.StrokeWidth)
}

func TestParse_Opacity(t *testing.T) {
	t.Parallel()

	s := Parse(map[string]string{"opacity": "0.5", "fill-opacity": "0.8"})

	assert.InDelta(t, 0.5, s.Opacity, 0.001)
	assert.InDelta(t, 0.8, s.FillOpacity, 0.001)
}

func TestParse_StyleAttrOverridesPresentation(t *testing.T) {
	t.Parallel()

	// Presentation attribute says red; style attribute says blue.
	// style wins per SVG specificity rules.
	s := Parse(map[string]string{
		"fill":  "#ff0000",
		"style": "fill:#0000ff",
	})

	assert.Equal(t, color.NRGBA{B: 0xff, A: 0xff}, s.Fill)
}

func TestParse_TextAttributes(t *testing.T) {
	t.Parallel()

	s := Parse(map[string]string{
		"font-size":   "24px",
		"font-weight": "bold",
		"text-anchor": "middle",
	})

	assert.Equal(t, float32(24), s.FontSize)
	assert.Equal(t, "bold", s.FontWeight)
	assert.Equal(t, "middle", s.TextAnchor)
}

func TestParse_DisplayNone(t *testing.T) {
	t.Parallel()

	s := Parse(map[string]string{"display": "none"})

	assert.Equal(t, "none", s.Display)
}

func TestInherit_UnsetFillInherits(t *testing.T) {
	t.Parallel()

	parent := Style{Fill: color.NRGBA{R: 0xff, A: 0xff}, FillOpacity: 0.8}
	child := Parse(map[string]string{})

	result := Inherit(parent, child, map[string]string{})

	assert.Equal(t, parent.Fill, result.Fill)
	assert.Equal(t, float32(0.8), result.FillOpacity)
}

func TestInherit_SetFillDoesNotInherit(t *testing.T) {
	t.Parallel()

	parent := Style{Fill: color.NRGBA{R: 0xff, A: 0xff}}

	result := Inherit(parent, Parse(map[string]string{"fill": "#00ff00"}), map[string]string{"fill": "#00ff00"})

	assert.Equal(t, color.NRGBA{G: 0xff, A: 0xff}, result.Fill)
}

func TestInherit_DisplayNotInherited(t *testing.T) {
	t.Parallel()

	parent := Style{Display: "none"}
	result := Inherit(parent, Parse(map[string]string{}), map[string]string{})

	// display is not inherited; should be the SVG default.
	assert.Equal(t, "inline", result.Display)
}

func TestInherit_DisplayFromChildWhenSet(t *testing.T) {
	t.Parallel()

	parent := Style{Display: "inline"}
	result := Inherit(parent, Parse(map[string]string{"display": "none"}), map[string]string{"display": "none"})

	assert.Equal(t, "none", result.Display)
}

func TestParse_FillRule(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		attrs    map[string]string
		wantRule string
	}{
		{
			name:     "evenodd",
			attrs:    map[string]string{"fill-rule": "evenodd"},
			wantRule: "evenodd",
		},
		{
			name:     "nonzero",
			attrs:    map[string]string{"fill-rule": "nonzero"},
			wantRule: "nonzero",
		},
		{
			name:     "unknown value ignored",
			attrs:    map[string]string{"fill-rule": "inherit"},
			wantRule: "",
		},
		{
			name:     "absent",
			attrs:    map[string]string{},
			wantRule: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := Parse(test.attrs)

			assert.Equal(t, test.wantRule, got.FillRule)
		})
	}
}

func TestInherit_FillRuleInherited(t *testing.T) {
	t.Parallel()

	parent := Style{FillRule: "evenodd"}
	result := Inherit(parent, Parse(map[string]string{}), map[string]string{})

	assert.Equal(t, "evenodd", result.FillRule)
}

func TestInherit_FillRuleChildOverrides(t *testing.T) {
	t.Parallel()

	parent := Style{FillRule: "evenodd"}
	result := Inherit(parent, Parse(map[string]string{"fill-rule": "nonzero"}), map[string]string{"fill-rule": "nonzero"})

	assert.Equal(t, "nonzero", result.FillRule)
}

func TestParseColor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  color.NRGBA
		ok    bool
	}{
		{
			name:  "#rrggbb",
			input: "#ff8800",
			want:  color.NRGBA{R: 0xff, G: 0x88, A: 0xff},
			ok:    true,
		},
		{
			name:  "#rgb short",
			input: "#f80",
			want:  color.NRGBA{R: 0xff, G: 0x88, A: 0xff},
			ok:    true,
		},
		{
			name:  "#rrggbbaa",
			input: "#ff880080",
			want:  color.NRGBA{R: 0xff, G: 0x88, A: 0x80},
			ok:    true,
		},
		{
			name:  "#rgba short",
			input: "#f808",
			want:  color.NRGBA{R: 0xff, G: 0x88, A: 0x88},
			ok:    true,
		},
		{
			name:  "rgb()",
			input: "rgb(255, 128, 0)",
			want:  color.NRGBA{R: 0xff, G: 0x80, A: 0xff},
			ok:    true,
		},
		{
			name:  "rgba()",
			input: "rgba(255, 128, 0, 0.5)",
			want:  color.NRGBA{R: 0xff, G: 0x80, A: 0x7f},
			ok:    true,
		},
		{
			name:  "named black",
			input: "black",
			want:  color.NRGBA{A: 0xff},
			ok:    true,
		},
		{
			name:  "named white",
			input: "white",
			want:  color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff},
			ok:    true,
		},
		{
			name:  "named red",
			input: "red",
			want:  color.NRGBA{R: 0xff, A: 0xff},
			ok:    true,
		},
		{
			name:  "named gray",
			input: "gray",
			want:  color.NRGBA{R: 0x80, G: 0x80, B: 0x80, A: 0xff},
			ok:    true,
		},
		{
			name:  "named grey alias",
			input: "grey",
			want:  color.NRGBA{R: 0x80, G: 0x80, B: 0x80, A: 0xff},
			ok:    true,
		},
		{
			name:  "named navy",
			input: "navy",
			want:  color.NRGBA{B: 0x80, A: 0xff},
			ok:    true,
		},
		{
			name:  "none returns false",
			input: "none",
			ok:    false,
		},
		{
			name:  "empty returns false",
			input: "",
			ok:    false,
		},
		{
			name:  "unknown name returns false",
			input: "chartreuse",
			ok:    false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, ok := ParseColor(test.input)

			assert.Equal(t, test.ok, ok)
			if test.ok {
				assert.Equal(t, test.want, got)
			}
		})
	}
}
