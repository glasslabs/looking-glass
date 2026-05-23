package style

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseSheet_IDRule(t *testing.T) {
	t.Parallel()

	sh := ParseSheet(`#light\.kitchen { fill: #ffd400; stroke: #ffd400; }`)

	assert.Equal(t, map[string]string{"light.kitchen": "fill: #ffd400; stroke: #ffd400;"}, sh.IDRules)
	assert.Empty(t, sh.ClassRules)
	assert.Empty(t, sh.CompoundClassRules)
}

func TestParseSheet_ClassRule(t *testing.T) {
	t.Parallel()

	sh := ParseSheet(`.light { fill: #ffd400; }`)

	assert.Equal(t, map[string]string{"light": "fill: #ffd400;"}, sh.ClassRules)
	assert.Empty(t, sh.IDRules)
	assert.Empty(t, sh.CompoundClassRules)
}

func TestParseSheet_MultipleRules(t *testing.T) {
	t.Parallel()

	css := `
		.light { fill: #ffd400; stroke: #ffd400; }
		.door  { fill: #42a2dd; stroke: #42a2dd; }
		#light\.kitchen { fill: #ff0000; }
	`

	sh := ParseSheet(css)

	assert.Equal(t, map[string]string{
		"light": "fill: #ffd400; stroke: #ffd400;",
		"door":  "fill: #42a2dd; stroke: #42a2dd;",
	}, sh.ClassRules)
	assert.Equal(t, map[string]string{
		"light.kitchen": "fill: #ff0000;",
	}, sh.IDRules)
	assert.Empty(t, sh.CompoundClassRules)
}

func TestParseSheet_ColorValuesNotMistakenForSelectors(t *testing.T) {
	t.Parallel()

	// Colour literals like #ffd400 inside declaration blocks must not be
	// parsed as ID selectors.
	sh := ParseSheet(`.on { fill: #ffd400; stroke: #ffd400; }`)

	assert.Equal(t, map[string]string{"on": "fill: #ffd400; stroke: #ffd400;"}, sh.ClassRules)
	assert.Empty(t, sh.IDRules)
	assert.Empty(t, sh.CompoundClassRules)
}

func TestParseSheet_CompoundClassIgnored(t *testing.T) {
	t.Parallel()

	// ".light .on" uses a descendant combinator — not a compound class selector.
	sh := ParseSheet(`.light .on { fill: #ffd400; }`)

	assert.Empty(t, sh.ClassRules)
	assert.Empty(t, sh.IDRules)
	assert.Empty(t, sh.CompoundClassRules)
}

func TestParseSheet_MultipleClassesOnElement(t *testing.T) {
	t.Parallel()

	// Two separate simple class rules — each stored independently.
	// An SVG element with class="light on" would match both.
	css := `
		.light { stroke: none; }
		.on    { fill: #ffd400; }
	`

	sh := ParseSheet(css)

	assert.Equal(t, map[string]string{
		"light": "stroke: none;",
		"on":    "fill: #ffd400;",
	}, sh.ClassRules)
	assert.Empty(t, sh.CompoundClassRules)
}

func TestParseSheet_ComboMultipleClassesOnElement(t *testing.T) {
	t.Parallel()

	// ".light.on" is a compound selector — only matches elements that have
	// BOTH classes simultaneously. It must not fire on class="light" alone.
	css := `
		.light    { fill: none; stroke: none; }
		.light.on { fill: #ffd400; stroke: #ffd400; }
	`

	sh := ParseSheet(css)

	assert.Equal(t, map[string]string{
		"light": "fill: none; stroke: none;",
	}, sh.ClassRules)
	assert.Equal(t, []CompoundRule{
		{Classes: []string{"light", "on"}, Decls: "fill: #ffd400; stroke: #ffd400;"},
	}, sh.CompoundClassRules)
}

func TestParseSheet_CompoundRulesSortedBySpecificity(t *testing.T) {
	t.Parallel()

	// Two-class rule has lower specificity than three-class rule regardless of
	// document order — the three-class rule must be applied last so it wins.
	css := `
		.a.b.c { fill: red; }
		.a.b   { fill: blue; }
	`

	sh := ParseSheet(css)

	assert.Equal(t, []CompoundRule{
		{Classes: []string{"a", "b"}, Decls: "fill: blue;"},
		{Classes: []string{"a", "b", "c"}, Decls: "fill: red;"},
	}, sh.CompoundClassRules)
}

func TestParseSheet_EmptyDeclarationsSkipped(t *testing.T) {
	t.Parallel()

	sh := ParseSheet(`#foo {}`)

	assert.Empty(t, sh.IDRules)
}

func TestParseSheet_UnknownSelectorIgnored(t *testing.T) {
	t.Parallel()

	sh := ParseSheet(`svg rect { fill: red; }`)

	assert.Empty(t, sh.IDRules)
	assert.Empty(t, sh.ClassRules)
	assert.Empty(t, sh.CompoundClassRules)
}

func TestParseSheet_SelectorUnescaping(t *testing.T) {
	t.Parallel()

	// CSS requires dots in identifiers to be escaped as \. — the parsed key
	// must be the unescaped form that matches the raw SVG id attribute.
	sh := ParseSheet(`#cover\.garage { fill: #42a2dd; }`)

	_, ok := sh.IDRules["cover.garage"]
	assert.True(t, ok)
}

func TestParseSheet_Empty(t *testing.T) {
	t.Parallel()

	sh := ParseSheet("")

	assert.Empty(t, sh.IDRules)
	assert.Empty(t, sh.ClassRules)
	assert.Empty(t, sh.CompoundClassRules)
}
