// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"cogentcore.org/core/colors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/states"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/units"
)

// NewSnackbar returns a new [SnackbarStage] in the context
// of the given widget.
func (bd *Body) NewSnackbar(ctx Widget) *Stage {
	ctx = NonNilContext(ctx)
	bd.SnackbarStyles()
	bd.Scene.Stage = NewPopupStage(SnackbarStage, bd.Scene, ctx).
		SetTimeout(SystemSettings.SnackbarTimeout)
	return bd.Scene.Stage
}

// MessageSnackbar opens a [Snackbar] displaying the given message
// in the context of the given widget.
func MessageSnackbar(ctx Widget, msg string) {
	NewBody().AddSnackbarText(msg).NewSnackbar(ctx).Run()
}

// ErrorSnackbar opens a [Snackbar] displaying the given error
// in the context of the given widget. Optional label text can be
// provided; if it is not, the label text will default to "Error".
// If the given error is nil, no snackbar. is created.
func ErrorSnackbar(ctx Widget, err error, label ...string) {
	if err == nil {
		return
	}
	lbl := "Error"
	if len(label) > 0 {
		lbl = label[0]
	}
	MessageSnackbar(ctx, lbl+": "+err.Error())
}

// SnackbarStyles sets default stylers for snackbar bodies.
// It is automatically called in [Body.NewSnackbar].
func (bd *Body) SnackbarStyles() {
	bd.Style(func(s *styles.Style) {
		s.Direction = styles.Row
		s.Overflow.Set(styles.OverflowVisible) // key for avoiding sizing errors when re-rendering with small pref size
		s.Border.Radius = styles.BorderRadiusExtraSmall
		s.Padding.SetHoriz(units.Dp(16))
		s.Background = colors.C(colors.Scheme.InverseSurface)
		s.Color = colors.Scheme.InverseOnSurface
		// we go on top of things so we want no margin background
		s.FillMargin = false
		s.Align.Content = styles.Center
		s.Align.Items = styles.Center
		s.Gap.X.Dp(12)
		s.Grow.Set(1, 0)
		s.Min.Y.Dp(48)
		s.Min.X.SetCustom(func(uc *units.Context) float32 {
			return min(uc.Em(20), uc.Vw(70))
		})
	})
	bd.Scene.Style(func(s *styles.Style) {
		s.Background = nil
		s.Border.Radius = styles.BorderRadiusExtraSmall
		s.BoxShadow = styles.BoxShadow3()
	})
}

// AddSnackbarText adds a snackbar label with the given text
func (bd *Body) AddSnackbarText(text string) *Body {
	NewLabel(bd, "text").SetText(text).SetType(LabelBodyMedium).
		Style(func(s *styles.Style) {
			s.SetTextWrap(false)
			if s.Is(states.Selected) {
				s.Color = colors.Scheme.Select.OnContainer
			}
		})
	return bd
}

// AddSnackbarButton adds a snackbar button with the given text and optional OnClick
// event handler. Only the first of the given event handlers is used, and the
// snackbar is automatically closed when the button is clicked regardless of
// whether there is an event handler passed.
func (bd *Body) AddSnackbarButton(text string, onClick ...func(e events.Event)) *Body {
	NewStretch(bd, "stretch")
	bt := NewButton(bd, "button").SetType(ButtonText).SetText(text)
	bt.Style(func(s *styles.Style) {
		s.Color = colors.Scheme.InversePrimary
	})
	bt.OnClick(func(e events.Event) {
		if len(onClick) > 0 {
			onClick[0](e)
		}
		bd.Close()
	})
	return bd
}

// AddSnackbarIcon adds a snackbar icon button with the given icon and optional
// OnClick event handler. Only the first of the given event handlers is used, and the
// snackbar is automatically closed when the button is clicked regardless of whether
// there is an event handler passed.
func (bd *Body) AddSnackbarIcon(icon icons.Icon, onClick ...func(e events.Event)) *Body {
	ic := NewButton(bd, "icon").SetType(ButtonAction).SetIcon(icon)
	ic.Style(func(s *styles.Style) {
		s.Color = colors.Scheme.InverseOnSurface
	})
	ic.OnClick(func(e events.Event) {
		if len(onClick) > 0 {
			onClick[0](e)
		}
		bd.Close()
	})
	return bd
}
