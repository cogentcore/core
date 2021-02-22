// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/units"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/mat32"
)

func init() {
	gi.TheIconMgr = &IconMgr{}
	DefaultIconSet = MakeDefaultIcons()
	DefaultIconSet.OpenDefaultIcons()
	CurIconSet = make(IconSet, len(DefaultIconSet))
	for k, v := range DefaultIconSet {
		CurIconSet[k] = v
	}
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
	Rendered bool        `copy:"-" json:"-" xml:"-" desc:"we have already rendered at RenderedSize -- doesn't re-render at same size -- if the paint params change, set this to false to re-render"`
	RendSize image.Point `copy:"-" json:"-" xml:"-" desc:"size at which we previously rendered"`
}

var KiT_Icon = kit.Types.AddType(&Icon{}, IconProps)

// AddNewIcon adds a new icon to given parent node, with given name.
func AddNewIcon(parent ki.Ki, name string) *Icon {
	return parent.AddNewChild(KiT_Icon, name).(*Icon)
}

var IconProps = ki.Props{
	"EnumType:Flag":    gi.KiT_VpFlags,
	"background-color": color.Transparent,
}

func (ic *Icon) CopyFieldsFrom(frm interface{}) {
	fr := frm.(*Icon)
	ic.SVG.CopyFieldsFrom(&fr.SVG)
	ic.Filename = fr.Filename
}

// CopyFromIcon copies from a source icon, typically one from a library --
// preserves all the existing render state etc for the current icon, so that
// only a new render is required
func (ic *Icon) CopyFromIcon(cp *Icon) {
	if cp == nil {
		return
	}
	ic.CopyFrom(cp)
	// ic.FullInit2DTree()
	ic.Rendered = false
}

// IconAutoOpen controls auto-loading of icons -- can turn this off for debugging etc
var IconAutoOpen = true

func (ic *Icon) Init2D() {
	ic.SVG.Init2D()
	ic.Fill = true
}

func (ic *Icon) Size2D(iter int) {
	ic.Viewport2D.Size2D(iter)
}

func (ic *Icon) Layout2D(parBBox image.Rectangle, iter int) bool {
	ic.StyMu.RLock()
	needSty := ic.Sty.Font.Size.Val == 0
	ic.StyMu.RUnlock()
	if needSty {
		ic.StyleSVG()
	}
	redo := ic.SVG.Layout2D(parBBox, iter)
	ic.SetNormXForm()
	return redo
}

// NeedsReRender tests whether the last render parameters (size, color) have changed or not
func (ic *Icon) NeedsReRender() bool {
	if ic.NeedsFullReRender() || !ic.Rendered || ic.RendSize != ic.Geom.Size {
		return true
	}
	return false
}

func (ic *Icon) Render2D() {
	if ic.Viewport == nil {
		ic.FullRender2DTree()
		return
	}
	if ic.NeedsReRender() {
		if ic.PushBounds() {
			rs := &ic.Render
			if ic.Fill {
				ic.FillViewport()
			}
			ic.SetNormXForm()
			rs.PushXFormLock(ic.Pnt.XForm)
			ic.Render2DChildren() // we must do children first, then us!
			rs.PopXFormLock()
			ic.Rendered = true
			ic.RendSize = ic.Geom.Size
			ic.PopBounds()
		}
	}
	ic.RenderViewport2D() // update our parent image
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
	if _, ok := CurIconSet[iconName]; ok {
		return true
	}
	if _, ok := DefaultIconSet[iconName]; ok {
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
	ic, ok := CurIconSet[name]
	if !ok {
		ic = DefaultIconSet[name] // guaranteed above to exist
	}
	if ic.Filename != "" && !ic.HasChildren() && IconAutoOpen && ic.Filename != "blank.svg" {
		ic.OpenXML(gi.FileName(ic.Filename))
		ki.UniquifyNamesAll(ic.This())
	}
	return ic, nil
}

// SetIcon sets the icon by name into given Icon wrapper, returning error
// message if not found etc.  This is how gi.Icon is initialized from
// underlying svg.Icon items.
func (im *IconMgr) SetIcon(ic *gi.Icon, iconName string) error {
	sic, err := im.IconByName(iconName)
	if err != nil {
		return err
	}
	ic.SetNChildren(1, KiT_Icon, "icon")
	nic := ic.Child(0).(*Icon)
	nic.CopyFromIcon(sic)
	ic.Filename = sic.Filename
	wd := sic.Prop("width")
	if wd != nil {
		ic.SetProp("width", wd)
	}
	ht := sic.Prop("height")
	if ht != nil {
		ic.SetProp("height", ht)
	}
	return nil
}

// IconList returns the current list of all available icons,
// optionally sorted in alphabetical order.
func (im *IconMgr) IconList(alphaSort bool) []gi.IconName {
	return CurIconSet.IconList(alphaSort)
}

////////////////////////////////////////////////////////////////////////////////////////
// IconSet is a list of icons

// IconSet is a collection of icons
type IconSet map[string]*Icon

// DefaultIconSet is the default icon set, initialized by default
var DefaultIconSet IconSet

// CurIconSet is the current icon set -- defaults to default but can be
// changed to whatever you want
var CurIconSet IconSet

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

func (iset *IconSet) OpenDefaultIcons() error {
	err := iset.OpenIconsFromAssetDir("../icons", AssetDir, Asset)
	// path, err := dirs.GoSrcDir("github.com/goki/gi/icons")
	if err != nil {
		log.Println(err)
		return err
	}
	// // fmt.Printf("loading default icons: %v\n", path)
	// rval := iset.OpenIconsFromPath(path)
	// // tstpath := filepath.Join(gopath, "src/github.com/goki/gi/icons_svg_test")
	// // rval = iset.OpenIconsFromPath(tstpath)
	return nil
}

// OpenIconsFromAssetDir loads icons using Asset and AssetDir funcs
// as generated by e.g., https://github.com/shuLhan/go-bindata or similar
// such forks of go-bindata
func (iset *IconSet) OpenIconsFromAssetDir(path string, dirFunc func(path string) ([]string, error), assetFunc func(name string) ([]byte, error)) error {
	ext := ".svg"

	fnms, err := dirFunc(path)
	if err != nil {
		fmt.Printf("gi.IconSet: error accessing asset path %q: %v\n", path, err)
		return err
	}
	var lasterr error
	for _, fnm := range fnms {
		if filepath.Ext(fnm) != ext {
			continue
		}
		ps, fn := filepath.Split(fnm)
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
		ic.Filename = fnm
		b, err := assetFunc(filepath.Join(path, fnm))
		if err == nil {
			err = ic.ReadXML(bytes.NewBuffer(b))
			if err == nil || err == io.EOF {
				ki.UniquifyNamesAll(ic.This())
				(*iset)[nm] = &ic
			} else {
				lasterr = err
				log.Println(err)
			}
		} else {
			lasterr = err
			log.Println(err)
		}
	}
	return lasterr
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

func MakeDefaultIcons() IconSet {
	iset := make(IconSet, 100)
	if true {
		{
			ic := &Icon{}
			ic.InitName(ic, "wedge-down")
			ic.ViewBox.Size = mat32.Vec2{1, 1}
			p := AddNewPath(ic, "p", "M 0.05 0.05 .95 0.05 .5 .95 Z")
			p.SetProp("stroke-width", units.NewPct(1))
			iset[ic.Nm] = ic
		}
		{
			ic := &Icon{}
			ic.InitName(ic, "wedge-up")
			ic.ViewBox.Size = mat32.Vec2{1, 1}
			p := AddNewPath(ic, "p", "M 0.05 0.95 .95 0.95 .5 .05 Z")
			p.SetProp("stroke-width", units.NewPct(1))
			iset[ic.Nm] = ic
		}
		{
			ic := &Icon{}
			ic.InitName(ic, "wedge-left")
			ic.ViewBox.Size = mat32.Vec2{1, 1}
			p := AddNewPath(ic, "p", "M 0.95 0.05 .95 0.95 .05 .5 Z")
			p.SetProp("stroke-width", units.NewPct(1))
			iset[ic.Nm] = ic
		}
		{
			ic := &Icon{}
			ic.InitName(ic, "wedge-right")
			ic.ViewBox.Size = mat32.Vec2{1, 1}
			p := AddNewPath(ic, "p", "M 0.05 0.05 .05 0.95 .95 .5 Z")
			p.SetProp("stroke-width", units.NewPct(1))
			iset[ic.Nm] = ic
		}
		{
			ic := &Icon{}
			ic.InitName(ic, "checkmark")
			ic.ViewBox.Size = mat32.Vec2{1, 1}
			p := AddNewPath(ic, "p", "M 0.1 0.5 .5 0.9 .9 .1")
			p.SetProp("stroke-width", units.NewPct(20))
			p.SetProp("fill", "none")
			iset[ic.Nm] = ic
		}
		{
			ic := &Icon{}
			ic.InitName(ic, "checked-box")
			ic.ViewBox.Size = mat32.Vec2{1, 1}
			bx := AddNewRect(ic, "bx", 0.05, 0.05, 0.9, 0.9)
			bx.SetProp("stroke-width", units.NewPct(5))
			// bx.Radius.Set(0.02, 0.02)
			p := AddNewPath(ic, "p", "M 0.2 0.5 .5 0.8 .8 .2")
			p.SetProp("stroke-width", units.NewPct(20))
			p.SetProp("fill", "none")
			iset[ic.Nm] = ic
		}
		{
			ic := &Icon{}
			ic.InitName(ic, "unchecked-box")
			ic.ViewBox.Size = mat32.Vec2{1, 1}
			bx := AddNewRect(ic, "bx", 0.05, 0.05, 0.9, 0.9)
			bx.SetProp("stroke-width", units.NewPct(5))
			// bx.Radius.Set(0.02, 0.02) // not rendering well at small sizes
			iset[ic.Nm] = ic
		}
		{
			ic := &Icon{}
			ic.InitName(ic, "circlebutton-on")
			ic.ViewBox.Size = mat32.Vec2{1, 1}
			oc := AddNewCircle(ic, "oc", 0.5, 0.5, 0.4)
			oc.SetProp("fill", "none")
			oc.SetProp("stroke-width", units.NewPct(10))
			inc := AddNewCircle(ic, "ic", 0.5, 0.5, 0.2)
			inc.SetProp("stroke-width", units.NewPct(10))
			iset[ic.Nm] = ic
		}
		{
			ic := &Icon{}
			ic.InitName(ic, "circlebutton-off")
			ic.ViewBox.Size = mat32.Vec2{1, 1}
			oc := AddNewCircle(ic, "oc", 0.5, 0.5, 0.4)
			oc.SetProp("fill", "none")
			oc.SetProp("stroke-width", units.NewPct(10))
			iset[ic.Nm] = ic
		}
		{
			ic := &Icon{}
			rad := mat32.Vec2{0.25, 0.12}
			ic.InitName(ic, "handle-circles-vert")
			ic.ViewBox.Size = mat32.Vec2{1, 1}
			c0 := AddNewEllipse(ic, "c0", 0.5, 0.15, rad.X, rad.Y)
			c0.SetProp("stroke-width", units.NewPct(5))
			c1 := AddNewEllipse(ic, "c1", 0.5, 0.5, rad.X, rad.Y)
			c1.SetProp("stroke-width", units.NewPct(5))
			c2 := AddNewEllipse(ic, "c2", 0.5, 0.85, rad.X, rad.Y)
			c2.SetProp("stroke-width", units.NewPct(5))
			iset[ic.Nm] = ic
		}
		{
			ic := &Icon{}
			rad := mat32.Vec2{0.12, 0.25}
			ic.InitName(ic, "handle-circles-horiz")
			ic.ViewBox.Size = mat32.Vec2{1, 1}
			c0 := AddNewEllipse(ic, "c0", 0.15, 0.5, rad.X, rad.Y)
			c0.SetProp("stroke-width", units.NewPct(5))
			c1 := AddNewEllipse(ic, "c1", 0.5, 0.5, rad.X, rad.Y)
			c1.SetProp("stroke-width", units.NewPct(5))
			c2 := AddNewEllipse(ic, "c2", 0.85, 0.5, rad.X, rad.Y)
			c2.SetProp("stroke-width", units.NewPct(5))
			iset[ic.Nm] = ic
		}
	}
	return iset
}
