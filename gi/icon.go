// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"
	"log"

	"goki.dev/colors"
	"goki.dev/girl/gist"
	"goki.dev/icons"
	"goki.dev/ki/v2"
	"goki.dev/svg"
	"golang.org/x/image/draw"
)

// Icon contains a svg.SVG element.
// The rendered version is cached for a given size.
type Icon struct {
	WidgetBase

	// icon name that has been set.
	IconName icons.Icon

	// file name for the loaded icon, if loaded
	Filename string

	// SVG drawing
	SVG svg.SVG

	// RendSize is the last rendered size of the Icon SVG.
	// if the SVG.Name == IconName and this size is the same
	// then the current SVG image is used.
	RendSize image.Point
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
	ic.IconName = fr.IconName
	ic.Filename = fr.Filename
}

// SetIcon sets the icon by name into given Icon wrapper, returning error
// message if not found etc, and returning true if a new icon was actually set
// -- does nothing if IconName is already == icon name and has children, and deletes
// children if name is nil / none (both cases return false for new icon)
func (ic *Icon) SetIcon(name icons.Icon) (bool, error) {
	if name.IsNil() {
		ic.SVG.DeleteAll()
		return false, nil
	}
	if ic.SVG.Root.HasChildren() && ic.IconName == name {
		return false, nil
	}
	fnm := name.Filename()
	ic.SVG.Config(2, 2)
	err := ic.SVG.OpenFS(icons.Icons, fnm)
	if err != nil {
		log.Println("error opening icon named:", fnm, err)
	}
	// pr := prof.Start("IconSetIcon")
	// pr.End()
	// err := TheIconMgr.SetIcon(ic, name)
	if err == nil {
		ic.IconName = name
		return true, nil
	}
	return false, err
}

func (ic *Icon) GetSize(sc *Scene, iter int) {
	if iter > 0 {
		return
	}
	ic.GetSizeFromWH(2, 2)
	// todo: ?
	// ic.SVG.Nm = ic.Nm
	// ic.LayState.Alloc.Size = sic.LayState.Alloc.Size
}

func (ic *Icon) SetStyle(sc *Scene) {
	ic.StyMu.Lock()
	defer ic.StyMu.Unlock()

	ic.SVG.Norm = true
	// ic.SVG.Fill = true
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

// RenderSVG renders the SVG to Pixels if needs update
func (ic *Icon) RenderSVG(sc *Scene) {
	sv := &ic.SVG
	if !sc.HasFlag(ScRebuild) && sv.Pixels != nil { // if rebuilding rebuild..
		isz := sv.Pixels.Bounds().Size()
		if isz == ic.RendSize && sv.Name == string(ic.IconName) {
			return
		}
	}
	// todo: units context from us to SVG??
	zp := image.Point{}
	sz := ic.LayState.Alloc.Size.ToPoint()
	if sz == zp {
		ic.RendSize = zp
		return
	}
	sv.Resize(sz) // does Config if needed
	sv.Render()
	ic.RendSize = sz
	sv.Name = string(ic.IconName)
	// fmt.Println("re-rendered icon:", sv.Name, "size:", sz)
}

func (ic *Icon) Render(sc *Scene) {
	ic.RenderSVG(sc)

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

// todo: optimize by loading into lib and copying instead of loading fresh every time!

// IconMgr is the manager of all things Icon -- needed to allow svg to be a
// separate package, and implemented by svg.IconMgr
type IconMgr interface {
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
