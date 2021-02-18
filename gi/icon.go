// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"
	"image/color"

	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/mat32"
)

// SetIcon sets the icon by name into given Icon wrapper, returning error
// message if not found etc, and returning true if a new icon was actually set
// -- does nothing if IconNm is already == icon name and has children, and deletes
// children if name is nil / none (both cases return false for new icon)
func (inm IconName) SetIcon(ic *Icon) (bool, error) {
	return ic.SetIcon(string(inm))
}

// IsNil tests whether the icon name is empty, 'none' or 'nil' -- indicates to
// not use a icon
func (inm IconName) IsNil() bool {
	return inm == "" || inm == "none" || inm == "nil"
}

// IsValid tests whether the icon name is valid -- represents a non-nil icon
// available in the current or default icon set
func (inm IconName) IsValid() bool {
	return TheIconMgr.IsValid(string(inm))
}

// Icon is a wrapper around a child svg.Icon SVG element.  SVG should contain no
// color information -- it should just be a filled shape where the fill and
// stroke colors come from the surrounding context / paint settings.  The
// rendered version is cached for a given size. Icons are always copied from
// an original source icon and then can be customized from there.
type Icon struct {
	WidgetBase
	IconNm   string `desc:"icon name that has been set -- optimizes to prevent reloading of icon"`
	Filename string `desc:"file name for the loaded icon, if loaded"`
}

var KiT_Icon = kit.Types.AddType(&Icon{}, IconProps)

// AddNewIcon adds a new icon to given parent node, with given name, and icon name.
func AddNewIcon(parent ki.Ki, name string, icon string) *Icon {
	ic := parent.AddNewChild(KiT_Icon, name).(*Icon)
	ic.SetIcon(icon)
	return ic
}

func (ic *Icon) CopyFieldsFrom(frm interface{}) {
	fr := frm.(*Icon)
	ic.WidgetBase.CopyFieldsFrom(&fr.WidgetBase)
	ic.Filename = fr.Filename
}

var IconProps = ki.Props{
	"EnumType:Flag":    KiT_NodeFlags,
	"background-color": color.Transparent,
}

// SetIcon sets the icon by name into given Icon wrapper, returning error
// message if not found etc, and returning true if a new icon was actually set
// -- does nothing if IconNm is already == icon name and has children, and deletes
// children if name is nil / none (both cases return false for new icon)
func (ic *Icon) SetIcon(name string) (bool, error) {
	if IconName(name).IsNil() {
		ic.DeleteChildren(ki.DestroyKids)
		return false, nil
	}
	if ic.HasChildren() && ic.IconNm == name {
		return false, nil
	}
	// pr := prof.Start("IconSetIcon")
	// pr.End()
	err := TheIconMgr.SetIcon(ic, name)
	if err == nil {
		ic.IconNm = string(name)
		return true, nil
	}
	return false, err
}

// SVGIcon returns the child svg icon, or nil
func (ic *Icon) SVGIcon() *Viewport2D {
	if !ic.HasChildren() {
		return nil
	}
	sic := ic.Child(0).Embed(KiT_Viewport2D).(*Viewport2D)
	return sic
}

func (ic *Icon) Size2D(iter int) {
	if iter > 0 {
		return
	}
	sic := ic.SVGIcon()
	if sic != nil {
		sic.Nm = ic.Nm
		ic.LayState.Alloc.Size = sic.LayState.Alloc.Size
	}
}

func (ic *Icon) Style2D() {
	ic.StyMu.Lock()
	defer ic.StyMu.Unlock()

	hasTempl, saveTempl := ic.Sty.FromTemplate()
	if !hasTempl || saveTempl {
		ic.Style2DWidget()
	}
	if hasTempl && saveTempl {
		ic.Sty.SaveTemplate()
	}
	ic.LayState.SetFromStyle(&ic.Sty.Layout) // also does reset
	sic := ic.SVGIcon()
	if sic != nil {
		sic.Nm = ic.Nm
		sic.Props = ic.Props
		sic.CSS = ic.CSS
		sic.Sty = ic.Sty
		sic.DefStyle = ic.DefStyle
		if ic.NeedsFullReRender() {
			sic.SetFullReRender()
		}
	}
}

func (ic *Icon) Layout2D(parBBox image.Rectangle, iter int) bool {
	sic := ic.SVGIcon()
	ic.Layout2DBase(parBBox, true, iter)
	if sic != nil {
		sic.LayState = ic.LayState
		sic.LayState.Alloc.PosRel = mat32.Vec2Zero
	}
	return ic.Layout2DChildren(iter)
}

func (ic *Icon) Render2D() {
	if ic.FullReRenderIfNeeded() {
		return
	}
	if ic.PushBounds() {
		ic.Render2DChildren()
		ic.PopBounds()
	}
}

////////////////////////////////////////////////////////////////////////////////////////
//  IconMgr

// IconMgr is the manager of all things Icon -- needed to allow svg to be a
// separate package, and implemented by svg.IconMgr
type IconMgr interface {
	// IsValid checks if given icon name is a valid name for an available icon
	// (also checks that the icon manager is non-nil and issues appropriate error)
	IsValid(iconName string) bool

	// SetIcon sets the icon by name into given Icon wrapper, returning error
	// message if not found etc.  This is how gi.Icon is initialized from
	// underlying svg.Icon items.
	SetIcon(ic *Icon, iconName string) error

	// IconList returns the list of available icon names, optionally sorted
	// alphabetically (otherwise in map-random order)
	IconList(alphaSort bool) []IconName
}

// TheIconMgr is set by loading the gi/svg package -- all final users must
// import github/goki/gi/svg to get its init function
var TheIconMgr IconMgr

// CurIconList holds the current icon list, alpha sorted -- set at startup
var CurIconList []IconName
