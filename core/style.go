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
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/tree"
)

// Styler adds the given function for setting the style properties of the widget
// to [WidgetBase.Stylers.Normal]. It is one of the main ways to specify the styles of
// a widget, in addition to FirstStyler and FinalStyler, which add stylers that
// are called before and after the stylers added by this function, respectively.
func (wb *WidgetBase) Styler(s func(s *styles.Style)) {
	wb.Stylers.Normal = append(wb.Stylers.Normal, s)
}

// FirstStyler adds the given function for setting the style properties of the widget
// to [WidgetBase.Stylers.First]. It is one of the main ways to specify the styles of
// a widget, in addition to Styler and FinalStyler, which add stylers that
// are called after the stylers added by this function.
func (wb *WidgetBase) FirstStyler(s func(s *styles.Style)) {
	wb.Stylers.First = append(wb.Stylers.First, s)
}

// FinalStyler adds the given function for setting the style properties of the widget
// to [WidgetBase.Stylers.Final]. It is one of the main ways to specify the styles of
// a widget, in addition to FirstStyler and Styler, which add stylers that are called
// before the stylers added by this function.
func (wb *WidgetBase) FinalStyler(s func(s *styles.Style)) {
	wb.Stylers.Final = append(wb.Stylers.Final, s)
}

// Style updates the style properties of the widget based on [WidgetBase.Stylers].
// To specify the style properties of a widget, use [WidgetBase.Styler].
func (wb *WidgetBase) Style() {
	if wb.This == nil {
		return
	}

	pw := wb.parentWidget()

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
		setUnitContext(&wb.Styles, wb.Scene, wb.Geom.Size.Alloc.Content, psz)
	}()

	if wb.OverrideStyle {
		return
	}
	wb.resetStyleWidget()

	if pw != nil {
		wb.Styles.InheritFields(&pw.Styles)
	}

	wb.resetStyleSettings()
	wb.runStylers()
	wb.styleSettings()
}

// resetStyleWidget resets the widget styles and applies the basic
// default styles specified in [styles.Style.Defaults].
func (wb *WidgetBase) resetStyleWidget() {
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
	s.Font.Family = rich.SansSerif
}

// runStylers runs the [WidgetBase.Stylers].
func (wb *WidgetBase) runStylers() {
	wb.Stylers.Do(func(s *[]func(s *styles.Style)) {
		for _, f := range *s {
			f(&wb.Styles)
		}
	})
}

// resetStyleSettings reverses the effects of [WidgetBase.styleSettings]
// for the widget's font size so that it does not create cascading
// inhereted font size values. It only does this for non-root elements,
// as the root element must receive the larger font size so that
// all other widgets inherit it. It must be called before
// [WidgetBase.runStylers] and [WidgetBase.styleSettings].
func (wb *WidgetBase) resetStyleSettings() {
	if tree.IsRoot(wb) {
		return
	}
	fsz := AppearanceSettings.FontSize / 100
	wb.Styles.Font.Size.Value /= fsz
}

// styleSettings applies [AppearanceSettingsData.Spacing]
// and [AppearanceSettingsData.FontSize] to the style values for the widget.
func (wb *WidgetBase) styleSettings() {
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
}

// StyleTree calls [WidgetBase.Style] on every widget in tree
// underneath and including this widget.
func (wb *WidgetBase) StyleTree() {
	wb.WidgetWalkDown(func(cw Widget, cwb *WidgetBase) bool {
		cw.Style()
		return tree.Continue
	})
}

// Restyle ensures that the styling of the widget and all of its children
// is updated and rendered by calling [WidgetBase.StyleTree] and
// [WidgetBase.NeedsRender]. It does not trigger a new update or layout
// pass, so it should only be used for non-structural styling changes.
func (wb *WidgetBase) Restyle() {
	wb.StyleTree()
	wb.NeedsRender()
}

// setUnitContext sets the unit context based on size of scene, element, and parent
// element (from bbox) and then caches everything out in terms of raw pixel
// dots for rendering.
// Zero values for element and parent size are ignored.
func setUnitContext(st *styles.Style, sc *Scene, el, parent math32.Vector2) {
	var rc *renderContext
	sz := image.Point{1920, 1080}
	if sc != nil {
		rc = sc.renderContext()
		sz = sc.SceneGeom.Size
	}
	if rc != nil {
		st.UnitContext.DPI = rc.logicalDPI
	} else {
		st.UnitContext.DPI = 160
	}
	st.UnitContext.SetSizes(float32(sz.X), float32(sz.Y), el.X, el.Y, parent.X, parent.Y)
	st.Font.ToDots(&st.UnitContext) // key to set first
	st.Font.SetUnitContext(&st.UnitContext)
	st.ToDots()
}

// ChildBackground returns the background color (Image) for the given child Widget.
// By default, this is just our [styles.Style.ActualBackground] but it can be computed
// specifically for the child (e.g., for zebra stripes in [ListGrid])
func (wb *WidgetBase) ChildBackground(child Widget) image.Image {
	return wb.Styles.ActualBackground
}

// parentActualBackground returns the actual background of
// the parent of the widget. If it has no parent, it returns nil.
func (wb *WidgetBase) parentActualBackground() image.Image {
	pwb := wb.parentWidget()
	if pwb == nil {
		return nil
	}
	return pwb.This.(Widget).ChildBackground(wb.This.(Widget))
}

// setFromTag uses the given tags to call the given set function for the given tag.
func setFromTag(tags reflect.StructTag, tag string, set func(v float32)) {
	if v, ok := tags.Lookup(tag); ok {
		f, err := reflectx.ToFloat32(v)
		if errors.Log(err) == nil {
			set(f)
		}
	}
}

// styleFromTags adds a [WidgetBase.Styler] to the given widget
// to set its style properties based on the given [reflect.StructTag].
// Width, height, and grow properties are supported.
func styleFromTags(w Widget, tags reflect.StructTag) {
	w.AsWidget().Styler(func(s *styles.Style) {
		setFromTag(tags, "width", s.Min.X.Ch)
		setFromTag(tags, "max-width", s.Max.X.Ch)
		setFromTag(tags, "height", s.Min.Y.Em)
		setFromTag(tags, "max-height", s.Max.Y.Em)
		setFromTag(tags, "grow", func(v float32) { s.Grow.X = v })
		setFromTag(tags, "grow-y", func(v float32) { s.Grow.Y = v })
		setFromTag(tags, "icon-width", s.IconSize.X.Em)
		setFromTag(tags, "icon-height", s.IconSize.Y.Em)
	})
	if tags.Get("new-window") == "+" {
		w.AsWidget().setFlag(true, widgetValueNewWindow)
	}
}
