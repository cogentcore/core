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
	s := &Snackbar{}
	nm := ""
	if len(name) > 0 {
		nm = name[0]
	} else {
		nm = ctx.Name() + "-dialog"
	}

	s.InitName(s, nm)
	s.EventMgr.Scene = &s.Scene
	s.BgColor.SetSolid(colors.Transparent)
	s.Lay = LayoutVert

	s.Scene.Stage = NewPopupStage(SnackbarStage, &s.Scene, ctx)
	return s
}

// NewSnackbarScene returns a new snackbar scene based on the given context widget
// and options.
func NewSnackbarScene(w Widget, opts SnackbarOpts) *Scene {
	sc := NewScene(w.Name() + "-snackbar")
	sc.SetLayout(LayoutHoriz)
	wsc := w.AsWidget().Sc
	sc.Style(func(s *styles.Style) {
		s.Border.Radius = styles.BorderRadiusExtraSmall
		s.Padding.SetHoriz(units.Dp(8))
		s.BackgroundColor.SetSolid(colors.Scheme.InverseSurface)
		s.Color = colors.Scheme.InverseOnSurface
		s.BoxShadow = styles.BoxShadow3()
		s.AlignV = styles.AlignMiddle
		sc.Spacing.Dp(12)
		s.SetStretchMaxWidth()
		s.Height = units.Dp(48)
	})
	NewLabel(sc, "text").SetText(opts.Text).SetType(LabelBodyMedium).
		Style(func(s *styles.Style) {
			s.Text.WhiteSpace = styles.WhiteSpaceNowrap
			if s.Is(states.Selected) {
				s.Color = colors.Scheme.Select.OnContainer
			}
		})
	if opts.Button != "" || !opts.Icon.IsNil() {
		NewStretch(sc, "stretch")
	}
	if opts.Button != "" {
		bt := NewButton(sc, "button").SetType(ButtonText).SetText(opts.Button)
		bt.Style(func(s *styles.Style) {
			s.Color = colors.Scheme.InversePrimary
		})
		bt.OnClick(func(e events.Event) {
			if opts.ButtonOnClick != nil {
				opts.ButtonOnClick(bt)
			}
			wsc.MainStage().PopupMgr.PopDeleteType(SnackbarStage)
		})
	}
	if !opts.Icon.IsNil() {
		ic := NewButton(sc, "icon").SetType(ButtonAction).SetIcon(opts.Icon)
		ic.Style(func(s *styles.Style) {
			s.Color = colors.Scheme.InverseOnSurface
		})
		ic.OnClick(func(e events.Event) {
			if opts.IconOnClick != nil {
				opts.IconOnClick(ic)
			}
			wsc.MainStage().PopupMgr.PopDeleteType(SnackbarStage)
		})
	}
	return sc
}
