package element

import (
	"encoding/binary"
	"hash/fnv"
	"image/color"
	"io"
	"math"

	"gioui.org/f32"
	"github.com/glasslabs/looking-glass/ui/svg/path"
	"github.com/glasslabs/looking-glass/ui/svg/style"
)

func computePathHash(segs []path.Segment, s style.Style, xform f32.Affine2D) uint64 {
	h := fnv.New64a()
	writeSegments(h, segs)
	writeStyle(h, s)
	writeTransform(h, xform)
	return h.Sum64()
}

func computeRectHash(x, y, w, ht, rx, ry float64, s style.Style, xform f32.Affine2D) uint64 {
	h := fnv.New64a()
	writeFloat64s(h, x, y, w, ht, rx, ry)
	writeStyle(h, s)
	writeTransform(h, xform)
	return h.Sum64()
}

func computeCircleHash(cx, cy, rx, ry float64, s style.Style, xform f32.Affine2D) uint64 {
	h := fnv.New64a()
	writeFloat64s(h, cx, cy, rx, ry)
	writeStyle(h, s)
	writeTransform(h, xform)
	return h.Sum64()
}

func computeTextHash(x, y float64, content string, s style.Style, xform f32.Affine2D) uint64 {
	h := fnv.New64a()
	writeFloat64s(h, x, y)
	writeStr(h, content)
	writeStyle(h, s)
	writeTransform(h, xform)
	return h.Sum64()
}

func computeGroupHash(children []DrawCmd, xform f32.Affine2D) uint64 {
	h := fnv.New64a()
	writeTransform(h, xform)
	for _, child := range children {
		writeUint64(h, child.hash())
	}
	return h.Sum64()
}

func writeSegments(w io.Writer, segs []path.Segment) {
	var buf [8]byte
	for _, seg := range segs {
		buf[0] = seg.Cmd
		_, _ = w.Write(buf[:1])
		for _, arg := range seg.Args {
			binary.LittleEndian.PutUint64(buf[:], math.Float64bits(arg))
			_, _ = w.Write(buf[:])
		}
	}
}

func writeStyle(w io.Writer, s style.Style) {
	writeColor(w, s.Fill)
	writeBool(w, s.FillNone)
	writeFloat32(w, s.FillOpacity)
	writeStr(w, s.FillRule)
	writeColor(w, s.Stroke)
	writeBool(w, s.StrokeNone)
	writeFloat32(w, s.StrokeWidth)
	writeFloat32(w, s.StrokeOpacity)
	writeFloat32(w, s.Opacity)
	writeFloat32(w, s.FontSize)
	writeStr(w, s.FontWeight)
	writeStr(w, s.TextAnchor)
}

func writeTransform(w io.Writer, xform f32.Affine2D) {
	a0, a1, a2, a3, a4, a5 := xform.Elems()
	writeFloat32(w, a0)
	writeFloat32(w, a1)
	writeFloat32(w, a2)
	writeFloat32(w, a3)
	writeFloat32(w, a4)
	writeFloat32(w, a5)
}

func writeFloat64s(w io.Writer, vals ...float64) {
	for _, v := range vals {
		writeFloat64(w, v)
	}
}

func writeFloat64(w io.Writer, v float64) {
	var b [8]byte
	binary.LittleEndian.PutUint64(b[:], math.Float64bits(v))
	_, _ = w.Write(b[:])
}

func writeFloat32(w io.Writer, v float32) {
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], math.Float32bits(v))
	_, _ = w.Write(b[:])
}

func writeColor(w io.Writer, c color.NRGBA) {
	_, _ = w.Write([]byte{c.R, c.G, c.B, c.A})
}

func writeBool(w io.Writer, v bool) {
	if v {
		_, _ = w.Write([]byte{1})
	} else {
		_, _ = w.Write([]byte{0})
	}
}

func writeStr(w io.Writer, s string) {
	_, _ = w.Write([]byte(s))
	_, _ = w.Write([]byte{0}) // null terminator prevents adjacent-string collisions
}

func writeUint64(w io.Writer, v uint64) {
	var b [8]byte
	binary.LittleEndian.PutUint64(b[:], v)
	_, _ = w.Write(b[:])
}
