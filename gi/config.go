// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"log"

	"goki.dev/icons"
	"goki.dev/ki/v2"
)

// HasVp checks that the Vp Viewport has been set.
// Called prior to using -- logs an error if not.
// todo: need slog Debug mode for this kind of thing.
func (wb *WidgetBase) HasVp() bool {
	if wb.This() == nil || wb.Vp == nil {
		log.Printf("gi.WidgetBase.ReConfig: object or viewport is nil\n") // todo: slog.Debug
		return false
	}
	return true
}

// ReConfig is a convenience method for reconfiguring a widget after changes
// have been made.  In general it is more efficient to call Set* methods that
// automatically determine if Config is needed.
// The plain Config method is used during initial configuration,
// called by the Viewport and caches the Vp pointer.
func (wb *WidgetBase) ReConfig() {
	if !wb.HasVp() {
		return
	}
	wi := wb.This().(Widget)
	wi.Config(wb.Vp)
}

func (wb *WidgetBase) Config(vp *Viewport) {
	if wb.This() == nil {
		return
	}
	wi := wb.This().(Widget)
	updt := wi.UpdateStart()
	wb.Vp = vp
	wb.Style.Defaults()    // reset
	wb.LayState.Defaults() // doesn't overwrite
	wi.ConfigWidget(vp)    // where everything actually happens
	wi.SetStyle(vp)
	wb.UpdateEnd(updt)
	wb.SetNeedsLayout(vp, updt)
}

func (wb *WidgetBase) ConfigWidget(vp *Viewport) {
	// this must be defined for each widget type
}

// ConfigPartsIconLabel adds to config to create parts, of icon
// and label left-to right in a row, based on whether items are nil or empty
func (wb *WidgetBase) ConfigPartsIconLabel(config *ki.TypeAndNameList, icnm icons.Icon, txt string) (icIdx, lbIdx int) {
	if wb.Style.Template != "" {
		wb.Parts.Style.Template = wb.Style.Template + ".Parts"
	}
	icIdx = -1
	lbIdx = -1
	if TheIconMgr.IsValid(icnm) {
		icIdx = len(*config)
		config.Add(IconType, "icon")
		if txt != "" {
			config.Add(SpaceType, "space")
		}
	}
	if txt != "" {
		lbIdx = len(*config)
		config.Add(LabelType, "label")
	}
	return
}

// ConfigPartsSetIconLabel sets the icon and text values in parts, and get
// part style props, using given props if not set in object props
func (wb *WidgetBase) ConfigPartsSetIconLabel(icnm icons.Icon, txt string, icIdx, lbIdx int) {
	if icIdx >= 0 {
		ic := wb.Parts.Child(icIdx).(*Icon)
		if wb.Style.Template != "" {
			ic.Style.Template = wb.Style.Template + ".icon"
		}
		ic.SetIcon(icnm)
	}
	if lbIdx >= 0 {
		lbl := wb.Parts.Child(lbIdx).(*Label)
		if wb.Style.Template != "" {
			lbl.Style.Template = wb.Style.Template + ".icon"
		}
		if lbl.Text != txt {
			// avoiding SetText here makes it so label default
			// styles don't end up first, which is needed for
			// parent styles to override. However, there might have
			// been a reason for calling SetText, so we will see if
			// any bugs show up. TODO: figure out a good long-term solution for this.
			lbl.Text = txt
			// lbl.SetText(txt)
		}
	}
}
