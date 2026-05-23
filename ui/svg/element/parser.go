package element

import (
	"encoding/xml"
	"fmt"
	"maps"
	"strconv"
	"strings"

	"gioui.org/f32"
	"github.com/glasslabs/looking-glass/ui/svg/path"
	"github.com/glasslabs/looking-glass/ui/svg/style"
	"github.com/glasslabs/looking-glass/ui/svg/transform"
)

// Parse parses an SVG document and returns its draw command tree.
// Unknown elements are silently skipped; a malformed path d-string causes that
// element to be skipped while the rest of the document continues parsing.
// Returns an error only for document-level XML failures.
func Parse(content string) (*Doc, error) {
	p := &parser{
		defs:  make(map[string]DrawCmd),
		sheet: style.NewSheet(),
	}
	doc, err := p.parse(content)
	if err != nil {
		return nil, fmt.Errorf("parsing svg: %w", err)
	}
	return doc, nil
}

// parser holds state accumulated while walking the XML token stream.
type parser struct {
	defs  map[string]DrawCmd
	sheet style.Sheet
}

func (p *parser) parse(content string) (*Doc, error) {
	doc := &Doc{}
	dec := xml.NewDecoder(strings.NewReader(content))

	// Advance to the root <svg> element.
	for {
		tok, err := dec.Token()
		if err != nil {
			return nil, fmt.Errorf("reading svg root: %w", err)
		}
		el, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		if el.Name.Local != "svg" {
			return nil, fmt.Errorf("expected <svg> root element, got <%s>", el.Name.Local)
		}
		attrs := attrsToMap(el.Attr)
		doc.Width = parseLength(attrs["width"])
		doc.Height = parseLength(attrs["height"])
		doc.ViewBox = parseViewBox(attrs["viewBox"])
		break
	}

	rootStyle := style.Default()
	doc.Children = p.parseChildren(dec, rootStyle, "svg")
	return doc, nil
}

// parseChildren walks the token stream collecting draw commands until the
// matching end element is reached.
func (p *parser) parseChildren(dec *xml.Decoder, parentStyle style.Style, endTag string) []DrawCmd {
	var cmds []DrawCmd
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			cmd := p.parseElement(dec, t, parentStyle)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		case xml.EndElement:
			if t.Name.Local == endTag {
				return cmds
			}
		}
	}
	return cmds
}

// parseElement dispatches a StartElement to the appropriate handler.
// Returns nil for elements that produce no drawable output (defs, unknown, display:none).
// Each returned DrawCmd carries only the element's own local transform, not accumulated ancestors.
//
//nolint:cyclop,gocognit,gocyclo // Element dispatch is an inherently large switch.
func (p *parser) parseElement(dec *xml.Decoder, el xml.StartElement, parentStyle style.Style) DrawCmd {
	attrs := attrsToMap(el.Attr)

	// Apply matching CSS sheet rules respecting specificity order:
	// single-class rules, then compound class rules (ascending class count),
	// then ID rule — all before inline style="" which takes highest priority.
	var sheetDecls strings.Builder

	elementClasses := strings.Fields(attrs["class"])
	for _, c := range elementClasses {
		if decls, ok := p.sheet.ClassRules[c]; ok {
			sheetDecls.WriteString(decls)
			sheetDecls.WriteByte(';')
		}
	}

	if len(elementClasses) > 1 && len(p.sheet.CompoundClassRules) > 0 {
		classSet := make(map[string]struct{}, len(elementClasses))
		for _, c := range elementClasses {
			classSet[c] = struct{}{}
		}
		for _, cr := range p.sheet.CompoundClassRules {
			if len(cr.Classes) > len(elementClasses) {
				continue
			}
			match := true
			for _, c := range cr.Classes {
				if _, ok := classSet[c]; !ok {
					match = false
					break
				}
			}
			if match {
				sheetDecls.WriteString(cr.Decls)
				sheetDecls.WriteByte(';')
			}
		}
	}

	if id := attrs["id"]; id != "" {
		if decls, ok := p.sheet.IDRules[id]; ok {
			sheetDecls.WriteString(decls)
			sheetDecls.WriteByte(';')
		}
	}

	if sheetDecls.Len() > 0 {
		if existing := attrs["style"]; existing != "" {
			attrs["style"] = sheetDecls.String() + existing
		} else {
			attrs["style"] = sheetDecls.String()
		}
	}

	s := style.Inherit(parentStyle, style.Parse(attrs), attrs)
	if s.Display == "none" {
		skipElement(dec, el.Name.Local)
		return nil
	}

	xform := f32.AffineId()
	if v, ok := attrs["transform"]; ok {
		if t, err := transform.Parse(v); err == nil {
			xform = t
		}
	}

	switch el.Name.Local {
	case "g":
		children := p.parseChildren(dec, s, "g")
		if len(children) == 0 {
			return nil
		}
		return &GroupCmd{Children: children, Transform: xform, Hash: computeGroupHash(children, xform)}

	case "defs":
		p.parseDefs(dec, s)
		return nil

	case "style":
		p.parseStyleElement(dec)
		return nil

	case "use":
		return p.parseUse(attrs, xform)

	case "symbol":
		id := attrs["id"]
		children := p.parseChildren(dec, s, "symbol")
		if id != "" && len(children) > 0 {
			p.defs[id] = &GroupCmd{Children: children, Transform: f32.AffineId(), Hash: computeGroupHash(children, f32.AffineId())}
		}
		return nil

	case "path":
		return parsePath(attrs, s, xform)

	case "rect":
		return parseRect(attrs, s, xform)

	case "circle":
		return parseCircle(attrs, s, xform)

	case "ellipse":
		return parseEllipse(attrs, s, xform)

	case "line":
		skipElement(dec, el.Name.Local)
		return parseLine(attrs, s, xform)

	case "polyline":
		skipElement(dec, el.Name.Local)
		return parsePolyline(attrs, s, xform, false)

	case "polygon":
		skipElement(dec, el.Name.Local)
		return parsePolyline(attrs, s, xform, true)

	case "text":
		return p.parseText(dec, attrs, s, xform)

	default:
		skipElement(dec, el.Name.Local)
		return nil
	}
}

// parseDefs walks a <defs> block, registering each identified element.
func (p *parser) parseDefs(dec *xml.Decoder, parentStyle style.Style) {
	for {
		tok, err := dec.Token()
		if err != nil {
			return
		}
		switch t := tok.(type) {
		case xml.StartElement:
			attrs := attrsToMap(t.Attr)
			s := style.Inherit(parentStyle, style.Parse(attrs), attrs)
			id := attrs["id"]
			cmd := p.parseElement(dec, t, s)
			if id != "" && cmd != nil {
				p.defs[id] = cmd
			}
		case xml.EndElement:
			if t.Name.Local == "defs" {
				return
			}
		}
	}
}

// parseStyleElement reads a <style> element's text content and registers any
// #id and .class CSS rules it contains into p.sheet.
func (p *parser) parseStyleElement(dec *xml.Decoder) {
	var content strings.Builder
	for {
		tok, err := dec.Token()
		if err != nil {
			return
		}
		switch t := tok.(type) {
		case xml.CharData:
			content.Write(t)
		case xml.EndElement:
			if t.Name.Local == "style" {
				sh := style.ParseSheet(content.String())
				maps.Copy(p.sheet.IDRules, sh.IDRules)
				maps.Copy(p.sheet.ClassRules, sh.ClassRules)
				p.sheet.CompoundClassRules = append(p.sheet.CompoundClassRules, sh.CompoundClassRules...)
				return
			}
		}
	}
}

// parseUse inlines the element referenced by href/xlink:href, wrapping it in
// a GroupCmd that carries the <use> element's own transform and x/y offset.
func (p *parser) parseUse(attrs map[string]string, xform f32.Affine2D) DrawCmd {
	href := attrs["href"]
	if href == "" {
		href = attrs["xlink:href"]
	}
	id := strings.TrimPrefix(href, "#")
	if id == "" {
		return nil
	}
	ref, ok := p.defs[id]
	if !ok {
		return nil
	}

	dx := parseFloat(attrs["x"])
	dy := parseFloat(attrs["y"])
	if dx != 0 || dy != 0 {
		offset := f32.AffineId().Offset(f32.Pt(float32(dx), float32(dy)))
		xform = xform.Mul(offset)
	}

	children := []DrawCmd{ref}
	return &GroupCmd{Children: children, Transform: xform, Hash: computeGroupHash(children, xform)}
}

func parsePath(attrs map[string]string, s style.Style, xform f32.Affine2D) DrawCmd {
	d := attrs["d"]
	if d == "" {
		return nil
	}
	segs, err := path.Parse(d)
	if err != nil || len(segs) == 0 {
		return nil
	}
	return &PathCmd{Segments: segs, Style: s, Transform: xform, Hash: computePathHash(segs, s, xform)}
}

func parseRect(attrs map[string]string, s style.Style, xform f32.Affine2D) DrawCmd {
	w := parseFloat(attrs["width"])
	h := parseFloat(attrs["height"])
	if w <= 0 || h <= 0 {
		return nil
	}
	x := parseFloat(attrs["x"])
	y := parseFloat(attrs["y"])
	rx := parseFloat(attrs["rx"])
	ry := parseFloat(attrs["ry"])
	return &RectCmd{
		X:         x,
		Y:         y,
		W:         w,
		H:         h,
		RX:        rx,
		RY:        ry,
		Style:     s,
		Transform: xform,
		Hash:      computeRectHash(x, y, w, h, rx, ry, s, xform),
	}
}

func parseCircle(attrs map[string]string, s style.Style, xform f32.Affine2D) DrawCmd {
	r := parseFloat(attrs["r"])
	if r <= 0 {
		return nil
	}
	cx := parseFloat(attrs["cx"])
	cy := parseFloat(attrs["cy"])
	return &CircleCmd{
		CX:        cx,
		CY:        cy,
		RX:        r,
		RY:        r,
		Style:     s,
		Transform: xform,
		Hash:      computeCircleHash(cx, cy, r, r, s, xform),
	}
}

func parseEllipse(attrs map[string]string, s style.Style, xform f32.Affine2D) DrawCmd {
	rx := parseFloat(attrs["rx"])
	ry := parseFloat(attrs["ry"])
	if rx <= 0 || ry <= 0 {
		return nil
	}
	cx := parseFloat(attrs["cx"])
	cy := parseFloat(attrs["cy"])
	return &CircleCmd{
		CX:        cx,
		CY:        cy,
		RX:        rx,
		RY:        ry,
		Style:     s,
		Transform: xform,
		Hash:      computeCircleHash(cx, cy, rx, ry, s, xform),
	}
}

// parseLine converts a <line> into a PathCmd (M x1,y1 L x2,y2).
func parseLine(attrs map[string]string, s style.Style, xform f32.Affine2D) DrawCmd {
	x1 := parseFloat(attrs["x1"])
	y1 := parseFloat(attrs["y1"])
	x2 := parseFloat(attrs["x2"])
	y2 := parseFloat(attrs["y2"])
	segs := []path.Segment{
		{Cmd: 'M', Args: []float64{x1, y1}},
		{Cmd: 'L', Args: []float64{x2, y2}},
	}
	return &PathCmd{Segments: segs, Style: s, Transform: xform, Hash: computePathHash(segs, s, xform)}
}

// parsePolyline converts <polyline> or <polygon> into a PathCmd.
// When closed is true (polygon), a Z segment is appended.
func parsePolyline(attrs map[string]string, s style.Style, xform f32.Affine2D, closed bool) DrawCmd {
	pts := parsePoints(attrs["points"])
	if len(pts) < 2 {
		return nil
	}
	segs := make([]path.Segment, 0, len(pts)+1)
	for i, pt := range pts {
		cmd := byte('L')
		if i == 0 {
			cmd = 'M'
		}
		segs = append(segs, path.Segment{Cmd: cmd, Args: []float64{pt[0], pt[1]}})
	}
	if closed {
		segs = append(segs, path.Segment{Cmd: 'Z'})
	}
	return &PathCmd{Segments: segs, Style: s, Transform: xform, Hash: computePathHash(segs, s, xform)}
}

func (p *parser) parseText(dec *xml.Decoder, attrs map[string]string, s style.Style, xform f32.Affine2D) DrawCmd {
	var content strings.Builder
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.CharData:
			content.Write(t)
		case xml.EndElement:
			if t.Name.Local == "text" {
				text := strings.TrimSpace(content.String())
				if text == "" {
					return nil
				}
				x := parseFloat(attrs["x"])
				y := parseFloat(attrs["y"])
				return &TextCmd{
					X:         x,
					Y:         y,
					Content:   text,
					Style:     s,
					Transform: xform,
					Hash:      computeTextHash(x, y, text, s, xform),
				}
			}
		}
	}
	return nil
}

// skipElement consumes all tokens up to and including the matching end element.
func skipElement(dec *xml.Decoder, name string) {
	depth := 1
	for depth > 0 {
		tok, err := dec.Token()
		if err != nil {
			return
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == name {
				depth++
			}
		case xml.EndElement:
			if t.Name.Local == name {
				depth--
			}
		}
	}
}

func attrsToMap(attrs []xml.Attr) map[string]string {
	m := make(map[string]string, len(attrs))
	for _, a := range attrs {
		key := a.Name.Local
		if a.Name.Space != "" {
			key = a.Name.Space + ":" + key
		}
		m[key] = a.Value
	}
	return m
}

// parseFloat parses a float64 from s; returns 0 on failure.
func parseFloat(s string) float64 {
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0
	}
	return v
}

func parseLength(s string) float64 {
	s = strings.TrimSpace(s)
	for _, suffix := range []string{"px", "pt", "em", "rem", "%"} {
		if strings.HasSuffix(s, suffix) {
			if suffix == "%" {
				return 0
			}
			s = s[:len(s)-len(suffix)]
			break
		}
	}
	return parseFloat(s)
}

func parseViewBox(s string) [4]float64 {
	fields := strings.Fields(strings.ReplaceAll(s, ",", " "))
	if len(fields) != 4 {
		return [4]float64{}
	}
	var vb [4]float64
	for i, f := range fields {
		vb[i] = parseFloat(f)
	}
	return vb
}

func parsePoints(s string) [][2]float64 {
	fields := strings.FieldsFunc(s, func(r rune) bool {
		return r == ' ' || r == ',' || r == '\t' || r == '\n'
	})
	var pts [][2]float64
	for i := 0; i+1 < len(fields); i += 2 {
		x := parseFloat(fields[i])
		y := parseFloat(fields[i+1])
		pts = append(pts, [2]float64{x, y})
	}
	return pts
}
