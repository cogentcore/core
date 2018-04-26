// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"
	"log"
	"sort"

	"github.com/rcoreilly/goki/gi/units"
	"github.com/rcoreilly/goki/ki"
	"github.com/rcoreilly/goki/ki/kit"
)

// Qt has different icon states -- seems over-complicated -- just use a map of icons

// different types of icon state
// QIcon::Normal	0	Display the pixmap when the user is not interacting with the icon, but the functionality represented by the icon is available.
// QIcon::Disabled	1	Display the pixmap when the functionality represented by the icon is not available.
// QIcon::Active	2	Display the pixmap when the functionality represented by the icon is available and the user is interacting with the icon, for example, moving the mouse over it or clicking it.
// QIcon::Selected	3	Display the pixmap when the item represented by the icon is selected.

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
	Rendered     bool        `json:"-" xml:"-" desc:"we have already rendered at RenderedSize -- doesn't re-render at same size -- if the paint params change, set this to false to re-render"`
	RenderedSize image.Point `json:"-" xml:"-" desc:"size at which we previously rendered"`
}

var KiT_Icon = kit.Types.AddType(&Icon{}, nil)

func (vp *Icon) Init2D() {
	vp.SVG.Init2D()
	vp.Fill = true
}

// copy from a source icon, typically one from a library -- preserves all the exisiting render state etc for the current icon, so that only a new render is required
func (vp *Icon) CopyFromIcon(cp *Icon) {
	oldIc := *vp
	vp.CopyFrom(cp)
	vp.Rendered = false
	vp.Viewport = oldIc.Viewport
	vp.LayData = oldIc.LayData
	vp.VpBBox = oldIc.VpBBox
	vp.WinBBox = oldIc.WinBBox
	vp.ViewBox = oldIc.ViewBox
	vp.Style = oldIc.Style
	vp.Fill = oldIc.Fill
	vp.Pixels = nil
	vp.Resize(vp.ViewBox.Size.X, vp.ViewBox.Size.Y)
	// vp.FullRender2DTree()
	vp.LayData = oldIc.LayData
	vp.VpBBox = oldIc.VpBBox
	vp.WinBBox = oldIc.WinBBox
	vp.ViewBox = oldIc.ViewBox
	vp.Rendered = false // not yet..
}

var IconProps = []ki.Props{
	{ // widget props
		"background-color": "transparent",
	}, { // paint props
		"fill":   "blue",
		"stroke": "black",
	},
}

func (vp *Icon) Style2D() {
	vp.Style2DWidget(IconProps[0])
	vp.Style2DSVG(IconProps[1])
}

func (vp *Icon) ReStyle2D() {
	vp.ReStyle2DWidget()
	vp.ReStyle2DSVG()
}

func (vp *Icon) Size2D() {
	vp.SVG.Size2D()
}

func (vp *Icon) Layout2D(parBBox image.Rectangle) {
	vp.Layout2DBase(parBBox, true)
	pc := &vp.Paint
	rs := &vp.Render
	vp.SetNormXForm()
	rs.PushXForm(pc.XForm) // need xforms to get proper bboxes during layout
	vp.Layout2DChildren()
	rs.PopXForm()
}

func (vp *Icon) Render2D() {
	if vp.PushBounds() {
		if !(vp.Rendered && vp.RenderedSize == vp.ViewBox.Size) {
			pc := &vp.Paint
			rs := &vp.Render
			if vp.Fill {
				// fmt.Printf("icon %v fill bg %v\n", vp.PathUnique(), vp.Style.Background.Color)
				vp.FillViewport()
			}
			vp.SetNormXForm()
			rs.PushXForm(pc.XForm)
			// fmt.Printf("IconRender: %v Bg: %v Fill: %v Clr: %v Stroke: %v\n",
			// 	vp.PathUnique(), vp.Style.Background.Color, vp.Paint.FillStyle.Color, vp.Style.Color, vp.Paint.StrokeStyle.Color)
			vp.Render2DChildren() // we must do children first, then us!
			vp.PopBounds()
			rs.PopXForm()
			vp.Rendered = true
			vp.RenderedSize = vp.ViewBox.Size
		}
		vp.RenderViewport2D() // update our parent image
	}
}

// check for interface implementation
var _ Node2D = &Icon{}

// icon lists
// https://fontawesome.com/
// https://joekuan.wordpress.com/2015/09/23/list-of-qt-icons/
// https://leungwensen.github.io/svg-icon/
// golang.org/x/exp/shiny/materialdesign/icons/ -- material encoded icons

// different types of standard icon name spaces, from https://standards.freedesktop.org/icon-naming-spec/icon-naming-spec-latest.html -- we organize our IconSets into these different contexts
type IconContexts int32

const (
	// Icons that are used as parts of standard widgets -- these are available built-in
	WidgetIcons IconContexts = iota
	// Icons which are generally used in menus and dialogs for interacting with the user.
	ActionIcons
	// Animated images used to represent loading web sites, or other background processing which may be less suited to more verbose progress reporting in the user interface.
	AnimationIcons
	// Icons that describe what an application is, for use in the Programs menu, window decorations, and the task list. These may or may not be generic depending on the application and its purpose.
	ApplicationIcons
	// Icons that are used for categories in the Programs menu, or the Control Center, for separating applications, preferences, and settings for display to the user.
	CategoryIcons
	// Icons for hardware that is contained within or connected to the computing device. Naming for extended devices in this group, is of the form <primary function>-<manufacturer>-<model>. This allows ease of fallback to the primary function device name, or ones more targeted for a specific series of models from a manufacturer.
	DeviceIcons
	// Icons for tags and properties of files, that are displayed in the file manager. This context contains emblems for such things as read-only or photos
	EmblemIcons
	// Icons for emotions that are expressed through text chat applications such as :-) or :-P in IRC or instant messengers.
	EmoteIcons
	// Icons for international denominations such as flags.
	IntnlIcons
	// Icons for different types of data, such as audio or image files.
	MimeIcons
	// Icons used to represent locations, either on the local filesystem, or through remote connections. Folders, trash, and workgroups are some example.
	PlaceIcons
	// Icons for presenting status to the user. This context contains icons for warning and error dialogs, as well as for the current weather, appointment alarms, and battery status
	StatusIcons
	IconContextsN
)

//go:generate stringer -type=IconContexts

var KiT_IconContexts = kit.Enums.AddEnum(IconContextsN, false, nil)

// list of standard icon names that we expect to find in an IconSet
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

// IconByName is main function to get icon by name -- looks in CurIconSet and falls back to DefaultIconSet if not found there -- logs a message and returns nil if not found
func IconByName(name string) *Icon {
	ic, ok := (*CurIconSet)[name]
	if !ok {
		ic, ok = (*DefaultIconSet)[name]
		if !ok {
			// todo: look on StdIconNames to see if it is not a standard name..
			log.Printf("gi.IconByName: unable to find icon name in either CurIconSet or DefaultIconSet: %v\n", name)

			return nil
		}
	}
	return ic
}

// IconListSorted returns a slice of all the icons in the icon set sorted by name
func IconListSorted(is IconSet) []*Icon {
	il := make([]*Icon, len(is))
	idx := 0
	for _, ic := range is {
		il[idx] = ic
		idx++
	}
	sort.Slice(il, func(i, j int) bool {
		return il[i].Name() < il[j].Name()
	})
	return il
}

// note: icons must use a normalized 0-1 coordinate system!
func MakeDefaultIcons() *IconSet {
	iset := make(IconSet, 100)
	{
		wd := Icon{}
		wd.InitName(&wd, "widget-wedge-down")
		p := wd.AddNewChild(KiT_Path, "p").(*Path)
		p.Data = PathDataParse("M 0.05 0.05 .95 0.05 .5 .95 Z")
		iset[wd.Nm] = &wd
	}
	{
		wd := Icon{}
		wd.InitName(&wd, "widget-wedge-up")
		p := wd.AddNewChild(KiT_Path, "p").(*Path)
		p.Data = PathDataParse("M 0.05 0.95 .95 0.95 .5 .05 Z")
		iset[wd.Nm] = &wd
	}
	{
		wd := Icon{}
		wd.InitName(&wd, "widget-wedge-left")
		p := wd.AddNewChild(KiT_Path, "p").(*Path)
		p.Data = PathDataParse("M 0.95 0.05 .95 0.95 .05 .5 Z")
		iset[wd.Nm] = &wd
	}
	{
		wd := Icon{}
		wd.InitName(&wd, "widget-wedge-right")
		p := wd.AddNewChild(KiT_Path, "p").(*Path)
		p.Data = PathDataParse("M 0.05 0.05 .05 0.95 .95 .5 Z")
		iset[wd.Nm] = &wd
	}
	{
		wd := Icon{}
		wd.InitName(&wd, "widget-checkmark")
		p := wd.AddNewChild(KiT_Path, "p").(*Path)
		p.SetProp("stroke-width", units.NewValue(0.15, units.Pct))
		p.SetProp("fill", "none")
		p.Data = PathDataParse("M 0.1 0.5 .5 0.9 .9 .1")
		iset[wd.Nm] = &wd
	}
	{
		wd := Icon{}
		wd.InitName(&wd, "widget-checked-box")
		bx := wd.AddNewChild(KiT_Rect, "bx").(*Rect)
		bx.Pos.Set(0.05, 0.05)
		bx.Size.Set(0.9, 0.9)
		// bx.Radius.Set(0.02, 0.02)
		p := wd.AddNewChild(KiT_Path, "p").(*Path)
		p.SetProp("stroke-width", units.NewValue(0.15, units.Pct))
		p.SetProp("fill", "none")
		p.Data = PathDataParse("M 0.2 0.5 .5 0.8 .8 .2")
		iset[wd.Nm] = &wd
	}
	{
		wd := Icon{}
		wd.InitName(&wd, "widget-unchecked-box")
		bx := wd.AddNewChild(KiT_Rect, "bx").(*Rect)
		bx.Pos.Set(0.05, 0.05)
		bx.Size.Set(0.9, 0.9)
		// bx.Radius.Set(0.02, 0.02) // not rendering well at small sizes
		iset[wd.Nm] = &wd
	}
	{
		wd := Icon{}
		wd.InitName(&wd, "widget-circlebutton-on")
		oc := wd.AddNewChild(KiT_Circle, "oc").(*Circle)
		oc.Pos.Set(0.5, 0.5)
		oc.Radius = 0.4
		oc.SetProp("fill", "none")
		oc.SetProp("stroke-width", units.NewValue(0.1, units.Pct))
		ic := wd.AddNewChild(KiT_Circle, "ic").(*Circle)
		ic.Pos.Set(0.5, 0.5)
		ic.Radius = 0.2
		iset[wd.Nm] = &wd
	}
	{
		wd := Icon{}
		wd.InitName(&wd, "widget-circlebutton-off")
		oc := wd.AddNewChild(KiT_Circle, "oc").(*Circle)
		oc.Pos.Set(0.5, 0.5)
		oc.Radius = 0.4
		oc.SetProp("fill", "none")
		oc.SetProp("stroke-width", units.NewValue(0.1, units.Pct))
		iset[wd.Nm] = &wd
	}
	{
		wd := Icon{}
		wd.InitName(&wd, "widget-handle-circles")
		c0 := wd.AddNewChild(KiT_Circle, "c0").(*Circle)
		c0.Pos.Set(0.5, 0.15)
		c0.Radius = 0.1
		c1 := wd.AddNewChild(KiT_Circle, "c1").(*Circle)
		c1.Pos.Set(0.5, 0.5)
		c1.Radius = 0.1
		c2 := wd.AddNewChild(KiT_Circle, "c2").(*Circle)
		c2.Pos.Set(0.5, 0.85)
		c2.Radius = 0.1
		iset[wd.Nm] = &wd
	}
	return &iset
}
