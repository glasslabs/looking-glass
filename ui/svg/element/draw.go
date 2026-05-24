// Package element parses SVG XML into a tree of draw commands that can be
// walked by a renderer. It has no Gio dependency; rendering is the caller's
// responsibility.
package element

import (
	"gioui.org/f32"
	"github.com/glasslabs/looking-glass/ui/svg/path"
	"github.com/glasslabs/looking-glass/ui/svg/style"
)

// DrawCmd is implemented by every drawable SVG element.
type DrawCmd interface {
	drawCmd()
	hash() uint64
}

// PathCmd draws a shape defined by SVG path segments.
type PathCmd struct {
	Segments  []path.Segment
	Style     style.Style
	Transform f32.Affine2D
	Hash      uint64
}

func (*PathCmd) drawCmd()       {}
func (c *PathCmd) hash() uint64 { return c.Hash }

// RectCmd draws a rectangle, optionally with rounded corners.
type RectCmd struct {
	X, Y      float64
	W, H      float64
	RX, RY    float64 // corner radii; zero means sharp corners
	Style     style.Style
	Transform f32.Affine2D
	Hash      uint64
}

func (*RectCmd) drawCmd()       {}
func (c *RectCmd) hash() uint64 { return c.Hash }

// CircleCmd draws a circle or ellipse.
// For a circle, RX and RY are equal.
type CircleCmd struct {
	CX, CY    float64
	RX, RY    float64
	Style     style.Style
	Transform f32.Affine2D
	Hash      uint64
}

func (*CircleCmd) drawCmd()       {}
func (c *CircleCmd) hash() uint64 { return c.Hash }

// TextCmd draws a text string at a canvas position.
type TextCmd struct {
	X, Y      float64
	Content   string
	Style     style.Style
	Transform f32.Affine2D
	Hash      uint64
}

func (*TextCmd) drawCmd()       {}
func (c *TextCmd) hash() uint64 { return c.Hash }

// GroupCmd groups child draw commands under a shared transform.
type GroupCmd struct {
	Children  []DrawCmd
	Transform f32.Affine2D
	Hash      uint64
}

func (*GroupCmd) drawCmd()       {}
func (c *GroupCmd) hash() uint64 { return c.Hash }

// Doc is the parsed representation of an SVG document.
type Doc struct {
	// Width and Height are the intrinsic dimensions from the root <svg>
	// element's width and height attributes, in SVG user units.
	// Zero when the attributes are absent or use relative units.
	Width, Height float64
	// ViewBox is [minX, minY, width, height] from the viewBox attribute.
	// Zero when the attribute is absent.
	ViewBox [4]float64
	// Children are the top-level draw commands in document order.
	Children []DrawCmd
}
