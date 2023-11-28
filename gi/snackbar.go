// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"time"

	"goki.dev/colors"
	"goki.dev/girl/states"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/goosi/events"
	"goki.dev/icons"
)

var (
	// SnackbarTimeout is the default timeout for [SnackbarStage]s
	SnackbarTimeout = 7 * time.Second // todo: put in prefs
)

// NewSnackbar returns a new [Snackbar] Scene in the context of the
// given widget, optionally with the given name.
func NewSnackbar(ctx Widget, name ...string) *Scene {
	nm := ""
	if len(name) > 0 {
		nm = name[0]
	} else {
		nm = ctx.Name() + "-snackbar"
	}
	sc := NewScene(nm)
	sc.BgColor.SetSolid(colors.Transparent)
	sc.SnackbarStyles()

	sc.Stage = NewPopupStage(SnackbarStage, sc, ctx).SetTimeout(SnackbarTimeout)
	return sc
}

// ErrorSnackbar returns a new [Snackbar] displaying the given error
// in the context of the given widget.
func ErrorSnackbar(ctx Widget, err error) *Scene {
	return NewSnackbar(ctx, ctx.Name()+"-error-snackbar").AddSnackbarText("Error: " + err.Error())
}

func (sc *Scene) SnackbarStyles() {
	sc.Style(func(s *styles.Style) {
		s.Direction = styles.Row
		s.Overflow.Set(styles.OverflowVisible) // key for avoiding sizing errors when re-rendering with small pref size
		s.Border.Radius = styles.BorderRadiusExtraSmall
		s.Padding.SetHoriz(units.Dp(8))
		s.BackgroundColor.SetSolid(colors.Scheme.InverseSurface)
		s.Color = colors.Scheme.InverseOnSurface
		s.BoxShadow = styles.BoxShadow3()
		s.Align.Content = styles.Center
		s.Align.Items = styles.Center
		s.Gap.X.Dp(12)
		s.Grow.Set(1, 0)
		s.Min.Y.Dp(48)
	})
}

// AddSnackbarText adds a label with the given text to a snackbar
func (sc *Scene) AddSnackbarText(text string) *Scene {
	NewLabel(sc, "text").SetText(text).SetType(LabelBodyMedium).
		Style(func(s *styles.Style) {
			s.SetTextWrap(false)
			if s.Is(states.Selected) {
				s.Color = colors.Scheme.Select.OnContainer
			}
		})
	return sc
}

// AddSnackbarButton adds a button with the given text and optional OnClick
// event handler to the snackbar. Only the first of the given
// event handlers is used, and the popup is dismissed automatically
// regardless of whether there is an event handler passed.
func (sc *Scene) AddSnackbarButton(text string, onClick ...func(e events.Event)) *Scene {
	NewStretch(sc, "stretch")
	bt := NewButton(sc, "button").SetType(ButtonText).SetText(text)
	bt.Style(func(s *styles.Style) {
		s.Color = colors.Scheme.InversePrimary
	})
	bt.OnClick(func(e events.Event) {
		if len(onClick) > 0 {
			onClick[0](e)
		}
		sc.DeleteSnackbar()
	})
	return sc
}

// AddSnackbarIcon adds an icon button to the snackbar with the given icon and
// given OnClick event handler to the snackbar. Only the first of the given
// event handlers is used, and the popup is dismissed automatically
// regardless of whether there is an event handler passed.
func (sc *Scene) AddSnackbarIcon(icon icons.Icon, onClick ...func(e events.Event)) *Scene {
	ic := NewButton(sc, "icon").SetType(ButtonAction).SetIcon(icon)
	ic.Style(func(s *styles.Style) {
		s.Color = colors.Scheme.InverseOnSurface
	})
	ic.OnClick(func(e events.Event) {
		if len(onClick) > 0 {
			onClick[0](e)
		}
		sc.DeleteSnackbar()
	})
	return sc
}

// DeleteSnackbar deletes the popup associated with the snackbar.
func (sc *Scene) DeleteSnackbar() {
	sc.Stage.Context.AsWidget().Sc.Stage.Main.
		PopupMgr.PopDeleteType(SnackbarStage)
}
