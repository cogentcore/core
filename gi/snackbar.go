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

// Snackbar is a scene with methods for configuring a snackbar
type Snackbar struct { //goki:no-new
	Scene

	// Stage is the popup stage associated with the dialog
	Stage *PopupStage

	// // The main text message to display in the snackbar
	// Text string
	// // If not "", the text of an button to display in the snackbar
	// Button string
	// // If non-nil, the function to call when the main text button
	// // in the snackbar is clicked
	// ButtonOnClick func(bt *Button)
	// // If not [icons.None], the icon to display as a button
	// // on the right side of the snack bar
	// Icon icons.Icon
	// // If non-nil, the function to call when the icon button in the
	// // snackabr is clicked
	// IconOnClick func(bt *Button)
}

// // NewSnackbarFromScene returns a new Snackbar stage with given scene contents,
// // in connection with given widget (which provides key context).
// // Make further configuration choices using Set* methods, which
// // can be chained directly after the New call.
// // Use an appropriate Run call at the end to start the Stage running.
// func NewSnackbarFromScene(sc *Scene, ctx Widget) *PopupStage {
// 	return NewPopupStage(SnackbarStage, sc, ctx).SetTimeout(SnackbarTimeout)
// }

// NewSnackbar returns a new [Snackbar] in the context of the given widget,
// optionally with the given name.
func NewSnackbar(ctx Widget, name ...string) *Snackbar {
	sb := &Snackbar{}
	nm := ""
	if len(name) > 0 {
		nm = name[0]
	} else {
		nm = ctx.Name() + "-dialog"
	}

	sb.InitName(sb, nm)
	sb.EventMgr.Scene = &sb.Scene
	sb.BgColor.SetSolid(colors.Transparent)
	sb.Lay = LayoutHoriz
	sb.SnackbarStyles()

	sb.Stage = NewPopupStage(SnackbarStage, &sb.Scene, ctx)
	return sb
}

func (sb *Snackbar) SnackbarStyles() {
	sb.Style(func(s *styles.Style) {
		s.Border.Radius = styles.BorderRadiusExtraSmall
		s.Padding.SetHoriz(units.Dp(8))
		s.BackgroundColor.SetSolid(colors.Scheme.InverseSurface)
		s.Color = colors.Scheme.InverseOnSurface
		s.BoxShadow = styles.BoxShadow3()
		s.AlignV = styles.AlignMiddle
		sb.Spacing.Dp(12)
		s.SetStretchMaxWidth()
		s.Height = units.Dp(48)
	})
}

// Text adds a label with the given text to the snackbar
func (sb *Snackbar) Text(text string) *Snackbar {
	NewLabel(sb, "text").SetText(text).SetType(LabelBodyMedium).
		Style(func(s *styles.Style) {
			s.Text.WhiteSpace = styles.WhiteSpaceNowrap
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

// Run runs (shows) the snackbar.
func (sb *Snackbar) Run() {
	sb.Stage.Run()
}

// Delete deletes the popup associated with the snackbar.
func (sb *Snackbar) DeletePopup() {
	sb.Stage.CtxWidget.AsWidget().Sc.MainStage().PopupMgr.PopDeleteType(SnackbarStage)
}
