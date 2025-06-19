// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
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
