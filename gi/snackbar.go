// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"

	"goki.dev/colors"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/icons"
)

// SnackbarOpts contains the options used to create a snackbar.
type SnackbarOpts struct {
	Text          string
	Action        *ActOpts
	ActionOnClick func(act *Action)
	Icon          icons.Icon
	IconOnClick   func(act *Action)
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
	})
	NewLabel(sc, "text").SetText(opts.Text).SetType(LabelBodyMedium)
	// TODO: other elements
	return sc
}
