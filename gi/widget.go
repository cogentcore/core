// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"
	"log"
	"strings"
	"sync"

	"github.com/goki/gi/girl"
	"github.com/goki/gi/gist"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/mouse"
	"github.com/goki/gi/units"
	"github.com/goki/ki/ints"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/mat32"
)

// WidgetBase is the base type for all Widget Node2D elements, which are
// managed by a containing Layout, and use all 5 rendering passes.  All
// elemental widgets must support the Node Inactive and Selected states in a
// reasonable way (Selected only essential when also Inactive), so they can
// function appropriately in a chooser (e.g., SliceView or TableView) -- this
// includes toggling selection on left mouse press.
type WidgetBase struct {
	Node2DBase
	Tooltip      string       `desc:"text for tooltip for this widget -- can use HTML formatting"`
	Sty          gist.Style   `json:"-" xml:"-" desc:"styling settings for this widget -- set in SetStyle2D during an initialization step, and when the structure changes"`
	DefStyle     *gist.Style  `copy:"-" view:"-" json:"-" xml:"-" desc:"default style values computed by a parent widget for us -- if set, we are a part of a parent widget and should use these as our starting styles instead of type-based defaults"`
	LayState     LayoutState  `copy:"-" json:"-" xml:"-" desc:"all the layout state information for this item"`
	WidgetSig    ki.Signal    `copy:"-" json:"-" xml:"-" view:"-" desc:"general widget signals supported by all widgets, including select, focus, and context menu (right mouse button) events, which can be used by views and other compound widgets"`
	CtxtMenuFunc CtxtMenuFunc `copy:"-" view:"-" json:"-" xml:"-" desc:"optional context menu function called by MakeContextMenu AFTER any native items are added -- this function can decide where to insert new elements -- typically add a separator to disambiguate"`
	StyMu        sync.RWMutex `copy:"-" view:"-" json:"-" xml:"-" desc:"mutex protecting updates to the style"`
}

var KiT_WidgetBase = kit.Types.AddType(&WidgetBase{}, WidgetBaseProps)

var WidgetBaseProps = ki.Props{
	"base-type":     true,
	"EnumType:Flag": KiT_NodeFlags,
}

func (wb *WidgetBase) CopyFieldsFrom(frm interface{}) {
	fr, ok := frm.(*WidgetBase)
	if !ok {
		log.Printf("GoGi node of type: %v needs a CopyFieldsFrom method defined -- currently falling back on earlier WidgetBase one\n", ki.Type(wb).Name())
		ki.GenCopyFieldsFrom(wb.This(), frm)
		return
	}
	wb.Node2DBase.CopyFieldsFrom(&fr.Node2DBase)
	wb.Tooltip = fr.Tooltip
	wb.Sty.CopyFrom(&fr.Sty)
}

func (wb *WidgetBase) Disconnect() {
	wb.Node2DBase.Disconnect()
	wb.WidgetSig.DisconnectAll()
}

func (wb *WidgetBase) AsWidget() *WidgetBase {
	return wb
}

// Style satisfies the Styler interface
func (wb *WidgetBase) Style() *gist.Style {
	return &wb.Sty
}

// StyleRLock does a read-lock for reading the style
func (wb *WidgetBase) StyleRLock() {
	wb.StyMu.RLock()
}

// StyleRUnlock unlocks the read-lock
func (wb *WidgetBase) StyleRUnlock() {
	wb.StyMu.RUnlock()
}

// BoxSpace returns the style BoxSpace value under read lock
func (wb *WidgetBase) BoxSpace() float32 {
	wb.StyMu.RLock()
	bs := wb.Sty.BoxSpace()
	wb.StyMu.RUnlock()
	return bs
}

// Init2DWidget handles basic node initialization -- Init2D can then do special things
func (wb *WidgetBase) Init2DWidget() {
	wb.BBoxMu.Lock()
	wb.StyMu.Lock()
	wb.Viewport = wb.ParentViewport()
	wb.Sty.Defaults()
	wb.StyMu.Unlock()
	wb.LayState.Defaults() // doesn't overwrite
	wb.BBoxMu.Unlock()
	wb.ConnectToViewport()
}

func (wb *WidgetBase) Init2D() {
	wb.Init2DWidget()
}

// WidgetDefStyleKey is the key for accessing the default style stored in the
// type-properties for a given type -- also ones with sub-selectors for parts
// in there with selector appended to this key
var WidgetDefStyleKey = "__DefStyle"

// WidgetDefPropsKey is the key for accessing the default style properties
// stored in the type-properties for a given type -- also ones with
// sub-selectors for parts in there with selector appended to this key
var WidgetDefPropsKey = "__DefProps"

// DefaultStyle2DWidget retrieves default style object for the type, from type
// properties -- selector is optional selector for state etc.  Property key is
// "__DefStyle" + selector -- if part != nil, then use that obj for getting
// the default style starting point when creating a new style.  Also stores a
// "__DefProps"+selector type property of the props used for styling here, for
// accessing properties that are not compiled into standard Style object.
func DefaultStyle2DWidget(wb *WidgetBase, selector string, part *WidgetBase) *gist.Style {
	tprops := *kit.Types.Properties(ki.Type(wb), true) // true = makeNew
	styprops := tprops
	if selector != "" {
		sp, ok := kit.TypeProp(tprops, selector)
		if !ok {
			// log.Printf("gi.DefaultStyle2DWidget: did not find props for style selector: %v for node type: %v\n", selector, ki.Type(wb).Name())
		} else {
			spm, ok := sp.(ki.Props)
			if !ok {
				log.Printf("gi.DefaultStyle2DWidget: looking for a ki.Props for style selector: %v, instead got type: %T, for node type: %v\n", selector, spm, ki.Type(wb).Name())
			} else {
				styprops = spm
			}
		}
	}

	parSty := wb.ParentStyle()

	var dsty *gist.Style
	stKey := WidgetDefStyleKey + selector
	prKey := WidgetDefPropsKey + selector
	dstyi, ok := kit.TypeProp(tprops, stKey)
	if !ok || gist.RebuildDefaultStyles {
		dsty = &gist.Style{}
		dsty.Defaults()
		if selector != "" {
			var baseStyle *gist.Style
			if part != nil {
				baseStyle = DefaultStyle2DWidget(part, "", nil)
			} else {
				baseStyle = DefaultStyle2DWidget(wb, "", nil)
			}
			*dsty = *baseStyle
		}
		kit.TypesMu.Lock() // write lock
		dsty.SetStyleProps(parSty, styprops, wb.Viewport)
		dsty.IsSet = false // keep as non-set
		tprops[stKey] = dsty
		tprops[prKey] = styprops
		kit.TypesMu.Unlock()
	} else {
		dsty, _ = dstyi.(*gist.Style)
	}
	wb.ParentStyleRUnlock()
	return dsty
}

// Style2DWidget styles the Style values from node properties and optional
// base-level defaults -- for Widget-style nodes.
// must be called under a StyMu Lock
func (wb *WidgetBase) Style2DWidget() {
	// pr := prof.Start("Style2DWidget")
	// defer pr.End()

	gii, _ := wb.This().(Node2D)
	wb.Viewport.SetCurStyleNode(gii)
	defer wb.Viewport.SetCurStyleNode(nil)

	if !gist.RebuildDefaultStyles && wb.DefStyle != nil {
		wb.Sty.CopyFrom(wb.DefStyle)
	} else {
		wb.Sty.CopyFrom(DefaultStyle2DWidget(wb, "", nil))
	}
	wb.Sty.IsSet = false    // this is always first call, restart
	if wb.Viewport == nil { // robust
		wb.StyMu.Unlock()
		gii.Init2D()
		wb.StyMu.Lock()
	}
	styprops := *wb.Properties()
	parSty := wb.ParentStyle()
	wb.Sty.SetStyleProps(parSty, styprops, wb.Viewport)

	// look for class-specific style sheets among defaults -- have to do these
	// dynamically now -- cannot compile into default which is type-general
	tprops := *kit.Types.Properties(ki.Type(wb), true) // true = makeNew
	kit.TypesMu.RLock()
	classes := strings.Split(strings.ToLower(wb.Class), " ")
	for _, cl := range classes {
		clsty := "." + strings.TrimSpace(cl)
		if sp, ok := ki.SubProps(tprops, clsty); ok {
			wb.Sty.SetStyleProps(parSty, sp, wb.Viewport)
		}
	}
	kit.TypesMu.RUnlock()
	wb.ParentStyleRUnlock()

	pagg := wb.ParentCSSAgg()
	if pagg != nil {
		AggCSS(&wb.CSSAgg, *pagg)
	} else {
		wb.CSSAgg = nil // restart
	}
	AggCSS(&wb.CSSAgg, wb.CSS)
	StyleCSS(gii, wb.Viewport, &wb.Sty, wb.CSSAgg, "")

	SetUnitContext(&wb.Sty, wb.Viewport, mat32.Vec2Zero) // todo: test for use of el-relative
	if wb.Sty.Inactive {                                 // inactive can only set, not clear
		wb.SetInactive()
	}

	wb.Viewport.SetCurrentColor(wb.Sty.Font.Color)
}

// StylePart sets the style properties for a child in parts (or any other
// child) based on its name -- only call this when new parts were created --
// name of properties is #partname (lower cased) and it should contain a
// ki.Props which is then added to the part's props -- this provides built-in
// defaults for parts, so it is separate from the CSS process
func (wb *WidgetBase) StylePart(pk Node2D) {
	if pk == nil {
		return
	}
	pg := pk.AsWidget()
	if pg == nil {
		return
	}
	// pr := prof.Start("StylePart")
	// defer pr.End()
	// if pg.DefStyle != nil && !RebuildDefaultStyles { // already set
	// 	return
	// }
	stynm := "#" + strings.ToLower(pk.Name())
	// this is called on US (the parent object) so we store the #partname
	// default style within our type properties..  that's good -- HOWEVER we
	// cannot put any sub-selector properties within these part styles -- must
	// all be in the base-level.. hopefully that works..
	pdst := DefaultStyle2DWidget(wb, stynm, pg)
	pg.DefStyle = pdst // will use this as starting point for all styles now..

	if ics := pk.Embed(KiT_Icon); ics != nil {
		ic := ics.(*Icon)
		styprops := kit.Types.Properties(ki.Type(wb), true)
		if sp, ok := ki.SubProps(*styprops, stynm); ok {
			if fill, ok := sp["fill"]; ok {
				ic.SetProp("fill", fill)
			}
			if stroke, ok := sp["stroke"]; ok {
				ic.SetProp("stroke", stroke)
			}
		}
		if sp, ok := ki.SubProps(*wb.Properties(), stynm); ok {
			for k, v := range sp {
				ic.SetProp(k, v)
			}
		}
		ic.SetFullReRender()
	}
}

// ApplyCSS applies css styles for given node, using key to select sub-props
// from overall properties list, and optional selector to select a further
// :name selector within that key
func ApplyCSS(node Node2D, vp *Viewport2D, st *gist.Style, css ki.Props, key, selector string) bool {
	pp, got := css[key]
	if !got {
		return false
	}
	pmap, ok := pp.(ki.Props) // must be a props map
	if !ok {
		return false
	}
	if selector != "" {
		pmap, ok = gist.SubProps(pmap, selector)
		if !ok {
			return false
		}
	}
	parSty := node.AsNode2D().ParentStyle()
	st.SetStyleProps(parSty, pmap, vp)
	node.AsNode2D().ParentStyleRUnlock()
	return true
}

// StyleCSS applies css style properties to given Widget node, parsing out
// type, .class, and #name selectors, along with optional sub-selector
// (:hover, :active etc)
func StyleCSS(node Node2D, vp *Viewport2D, st *gist.Style, css ki.Props, selector string) {
	tyn := strings.ToLower(ki.Type(node).Name()) // type is most general, first
	ApplyCSS(node, vp, st, css, tyn, selector)
	classes := strings.Split(strings.ToLower(node.AsNode2D().Class), " ")
	for _, cl := range classes {
		cln := "." + strings.TrimSpace(cl)
		ApplyCSS(node, vp, st, css, cln, selector)
	}
	idnm := "#" + strings.ToLower(node.Name()) // then name
	ApplyCSS(node, vp, st, css, idnm, selector)
}

func (wb *WidgetBase) Style2D() {
	wb.StyMu.Lock()
	defer wb.StyMu.Unlock()

	hasTempl, saveTempl := wb.Sty.FromTemplate()
	if !hasTempl || saveTempl {
		wb.Style2DWidget()
	}
	if hasTempl && saveTempl {
		wb.Sty.SaveTemplate()
	}
	wb.LayState.SetFromStyle(&wb.Sty.Layout) // also does reset
}

// SetUnitContext sets the unit context based on size of viewport and parent
// element (from bbox) and then cache everything out in terms of raw pixel
// dots for rendering -- call at start of render
func SetUnitContext(st *gist.Style, vp *Viewport2D, el mat32.Vec2) {
	if vp != nil {
		if vp.Win != nil {
			st.UnContext.DPI = vp.Win.LogicalDPI()
		}
		if vp.Render.Image != nil {
			sz := vp.Render.Image.Bounds().Size()
			st.UnContext.SetSizes(float32(sz.X), float32(sz.Y), el.X, el.Y)
		}
	}
	girl.OpenFont(&st.Font, &st.UnContext) // calls SetUnContext after updating metrics
	st.ToDots()
}

func (wb *WidgetBase) InitLayout2D() bool {
	wb.StyMu.Lock()
	defer wb.StyMu.Unlock()
	wb.LayState.SetFromStyle(&wb.Sty.Layout)
	return false
}

func (wb *WidgetBase) Size2DBase(iter int) {
	wb.InitLayout2D()
}

func (wb *WidgetBase) Size2D(iter int) {
	wb.Size2DBase(iter)
}

// AddParentPos adds the position of our parent to our layout position --
// layout computations are all relative to parent position, so they are
// finally cached out at this stage also returns the size of the parent for
// setting units context relative to parent objects
func (wb *WidgetBase) AddParentPos() mat32.Vec2 {
	if pni, _ := KiToNode2D(wb.Par); pni != nil {
		if pw := pni.AsWidget(); pw != nil {
			if !wb.IsField() {
				wb.LayState.Alloc.Pos = pw.LayState.Alloc.PosOrig.Add(wb.LayState.Alloc.PosRel)
			}
			return pw.LayState.Alloc.Size
		}
	}
	return mat32.Vec2Zero
}

// BBoxFromAlloc gets our bbox from Layout allocation.
func (wb *WidgetBase) BBoxFromAlloc() image.Rectangle {
	return mat32.RectFromPosSizeMax(wb.LayState.Alloc.Pos, wb.LayState.Alloc.Size)
}

func (wb *WidgetBase) BBox2D() image.Rectangle {
	return wb.BBoxFromAlloc()
}

func (wb *WidgetBase) ComputeBBox2D(parBBox image.Rectangle, delta image.Point) {
	wb.ComputeBBox2DBase(parBBox, delta)
}

// Layout2DBase provides basic Layout2D functions -- good for most cases
func (wb *WidgetBase) Layout2DBase(parBBox image.Rectangle, initStyle bool, iter int) {
	nii, _ := wb.This().(Node2D)
	mvp := wb.ViewportSafe()
	if mvp == nil { // robust
		if nii.AsViewport2D() == nil {
			// todo: not so clear that this will do anything useful at this point
			// but at least it gets the viewport
			nii.Init2D()
			nii.Style2D()
			nii.Size2D(0)
			// fmt.Printf("node not init in Layout2DBase: %v\n", wb.Path())
		}
	}
	psize := wb.AddParentPos()
	wb.LayState.Alloc.PosOrig = wb.LayState.Alloc.Pos
	if initStyle {
		mvp := wb.ViewportSafe()
		SetUnitContext(&wb.Sty, mvp, psize) // update units with final layout
	}
	wb.BBox = nii.BBox2D() // only compute once, at this point
	// note: if other styles are maintained, they also need to be updated!
	nii.ComputeBBox2D(parBBox, image.ZP) // other bboxes from BBox
	if Layout2DTrace {
		fmt.Printf("Layout: %v alloc pos: %v size: %v vpbb: %v winbb: %v\n", wb.Path(), wb.LayState.Alloc.Pos, wb.LayState.Alloc.Size, wb.VpBBox, wb.WinBBox)
	}
	// typically Layout2DChildren must be called after this!
}

func (wb *WidgetBase) Layout2D(parBBox image.Rectangle, iter int) bool {
	wb.Layout2DBase(parBBox, true, iter)
	return wb.Layout2DChildren(iter)
}

// ChildrenBBox2DWidget provides a basic widget box-model subtraction of
// margin and padding to children -- call in ChildrenBBox2D for most widgets
func (wb *WidgetBase) ChildrenBBox2DWidget() image.Rectangle {
	nb := wb.VpBBox
	spc := int(wb.BoxSpace())
	nb.Min.X += spc
	nb.Min.Y += spc
	nb.Max.X -= spc
	nb.Max.Y -= spc
	return nb
}

func (wb *WidgetBase) ChildrenBBox2D() image.Rectangle {
	return wb.ChildrenBBox2DWidget()
}

// FullReRenderIfNeeded tests if the FullReRender flag has been set, and if
// so, calls ReRender2DTree and returns true -- call this at start of each
// Render2D
func (wb *WidgetBase) FullReRenderIfNeeded() bool {
	mvp := wb.ViewportSafe()
	if wb.This().(Node2D).IsVisible() && wb.NeedsFullReRender() && !mvp.IsDoingFullRender() {
		if Render2DTrace {
			fmt.Printf("Render: NeedsFullReRender for %v at %v\n", wb.Path(), wb.VpBBox)
		}
		wb.ClearFullReRender()
		wb.ReRender2DTree()
		return true
	}
	return false
}

// PushBounds pushes our bounding-box bounds onto the bounds stack if non-empty
// -- this limits our drawing to our own bounding box, automatically -- must
// be called as first step in Render2D returns whether the new bounds are
// empty or not -- if empty then don't render!
func (wb *WidgetBase) PushBounds() bool {
	if wb == nil || wb.This() == nil {
		return false
	}
	if !wb.This().(Node2D).IsVisible() {
		return false
	}
	if wb.VpBBox.Empty() {
		wb.ClearFullReRender()
		return false
	}
	mvp := wb.ViewportSafe()
	rs := &mvp.Render
	rs.PushBounds(wb.VpBBox)
	wb.ConnectToViewport()
	if Render2DTrace {
		fmt.Printf("Render: %v at %v\n", wb.Path(), wb.VpBBox)
	}
	return true
}

// PopBounds pops our bounding-box bounds -- last step in Render2D after
// rendering children
func (wb *WidgetBase) PopBounds() {
	wb.ClearFullReRender()
	if wb.This() == nil {
		return
	}
	mvp := wb.ViewportSafe()
	if mvp == nil {
		return
	}
	rs := &mvp.Render
	rs.PopBounds()
}

func (wb *WidgetBase) Render2D() {
	if wb.FullReRenderIfNeeded() {
		return
	}
	if wb.PushBounds() {
		wb.This().(Node2D).ConnectEvents2D()
		wb.Render2DChildren()
		wb.PopBounds()
	} else {
		wb.DisconnectAllEvents(RegPri)
	}
}

// ReRender2DTree does a re-render of the tree -- after it has already been
// initialized and styled -- redoes the full stack
func (wb *WidgetBase) ReRender2DTree() {
	parBBox := image.ZR
	pni, _ := KiToNode2D(wb.Par)
	if pni != nil {
		parBBox = pni.ChildrenBBox2D()
	}
	delta := wb.LayState.Alloc.Pos.Sub(wb.LayState.Alloc.PosOrig)
	wb.LayState.Alloc.Pos = wb.LayState.Alloc.PosOrig
	ld := wb.LayState // save our current layout data
	updt := wb.UpdateStart()
	wb.Init2DTree()
	wb.Style2DTree()
	wb.Size2DTree(0)
	wb.LayState = ld // restore
	wb.Layout2DTree()
	if !delta.IsNil() {
		wb.Move2D(delta.ToPointFloor(), parBBox)
	}
	wb.Render2DTree()
	wb.UpdateEndNoSig(updt)
}

// Move2DBase does the basic move on this node
func (wb *WidgetBase) Move2DBase(delta image.Point, parBBox image.Rectangle) {
	wb.LayState.Alloc.Pos = wb.LayState.Alloc.PosOrig.Add(mat32.NewVec2FmPoint(delta))
	wb.This().(Node2D).ComputeBBox2D(parBBox, delta)
}

func (wb *WidgetBase) Move2D(delta image.Point, parBBox image.Rectangle) {
	wb.Move2DBase(delta, parBBox)
	wb.Move2DChildren(delta)
}

// Move2DTree does move2d pass -- each node iterates over children for maximum
// control -- this starts with parent ChildrenBBox and current delta -- can be
// called de novo
func (wb *WidgetBase) Move2DTree() {
	if wb.HasNoLayout() {
		return
	}
	parBBox := image.ZR
	pnii, pn := KiToNode2D(wb.Par)
	if pn != nil {
		parBBox = pnii.ChildrenBBox2D()
	}
	delta := wb.LayState.Alloc.Pos.Sub(wb.LayState.Alloc.PosOrig).ToPoint()
	wb.This().(Node2D).Move2D(delta, parBBox) // important to use interface version to get interface!
}

// ParentLayout returns the parent layout
func (wb *WidgetBase) ParentLayout() *Layout {
	var parLy *Layout
	wb.FuncUpParent(0, wb.This(), func(k ki.Ki, level int, d interface{}) bool {
		nii, ok := k.(Node2D)
		if !ok {
			return ki.Break // don't keep going up
		}
		ly := nii.AsLayout2D()
		if ly != nil {
			parLy = ly
			return ki.Break // done
		}
		return ki.Continue
	})
	return parLy
}

// CtxtMenuFunc is a function for creating a context menu for given node
type CtxtMenuFunc func(g Node2D, m *Menu)

func (wb *WidgetBase) MakeContextMenu(m *Menu) {
	// derived types put native menu code here
	if wb.CtxtMenuFunc != nil {
		wb.CtxtMenuFunc(wb.This().(Node2D), m)
	}
	mvp := wb.ViewportSafe()
	TheViewIFace.CtxtMenuView(wb.This(), wb.IsInactive(), mvp, m)
}

var TooltipFrameProps = ki.Props{
	"background-color":    &Prefs.Colors.Highlight,
	"border-width":        units.NewPx(0),
	"border-color":        "none",
	"margin":              units.NewPx(0),
	"padding":             units.NewPx(2),
	"box-shadow.h-offset": units.NewPx(0),
	"box-shadow.v-offset": units.NewPx(0),
	"box-shadow.blur":     units.NewPx(0),
	"box-shadow.color":    &Prefs.Colors.Shadow,
}

// PopupTooltip pops up a viewport displaying the tooltip text
func PopupTooltip(tooltip string, x, y int, parVp *Viewport2D, name string) *Viewport2D {
	win := parVp.Win
	mainVp := win.Viewport
	pvp := Viewport2D{}
	pvp.InitName(&pvp, name+"Tooltip")
	pvp.Win = win
	updt := pvp.UpdateStart()
	pvp.SetProp("color", &Prefs.Colors.Font)
	pvp.Fill = false
	pvp.SetFlag(int(VpFlagPopup))
	pvp.SetFlag(int(VpFlagTooltip))

	pvp.Geom.Pos = image.Point{x, y}
	pvp.SetFlag(int(VpFlagPopupDestroyAll)) // nuke it all
	frame := pvp.AddNewChild(KiT_Frame, "Frame").(*Frame)
	frame.Lay = LayoutVert
	frame.Properties().CopyFrom(TooltipFrameProps, ki.DeepCopy)
	lbl := frame.AddNewChild(KiT_Label, "ttlbl").(*Label)
	lbl.SetProp("white-space", gist.WhiteSpaceNormal) // wrap

	mwdots := parVp.Sty.UnContext.ToDots(40, units.Em)
	mwdots = mat32.Min(mwdots, float32(mainVp.Geom.Size.X-20))

	lbl.SetProp("max-width", units.NewDot(mwdots))
	lbl.Text = tooltip
	frame.Init2DTree()
	frame.Style2DTree()                                    // sufficient to get sizes
	frame.LayState.Alloc.Size = mainVp.LayState.Alloc.Size // give it the whole vp initially
	frame.Size2DTree(0)                                    // collect sizes
	pvp.Win = nil
	vpsz := frame.LayState.Size.Pref.Min(mainVp.LayState.Alloc.Size).ToPoint()

	x = ints.MinInt(x, mainVp.Geom.Size.X-vpsz.X) // fit
	y = ints.MinInt(y, mainVp.Geom.Size.Y-vpsz.Y) // fit
	pvp.Resize(vpsz)
	pvp.Geom.Pos = image.Point{x, y}
	pvp.UpdateEndNoSig(updt)

	win.PushPopup(pvp.This())
	return &pvp
}

// WidgetSignals are general signals that all widgets can send, via WidgetSig
// signal
type WidgetSignals int64

const (
	// WidgetSelected is triggered when a widget is selected, typically via
	// left mouse button click (see EmitSelectedSignal) -- is NOT contingent
	// on actual IsSelected status -- just reports the click event.
	// The data is the index of the selected item for multi-item widgets
	// (-1 = none / unselected)
	WidgetSelected WidgetSignals = iota

	// WidgetFocused is triggered when a widget receives keyboard focus (see
	// EmitFocusedSignal -- call in FocusChanged2D for gotFocus
	WidgetFocused

	// WidgetContextMenu is triggered when a widget receives a
	// right-mouse-button press, BEFORE generating and displaying the context
	// menu, so that relevant state can be updated etc (see
	// EmitContextMenuSignal)
	WidgetContextMenu

	WidgetSignalsN
)

//go:generate stringer -type=WidgetSignals

// EmitSelectedSignal emits the WidgetSelected signal for this widget
func (wb *WidgetBase) EmitSelectedSignal() {
	wb.WidgetSig.Emit(wb.This(), int64(WidgetSelected), nil)
}

// EmitFocusedSignal emits the WidgetFocused signal for this widget
func (wb *WidgetBase) EmitFocusedSignal() {
	wb.WidgetSig.Emit(wb.This(), int64(WidgetFocused), nil)
}

// EmitContextMenuSignal emits the WidgetContextMenu signal for this widget
func (wb *WidgetBase) EmitContextMenuSignal() {
	wb.WidgetSig.Emit(wb.This(), int64(WidgetContextMenu), nil)
}

// HoverTooltipEvent connects to HoverEvent and pops up a tooltip -- most
// widgets should call this as part of their event connection method
func (wb *WidgetBase) HoverTooltipEvent() {
	wb.ConnectEvent(oswin.MouseHoverEvent, RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		me := d.(*mouse.HoverEvent)
		wbb := recv.Embed(KiT_WidgetBase).(*WidgetBase)
		if wbb.Tooltip != "" {
			me.SetProcessed()
			pos := wbb.WinBBox.Max
			pos.X -= 20
			mvp := wbb.ViewportSafe()
			PopupTooltip(wbb.Tooltip, pos.X, pos.Y, mvp, wbb.Nm)
		}
	})
}

// WidgetMouseEvents connects to either or both mouse events -- IMPORTANT: if
// you need to also connect to other mouse events, you must copy this code --
// all processing of a mouse event must happen within one function b/c there
// can only be one registered per receiver and event type.  sel = Left button
// mouse.Press event, toggles the selected state, and emits a SelectedEvent.
// ctxtMenu = connects to Right button mouse.Press event, and sends a
// WidgetSig WidgetContextMenu signal, followed by calling ContextMenu method
// -- signal can be used to change state prior to generating context menu,
// including setting a CtxtMenuFunc that removes all items and thus negates
// the presentation of any menu
func (wb *WidgetBase) WidgetMouseEvents(sel, ctxtMenu bool) {
	if !sel && !ctxtMenu {
		return
	}
	wb.ConnectEvent(oswin.MouseEvent, RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		me := d.(*mouse.Event)
		if sel {
			if me.Action == mouse.Press && me.Button == mouse.Left {
				me.SetProcessed()
				wbb := recv.Embed(KiT_WidgetBase).(*WidgetBase)
				wbb.SetSelectedState(!wbb.IsSelected())
				wbb.EmitSelectedSignal()
				wbb.UpdateSig()
			}
		}
		if ctxtMenu {
			if me.Action == mouse.Release && me.Button == mouse.Right {
				me.SetProcessed()
				wbb := recv.Embed(KiT_WidgetBase).(*WidgetBase)
				wbb.EmitContextMenuSignal()
				wbb.This().(Node2D).ContextMenu()
			}
		}
	})
}

////////////////////////////////////////////////////////////////////////////////
//  Standard rendering

// RenderLock returns the locked girl.State, Paint, and Style with StyMu locked.
// This should be called at start of widget-level rendering.
func (wb *WidgetBase) RenderLock() (*girl.State, *girl.Paint, *gist.Style) {
	wb.StyMu.RLock()
	rs := &wb.Viewport.Render
	rs.Lock()
	return rs, &rs.Paint, &wb.Sty
}

// RenderUnlock unlocks girl.State and style
func (wb *WidgetBase) RenderUnlock(rs *girl.State) {
	rs.Unlock()
	wb.StyMu.RUnlock()
}

// RenderBoxImpl implements the standard box model rendering -- assumes all
// paint params have already been set
func (wb *WidgetBase) RenderBoxImpl(pos mat32.Vec2, sz mat32.Vec2, rad float32) {
	rs := &wb.Viewport.Render
	pc := &rs.Paint
	if rad == 0.0 {
		pc.DrawRectangle(rs, pos.X, pos.Y, sz.X, sz.Y)
	} else {
		pc.DrawRoundedRectangle(rs, pos.X, pos.Y, sz.X, sz.Y, rad)
	}
	pc.FillStrokeClear(rs)
}

// RenderStdBox draws standard box using given style.
// girl.State and Style must already be locked at this point (RenderLock)
func (wb *WidgetBase) RenderStdBox(st *gist.Style) {
	wb.StyMu.RLock()
	defer wb.StyMu.RUnlock()

	rs := &wb.Viewport.Render
	pc := &rs.Paint

	pos := wb.LayState.Alloc.Pos.AddScalar(st.Layout.Margin.Dots)
	sz := wb.LayState.Alloc.Size.AddScalar(-2.0 * st.Layout.Margin.Dots)
	rad := st.Border.Radius.Dots

	// first do any shadow
	if st.BoxShadow.HasShadow() {
		spos := pos.Add(mat32.Vec2{st.BoxShadow.HOffset.Dots, st.BoxShadow.VOffset.Dots})
		pc.StrokeStyle.SetColor(nil)
		pc.FillStyle.Color.SetShadowGradient(st.BoxShadow.Color, "")
		// todo: this is not rendering a transparent gradient
		// pc.FillStyle.Opacity = .5
		wb.RenderBoxImpl(spos, sz, rad)
		// pc.FillStyle.Opacity = 1.0
	}
	// then draw the box over top of that -- note: won't work well for
	// transparent! need to set clipping to box first..
	if !st.Font.BgColor.IsNil() {
		if rad == 0 {
			pc.FillBox(rs, pos, sz, &st.Font.BgColor)
		} else {
			pc.FillStyle.SetColorSpec(&st.Font.BgColor)
			pc.DrawRoundedRectangle(rs, pos.X, pos.Y, sz.X, sz.Y, rad)
			pc.Fill(rs)
		}
	}

	pc.StrokeStyle.SetColor(&st.Border.Color)
	pc.StrokeStyle.Width = st.Border.Width
	// pc.FillStyle.SetColor(&st.Font.BgColor)
	pos = pos.AddScalar(0.5 * st.Border.Width.Dots)
	sz = sz.SubScalar(st.Border.Width.Dots)
	pc.FillStyle.SetColor(nil)
	wb.RenderBoxImpl(pos, sz, st.Border.Radius.Dots)
}

// set our LayState.Alloc.Size from constraints
func (wb *WidgetBase) Size2DFromWH(w, h float32) {
	st := &wb.Sty
	if st.Layout.Width.Dots > 0 {
		w = mat32.Max(st.Layout.Width.Dots, w)
	}
	if st.Layout.Height.Dots > 0 {
		h = mat32.Max(st.Layout.Height.Dots, h)
	}
	spc := st.BoxSpace()
	w += 2.0 * spc
	h += 2.0 * spc
	wb.LayState.Alloc.Size = mat32.Vec2{w, h}
}

// Size2DAddSpace adds space to existing AllocSize
func (wb *WidgetBase) Size2DAddSpace() {
	spc := wb.BoxSpace()
	wb.LayState.Alloc.Size.SetAddScalar(2 * spc)
}

// Size2DSubSpace returns AllocSize minus 2 * BoxSpace -- the amount avail to the internal elements
func (wb *WidgetBase) Size2DSubSpace() mat32.Vec2 {
	spc := wb.BoxSpace()
	return wb.LayState.Alloc.Size.SubScalar(2 * spc)
}

///////////////////////////////////////////////////////////////////
// PartsWidgetBase

// PartsWidgetBase is the base type for all Widget Node2D elements that manage
// a set of constituent parts
type PartsWidgetBase struct {
	WidgetBase
	Parts Layout `json:"-" xml:"-" view-closed:"true" desc:"a separate tree of sub-widgets that implement discrete parts of a widget -- positions are always relative to the parent widget -- fully managed by the widget and not saved"`
}

var KiT_PartsWidgetBase = kit.Types.AddType(&PartsWidgetBase{}, PartsWidgetBaseProps)

var PartsWidgetBaseProps = ki.Props{
	"base-type":     true,
	"EnumType:Flag": KiT_NodeFlags,
}

func (wb *PartsWidgetBase) CopyFieldsFrom(frm interface{}) {
	fr, ok := frm.(*PartsWidgetBase)
	if !ok {
		log.Printf("GoGi node of type: %v needs a CopyFieldsFrom method defined -- currently falling back on earlier PartsWidgetBase one\n", wb.This().Name())
		ki.GenCopyFieldsFrom(wb.This(), frm)
		return
	}
	wb.WidgetBase.CopyFieldsFrom(&fr.WidgetBase)
	// wb.Parts.CopyFrom(&fr.Parts) // managed by widget -- we don't copy..
}

// standard FunDownMeFirst etc operate automatically on Field structs such as
// Parts -- custom calls only needed for manually-recursive traversal in
// Layout and Render

// SizeFromParts sets our size from those of our parts -- default..
func (wb *PartsWidgetBase) SizeFromParts(iter int) {
	wb.LayState.Alloc.Size = wb.Parts.LayState.Size.Pref // get from parts
	wb.Size2DAddSpace()
	if Layout2DTrace {
		fmt.Printf("Size:   %v size from parts: %v, parts pref: %v\n", wb.Path(), wb.LayState.Alloc.Size, wb.Parts.LayState.Size.Pref)
	}
}

func (wb *PartsWidgetBase) Size2DParts(iter int) {
	wb.InitLayout2D()
	wb.SizeFromParts(iter) // get our size from parts
}

func (wb *PartsWidgetBase) Size2D(iter int) {
	wb.Size2DParts(iter)
}

func (wb *PartsWidgetBase) ComputeBBox2DParts(parBBox image.Rectangle, delta image.Point) {
	wb.ComputeBBox2DBase(parBBox, delta)
	wb.Parts.This().(Node2D).ComputeBBox2D(parBBox, delta)
}

func (wb *PartsWidgetBase) ComputeBBox2D(parBBox image.Rectangle, delta image.Point) {
	wb.ComputeBBox2DParts(parBBox, delta)
}

func (wb *PartsWidgetBase) Layout2DParts(parBBox image.Rectangle, iter int) {
	spc := wb.BoxSpace()
	wb.Parts.LayState.Alloc.Pos = wb.LayState.Alloc.Pos.AddScalar(spc)
	wb.Parts.LayState.Alloc.Size = wb.LayState.Alloc.Size.AddScalar(-2.0 * spc)
	wb.Parts.Layout2D(parBBox, iter)
}

func (wb *PartsWidgetBase) Layout2D(parBBox image.Rectangle, iter int) bool {
	wb.Layout2DBase(parBBox, true, iter) // init style
	wb.Layout2DParts(parBBox, iter)
	return wb.Layout2DChildren(iter)
}

func (wb *PartsWidgetBase) Render2DParts() {
	wb.Parts.Render2DTree()
}

func (wb *PartsWidgetBase) Move2D(delta image.Point, parBBox image.Rectangle) {
	wb.Move2DBase(delta, parBBox)
	wb.Parts.This().(Node2D).Move2D(delta, parBBox)
	wb.Move2DChildren(delta)
}

///////////////////////////////////////////////////////////////////
// ConfigParts building-blocks

// ConfigPartsIconLabel adds to config to create parts, of icon
// and label left-to right in a row, based on whether items are nil or empty
func (wb *PartsWidgetBase) ConfigPartsIconLabel(config *kit.TypeAndNameList, icnm string, txt string) (icIdx, lbIdx int) {
	wb.Parts.SetProp("overflow", gist.OverflowHidden) // no scrollbars!
	if wb.Sty.Template != "" {
		wb.Parts.Sty.Template = wb.Sty.Template + ".Parts"
	}
	icIdx = -1
	lbIdx = -1
	if IconName(icnm).IsValid() {
		icIdx = len(*config)
		config.Add(KiT_Icon, "icon")
		if txt != "" {
			config.Add(KiT_Space, "space")
		}
	}
	if txt != "" {
		lbIdx = len(*config)
		config.Add(KiT_Label, "label")
	}
	return
}

// ConfigPartsSetIconLabel sets the icon and text values in parts, and get
// part style props, using given props if not set in object props
func (wb *PartsWidgetBase) ConfigPartsSetIconLabel(icnm string, txt string, icIdx, lbIdx int) {
	if icIdx >= 0 {
		ic := wb.Parts.Child(icIdx).(*Icon)
		if wb.Sty.Template != "" {
			ic.Sty.Template = wb.Sty.Template + ".icon"
		}
		if set, _ := ic.SetIcon(icnm); set || wb.NeedsFullReRender() {
			wb.StylePart(Node2D(ic))
		}
	}
	if lbIdx >= 0 {
		lbl := wb.Parts.Child(lbIdx).(*Label)
		if wb.Sty.Template != "" {
			lbl.Sty.Template = wb.Sty.Template + ".icon"
		}
		if lbl.Text != txt {
			wb.StylePart(Node2D(lbl))
			if icIdx >= 0 {
				wb.StylePart(wb.Parts.Child(lbIdx - 1).(Node2D)) // also get the space
			}
			lbl.SetText(txt)
		}
	}
}

// PartsNeedUpdateIconLabel check if parts need to be updated -- for ConfigPartsIfNeeded
func (wb *PartsWidgetBase) PartsNeedUpdateIconLabel(icnm string, txt string) bool {
	if IconName(icnm).IsValid() {
		ick := wb.Parts.ChildByName("icon", 0)
		if ick == nil {
			return true
		}
		ic := ick.(*Icon)
		if !ic.HasChildren() || ic.IconNm != icnm || wb.NeedsFullReRender() {
			return true
		}
	} else {
		cn := wb.Parts.ChildByName("icon", 0)
		if cn != nil { // need to remove it
			return true
		}
	}
	if txt != "" {
		lblk := wb.Parts.ChildByName("label", 2)
		if lblk == nil {
			return true
		}
		lbl := lblk.(*Label)
		lbl.Sty.Font.Color = wb.Sty.Font.Color
		if lbl.Text != txt {
			return true
		}
	} else {
		cn := wb.Parts.ChildByName("label", 2)
		if cn != nil {
			return true
		}
	}
	return false
}

// SetFullReRenderIconLabel sets the icon and label to be re-rendered, needed
// when styles change
func (wb *PartsWidgetBase) SetFullReRenderIconLabel() {
	if ick := wb.Parts.ChildByName("icon", 0); ick != nil {
		ic := ick.(*Icon)
		ic.SetFullReRender()
	}
	if lblk := wb.Parts.ChildByName("label", 2); lblk != nil {
		lbl := lblk.(*Label)
		lbl.SetFullReRender()
	}
	wb.Parts.StyMu.Lock()
	wb.Parts.Style2DWidget() // restyle parent so parts inherit
	wb.Parts.StyMu.Unlock()
}
