// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"

	"goki.dev/girl/gist"
	"goki.dev/girl/units"
	"goki.dev/goosi"
	"goki.dev/goosi/mouse"
	"goki.dev/ki/v2"
	"goki.dev/mat32/v2"
)

// TooltipConfigStyles configures the default styles
// for the given tooltip frame with the given parent.
// It should be called on tooltips when they are created.
func TooltipConfigStyles(tooltip *Frame) {
	tooltip.AddStyler(func(w *WidgetBase, s *gist.Style) {
		s.Border.Style.Set(gist.BorderNone)
		s.Border.Radius = gist.BorderRadiusExtraSmall
		s.Padding.Set(units.Px(8 * Prefs.DensityMul()))
		s.BackgroundColor.SetSolid(ColorScheme.InverseSurface)
		s.Color = ColorScheme.InverseOnSurface
		s.BoxShadow = BoxShadow1 // STYTODO: not sure whether we should have this
	})
}

// PopupTooltip pops up a viewport displaying the tooltip text
func PopupTooltip(tooltip string, x, y int, parVp *Viewport, name string) *Viewport {
	win := parVp.Win
	mainVp := win.Viewport
	pvp := &Viewport{}
	pvp.Name = name + "Tooltip"
	pvp.Win = win
	pvp.Type = VpTooltip

	pvp.Frame.AddStyler(func(w *WidgetBase, s *gist.Style) {
		// TOOD: get border radius actually working
		// without having parent background color workaround
		s.Border.Radius = gist.BorderRadiusExtraSmall
		s.BackgroundColor = pvp.Frame.ParentBackgroundColor()
	})

	pvp.Geom.Pos = image.Point{x, y}
	pvp.SetFlag(true, VpPopupDestroyAll) // nuke it all

	frame := &pvp.Frame
	lbl := NewLabel(frame, "ttlbl")
	lbl.Text = tooltip
	lbl.Type = LabelBodyMedium

	TooltipConfigStyles(frame)

	lbl.AddStyler(func(w *WidgetBase, s *gist.Style) {
		mwdots := parVp.Frame.Style.UnContext.ToDots(40, units.UnitEm)
		mwdots = mat32.Min(mwdots, float32(mainVp.Geom.Size.X-20))

		s.MaxWidth.SetDot(mwdots)
	})

	frame.ConfigTree(pvp)
	frame.SetStyleTree(pvp) // sufficient to get sizes
	mainSz := mat32.NewVec2FmPoint(mainVp.Geom.Size)
	frame.LayState.Alloc.Size = mainSz // give it the whole vp initially
	frame.GetSizeTree(pvp, 0)          // collect sizes
	pvp.Win = nil
	vpsz := frame.LayState.Size.Pref.Min(mainSz).ToPoint()

	x = min(x, mainVp.Geom.Size.X-vpsz.X) // fit
	y = min(y, mainVp.Geom.Size.Y-vpsz.Y) // fit
	pvp.Resize(vpsz)
	pvp.Geom.Pos = image.Point{x, y}

	// win.PushPopup(pvp)
	return pvp
}

// HoverTooltipEvent connects to HoverEvent and pops up a tooltip -- most
// widgets should call this as part of their event connection method
func (wb *WidgetBase) HoverTooltipEvent() {
	wb.ConnectEvent(goosi.MouseHoverEvent, RegPri, func(recv, send ki.Ki, sig int64, d any) {
		me := d.(*mouse.HoverEvent)
		wbb := AsWidgetBase(recv)
		if wbb.Tooltip != "" {
			me.SetProcessed()
			pos := wbb.WinBBox.Max
			pos.X -= 20
			mvp := wbb.Vp
			PopupTooltip(wbb.Tooltip, pos.X, pos.Y, mvp, wbb.Nm)
		}
	})
}
