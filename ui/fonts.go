package ui

import (
	"embed"
	"image/color"
	"io/fs"
	"strings"

	"gioui.org/font"
	"gioui.org/font/opentype"
	"gioui.org/unit"
	"github.com/hamba/logger/v2"
	lctx "github.com/hamba/logger/v2/ctx"
)

//go:embed all:fonts
var embeddedFonts embed.FS

const (
	robotoTypeface          font.Typeface = "Roboto"
	robotoCondensedTypeface font.Typeface = "Roboto Condensed"

	defaultFontSizeSp unit.Sp = 24
)

var defaultTextColor = color.NRGBA{R: 255, G: 255, B: 255, A: 255}

type fontSpec struct {
	filename string
	typeface font.Typeface
	weight   font.Weight
	style    font.Style
}

var robotoSpecs = []fontSpec{
	{"Roboto-Thin.ttf", robotoTypeface, font.Thin, font.Regular},
	{"Roboto-Light.ttf", robotoTypeface, font.Light, font.Regular},
	{"Roboto-LightItalic.ttf", robotoTypeface, font.Light, font.Italic},
	{"Roboto-Regular.ttf", robotoTypeface, font.Normal, font.Regular},
	{"Roboto-Italic.ttf", robotoTypeface, font.Normal, font.Italic},
	{"Roboto-Medium.ttf", robotoTypeface, font.Medium, font.Regular},
	{"Roboto-MediumItalic.ttf", robotoTypeface, font.Medium, font.Italic},
	{"Roboto-Bold.ttf", robotoTypeface, font.Bold, font.Regular},
	{"Roboto-BoldItalic.ttf", robotoTypeface, font.Bold, font.Italic},
	{"RobotoCondensed-Light.ttf", robotoCondensedTypeface, font.Light, font.Regular},
	{"RobotoCondensed-LightItalic.ttf", robotoCondensedTypeface, font.Light, font.Italic},
	{"RobotoCondensed-Regular.ttf", robotoCondensedTypeface, font.Normal, font.Regular},
	{"RobotoCondensed-Italic.ttf", robotoCondensedTypeface, font.Normal, font.Italic},
	{"RobotoCondensed-Bold.ttf", robotoCondensedTypeface, font.Bold, font.Regular},
	{"RobotoCondensed-BoldItalic.ttf", robotoCondensedTypeface, font.Bold, font.Italic},
}

func loadFontFaces(log *logger.Logger) []font.FontFace {
	var faces []font.FontFace

	for _, spec := range robotoSpecs {
		path := "fonts/" + spec.filename

		data, err := fs.ReadFile(embeddedFonts, path)
		if err != nil {
			continue
		}

		face, err := opentype.Parse(data)
		if err != nil {
			log.Error("fonts: could not parse embedded font",
				lctx.Str("file", spec.filename), lctx.Err(err))
			continue
		}

		faces = append(faces, font.FontFace{
			Font: font.Font{
				Typeface: spec.typeface,
				Weight:   spec.weight,
				Style:    spec.style,
			},
			Face: face,
		})
	}

	if len(faces) > 0 {
		families := uniqueFamilies(faces)
		log.Info("fonts: loaded embedded font variants",
			lctx.Int("variants", len(faces)),
			lctx.Str("families", strings.Join(families, ", ")))
	} else {
		log.Error("fonts: no Roboto TTF files found in embedded fonts directory; " +
			"run `make fonts` and rebuild. falling back to Go font")
	}

	return faces
}

func uniqueFamilies(faces []font.FontFace) []string {
	seen := make(map[font.Typeface]struct{})
	var out []string
	for _, f := range faces {
		if _, ok := seen[f.Font.Typeface]; !ok {
			seen[f.Font.Typeface] = struct{}{}
			out = append(out, string(f.Font.Typeface))
		}
	}
	return out
}
