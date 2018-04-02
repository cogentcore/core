// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	// "fmt"
	"image"

	"github.com/rcoreilly/goki/ki/kit"
	// "image/draw"
	// "log"
	// "sync"
	// "time"
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
// for a given set of fill / stroke paint values, as an optimization (later).
type Icon struct {
	SVG
	Rendered bool `json:"-" xml:"-" desc:"we have already rendered at our current size -- doesn't re-render"`
}

var KiT_Icon = kit.Types.AddType(&Icon{}, nil)

func (vp *Icon) AsNode2D() *Node2DBase {
	return &vp.Node2DBase
}

func (vp *Icon) AsViewport2D() *Viewport2D {
	return &vp.Viewport2D
}

func (vp *Icon) AsLayout2D() *Layout {
	return nil
}

func (vp *Icon) Init2D() {
	vp.SVG.Init2D()
}

func (vp *Icon) Style2D() {
	vp.SVG.Style2D()
}

func (vp *Icon) Size2D() {
	vp.SVG.Size2D()
}

func (vp *Icon) Layout2D(parBBox image.Rectangle) {
	vp.SVG.Layout2D(parBBox)
}

func (vp *Icon) BBox2D() image.Rectangle {
	return vp.SVG.BBox2D()
}

func (vp *Icon) ComputeBBox2D(parBBox image.Rectangle) Vec2D {
	return vp.SVG.ComputeBBox2D(parBBox)
}

func (vp *Icon) ChildrenBBox2D() image.Rectangle {
	return vp.SVG.ChildrenBBox2D()
}

func (vp *Icon) Render2D() {
	// todo: check rendered -- don't re-render
	// set scaling to normalized 0-1 coords -- todo: check actual width, height etc
	pc := &vp.Paint
	vps := Vec2D{}
	vps.SetPoint(vp.ViewBox.Size)
	pc.Identity()
	pc.Scale(vps.X, vps.Y)
	vp.SVG.Render2D()
}

func (vp *Icon) CanReRender2D() bool {
	return true
}

func (vp *Icon) FocusChanged2D(gotFocus bool) {
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
		"widget-down-wedge",
		"widget-up-wedge",
		"widget-left-wedge",
		"widget-right-wedge",
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

// note: icons must use a normalized 0-1 coordinate system!
func MakeDefaultIcons() *IconSet {
	iset := make(IconSet, 100)
	wd := Icon{}
	wd.InitName(&wd, "widget-down-wedge")
	p := wd.AddNewChildNamed(KiT_Path, "p").(*Path)
	p.Data = ParsePathData("M 0 0 1 0 .5 1 Z")
	iset[wd.Nm] = &wd
	return &iset
}
