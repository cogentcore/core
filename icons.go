// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"go/build"
	"image"
	"image/color"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/goki/gi/units"
	"github.com/goki/ki"
	"github.com/goki/ki/kit"
)

// Plan: modern gui icons do NOT specify any color and are colored by the
// user, applying either transparency or tinting etc to indicate different
// states.  Thus, there is NO NEED for state information in the icon itself!

// however, this does mean that caching needs to be sensitive to color settings

// Icon is an SVG that is assumed to contain no color information -- it should
// just be a filled shape where the fill and stroke colors come from the
// surrounding context / paint settings.  The rendered version can be cached
// for a given set of fill / stroke paint values, as an optimization.
type Icon struct {
	SVG
	Filename            string      `desc:"file name (typically with relevant path -- todo: special paths for finding installed defaults) for icon -- lazy loading is used, so icon files are loaded only when needed"`
	Rendered            bool        `json:"-" xml:"-" desc:"we have already rendered at RenderedSize -- doesn't re-render at same size -- if the paint params change, set this to false to re-render"`
	RenderedSize        image.Point `json:"-" xml:"-" desc:"size at which we previously rendered"`
	RenderedStrokeColor Color       `json:"-" xml:"-" desc:"stroke color rendered"`
	RenderedFillColor   Color       `json:"-" xml:"-" desc:"fill color rendered"`
}

var KiT_Icon = kit.Types.AddType(&Icon{}, IconProps)

// InitFromName initializes icon from an icon looked up by name -- returns
// false if name is not valid
func (ic *Icon) InitFromName(iconName string) bool {
	cp := IconByName(iconName)
	if cp == nil {
		return false
	}
	ic.CopyFromIcon(cp)
	return true
}

// CopyFromIcon copies from a source icon, typically one from a library --
// preserves all the exisiting render state etc for the current icon, so that
// only a new render is required
func (ic *Icon) CopyFromIcon(cp *Icon) {
	if cp == nil {
		return
	}
	oldIc := *ic
	ic.CopyFrom(cp)
	ic.Rendered = false
	ic.Viewport = oldIc.Viewport
	ic.LayData = oldIc.LayData
	ic.VpBBox = oldIc.VpBBox
	ic.WinBBox = oldIc.WinBBox
	ic.Geom = oldIc.Geom
	ic.Sty = oldIc.Sty
	ic.Fill = oldIc.Fill
	ic.Pixels = nil
	ic.Resize(ic.Geom.Size)
	ic.FullRender2DTree()
	ic.LayData = oldIc.LayData
	ic.VpBBox = oldIc.VpBBox
	ic.WinBBox = oldIc.WinBBox
	ic.Rendered = false // not yet..
}

var IconProps = ki.Props{
	"background-color": color.Transparent,
}

// IconAutoLoad controls auto-loading of icons -- can turn this off for debugging etc
var IconAutoLoad = true

func (ic *Icon) Init2D() {
	if ic.Filename != "" && !ic.HasChildren() && IconAutoLoad {
		// fmt.Printf("loading icon: %v\n", ic.Filename)
		ic.LoadXML(ic.Filename)
		// if err != nil {
		// 	IconAutoLoad = false
		// }
	}
	ic.SVG.Init2D()
	ic.Fill = true
}

func (ic *Icon) Size2D() {
	ic.Viewport.Size2D()
}

func (ic *Icon) Layout2D(parBBox image.Rectangle) {
	ic.SVG.Layout2D(parBBox)
	ic.SetNormXForm()
}

// NeedsReRender tests whether the last render parameters (size, color) have changed or not
func (ic *Icon) NeedsReRender() bool {
	return !ic.Rendered || ic.RenderedSize != ic.Geom.Size
	// || ic.RenderedStrokeColor != pc.StrokeStyle.Color.Color || ic.RenderedFillColor != pc.FillStyle.Color.Color
}

func (ic *Icon) Render2D() {
	if ic.PushBounds() {
		if ic.NeedsReRender() {
			rs := &ic.Render
			if ic.Fill {
				ic.FillViewport()
			}
			ic.SetNormXForm()
			rs.PushXForm(ic.Pnt.XForm)
			// fmt.Printf("IconRender: %v Bg: %v Fill: %v Clr: %v Stroke: %v\n",
			// 	ic.PathUnique(), ic.Sty.Background.Color, ic.Pnt.FillStyle.Color, ic.Sty.Color, ic.Pnt.StrokeStyle.Color)
			ic.Render2DChildren() // we must do children first, then us!
			ic.PopBounds()
			rs.PopXForm()
			ic.Rendered = true
			ic.RenderedSize = ic.Geom.Size
			// ic.RenderedStrokeColor = pc.StrokeStyle.Color.Color
			// ic.RenderedFillColor = pc.FillStyle.Color.Color
			// ic.SavePNG(ic.Nm + ".png")
		}
		ic.RenderViewport2D() // update our parent image
	}
}

////////////////////////////////////////////////////////////////////////////////////////
// IconName

// IconName is used to specify an icon -- currently just the unique name of
// the icon -- automtically provides a chooser menu for icons using ValueView
// system
type IconName string

// Icon() returns the icon from current IconSet of the given name, or nil if
// not available -- icon should be copied before inserting into a widget
func (inm IconName) Icon() *Icon {
	return IconByName(string(inm))
}

// IsNil tests whether the icon name is empty, 'none' or 'nil' -- indicates to
// not use a icon
func (inm IconName) IsNil() bool {
	return IconNameNil(string(inm))
}

// IsValid tests whether the icon name is valid -- represents a non-nil icon
// available in the current or default icon set
func (inm IconName) IsValid() bool {
	return IconNameValid(string(inm))
}

// ValueView() returns the ValueView representation for the icon name --
// presents a chooser
func (inm IconName) ValueView() ValueView {
	vv := IconValueView{}
	vv.Init(&vv)
	return &vv
}

////////////////////////////////////////////////////////////////////////////////////////
//  IconValueView

// IconValueView presents a StructViewInline for a struct plus a IconView button..
type IconValueView struct {
	ValueViewBase
}

var KiT_IconValueView = kit.Types.AddType(&IconValueView{}, nil)

func (vv *IconValueView) WidgetType() reflect.Type {
	vv.WidgetTyp = KiT_ComboBox
	return vv.WidgetTyp
}

func (vv *IconValueView) UpdateWidget() {
	cb := vv.Widget.(*ComboBox)
	txt := kit.ToString(vv.Value.Interface())
	cb.SetCurVal(txt)
}

func (vv *IconValueView) ConfigWidget(widg Node2D) {
	vv.Widget = widg

	cb := vv.Widget.(*ComboBox)
	cb.ItemsFromStringList(IconListSorted(*CurIconSet), false, 30)

	vv.UpdateWidget()

	cb.ComboSig.ConnectOnly(vv.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		vvv, _ := recv.EmbeddedStruct(KiT_IconValueView).(*IconValueView)
		cbb := vvv.Widget.(*ComboBox)
		eval := cbb.CurVal.(string)
		if vvv.SetValue(eval) {
			vvv.UpdateWidget()
		}
	})
}

////////////////////////////////////////////////////////////////////////////////////////
// IconContexts, IconSets

// TODO: not clear if there is actual value in organizing everything into
// these contexts, etc, as opposed to just using a flat unique name, which is
// simpler as a basic map.  for the time being, using the flat map but have
// copied the context here for further consideration..

// icon lists
// https://fontawesome.com/
// https://joekuan.wordpress.com/2015/09/23/list-of-qt-icons/
// https://leungwensen.github.io/svg-icon/
// golang.org/x/exp/shiny/materialdesign/icons/ -- material encoded icons
// https://community.kde.org/KDE_Visual_Design_Group/HIG/IconDesign https://github.com/KDE/breeze-icons

// different types of standard icon name spaces, from
// https://standards.freedesktop.org/icon-naming-spec/icon-naming-spec-latest.html
// -- we organize our IconSets into these different contexts
type IconContexts int32

const (
	// WidgetIcons are used as parts of standard widgets -- these are
	// available built-in
	WidgetIcons IconContexts = iota

	// ActionIcons are generally used in menus and dialogs for interacting
	// with the user.
	ActionIcons

	// AnimationIcons are images used to represent loading web sites, or other
	// background processing which may be less suited to more verbose progress
	// reporting in the user interface.
	AnimationIcons

	// ApplicationIcons describe what an application is, for use in the
	// Programs menu, window decorations, and the task list. These may or may
	// not be generic depending on the application and its purpose.
	ApplicationIcons

	// CategoryIcons are used for categories in the Programs menu, or the
	// Control Center, for separating applications, preferences, and settings
	// for display to the user.
	CategoryIcons

	// DeviceIcons are for hardware that is contained within or connected to the
	// computing device. Naming for extended devices in this group, is of the
	// form <primary function>-<manufacturer>-<model>. This allows ease of
	// fallback to the primary function device name, or ones more targeted for
	// a specific series of models from a manufacturer.
	DeviceIcons

	// EmblemIcons are for tags and properties of files, that are displayed in
	// the file manager. This context contains emblems for such things as
	// read-only or photos
	EmblemIcons

	// EmoteIcons for emotions that are expressed through text chat
	// applications such as :-) or :-P in IRC or instant messengers.
	EmoteIcons

	// IntnlIcons for international denominations such as flags.
	IntnlIcons

	// MimeIcons for different types of data, such as audio or image files.
	MimeIcons

	// PlaceIcons used to represent locations, either on the local filesystem,
	// or through remote connections. Folders, trash, and workgroups are some
	// example.
	PlaceIcons

	// StatusIcons for presenting status to the user. This context contains
	// icons for warning and error dialogs, as well as for the current
	// weather, appointment alarms, and battery status
	StatusIcons

	IconContextsN
)

//go:generate stringer -type=IconContexts

var KiT_IconContexts = kit.Enums.AddEnum(IconContextsN, false, nil)

func (ev IconContexts) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *IconContexts) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

// StdIconNames is a list of standard icon names that we expect to find in an
// IconSet -- used for loookup
var StdIconNames = [IconContextsN][]string{
	{ // WidgetIcons
		"widget-wedge-down",
		"widget-wedge-up",
		"widget-wedge-left",
		"widget-wedge-right",
		"widget-checkmark",
		"widget-checked-box",
		"widget-unchecked-box",
		"widget-circlebutton-on",
		"widget-circlebutton-off",
		"widget-handle-circles",
	}, { // ActionIcons
		"edit-clear",
		"edit-copy",
		"edit-cut",
		"edit-delete",
		"edit-find",
		"edit-find-replace",
		"edit-paste",
		"edit-redo",
		"edit-select-all",
		"edit-undo",
		"list-add",
		"list-remove",
	}, { // AnimationIcons
	}, { // ApplicationIcons
	}, { // CategoryIcons
	}, { // DeviceIcons
	}, { // EmblemIcons
	}, { // EmoteIcons
	}, { // IntnlIcons
	}, { // MimeIcons
	}, { // PlaceIcons
	}, { // StatusIcons
	},
}

// an IconSet is a collection of icons styled in the same themes - lookup by name
type IconSet map[string]*Icon

// the default icon set is loaded by default
var DefaultIconSet *IconSet = MakeDefaultIcons()

// the current icon set can be set to any icon set
var CurIconSet *IconSet = DefaultIconSet

// IconNameNil tests whether the icon name is empty, 'none' or 'nil' --
// indicates to not use a icon
func IconNameNil(name string) bool {
	return name == "" || name == "none" || name == "nil"
}

// IconNameValid tests whether the icon name refers to a valid icon in current IconSet
func IconNameValid(name string) bool {
	return IconByName(name) != nil
}

// IconByName is main function to get icon by name -- looks in CurIconSet and
// falls back to DefaultIconSet if not found there -- logs a message and
// returns nil if not found
func IconByName(name string) *Icon {
	if IconNameNil(name) {
		return nil
	}
	ic, ok := (*CurIconSet)[name]
	if !ok {
		ic, ok = (*DefaultIconSet)[name]
		if !ok {
			// todo: look on StdIconNames to see if it is not a standard name..
			// log.Printf("gi.IconByName: unable to find icon name in either CurIconSet or DefaultIconSet: %v\n", name)

			return nil
		}
	}
	return ic
}

// IconListSorted returns a slice of all the icon names in the icon set alpha
// sorted, including 'none'
func IconListSorted(is IconSet) []string {
	il := make([]string, len(is)+1)
	il[0] = "none"
	idx := 1
	for _, ic := range is {
		il[idx] = ic.Nm
		idx++
	}
	sort.Slice(il, func(i, j int) bool {
		return il[i] < il[j]
	})
	return il
}

func MakeDefaultIcons() *IconSet {
	iset := make(IconSet, 100)
	if true {
		{
			ic := Icon{}
			ic.InitName(&ic, "widget-wedge-down")
			ic.ViewBox.Size = Vec2D{1, 1}
			p := ic.AddNewChild(KiT_Path, "p").(*Path)
			p.SetData("M 0.05 0.05 .95 0.05 .5 .95 Z")
			iset[ic.Nm] = &ic
		}
		{
			ic := Icon{}
			ic.InitName(&ic, "widget-wedge-up")
			ic.ViewBox.Size = Vec2D{1, 1}
			p := ic.AddNewChild(KiT_Path, "p").(*Path)
			p.SetData("M 0.05 0.95 .95 0.95 .5 .05 Z")
			iset[ic.Nm] = &ic
		}
		{
			ic := Icon{}
			ic.InitName(&ic, "widget-wedge-left")
			ic.ViewBox.Size = Vec2D{1, 1}
			p := ic.AddNewChild(KiT_Path, "p").(*Path)
			p.SetData("M 0.95 0.05 .95 0.95 .05 .5 Z")
			iset[ic.Nm] = &ic
		}
		{
			ic := Icon{}
			ic.InitName(&ic, "widget-wedge-right")
			ic.ViewBox.Size = Vec2D{1, 1}
			p := ic.AddNewChild(KiT_Path, "p").(*Path)
			p.SetData("M 0.05 0.05 .05 0.95 .95 .5 Z")
			iset[ic.Nm] = &ic
		}
		{
			ic := Icon{}
			ic.InitName(&ic, "widget-checkmark")
			ic.ViewBox.Size = Vec2D{1, 1}
			p := ic.AddNewChild(KiT_Path, "p").(*Path)
			p.SetProp("stroke-width", units.NewValue(20, units.Pct))
			p.SetProp("fill", "none")
			p.SetData("M 0.1 0.5 .5 0.9 .9 .1")
			iset[ic.Nm] = &ic
		}
		{
			ic := Icon{}
			ic.InitName(&ic, "widget-checked-box")
			ic.ViewBox.Size = Vec2D{1, 1}
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
			ic.ViewBox.Size = Vec2D{1, 1}
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
			ic.ViewBox.Size = Vec2D{1, 1}
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
			ic.ViewBox.Size = Vec2D{1, 1}
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
			ic.SetProp("stroke-width", units.NewValue(5, units.Pct))
			ic.ViewBox.Size = Vec2D{1, 1}
			c0 := ic.AddNewChild(KiT_Circle, "c0").(*Circle)
			c0.Pos.Set(0.5, 0.15)
			c0.Radius = 0.12
			c1 := ic.AddNewChild(KiT_Circle, "c1").(*Circle)
			c1.Pos.Set(0.5, 0.5)
			c1.Radius = 0.12
			c2 := ic.AddNewChild(KiT_Circle, "c2").(*Circle)
			c2.Pos.Set(0.5, 0.85)
			c2.Radius = 0.12
			iset[ic.Nm] = &ic
		}
	}
	if false {
		ic := Icon{}
		ic.InitName(&ic, "astronaut")
		ic.Filename = "/Users/oreilly/go/src/github.com/srwiley/oksvg/testdata/testIcons/astronaut.svg"
		iset[ic.Nm] = &ic
	}
	if false {
		ic := Icon{}
		ic.InitName(&ic, "test")
		//		ic.Filename = "/Users/oreilly/go/src/github.com/goki/gi/icons/actions/adjusthsl.svg"
		ic.Filename = "/Users/oreilly/github/svg-icon/dist/svg/awesome/adn.svg"
		iset[ic.Nm] = &ic
	}

	iset.LoadDefaultIcons()

	return &iset
}

func (iset *IconSet) LoadDefaultIcons() error {
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		gopath = build.Default.GOPATH
	}
	path := filepath.Join(gopath, "src/github.com/goki/gi/icons")
	// path = "/Users/oreilly/github/svg-icon/dist/svg/simple"
	fmt.Printf("loading default icons: %v\n", path)
	return iset.LoadIconsFromPath(path)
}

// LoadIconsFromPath scans for .svg icon files in given path, adding them to
// the given IconSet, just storing the filename for later lazy loading
func (iset *IconSet) LoadIconsFromPath(path string) error {
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
