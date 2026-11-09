// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"encoding/xml"
	"errors"
	"fmt"
	"strings"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
)

////////////////////////////////////////////////////////////////////////////////////////
//////// ViewBox defines the SVG viewbox

// ViewBox is used in SVG to define the coordinate system
type ViewBox struct {

	// offset or starting point in parent Viewport2D
	Min math32.Vector2

	// size of viewbox within parent Viewport2D
	Size math32.Vector2

	// how to scale the view box within parent
	PreserveAspectRatio ViewBoxPreserveAspectRatio
}

// Defaults returns viewbox to defaults
func (vb *ViewBox) Defaults() {
	vb.Min = math32.Vector2{}
	vb.Size = math32.Vec2(100, 100)
	vb.PreserveAspectRatio.Align.Set(AlignMid)
	vb.PreserveAspectRatio.MeetOrSlice = Meet
}

func (vb *ViewBox) Max() math32.Vector2 {
	return vb.Min.Add(vb.Size)
}

// BoxString returns the string representation of just the viewbox:
// "min.X min.Y size.X size.Y"
func (vb *ViewBox) BoxString() string {
	return fmt.Sprintf(`viewbox="%g %g %g %g"`, vb.Min.X, vb.Min.Y, vb.Size.X, vb.Size.Y)
}

func (vb *ViewBox) String() string {
	return vb.BoxString() + ` preserveAspectRatio="` + vb.PreserveAspectRatio.String() + `"`
}

// Transform returns the transform based on viewbox size relative to given box
// (viewport) size that it will be rendered into
func (vb *ViewBox) Transform(box math32.Vector2) (size, trans, scale math32.Vector2) {
	of := styles.FitFill
	switch {
	case vb.PreserveAspectRatio.Align.X == AlignNone:
		of = styles.FitFill
	case vb.PreserveAspectRatio.MeetOrSlice == Meet:
		of = styles.FitContain
	case vb.PreserveAspectRatio.MeetOrSlice == Slice:
		of = styles.FitCover
	}
	if vb.Size.X == 0 || vb.Size.Y == 0 {
		vb.Size = math32.Vec2(100, 100)
	}
	size = styles.ObjectSizeFromFit(of, vb.Size, box)
	scale = size.Div(vb.Size)
	extra := box.Sub(size)
	if extra.X > 0 {
		trans.X = extra.X * vb.PreserveAspectRatio.Align.X.AlignFactor()
	}
	if extra.Y > 0 {
		trans.Y = extra.Y * vb.PreserveAspectRatio.Align.Y.AlignFactor()
	}
	trans.SetDiv(scale)
	return
}

// ViewBoxAlign defines values for the PreserveAspectRatio alignment factor
type ViewBoxAligns int32 //enums:enum -trim-prefix Align -transform lower

const (
	// align ViewBox.Min with midpoint of Viewport (default)
	AlignMid ViewBoxAligns = iota

	// do not preserve uniform scaling (if either X or Y is None, both are treated as such).
	// In this case, the Meet / Slice value is ignored.
	// This is the same as FitFill from styles.ObjectFits
	AlignNone

	// align ViewBox.Min with top / left of Viewport
	AlignMin

	// align ViewBox.Min+Size with bottom / right of Viewport
	AlignMax
)

// Aligns returns the styles.Aligns version of ViewBoxAligns
func (va ViewBoxAligns) Aligns() styles.Aligns {
	switch va {
	case AlignNone:
		return styles.Start
	case AlignMin:
		return styles.Start
	case AlignMax:
		return styles.End
	default:
		return styles.Center
	}
}

// SetFromAligns sets alignment from the styles.Aligns version of ViewBoxAligns
func (va *ViewBoxAligns) SetFromAligns(a styles.Aligns) {
	switch a {
	case styles.Start:
		*va = AlignMin
	case styles.End:
		*va = AlignMax
	case styles.Center:
		*va = AlignMid
	}
}

// AlignFactor returns the alignment factor for proportion offset
func (va ViewBoxAligns) AlignFactor() float32 {
	return styles.AlignFactor(va.Aligns())
}

// ViewBoxMeetOrSlice defines values for the PreserveAspectRatio meet or slice factor
type ViewBoxMeetOrSlice int32 //enums:enum -transform lower

const (
	// Meet only applies if Align != None (i.e., only for uniform scaling),
	// and means the entire ViewBox is visible within Viewport,
	// and it is scaled up as much as possible to meet the align constraints.
	// This is the same as FitContain from styles.ObjectFits
	Meet ViewBoxMeetOrSlice = iota

	// Slice only applies if Align != None (i.e., only for uniform scaling),
	// and means the entire ViewBox is covered by the ViewBox, and the
	// ViewBox is scaled down as much as possible, while still meeting the
	// align constraints.
	// This is the same as FitCover from styles.ObjectFits
	Slice
)

// ViewBoxPreserveAspectRatio determines how to scale the view box within parent Viewport2D
type ViewBoxPreserveAspectRatio struct {

	// how to align X, Y coordinates within viewbox
	Align styles.XY[ViewBoxAligns] `xml:"align"`

	// how to scale the view box relative to the viewport
	MeetOrSlice ViewBoxMeetOrSlice `xml:"meetOrSlice"`
}

func (pa *ViewBoxPreserveAspectRatio) String() string {
	if pa.Align.X == AlignNone {
		return "none"
	}
	xs := "xM" + pa.Align.X.String()[1:]
	ys := "YM" + pa.Align.Y.String()[1:]
	s := xs + ys
	if pa.MeetOrSlice != Meet {
		s += " slice"
	}
	return s
}

// SetString sets from a standard svg-formatted string,
// consisting of:
// none | x[Min, Mid, Max]Y[Min, Mid, Max] [ meet | slice]
// e.g., "xMidYMid meet" (default)
// It does not make sense to specify "meet | slice" for "none"
// as they do not apply in that case.
func (pa *ViewBoxPreserveAspectRatio) SetString(s string) error {
	s = strings.TrimSpace(s)
	if len(s) == 0 {
		pa.Align.Set(AlignMid, AlignMid)
		pa.MeetOrSlice = Meet
		return nil
	}
	sl := strings.ToLower(s)
	f := strings.Fields(sl)
	if strings.HasPrefix(f[0], "none") {
		pa.Align.Set(AlignNone)
		pa.MeetOrSlice = Meet
		return nil
	}
	var errs []error
	if len(f) > 1 {
		switch f[1] {
		case "slice":
			pa.MeetOrSlice = Slice
		case "meet":
			pa.MeetOrSlice = Meet
		default:
			errs = append(errs, fmt.Errorf("ViewBoxPreserveAspectRatio: 2nd value must be meet or slice, not %q", f[1]))
		}
	}

	yi := strings.Index(f[0], "y")
	if yi < 0 {
		return fmt.Errorf("ViewBoxPreserveAspectRatio: string %q must contain a 'y'", s)
	}
	xs := f[0][1:yi]
	ys := f[0][yi+1:]

	err := pa.Align.X.SetString(xs)
	if err != nil {
		errs = append(errs, fmt.Errorf("ViewBoxPreserveAspectRatio: X align be min, mid, or max, not %q", xs))
	}

	err = pa.Align.Y.SetString(ys)
	if err != nil {
		errs = append(errs, fmt.Errorf("ViewBoxPreserveAspectRatio: Y align be min, mid, or max, not %q", ys))
	}
	if len(errs) == 0 {
		return nil
	}
	return errors.Join(errs...)
}

// SetFromStyle sets from ObjectFit and Justify (X) and Align (Y) Content
// in given style.
func (pa *ViewBoxPreserveAspectRatio) SetFromStyle(s *styles.Style) {
	pa.Align.X.SetFromAligns(s.Justify.Content)
	pa.Align.Y.SetFromAligns(s.Align.Content)

	// todo: could override with ObjectPosition but maybe not worth it?

	switch s.ObjectFit {
	case styles.FitFill:
		pa.Align.Set(AlignNone)
	case styles.FitContain:
		pa.MeetOrSlice = Meet
	case styles.FitCover, styles.FitScaleDown: // note: FitScaleDown not handled
		pa.MeetOrSlice = Slice
	}
}

// parseLength parses the given SVG length attribute (as x or width) of an
// element in the given root: a number of user units, with an optional
// unit (px, or absolute units, or em for 16px), or a percentage of the
// size of the view box of the root (its width for the x axis, its height
// for y, and their normalized diagonal for r, as for a circle radius).
func (sv *SVG) parseLength(root *Root, v string, axis byte) (float32, error) {
	v = strings.TrimSpace(v)
	if pct, ok := strings.CutSuffix(v, "%"); ok {
		f, err := math32.ParseFloat32(strings.TrimSpace(pct))
		if err != nil {
			return 0, err
		}
		size := root.ViewBox.Size
		if size.X <= 0 || size.Y <= 0 {
			size = math32.FromPoint(sv.Geom.Size)
		}
		switch axis {
		case 'x':
			return f / 100 * size.X, nil
		case 'y':
			return f / 100 * size.Y, nil
		default:
			return f / 100 * math32.Sqrt((size.X*size.X+size.Y*size.Y)/2), nil
		}
	}
	scale := float32(1)
	for _, u := range []struct {
		suffix string
		scale  float32
	}{{"px", 1}, {"pt", 4.0 / 3}, {"pc", 16}, {"mm", 96 / 25.4}, {"cm", 96 / 2.54}, {"in", 96}, {"em", 16}} {
		if rest, ok := strings.CutSuffix(v, u.suffix); ok {
			v, scale = strings.TrimSpace(rest), u.scale
			break
		}
	}
	f, err := math32.ParseFloat32(v)
	return f * scale, err
}

// nestedSVG adds the groups of a nested svg element with the given
// attributes (in the given root) to the given parent: an outer group
// clipped to its viewport (x, y, width, height, 100% by default), as
// browsers clip nested svg elements, and an inner group with the transform
// of its view box to its viewport, which it returns, for its children.
func (sv *SVG) nestedSVG(parent Node, root *Root, attrs []xml.Attr, n int) Node {
	outer := NewGroup(parent)
	inner := NewGroup(outer)
	var x, y float32
	w, _ := sv.parseLength(root, "100%", 'x')
	h, _ := sv.parseLength(root, "100%", 'y')
	var vb ViewBox
	for _, attr := range attrs {
		if SetStandardXMLAttr(inner, attr.Name.Local, attr.Value) {
			continue
		}
		switch attr.Name.Local {
		case "x":
			x, _ = sv.parseLength(root, attr.Value, 'x')
		case "y":
			y, _ = sv.parseLength(root, attr.Value, 'y')
		case "width":
			w, _ = sv.parseLength(root, attr.Value, 'x')
		case "height":
			h, _ = sv.parseLength(root, attr.Value, 'y')
		case "viewBox":
			if pts := math32.ReadPoints(attr.Value); len(pts) == 4 {
				vb.Min.Set(pts[0], pts[1])
				vb.Size.Set(pts[2], pts[3])
			}
		case "preserveAspectRatio":
			vb.PreserveAspectRatio.SetString(attr.Value)
		case "xmlns", "version", "overflow":
		default:
			inner.SetProperty(attr.Name.Local, attr.Value)
		}
	}
	xf := math32.Translate2D(x, y)
	if vb.Size.X > 0 && vb.Size.Y > 0 {
		_, trans, scale := vb.Transform(math32.Vec2(w, h))
		trans.SetSub(vb.Min)
		xf = xf.Mul(math32.Scale2D(scale.X, scale.Y).Translate(trans.X, trans.Y))
	}
	inner.SetProperty("transform", xf.String())
	cp := NewClipPath(sv.Defs)
	cp.SetName(fmt.Sprintf("nested-svg-clip-%d-%p", n, outer))
	rect := NewRect(cp)
	rect.Pos.Set(x, y)
	rect.Size.Set(w, h)
	outer.SetProperty("clip-path", "url(#"+cp.Name+")")
	return inner
}
