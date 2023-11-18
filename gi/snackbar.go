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
	"goki.dev/mat32/v2"
)

var (
	// SnackbarTimeout is the default timeout for [SnackbarStage]s
	SnackbarTimeout = 7 * time.Second // todo: put in prefs
)

// Snackbar is a scene with methods for configuring a snackbar
type Snackbar struct { //goki:no-new
	Scene
}

// NewSnackbar returns a new [Snackbar] in the context of the given widget,
// optionally with the given name.
func NewSnackbar(ctx Widget, name ...string) *Snackbar {
	sb := &Snackbar{}
	nm := ""
	if len(name) > 0 {
		nm = name[0]
	} else {
		nm = ctx.Name() + "-snackbar"
	}

	sb.InitName(sb, nm)
	sb.EventMgr.Scene = &sb.Scene
	sb.BgColor.SetSolid(colors.Transparent)
	sb.SnackbarStyles()

	sb.Stage = NewPopupStage(SnackbarStage, &sb.Scene, ctx)
	sb.SetTimeout(SnackbarTimeout)
	return sb
}

// ErrorSnackbar returns a new [Snackbar] displaying the given error
// in the context of the given widget.
func ErrorSnackbar(ctx Widget, err error) *Snackbar {
	return NewSnackbar(ctx, ctx.Name()+"-error-snackbar").Text("Error: " + err.Error())
}

func (sb *Snackbar) SnackbarStyles() {
	sb.Style(func(s *styles.Style) {
		s.SetMainAxis(mat32.X)
		s.Overflow.Set(styles.OverflowVisible) // key for avoiding sizing errors when re-rendering with small pref size
		s.Border.Radius = styles.BorderRadiusExtraSmall
		s.Padding.SetHoriz(units.Dp(8))
		s.BackgroundColor.SetSolid(colors.Scheme.InverseSurface)
		s.Color = colors.Scheme.InverseOnSurface
		s.BoxShadow = styles.BoxShadow3()
		s.Align.Y = styles.AlignCenter
		s.Text.AlignV = styles.AlignCenter
		s.Gap.X.Dp(12)
		s.Grow.Set(1, 0)
		s.Min.Y.Dp(48)
	})
}

// Text adds a label with the given text to the snackbar
func (sb *Snackbar) Text(text string) *Snackbar {
	NewLabel(sb, "text").SetText(text).SetType(LabelBodyMedium).
		Style(func(s *styles.Style) {
			s.SetTextWrap(false)
			if s.Is(states.Selected) {
				s.Color = colors.Scheme.Select.OnContainer
			}
		})
	return sb
}

// Button adds a button with the given text and optional OnClick
// event handler to the snackbar. Only the first of the given
// event handlers is used, and the popup is dismissed automatically
// regardless of whether there is an event handler passed.
func (sb *Snackbar) Button(text string, onClick ...func(e events.Event)) *Snackbar {
	NewStretch(sb, "stretch")
	bt := NewButton(sb, "button").SetType(ButtonText).SetText(text)
	bt.Style(func(s *styles.Style) {
		s.Color = colors.Scheme.InversePrimary
	})
	bt.OnClick(func(e events.Event) {
		if len(onClick) > 0 {
			onClick[0](e)
		}
		sb.DeletePopup()
	})
	return sb
}

// Icon adds an icon button to the snackbar with the given icon and
// given OnClick event handler to the snackbar. Only the first of the given
// event handlers is used, and the popup is dismissed automatically
// regardless of whether there is an event handler passed.
func (sb *Snackbar) Icon(icon icons.Icon, onClick ...func(e events.Event)) *Snackbar {
	ic := NewButton(sb, "icon").SetType(ButtonAction).SetIcon(icon)
	ic.Style(func(s *styles.Style) {
		s.Color = colors.Scheme.InverseOnSurface
	})
	ic.OnClick(func(e events.Event) {
		if len(onClick) > 0 {
			onClick[0](e)
		}
		sb.DeletePopup()
	})
	return sb
}

// SetTimeout sets the timeout of the snackbar
func (sb *Snackbar) SetTimeout(timeout time.Duration) *Snackbar {
	sb.Stage.AsPopup().SetTimeout(timeout)
	return sb
}

// Run runs (shows) the snackbar.
func (sb *Snackbar) Run() *Snackbar {
	sb.Stage.Run()
	return sb
}

// Delete deletes the popup associated with the snackbar.
func (sb *Snackbar) DeletePopup() {
	sb.Stage.AsPopup().CtxWidget.AsWidget().Sc.MainStage().PopupMgr.PopDeleteType(SnackbarStage)
}
