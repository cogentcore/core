// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"fmt"
	"go/build"
	"image"
	"image/color"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/goki/gi"
	"github.com/goki/gi/units"
	"github.com/goki/ki"
	"github.com/goki/ki/kit"
)

func init() {
	gi.TheIconMgr = &IconMgr{}
	DefaultIconSet = MakeDefaultIcons()
	DefaultIconSet.OpenDefaultIcons()
	CurIconSet = DefaultIconSet
	gi.CurIconList = gi.TheIconMgr.IconList(true)
}

// svg.Icon is the actual SVG for a gi.Icon -- it should contain no color
// information -- it should just be a filled shape where the fill and stroke
// colors come from the surrounding context / paint settings.  The rendered
// version is cached for a given size. Icons are always copied from an
// original source icon and then can be customized from there.
type Icon struct {
	SVG
	Filename string      `desc:"file name with full path for icon if loaded from file"`
	Rendered bool        `json:"-" xml:"-" desc:"we have already rendered at RenderedSize -- doesn't re-render at same size -- if the paint params change, set this to false to re-render"`
	RendSize image.Point `json:"-" xml:"-" desc:"size at which we previously rendered"`
}

var KiT_Icon = kit.Types.AddType(&Icon{}, IconProps)

var IconProps = ki.Props{
	"background-color": color.Transparent,
}

// CopyFromIcon copies from a source icon, typically one from a library --
// preserves all the exisiting render state etc for the current icon, so that
// only a new render is required
func (ic *Icon) CopyFromIcon(cp *Icon) {
	if cp == nil {
		return
	}
	ic.CopyFrom(cp)
	ic.Rendered = false
}

// IconAutoOpen controls auto-loading of icons -- can turn this off for debugging etc
var IconAutoOpen = true

func (ic *Icon) Init2D() {
	ic.SVG.Init2D()
	ic.Fill = true
}

func (ic *Icon) Size2D(iter int) {
	ic.Viewport.Size2D(iter)
}

func (ic *Icon) Layout2D(parBBox image.Rectangle, iter int) bool {
	redo := ic.SVG.Layout2D(parBBox, iter)
	ic.SetNormXForm()
	return redo
}

// NeedsReRender tests whether the last render parameters (size, color) have changed or not
func (ic *Icon) NeedsReRender() bool {
	if ic.FullReRenderIfNeeded() || !ic.Rendered || ic.RendSize != ic.Geom.Size {
		return true
	}
	return false
}

func (ic *Icon) Render2D() {
	if ic.Viewport == nil {
		ic.FullRender2DTree()
		return
	}
	if ic.PushBounds() {
		if ic.NeedsReRender() {
			rs := &ic.Render
			if ic.Fill {
				ic.FillViewport()
			}
			ic.SetNormXForm()
			rs.PushXForm(ic.Pnt.XForm)
			ic.Render2DChildren() // we must do children first, then us!
			rs.PopXForm()
			ic.Rendered = true
			ic.RendSize = ic.Geom.Size
		}
		ic.RenderViewport2D() // update our parent image
		ic.PopBounds()
	}
}

////////////////////////////////////////////////////////////////////////////////////////
// IconMgr

// svg.IconMgr is THE implementation of the gi.IconMgr interface
type IconMgr struct {
}

func (im *IconMgr) IsValid(iconName string) bool {
	if im == nil {
		fmt.Println("TheIconMgr is nil -- you MUST import gi/svg as e.g., import \"_ github.com/goki/gi/svg\" to properly initialize the SVG icon manager")
		return false
	}
	if gi.IconName(iconName).IsNil() {
		return false
	}
	if _, ok := (*CurIconSet)[iconName]; ok {
		return true
	}
	if _, ok := (*DefaultIconSet)[iconName]; ok {
		return true
	}
	return false
}

// IconByName is main function to get icon by name -- looks in CurIconSet and
// falls back to DefaultIconSet if not found there -- returns error and logs a
// message if not found
func (im *IconMgr) IconByName(name string) (*Icon, error) {
	if gi.IconName(name).IsNil() {
		return nil, nil
	}
	if !im.IsValid(name) {
		err := fmt.Errorf("svg.IconMgr.IconByName -- icon name not found in CurIconSet or DefaultIconSet: %v\n", name)
		return nil, err
	}
	ic, ok := (*CurIconSet)[name]
	if !ok {
		ic = (*DefaultIconSet)[name]
	}
	if ic.Filename != "" && !ic.HasChildren() && IconAutoOpen {
		ic.OpenXML(ic.Filename)
	}
	return ic, nil
}

func (im *IconMgr) SetIcon(ic *gi.Icon, iconName string) error {
	sic, err := im.IconByName(iconName)
	if err != nil {
		return err
	}
	ic.SetNChildren(1, KiT_Icon, "icon")
	nic := ic.KnownChild(0).(*Icon)
	nic.CopyFromIcon(sic)
	ic.Filename = sic.Filename
	return nil
}

func (im *IconMgr) IconList(alphaSort bool) []gi.IconName {
	return CurIconSet.IconList(alphaSort)
}

////////////////////////////////////////////////////////////////////////////////////////
// IconSet is a list of icons

// IconSet is a collection of icons
type IconSet map[string]*Icon

// DefaultIconSet is the default icon set, initialized by default
var DefaultIconSet *IconSet

// CurIconSet is the current icon set -- defaults to default but can be
// changed to whatever you want
var CurIconSet *IconSet

func (iset *IconSet) OpenDefaultIcons() error {
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		gopath = build.Default.GOPATH
	}
	path := filepath.Join(gopath, "src/github.com/goki/gi/icons")
	// fmt.Printf("loading default icons: %v\n", path)
	rval := iset.OpenIconsFromPath(path)
	// tstpath := filepath.Join(gopath, "src/github.com/goki/gi/icons_svg_test")
	// rval = iset.OpenIconsFromPath(tstpath)
	return rval
}

// OpenIconsFromPath scans for .svg icon files in given path, adding them to
// the given IconSet, just storing the filename for later lazy loading
func (iset *IconSet) OpenIconsFromPath(path string) error {
	ext := ".svg"

	err := filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("gi.IconSet: error accessing path %q: %v\n", p, err)
			return err
		}
		if filepath.Ext(p) == ext {
			ps, fn := filepath.Split(p)
			bfn := fn[:len(fn)-len(ext)]
			nm := strings.ToLower(bfn)
			pd := strings.TrimPrefix(ps, path)
			if pd != "" {
				pd = strings.ToLower(strings.Trim(strings.Trim(pd, string(filepath.Separator)), "/"))
				if pd != "" {
					nm = pd + "-" + nm
				}
			}
			ic := Icon{}
			ic.InitName(&ic, nm)
			ic.Filename = p
			(*iset)[nm] = &ic
		}
		return nil
	})
	if err != nil {
		log.Printf("gi.IconSet: error walking the path %q: %v\n", path, err)
	}
	return err
}

// IconList returns a list of names of icons in the icon set
func (iset *IconSet) IconList(alphaSort bool) []gi.IconName {
	il := make([]gi.IconName, len(*iset)+1)
	il[0] = gi.IconName("none")
	idx := 1
	for _, ic := range *iset {
		il[idx] = gi.IconName(ic.Nm)
		idx++
	}
	if alphaSort {
		sort.Slice(il, func(i, j int) bool {
			return il[i] < il[j]
		})
	}
	return il
}

func MakeDefaultIcons() *IconSet {
	iset := make(IconSet, 100)
	if true {
		{
			ic := Icon{}
			ic.InitName(&ic, "widget-wedge-down")
			ic.ViewBox.Size = gi.Vec2D{1, 1}
			p := ic.AddNewChild(KiT_Path, "p").(*Path)
			p.SetProp("stroke-width", units.NewValue(1, units.Pct))
			p.SetData("M 0.05 0.05 .95 0.05 .5 .95 Z")
			iset[ic.Nm] = &ic
		}
		{
			ic := Icon{}
			ic.InitName(&ic, "widget-wedge-up")
			ic.ViewBox.Size = gi.Vec2D{1, 1}
			p := ic.AddNewChild(KiT_Path, "p").(*Path)
			p.SetProp("stroke-width", units.NewValue(1, units.Pct))
			p.SetData("M 0.05 0.95 .95 0.95 .5 .05 Z")
			iset[ic.Nm] = &ic
		}
		{
			ic := Icon{}
			ic.InitName(&ic, "widget-wedge-left")
			ic.ViewBox.Size = gi.Vec2D{1, 1}
			p := ic.AddNewChild(KiT_Path, "p").(*Path)
			p.SetProp("stroke-width", units.NewValue(1, units.Pct))
			p.SetData("M 0.95 0.05 .95 0.95 .05 .5 Z")
			iset[ic.Nm] = &ic
		}
		{
			ic := Icon{}
			ic.InitName(&ic, "widget-wedge-right")
			ic.ViewBox.Size = gi.Vec2D{1, 1}
			p := ic.AddNewChild(KiT_Path, "p").(*Path)
			p.SetProp("stroke-width", units.NewValue(1, units.Pct))
			p.SetData("M 0.05 0.05 .05 0.95 .95 .5 Z")
			iset[ic.Nm] = &ic
		}
		{
			ic := Icon{}
			ic.InitName(&ic, "widget-checkmark")
			ic.ViewBox.Size = gi.Vec2D{1, 1}
			p := ic.AddNewChild(KiT_Path, "p").(*Path)
			p.SetProp("stroke-width", units.NewValue(20, units.Pct))
			p.SetProp("fill", "none")
			p.SetData("M 0.1 0.5 .5 0.9 .9 .1")
			iset[ic.Nm] = &ic
		}
		{
			ic := Icon{}
			ic.InitName(&ic, "widget-checked-box")
			ic.ViewBox.Size = gi.Vec2D{1, 1}
			bx := ic.AddNewChild(KiT_Rect, "bx").(*Rect)
			bx.Pos.Set(0.05, 0.05)
			bx.Size.Set(0.9, 0.9)
			bx.SetProp("stroke-width", units.NewValue(5, units.Pct))
			// bx.Radius.Set(0.02, 0.02)
			p := ic.AddNewChild(KiT_Path, "p").(*Path)
			p.SetProp("stroke-width", units.NewValue(20, units.Pct))
			p.SetProp("fill", "none")
			p.SetData("M 0.2 0.5 .5 0.8 .8 .2")
			iset[ic.Nm] = &ic
		}
		{
			ic := Icon{}
			ic.InitName(&ic, "widget-unchecked-box")
			ic.ViewBox.Size = gi.Vec2D{1, 1}
			bx := ic.AddNewChild(KiT_Rect, "bx").(*Rect)
			bx.SetProp("stroke-width", units.NewValue(5, units.Pct))
			bx.Pos.Set(0.05, 0.05)
			bx.Size.Set(0.9, 0.9)
			// bx.Radius.Set(0.02, 0.02) // not rendering well at small sizes
			iset[ic.Nm] = &ic
		}
		{
			ic := Icon{}
			ic.InitName(&ic, "widget-circlebutton-on")
			ic.ViewBox.Size = gi.Vec2D{1, 1}
			oc := ic.AddNewChild(KiT_Circle, "oc").(*Circle)
			oc.Pos.Set(0.5, 0.5)
			oc.Radius = 0.4
			oc.SetProp("fill", "none")
			oc.SetProp("stroke-width", units.NewValue(10, units.Pct))
			inc := ic.AddNewChild(KiT_Circle, "ic").(*Circle)
			inc.Pos.Set(0.5, 0.5)
			inc.Radius = 0.2
			inc.SetProp("stroke-width", units.NewValue(10, units.Pct))
			iset[ic.Nm] = &ic
		}
		{
			ic := Icon{}
			ic.InitName(&ic, "widget-circlebutton-off")
			ic.ViewBox.Size = gi.Vec2D{1, 1}
			oc := ic.AddNewChild(KiT_Circle, "oc").(*Circle)
			oc.Pos.Set(0.5, 0.5)
			oc.Radius = 0.4
			oc.SetProp("fill", "none")
			oc.SetProp("stroke-width", units.NewValue(10, units.Pct))
			iset[ic.Nm] = &ic
		}
		{
			ic := Icon{}
			ic.InitName(&ic, "widget-handle-circles")
			ic.ViewBox.Size = gi.Vec2D{1, 1}
			c0 := ic.AddNewChild(KiT_Circle, "c0").(*Circle)
			c0.SetProp("stroke-width", units.NewValue(5, units.Pct))
			c0.Pos.Set(0.5, 0.15)
			c0.Radius = 0.12
			c1 := ic.AddNewChild(KiT_Circle, "c1").(*Circle)
			c1.SetProp("stroke-width", units.NewValue(5, units.Pct))
			c1.Pos.Set(0.5, 0.5)
			c1.Radius = 0.12
			c2 := ic.AddNewChild(KiT_Circle, "c2").(*Circle)
			c2.SetProp("stroke-width", units.NewValue(5, units.Pct))
			c2.Pos.Set(0.5, 0.85)
			c2.Radius = 0.12
			iset[ic.Nm] = &ic
		}
	}
	return &iset
}
