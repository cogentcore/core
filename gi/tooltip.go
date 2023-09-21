// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

// TooltipConfigStyles configures the default styles
// for the given tooltip frame with the given parent.
// It should be called on tooltips when they are created.
func TooltipConfigStyles(par *WidgetBase, tooltip *Frame) {
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
	pvp := Viewport{}
	pvp.InitName(&pvp, name+"Tooltip")
	pvp.Win = win
	updt := pvp.UpdateStart()
	pvp.Fill = true
	pvp.SetFlag(int(VpFlagPopup))
	pvp.SetFlag(int(VpFlagTooltip))
	pvp.AddStyler(func(w *WidgetBase, s *gist.Style) {
		// TOOD: get border radius actually working
		// without having parent background color workaround

		s.Border.Radius = gist.BorderRadiusExtraSmall
		s.BackgroundColor = pvp.ParentBackgroundColor()
	})

	pvp.Geom.Pos = image.Point{x, y}
	pvp.SetFlag(int(VpFlagPopupDestroyAll)) // nuke it all
	frame := NewFrame(&pvp, "Frame", LayoutVert)
	lbl := NewLabel(frame, "ttlbl", tooltip)
	lbl.Type = LabelBodyMedium

	TooltipConfigStyles(&pvp.WidgetBase, frame)

	lbl.AddStyler(func(w *WidgetBase, s *gist.Style) {
		mwdots := parVp.Style.UnContext.ToDots(40, units.UnitEm)
		mwdots = mat32.Min(mwdots, float32(mainVp.Geom.Size.X-20))

		s.MaxWidth.SetDot(mwdots)
	})

	frame.ConfigTree()
	frame.SetStyleTree()                                   // sufficient to get sizes
	frame.LayState.Alloc.Size = mainVp.LayState.Alloc.Size // give it the whole vp initially
	frame.GetSizeTree(0)                                   // collect sizes
	pvp.Win = nil
	vpsz := frame.LayState.Size.Pref.Min(mainVp.LayState.Alloc.Size).ToPoint()

	x = min(x, mainVp.Geom.Size.X-vpsz.X) // fit
	y = min(y, mainVp.Geom.Size.Y-vpsz.Y) // fit
	pvp.Resize(vpsz)
	pvp.Geom.Pos = image.Point{x, y}
	pvp.UpdateEndNoSig(updt)

	win.PushPopup(pvp.This())
	return &pvp
}

// HoverTooltipEvent connects to HoverEvent and pops up a tooltip -- most
// widgets should call this as part of their event connection method
func (wb *WidgetBase) HoverTooltipEvent() {
	wb.ConnectEvent(goosi.MouseHoverEvent, RegPri, func(recv, send ki.Ki, sig int64, d any) {
		me := d.(*mouse.HoverEvent)
		wbb := recv.Embed(TypeWidgetBase).(*WidgetBase)
		if wbb.Tooltip != "" {
			me.SetProcessed()
			pos := wbb.WinBBox.Max
			pos.X -= 20
			mvp := wbb.ViewportSafe()
			PopupTooltip(wbb.Tooltip, pos.X, pos.Y, mvp, wbb.Nm)
		}
	})
}
