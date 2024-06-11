// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"image"
	"reflect"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/tree"
)

// Styler adds the given function for setting the style properties of the widget
// to [WidgetBase.Stylers]. It is one of the main ways to specify the styles of
// a widget, in addition to FirstStyler and FinalStyler, which add stylers that
// are called before and after the stylers added by this function, respectively.
func (wb *WidgetBase) Styler(s func(s *styles.Style)) *WidgetBase {
	wb.Stylers = append(wb.Stylers, s)
	return wb
}

// FirstStyler adds the given function for setting the style properties of the widget
// to [WidgetBase.FirstStylers]. It is one of the main ways to specify the styles of
// a widget, in addition to Styler and FinalStyler, which add stylers that are called
// after the stylers added by this function.
func (wb *WidgetBase) FirstStyler(s func(s *styles.Style)) *WidgetBase {
	wb.FirstStylers = append(wb.FirstStylers, s)
	return wb
}

// FinalStyler adds the given function for setting the style properties of the widget
// to [WidgetBase.FinalStylers]. It is one of the main ways to specify the styles of
// a widget, in addition to FirstStyler and Styler, which add stylers that are called
// before the stylers added by this function.
func (wb *WidgetBase) FinalStyler(s func(s *styles.Style)) *WidgetBase {
	wb.FinalStylers = append(wb.FinalStylers, s)
	return wb
}

// ApplyStyleWidget is the primary styling function for all Widgets.
// Handles inheritance and runs the Styler functions.
func (wb *WidgetBase) ApplyStyleWidget() {
	if wb.This() == nil {
		return
	}

	pw := wb.ParentWidget()

	// we do these things even if we are overriding the style
	defer func() {
		// note: this does not un-set the Invisible if not None, because all kinds of things
		// can turn invisible to off.
		if wb.Styles.Display == styles.DisplayNone {
			wb.SetState(true, states.Invisible)
		}
		psz := math32.Vector2{}
		if pw != nil {
			psz = pw.Geom.Size.Alloc.Content
		}
		SetUnitContext(&wb.Styles, wb.Scene, wb.Geom.Size.Alloc.Content, psz)
	}()

	if wb.OverrideStyle {
		return
	}
	wb.ResetStyleWidget()

	if pw != nil {
		wb.Styles.InheritFields(&pw.Styles)
	}

	wb.ResetStyleSettings()
	wb.RunStylers()
	wb.ApplyStyleSettings()
}

// ResetStyleWidget resets the widget styles and applies the basic
// default styles specified in [styles.Style.Defaults]. It is called
// automatically in [ApplyStyleWidget]
// and should not need to be called by end-user code.
func (wb *WidgetBase) ResetStyleWidget() {
	s := &wb.Styles

	// need to persist state
	state := s.State
	*s = styles.Style{}
	s.Defaults()
	s.State = state

	// default to state layer associated with the state,
	// which the developer can override in their stylers
	// wb.Transition(&s.StateLayer, s.State.StateLayer(), 200*time.Millisecond, LinearTransition)
	s.StateLayer = s.State.StateLayer()

	s.SetMono(false)
}

// RunStylers runs the stylers specified in the widget's FirstStylers,
// Stylers, and FinalStylers in that order in a sequential ascending order.
func (wb *WidgetBase) RunStylers() {
	for _, s := range wb.FirstStylers {
		s(&wb.Styles)
	}
	for _, s := range wb.Stylers {
		s(&wb.Styles)
	}
	for _, s := range wb.FinalStylers {
		s(&wb.Styles)
	}
}

// ResetStyleSettings reverses the effects of [ApplyStyleSettings]
// for the widget's font size so that it does not create cascading
// inhereted font size values. It only does this for non-root elements,
// as the root element must receive the larger font size so that
// all other widgets inherit it. It must be called before
// [WidgetBase.RunStylers] and [WidgetBase.ApplyStyleSettings].
func (wb *WidgetBase) ResetStyleSettings() {
	if tree.IsRoot(wb) {
		return
	}
	fsz := AppearanceSettings.FontSize / 100
	wb.Styles.Font.Size.Value /= fsz
	wb.Styles.Text.LineHeight.Value /= fsz
}

// ApplyStyleSettings applies [AppearanceSettingsData.Spacing]
// and [AppearanceSettings.FontSize] to the style values for the widget.
func (wb *WidgetBase) ApplyStyleSettings() {
	s := &wb.Styles

	spc := AppearanceSettings.Spacing / 100
	s.Margin.Top.Value *= spc
	s.Margin.Right.Value *= spc
	s.Margin.Bottom.Value *= spc
	s.Margin.Left.Value *= spc
	s.Padding.Top.Value *= spc
	s.Padding.Right.Value *= spc
	s.Padding.Bottom.Value *= spc
	s.Padding.Left.Value *= spc
	s.Gap.X.Value *= spc
	s.Gap.Y.Value *= spc

	fsz := AppearanceSettings.FontSize / 100
	s.Font.Size.Value *= fsz
	s.Text.LineHeight.Value *= fsz
}

// ApplyStyleUpdate calls ApplyStyleTree and NeedsRender.
// This is the main call needed to ensure that state-sensitive styling
// is updated, when the state changes.
func (wb *WidgetBase) ApplyStyleUpdate() {
	wb.ApplyStyleTree()
	wb.NeedsRender()
}

// Style updates the style properties of the widget based on [WidgetBase.Stylers].
// To specify the style properties of a widget, use [WidgetBase.Styler].
func (wb *WidgetBase) Style() {
	wb.ApplyStyleWidget()
}

// SetUnitContext sets the unit context based on size of scene, element, and parent
// element (from bbox) and then caches everything out in terms of raw pixel
// dots for rendering.
// Zero values for element and parent size are ignored.
func SetUnitContext(st *styles.Style, sc *Scene, el, parent math32.Vector2) {
	rebuild := false
	var rc *RenderContext
	sz := image.Point{1920, 1280}
	if sc != nil {
		rebuild = sc.NeedsRebuild()
		rc = sc.RenderContext()
		sz = sc.SceneGeom.Size
	}
	if rc != nil {
		st.UnitContext.DPI = rc.LogicalDPI
	} else {
		st.UnitContext.DPI = 160
	}
	st.UnitContext.SetSizes(float32(sz.X), float32(sz.Y), el.X, el.Y, parent.X, parent.Y)
	if st.Font.Face == nil || rebuild {
		st.Font = paint.OpenFont(st.FontRender(), &st.UnitContext) // calls SetUnContext after updating metrics
	}
	st.ToDots()
}

// ChildBackground returns the background color (Image) for given child Widget.
// By default, this is just our [Styles.Actualbackground] but it can be computed
// specifically for the child (e.g., for zebra stripes in views.SliceViewGrid)
func (wb *WidgetBase) ChildBackground(child Widget) image.Image {
	return wb.Styles.ActualBackground
}

// ParentActualBackground returns the actual background of
// the parent of the widget. If it has no parent, it returns nil.
func (wb *WidgetBase) ParentActualBackground() image.Image {
	pwb := wb.ParentWidget()
	if pwb == nil {
		return nil
	}
	return pwb.This().(Widget).ChildBackground(wb.This().(Widget))
}

// StyleFromTags adds a [WidgetBase.Styler] to the given widget
// to set its style properties based on the given [reflect.StructTag].
// Width, height, and grow properties are supported.
func StyleFromTags(w Widget, tags reflect.StructTag) {
	style := func(tag string, set func(v float32)) {
		if v, ok := tags.Lookup(tag); ok {
			f, err := reflectx.ToFloat32(v)
			if errors.Log(err) == nil {
				set(f)
			}
		}
	}
	w.Styler(func(s *styles.Style) {
		style("width", s.Min.X.Ch)
		style("max-width", s.Max.X.Ch)
		style("height", s.Min.Y.Em)
		style("max-height", s.Max.Y.Em)
		style("grow", func(v float32) { s.Grow.X = v })
		style("grow-y", func(v float32) { s.Grow.Y = v })
	})
}
