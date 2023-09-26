// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"

	"goki.dev/colors"
	"goki.dev/girl/gist"
	"goki.dev/icons"
	"goki.dev/ki/v2"
	"goki.dev/svg"
	"golang.org/x/image/draw"
)

// // SetIcon sets the icon by name into given Icon wrapper, returning error
// // message if not found etc, and returning true if a new icon was actually set
// // -- does nothing if IconNm is already == icon name and has children, and deletes
// // children if name is nil / none (both cases return false for new icon)
// func (inm IconName) SetIcon(ic *Icon) (bool, error) {
// 	return ic.SetIcon(string(inm))
// }

// // IsNil tests whether the icon name is empty, 'none' or 'nil' -- indicates to
// // not use a icon
// func (inm IconName) IsNil() bool {
// 	return inm == "" || inm == "none" || inm == "nil"
// }

// // IsValid tests whether the icon name is valid -- represents a non-nil icon
// // available in the current or default icon set
// func (inm IconName) IsValid() bool {
// 	return TheIconMgr.IsValid(string(inm))
// }

// Icon contains a svg.SVG element.
// The rendered version is cached for a given size.
type Icon struct {
	WidgetBase

	// icon name that has been set -- optimizes to prevent reloading of icon
	IconNm icons.Icon `desc:"icon name that has been set -- optimizes to prevent reloading of icon"`

	// file name for the loaded icon, if loaded
	Filename string `desc:"file name for the loaded icon, if loaded"`

	// SVG drawing
	SVG svg.SVG `desc:"SVG drawing"`
}

// event functions for this type
var IconEventFuncs WidgetEvents

func (ic *Icon) OnInit() {
	ic.AddEvents(&IconEventFuncs)
	ic.AddStyler(func(w *WidgetBase, s *gist.Style) {
		s.Width.SetEm(1)
		s.Height.SetEm(1)
		s.BackgroundColor.SetSolid(colors.Transparent)
	})
}

func (ic *Icon) CopyFieldsFrom(frm any) {
	fr := frm.(*Icon)
	ic.WidgetBase.CopyFieldsFrom(&fr.WidgetBase)
	ic.IconNm = fr.IconNm
	ic.Filename = fr.Filename
}

// SetIcon sets the icon by name into given Icon wrapper, returning error
// message if not found etc, and returning true if a new icon was actually set
// -- does nothing if IconNm is already == icon name and has children, and deletes
// children if name is nil / none (both cases return false for new icon)
func (ic *Icon) SetIcon(name icons.Icon) (bool, error) {
	if name.IsNil() {
		ic.SVG.DeleteAll()
		return false, nil
	}
	if ic.HasChildren() && ic.IconNm == name {
		return false, nil
	}
	// pr := prof.Start("IconSetIcon")
	// pr.End()
	err := TheIconMgr.SetIcon(ic, name)
	if err == nil {
		ic.IconNm = name
		return true, nil
	}
	return false, err
}

func (ic *Icon) GetSize(sc *Scene, iter int) {
	if iter > 0 {
		return
	}
	// ic.SVG.Nm = ic.Nm
	// ic.LayState.Alloc.Size = sic.LayState.Alloc.Size
}

func (ic *Icon) SetStyle(sc *Scene) {
	ic.StyMu.Lock()
	defer ic.StyMu.Unlock()

	ic.SetStyleWidget(sc)
	ic.LayState.SetFromStyle(&ic.Style) // also does reset
	// todo: set ic.SVG style
}

func (ic *Icon) DoLayout(sc *Scene, parBBox image.Rectangle, iter int) bool {
	ic.DoLayoutBase(sc, parBBox, true, iter)
	return ic.DoLayoutChildren(sc, iter)
}

func (ic *Icon) DrawIntoScene(sc *Scene) {
	if ic.SVG.Pixels == nil {
		return
	}
	pos := ic.LayState.Alloc.Pos.ToPointCeil()
	max := pos.Add(ic.LayState.Alloc.Size.ToPointCeil())
	r := image.Rectangle{Min: pos, Max: max}
	sp := image.Point{}
	if ic.Par != nil { // use parents children bbox to determine where we can draw
		pni, _ := AsWidget(ic.Par)
		pbb := pni.ChildrenBBoxes(sc)
		nr := r.Intersect(pbb)
		sp = nr.Min.Sub(r.Min)
		if sp.X < 0 || sp.Y < 0 || sp.X > 10000 || sp.Y > 10000 {
			fmt.Printf("aberrant sp: %v\n", sp)
			return
		}
		r = nr
	}
	draw.Draw(sc.Pixels, r, ic.SVG.Pixels, sp, draw.Over)
}

func (ic *Icon) FilterEvents() {
	ic.Events.CopyFrom(&IconEventFuncs)
}

func (ic *Icon) Render(sc *Scene) {
	// todo: cache rendered size, update render if diff size..

	wi := ic.This().(Widget)
	if ic.PushBounds(sc) {
		wi.FilterEvents()
		ic.RenderChildren(sc)
		ic.DrawIntoScene(sc)
		ic.PopBounds(sc)
	}
}

////////////////////////////////////////////////////////////////////////////////////////
//  IconMgr

// IconMgr is the manager of all things Icon -- needed to allow svg to be a
// separate package, and implemented by svg.IconMgr
type IconMgr interface {
	// IsValid checks if given icon name is a valid name for an available icon
	// (also checks that the icon manager is non-nil and issues appropriate error)
	IsValid(iconName icons.Icon) bool

	// SetIcon sets the icon by name into given Icon wrapper, returning error
	// message if not found etc.  This is how gi.Icon is initialized from
	// underlying svg.Icon items.
	SetIcon(ic *Icon, iconName icons.Icon) error

	// IconByName is main function to get icon by name -- looks in CurIconSet and
	// falls back to DefaultIconSet if not found there -- returns error
	// message if not found.  cast result to *svg.Icon
	IconByName(name icons.Icon) (ki.Ki, error)

	// IconList returns the list of available icon names, optionally sorted
	// alphabetically (otherwise in map-random order)
	IconList(alphaSort bool) []icons.Icon
}

// TheIconMgr is set by loading the gi/svg package -- all final users must
// import github/goki/gi/svg to get its init function
var TheIconMgr IconMgr

// CurIconList holds the current icon list, alpha sorted -- set at startup
var CurIconList []icons.Icon
