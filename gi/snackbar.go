// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"

	"goki.dev/colors"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/goosi/events"
	"goki.dev/icons"
)

// SnackbarOpts contains the options used to create a snackbar.
type SnackbarOpts struct {
	// The main text message to display in the snackbar
	Text string
	// If not "", the text of an action to display in the snackbar
	Action string
	// If non-nil, the function to call when the main text action
	// in the snackbar is clicked
	ActionOnClick func(ac *Action)
	// If not [icons.None], the icon to display as an action
	// on the right side of the snack bar
	Icon icons.Icon
	// If non-nil, the function to call when the icon action in the
	// snackabr is clicked
	IconOnClick func(ac *Action)
}

// NewSnackbarFromScene returns a new Snackbar stage with given scene contents,
// in connection with given widget (which provides key context).
// Make further configuration choices using Set* methods, which
// can be chained directly after the New call.
// Use an appropriate Run call at the end to start the Stage running.
func NewSnackbarFromScene(sc *Scene, ctx Widget) *PopupStage {
	return NewPopupStage(Snackbar, sc, ctx)
}

// NewSnackbar returns a new snackbar based on the given context widget and options.
func NewSnackbar(w Widget, opts SnackbarOpts) *PopupStage {
	return NewSnackbarFromScene(NewSnackbarScene(w, opts), w)
}

// NewSnackbarScene returns a new snackbar scene based on the given context widget
// and options.
func NewSnackbarScene(w Widget, opts SnackbarOpts) *Scene {
	sc := StageScene(w.Name() + "-snackbar")
	sc.SetLayout(LayoutHoriz)
	wsc := w.AsWidget().Sc
	wscm := wsc.Geom.Bounds().Max
	// TODO: improve positioning and sizing
	sc.Geom.Pos = wscm.Sub(image.Pt(3*wscm.X/4, 50))
	sc.AddStyles(func(s *styles.Style) {
		s.Border.Radius = styles.BorderRadiusExtraSmall
		s.Padding.Set(units.Dp(8 * Prefs.DensityMul()))
		s.BackgroundColor.SetSolid(colors.Scheme.InverseSurface)
		s.Color = colors.Scheme.InverseOnSurface
		s.BoxShadow = styles.BoxShadow3()
		s.AlignV = styles.AlignMiddle
		sc.Spacing.SetDp(12 * Prefs.DensityMul())
		s.Height.SetDp(40)
		s.Width.SetVw(100)
	})
	NewLabel(sc, "text").SetText(opts.Text).SetType(LabelBodyMedium)
	if opts.Action != "" || !opts.Icon.IsNil() {
		NewStretch(sc, "stretch")
	}
	if opts.Action != "" {
		ac := NewAction(sc, "action").SetType(ActionParts)
		ac.SetText(opts.Action)
		ac.AddStyles(func(s *styles.Style) {
			s.Color = colors.Scheme.InversePrimary
		})

		ac.On(events.Click, func(e events.Event) {
			if opts.ActionOnClick != nil {
				opts.ActionOnClick(ac)
			}
			wsc.MainStage().PopupMgr.PopDeleteType(Snackbar)
		})
	}
	if !opts.Icon.IsNil() {
		ic := NewAction(sc, "icon").SetType(ActionParts)
		ic.SetIcon(opts.Icon)
		ic.AddStyles(func(s *styles.Style) {
			s.Color = colors.Scheme.InverseOnSurface
		})
		ic.On(events.Click, func(e events.Event) {
			if opts.IconOnClick != nil {
				opts.IconOnClick(ic)
			}
			wsc.MainStage().PopupMgr.PopDeleteType(Snackbar)
		})
	}
	return sc
}
