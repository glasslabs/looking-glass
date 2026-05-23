package svg

import (
	"image"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFitSize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		vbW  float64
		vbH  float64
		max  image.Point
		want image.Point
	}{
		{
			name: "square viewbox fits square constraint exactly",
			vbW:  100, vbH: 100,
			max:  image.Pt(300, 300),
			want: image.Pt(300, 300),
		},
		{
			name: "wide viewbox letterboxed in square constraint",
			vbW:  200, vbH: 100,
			max:  image.Pt(300, 300),
			want: image.Pt(300, 150),
		},
		{
			name: "tall viewbox pillarboxed in square constraint",
			vbW:  100, vbH: 200,
			max:  image.Pt(300, 300),
			want: image.Pt(150, 300),
		},
		{
			name: "zero vbW falls back to max",
			vbW:  0, vbH: 100,
			max:  image.Pt(300, 300),
			want: image.Pt(300, 300),
		},
		{
			name: "zero max falls back to max",
			vbW:  100, vbH: 100,
			max:  image.Pt(0, 0),
			want: image.Pt(0, 0),
		},
		{
			name: "non-square max with matching aspect ratio",
			vbW:  100, vbH: 50,
			max:  image.Pt(400, 200),
			want: image.Pt(400, 200),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := FitSize(test.vbW, test.vbH, test.max)

			assert.Equal(t, test.want, got)
		})
	}
}

func TestParseLength(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  float64
	}{
		{name: "plain number", input: "24", want: 24},
		{name: "px suffix", input: "24px", want: 24},
		{name: "pt suffix", input: "12pt", want: 16},
		{name: "empty string", input: "", want: 0},
		{name: "relative em skipped", input: "1.5em", want: 0},
		{name: "relative percent skipped", input: "50%", want: 0},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := parseLength(test.input)

			assert.InDelta(t, test.want, got, 0.01)
		})
	}
}

func TestDimensions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		content      string
		wantW, wantH float64
	}{
		{
			name:    "explicit width and height",
			content: `<svg width="200" height="100" viewBox="0 0 400 200"/>`,
			wantW:   200, wantH: 100,
		},
		{
			name:    "px suffix",
			content: `<svg width="300px" height="150px"/>`,
			wantW:   300, wantH: 150,
		},
		{
			name:    "percentage returns zero",
			content: `<svg width="100%" height="100%"/>`,
			wantW:   0, wantH: 0,
		},
		{
			name:    "missing attributes returns zero",
			content: `<svg viewBox="0 0 100 100"/>`,
			wantW:   0, wantH: 0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			w, h := Dimensions(test.content)

			assert.InDelta(t, test.wantW, w, 0.01)
			assert.InDelta(t, test.wantH, h, 0.01)
		})
	}
}
