package style

import (
	"sort"
	"strings"
)

// CompoundRule holds a parsed multi-class CSS rule (.light.on { ... }).
// All Classes must be present on an element for the rule to match.
type CompoundRule struct {
	// Classes is the sorted list of class names that must all be present.
	Classes []string
	// Decls is the raw CSS declaration block for the rule.
	Decls string
}

// Sheet holds parsed CSS rules from a <style> element.
type Sheet struct {
	// IDRules maps an unescaped SVG element id to its CSS declaration block.
	IDRules map[string]string
	// ClassRules maps a single class name to its CSS declaration block.
	ClassRules map[string]string
	// CompoundClassRules holds multi-class rules (.light.on) sorted by
	// ascending number of required classes so that higher-specificity rules
	// are applied last and override lower-specificity ones.
	CompoundClassRules []CompoundRule
}

// NewSheet returns a Sheet with initialised maps ready for use.
func NewSheet() Sheet {
	return Sheet{
		IDRules:    make(map[string]string),
		ClassRules: make(map[string]string),
	}
}

// ParseSheet parses a CSS stylesheet and populates a Sheet with all simple
// #id, .class, and compound .class1.class2 rules it finds.
// Descendant combinators (spaces), attribute selectors, pseudo-classes, and
// other complex selectors are silently ignored.
//
// The parser scans { … } pairs rather than leading # or . characters so that
// colour values such as fill:#ffd400 inside a declaration block are never
// mistaken for selector tokens.
func ParseSheet(css string) Sheet {
	sh := NewSheet()
	rest := css
	for {
		braceIdx := strings.IndexByte(rest, '{')
		if braceIdx == -1 {
			break
		}
		selector := strings.TrimSpace(rest[:braceIdx])
		rest = rest[braceIdx+1:]

		closeIdx := strings.IndexByte(rest, '}')
		if closeIdx == -1 {
			break
		}
		declarations := strings.TrimSpace(rest[:closeIdx])
		rest = rest[closeIdx+1:]

		if selector == "" || declarations == "" {
			continue
		}

		switch {
		case strings.HasPrefix(selector, "#"):
			raw := selector[1:]
			if isSimpleToken(raw) {
				sh.IDRules[unescapeSelector(raw)] = declarations
			}

		case strings.HasPrefix(selector, "."):
			// Reject any selector containing combinators or structural characters
			// (e.g. ".light .on" is a descendant combinator, not a compound selector).
			if strings.ContainsAny(selector, " \t\n[]()>~+*,:") {
				continue
			}
			// Split on unescaped '.' to separate compound class names.
			var classes []string
			for part := range strings.SplitSeq(selector[1:], ".") {
				if isSimpleToken(part) {
					classes = append(classes, unescapeSelector(part))
				}
			}
			if len(classes) == 0 {
				continue
			}
			if len(classes) == 1 {
				sh.ClassRules[classes[0]] = declarations
			} else {
				sort.Strings(classes) // canonical order for subset matching
				sh.CompoundClassRules = append(sh.CompoundClassRules, CompoundRule{
					Classes: classes,
					Decls:   declarations,
				})
			}
		}
	}

	// Sort compound rules by ascending number of required classes so that
	// higher-specificity rules (more classes) are applied last and win.
	sort.SliceStable(sh.CompoundClassRules, func(i, j int) bool {
		return len(sh.CompoundClassRules[i].Classes) < len(sh.CompoundClassRules[j].Classes)
	})

	return sh
}

// isSimpleToken reports whether s is a bare CSS identifier with no unescaped
// structural characters. CSS escape sequences (e.g. \.) are treated as a
// single valid character and skipped over, so "light\.kitchen" is accepted
// while "light .kitchen" (space = descendant combinator) is rejected.
func isSimpleToken(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); {
		if s[i] == '\\' && i+1 < len(s) {
			i += 2 // valid escape sequence — skip both characters
			continue
		}
		if strings.ContainsAny(string(s[i]), " \t\n.:[]()>#~+*,") {
			return false
		}
		i++
	}
	return true
}

// unescapeSelector removes CSS escape sequences from a selector token.
// Handles \. → . which is the common case for HA entity IDs containing dots.
func unescapeSelector(s string) string {
	return strings.ReplaceAll(s, `\.`, ".")
}
