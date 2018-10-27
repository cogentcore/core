// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"
	"log"
	"strings"

	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/mouse"
	"github.com/goki/gi/units"
	"github.com/goki/ki"
	"github.com/goki/ki/bitflag"
	"github.com/goki/ki/ints"
	"github.com/goki/ki/kit"
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
	Sty          Style        `json:"-" xml:"-" desc:"styling settings for this widget -- set in SetStyle2D during an initialization step, and when the structure changes"`
	DefStyle     *Style       `view:"-" json:"-" xml:"-" desc:"default style values computed by a parent widget for us -- if set, we are a part of a parent widget and should use these as our starting styles instead of type-based defaults"`
	LayData      LayoutData   `json:"-" xml:"-" desc:"all the layout information for this item"`
	WidgetSig    ki.Signal    `json:"-" xml:"-" view:"-" desc:"general widget signals supported by all widgets, including select, focus, and context menu (right mouse button) events, which can be used by views and other compound widgets"`
	CtxtMenuFunc CtxtMenuFunc `view:"-" json:"-" xml:"-" desc:"optional context menu function called by MakeContextMenu AFTER any native items are added -- this function can decide where to insert new elements -- typically add a separator to disambiguate"`
}

var KiT_WidgetBase = kit.Types.AddType(&WidgetBase{}, WidgetBaseProps)

var WidgetBaseProps = ki.Props{
	"base-type": true,
}

func (wb *WidgetBase) AsWidget() *WidgetBase {
	return wb
}

// Style satisfies the Styler interface
func (wb *WidgetBase) Style() *Style {
	return &wb.Sty
}

// Init2DWidget handles basic node initialization -- Init2D can then do special things
func (wb *WidgetBase) Init2DWidget() {
	wb.Viewport = wb.ParentViewport()
	wb.Sty.Defaults()
	wb.LayData.Defaults() // doesn't overwrite
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
func (wb *WidgetBase) DefaultStyle2DWidget(selector string, part *WidgetBase) *Style {
	tprops := *kit.Types.Properties(wb.Type(), true) // true = makeNew
	styprops := tprops
	if selector != "" {
		sp, ok := kit.TypeProp(tprops, selector)
		if !ok {
			// log.Printf("gi.DefaultStyle2DWidget: did not find props for style selector: %v for node type: %v\n", selector, wb.Type().Name())
		} else {
			spm, ok := sp.(ki.Props)
			if !ok {
				log.Printf("gi.DefaultStyle2DWidget: looking for a ki.Props for style selector: %v, instead got type: %T, for node type: %v\n", selector, spm, wb.Type().Name())
			} else {
				styprops = spm
			}
		}
	}

	parSty := wb.ParentStyle()

	var dsty *Style
	stKey := WidgetDefStyleKey + selector
	prKey := WidgetDefPropsKey + selector
	dstyi, ok := kit.TypeProp(tprops, stKey)
	if !ok || RebuildDefaultStyles {
		dsty = &Style{}
		dsty.Defaults()
		if selector != "" {
			var baseStyle *Style
			if part != nil {
				baseStyle = part.DefaultStyle2DWidget("", nil)
			} else {
				baseStyle = wb.DefaultStyle2DWidget("", nil)
			}
			*dsty = *baseStyle
		}
		kit.TypesMu.Lock() // write lock
		dsty.SetStyleProps(parSty, styprops)
		dsty.IsSet = false // keep as non-set
		tprops[stKey] = dsty
		tprops[prKey] = styprops
		kit.TypesMu.Unlock()
	} else {
		dsty, _ = dstyi.(*Style)
	}
	return dsty
}

// Style2DWidget styles the Style values from node properties and optional
// base-level defaults -- for Widget-style nodes
func (wb *WidgetBase) Style2DWidget() {
	gii, _ := wb.This.(Node2D)
	SetCurStyleNode2D(gii)
	defer SetCurStyleNode2D(nil)
	if !RebuildDefaultStyles && wb.DefStyle != nil {
		wb.Sty.CopyFrom(wb.DefStyle)
	} else {
		wb.Sty.CopyFrom(wb.DefaultStyle2DWidget("", nil))
	}
	wb.Sty.IsSet = false    // this is always first call, restart
	if wb.Viewport == nil { // robust
		gii.Init2D()
	}
	styprops := *wb.Properties()
	parSty := wb.ParentStyle()
	wb.Sty.SetStyleProps(parSty, styprops)

	// look for class-specific style sheets among defaults -- have to do these
	// dynamically now -- cannot compile into default which is type-general
	tprops := *kit.Types.Properties(wb.Type(), true) // true = makeNew
	kit.TypesMu.RLock()
	clsty := "." + wb.Class
	if sp, ok := ki.SubProps(tprops, clsty); ok {
		wb.Sty.SetStyleProps(parSty, sp)
	}
	kit.TypesMu.RUnlock()

	pagg := wb.ParentCSSAgg()
	if pagg != nil {
		AggCSS(&wb.CSSAgg, *pagg)
	} else {
		wb.CSSAgg = nil // restart
	}
	AggCSS(&wb.CSSAgg, wb.CSS)
	wb.Sty.StyleCSS(gii, wb.CSSAgg, "")

	wb.Sty.SetUnitContext(wb.Viewport, Vec2DZero) // todo: test for use of el-relative
	if wb.Sty.Inactive {                          // inactive can only set, not clear
		wb.SetInactive()
	}
	wb.Sty.Use() // activates currentColor etc
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
	// if pg.DefStyle != nil && !RebuildDefaultStyles { // already set
	// 	return
	// }
	stynm := "#" + strings.ToLower(pk.Name())
	// this is called on US (the parent object) so we store the #partname
	// default style within our type properties..  that's good -- HOWEVER we
	// cannot put any sub-selector properties within these part styles -- must
	// all be in the base-level.. hopefully that works..
	pdst := wb.DefaultStyle2DWidget(stynm, pg)
	pg.DefStyle = pdst // will use this as starting point for all styles now..

	if ics := pk.Embed(KiT_Icon); ics != nil {
		ic := ics.(*Icon)
		styprops := kit.Types.Properties(wb.Type(), true)
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

func (wb *WidgetBase) Style2D() {
	wb.Style2DWidget()
	wb.LayData.SetFromStyle(&wb.Sty.Layout) // also does reset
}

func (wb *WidgetBase) InitLayout2D() bool {
	wb.LayData.SetFromStyle(&wb.Sty.Layout)
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
func (wb *WidgetBase) AddParentPos() Vec2D {
	if pni, _ := KiToNode2D(wb.Par); pni != nil {
		if pw := pni.AsWidget(); pw != nil {
			if !wb.IsField() {
				wb.LayData.AllocPos = pw.LayData.AllocPosOrig.Add(wb.LayData.AllocPosRel)
			}
			return pw.LayData.AllocSize
		}
	}
	return Vec2DZero
}

// BBoxFromAlloc gets our bbox from Layout allocation.
func (wb *WidgetBase) BBoxFromAlloc() image.Rectangle {
	return RectFromPosSizeMax(wb.LayData.AllocPos, wb.LayData.AllocSize)
}

func (wb *WidgetBase) BBox2D() image.Rectangle {
	return wb.BBoxFromAlloc()
}

func (wb *WidgetBase) ComputeBBox2D(parBBox image.Rectangle, delta image.Point) {
	wb.ComputeBBox2DBase(parBBox, delta)
}

// Layout2DBase provides basic Layout2D functions -- good for most cases
func (wb *WidgetBase) Layout2DBase(parBBox image.Rectangle, initStyle bool, iter int) {
	nii, _ := wb.This.(Node2D)
	if wb.Viewport == nil { // robust
		if nii.AsViewport2D() == nil {
			nii.Init2D()
			nii.Style2D()
			// fmt.Printf("node not init in Layout2DBase: %v\n", wb.PathUnique())
		}
	}
	psize := wb.AddParentPos()
	wb.LayData.AllocPosOrig = wb.LayData.AllocPos
	if initStyle {
		wb.Sty.SetUnitContext(wb.Viewport, psize) // update units with final layout
	}
	wb.BBox = nii.BBox2D() // only compute once, at this point
	// note: if other styles are maintained, they also need to be updated!
	nii.ComputeBBox2D(parBBox, image.ZP) // other bboxes from BBox
	if Layout2DTrace {
		fmt.Printf("Layout: %v alloc pos: %v size: %v vpbb: %v winbb: %v\n", wb.PathUnique(), wb.LayData.AllocPos, wb.LayData.AllocSize, wb.VpBBox, wb.WinBBox)
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
	spc := int(wb.Sty.BoxSpace())
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
	if wb.InBounds() && wb.NeedsFullReRender() {
		if Render2DTrace {
			fmt.Printf("Render: NeedsFullReRender for %v at %v\n", wb.PathUnique(), wb.VpBBox)
		}
		wb.ClearFullReRender()
		wb.ReRender2DTree()
		return true
	}
	return false
}

// InBounds returns true if our VpBBox is non-empty (and other stuff is non-nil)
func (wb *WidgetBase) InBounds() bool {
	if wb.This == nil || wb.Viewport == nil {
		return false
	}
	if wb.IsInvisible() {
		return false
	}
	return !wb.VpBBox.Empty()
}

// PushBounds pushes our bounding-box bounds onto the bounds stack if non-empty
// -- this limits our drawing to our own bounding box, automatically -- must
// be called as first step in Render2D returns whether the new bounds are
// empty or not -- if empty then don't render!
func (wb *WidgetBase) PushBounds() bool {
	if wb.This == nil || wb.Viewport == nil {
		return false
	}
	if wb.IsInvisible() {
		return false
	}
	if wb.IsOverlay() {
		wb.ClearFullReRender()
		if wb.Viewport != nil {
			wb.ConnectToViewport()
			wb.Viewport.Render.PushBounds(wb.Viewport.Pixels.Bounds())
		}
		return true
	}
	if wb.VpBBox.Empty() {
		wb.ClearFullReRender()
		return false
	}
	rs := &wb.Viewport.Render
	rs.PushBounds(wb.VpBBox)
	wb.ConnectToViewport()
	if Render2DTrace {
		fmt.Printf("Render: %v at %v\n", wb.PathUnique(), wb.VpBBox)
	}
	return true
}

// PopBounds pops our bounding-box bounds -- last step in Render2D after
// rendering children
func (wb *WidgetBase) PopBounds() {
	wb.ClearFullReRender()
	if wb.This == nil || wb.Viewport == nil {
		return
	}
	rs := &wb.Viewport.Render
	rs.PopBounds()
}

func (wb *WidgetBase) Render2D() {
	if wb.FullReRenderIfNeeded() {
		return
	}
	if wb.PushBounds() {
		wb.This.(Node2D).ConnectEvents2D()
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
	delta := wb.LayData.AllocPos.Sub(wb.LayData.AllocPosOrig)
	wb.LayData.AllocPos = wb.LayData.AllocPosOrig
	ld := wb.LayData // save our current layout data
	updt := wb.UpdateStart()
	wb.Init2DTree()
	wb.Style2DTree()
	wb.Size2DTree(0)
	wb.LayData = ld // restore
	wb.Layout2DTree()
	if !delta.IsZero() {
		wb.Move2D(delta.ToPointFloor(), parBBox)
	}
	wb.Render2DTree()
	wb.UpdateEndNoSig(updt)
}

// Move2DBase does the basic move on this node
func (wb *WidgetBase) Move2DBase(delta image.Point, parBBox image.Rectangle) {
	wb.LayData.AllocPos = wb.LayData.AllocPosOrig.Add(NewVec2DFmPoint(delta))
	wb.This.(Node2D).ComputeBBox2D(parBBox, delta)
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
	delta := wb.LayData.AllocPos.Sub(wb.LayData.AllocPosOrig).ToPoint()
	wb.This.(Node2D).Move2D(delta, parBBox) // important to use interface version to get interface!
}

// ParentLayout returns the parent layout
func (wb *WidgetBase) ParentLayout() *Layout {
	var parLy *Layout
	wb.FuncUpParent(0, wb.This, func(k ki.Ki, level int, d interface{}) bool {
		nii, ok := k.(Node2D)
		if !ok {
			return false // don't keep going up
		}
		ly := nii.AsLayout2D()
		if ly != nil {
			parLy = ly
			return false // done
		}
		return true
	})
	return parLy
}

// CtxtMenuFunc is a function for creating a context menu for given node
type CtxtMenuFunc func(g Node2D, m *Menu)

func (wb *WidgetBase) MakeContextMenu(m *Menu) {
	// derived types put native menu code here
	if wb.CtxtMenuFunc != nil {
		wb.CtxtMenuFunc(wb.This.(Node2D), m)
	}
	TheViewIFace.CtxtMenuView(wb.This, wb.IsInactive(), wb.Viewport, m)
}

var TooltipFrameProps = ki.Props{
	"background-color":    &Prefs.Colors.Highlight,
	"border-width":        units.NewValue(0, units.Px),
	"border-color":        "none",
	"margin":              units.NewValue(0, units.Px),
	"padding":             units.NewValue(2, units.Px),
	"box-shadow.h-offset": units.NewValue(0, units.Px),
	"box-shadow.v-offset": units.NewValue(0, units.Px),
	"box-shadow.blur":     units.NewValue(0, units.Px),
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
	bitflag.Set(&pvp.Flag, int(VpFlagPopup))
	bitflag.Set(&pvp.Flag, int(VpFlagTooltip))

	pvp.Geom.Pos = image.Point{x, y}
	bitflag.Set(&pvp.Flag, int(VpFlagPopupDestroyAll)) // nuke it all
	frame := pvp.AddNewChild(KiT_Frame, "Frame").(*Frame)
	frame.Lay = LayoutVert
	frame.SetProps(TooltipFrameProps, false)
	lbl := frame.AddNewChild(KiT_Label, "ttlbl").(*Label)
	lbl.SetProp("white-space", WhiteSpaceNormal) // wrap

	mwdots := parVp.Sty.UnContext.ToDots(40, units.Em)
	mwdots = Min32(mwdots, float32(mainVp.Geom.Size.X-20))

	lbl.SetProp("max-width", units.NewValue(mwdots, units.Dot))
	lbl.Text = tooltip
	frame.Init2DTree()
	frame.Style2DTree()                                // sufficient to get sizes
	frame.LayData.AllocSize = mainVp.LayData.AllocSize // give it the whole vp initially
	frame.Size2DTree(0)                                // collect sizes
	pvp.Win = nil
	vpsz := frame.LayData.Size.Pref.Min(mainVp.LayData.AllocSize).ToPoint()
	x = ints.MinInt(x, mainVp.Geom.Size.X-vpsz.X) // fit
	y = ints.MinInt(y, mainVp.Geom.Size.Y-vpsz.Y) // fit
	pvp.Resize(vpsz)
	pvp.Geom.Pos = image.Point{x, y}
	pvp.UpdateEndNoSig(updt)

	win.PushPopup(pvp.This)
	return &pvp
}

// WidgetSignals are general signals that all widgets can send, via WidgetSig
// signal
type WidgetSignals int64

const (
	// WidgetSelected is triggered when a widget is selected, typically via
	// left mouse button click (see EmitSelectedSignal) -- is NOT contingent
	// on actual IsSelected status -- just reports the click event
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
	wb.WidgetSig.Emit(wb.This, int64(WidgetSelected), nil)
}

// EmitFocusedSignal emits the WidgetFocused signal for this widget
func (wb *WidgetBase) EmitFocusedSignal() {
	wb.WidgetSig.Emit(wb.This, int64(WidgetFocused), nil)
}

// EmitContextMenuSignal emits the WidgetContextMenu signal for this widget
func (wb *WidgetBase) EmitContextMenuSignal() {
	wb.WidgetSig.Emit(wb.This, int64(WidgetContextMenu), nil)
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
			PopupTooltip(wbb.Tooltip, pos.X, pos.Y, wbb.Viewport, wbb.Nm)
		}
	})
}

// WidgetMouseEvents connects to eiher or both mouse events -- IMPORTANT: if
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
				wbb.This.(Node2D).ContextMenu()
			}
		}
	})
}

////////////////////////////////////////////////////////////////////////////////
//  Standard rendering

// RenderBoxImpl implements the standard box model rendering -- assumes all
// paint params have already been set
func (wb *WidgetBase) RenderBoxImpl(pos Vec2D, sz Vec2D, rad float32) {
	rs := &wb.Viewport.Render
	pc := &rs.Paint
	if rad == 0.0 {
		pc.DrawRectangle(rs, pos.X, pos.Y, sz.X, sz.Y)
	} else {
		pc.DrawRoundedRectangle(rs, pos.X, pos.Y, sz.X, sz.Y, rad)
	}
	pc.FillStrokeClear(rs)
}

// RenderStdBox draws standard box using given style
func (wb *WidgetBase) RenderStdBox(st *Style) {
	rs := &wb.Viewport.Render
	pc := &rs.Paint

	pos := wb.LayData.AllocPos.AddVal(st.Layout.Margin.Dots)
	sz := wb.LayData.AllocSize.AddVal(-2.0 * st.Layout.Margin.Dots)
	rad := st.Border.Radius.Dots

	// first do any shadow
	if st.BoxShadow.HasShadow() {
		spos := pos.Add(Vec2D{st.BoxShadow.HOffset.Dots, st.BoxShadow.VOffset.Dots})
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
	pos = pos.AddVal(0.5 * st.Border.Width.Dots)
	sz = sz.SubVal(st.Border.Width.Dots)
	pc.FillStyle.SetColor(nil)
	wb.RenderBoxImpl(pos, sz, st.Border.Radius.Dots)
}

// set our LayData.AllocSize from constraints
func (wb *WidgetBase) Size2DFromWH(w, h float32) {
	st := &wb.Sty
	if st.Layout.Width.Dots > 0 {
		w = Max32(st.Layout.Width.Dots, w)
	}
	if st.Layout.Height.Dots > 0 {
		h = Max32(st.Layout.Height.Dots, h)
	}
	spc := st.BoxSpace()
	w += 2.0 * spc
	h += 2.0 * spc
	wb.LayData.AllocSize = Vec2D{w, h}
}

// Size2DAddSpace adds space to existing AllocSize
func (wb *WidgetBase) Size2DAddSpace() {
	spc := wb.Sty.BoxSpace()
	wb.LayData.AllocSize.SetAddVal(2 * spc)
}

// Size2DSubSpace returns AllocSize minus 2 * BoxSpace -- the amount avail to the internal elements
func (wb *WidgetBase) Size2DSubSpace() Vec2D {
	spc := wb.Sty.BoxSpace()
	return wb.LayData.AllocSize.SubVal(2 * spc)
}

// SetMinPrefWidth sets minimum and preferred width -- will get at least this
// amount -- max unspecified
func (wb *WidgetBase) SetMinPrefWidth(val units.Value) {
	wb.SetProp("width", val)
	wb.SetProp("min-width", val)
}

// SetMinPrefHeight sets minimum and preferred height -- will get at least this
// amount -- max unspecified
func (wb *WidgetBase) SetMinPrefHeight(val units.Value) {
	wb.SetProp("height", val)
	wb.SetProp("min-height", val)
}

// SetStretchMaxWidth sets stretchy max width (-1) -- can grow to take up avail room
func (wb *WidgetBase) SetStretchMaxWidth() {
	wb.SetProp("max-width", units.NewValue(-1, units.Px))
}

// SetStretchMaxHeight sets stretchy max height (-1) -- can grow to take up avail room
func (wb *WidgetBase) SetStretchMaxHeight() {
	wb.SetProp("max-height", units.NewValue(-1, units.Px))
}

// SetFixedWidth sets all width options (width, min-width, max-width) to a fixed width value
func (wb *WidgetBase) SetFixedWidth(val units.Value) {
	wb.SetProp("width", val)
	wb.SetProp("min-width", val)
	wb.SetProp("max-width", val)
}

// SetFixedHeight sets all height options (height, min-height, max-height) to
// a fixed height value
func (wb *WidgetBase) SetFixedHeight(val units.Value) {
	wb.SetProp("height", val)
	wb.SetProp("min-height", val)
	wb.SetProp("max-height", val)
}

///////////////////////////////////////////////////////////////////
// PartsWidgetBase

// PartsWidgetBase is the base type for all Widget Node2D elements that manage
// a set of constitutent parts
type PartsWidgetBase struct {
	WidgetBase
	Parts Layout `json:"-" xml:"-" view-closed:"true" desc:"a separate tree of sub-widgets that implement discrete parts of a widget -- positions are always relative to the parent widget -- fully managed by the widget and not saved"`
}

var KiT_PartsWidgetBase = kit.Types.AddType(&PartsWidgetBase{}, PartsWidgetBaseProps)

var PartsWidgetBaseProps = ki.Props{
	"base-type": true,
}

// standard FunDownMeFirst etc operate automaticaly on Field structs such as
// Parts -- custom calls only needed for manually-recursive traversal in
// Layout and Render

// SizeFromParts sets our size from those of our parts -- default..
func (wb *PartsWidgetBase) SizeFromParts(iter int) {
	wb.LayData.AllocSize = wb.Parts.LayData.Size.Pref // get from parts
	wb.Size2DAddSpace()
	if Layout2DTrace {
		fmt.Printf("Size:   %v size from parts: %v, parts pref: %v\n", wb.PathUnique(), wb.LayData.AllocSize, wb.Parts.LayData.Size.Pref)
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
	wb.Parts.This.(Node2D).ComputeBBox2D(parBBox, delta)
}

func (wb *PartsWidgetBase) ComputeBBox2D(parBBox image.Rectangle, delta image.Point) {
	wb.ComputeBBox2DParts(parBBox, delta)
}

func (wb *PartsWidgetBase) Layout2DParts(parBBox image.Rectangle, iter int) {
	spc := wb.Sty.BoxSpace()
	wb.Parts.LayData.AllocPos = wb.LayData.AllocPos.AddVal(spc)
	wb.Parts.LayData.AllocSize = wb.LayData.AllocSize.AddVal(-2.0 * spc)
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
	wb.Parts.This.(Node2D).Move2D(delta, parBBox)
	wb.Move2DChildren(delta)
}

///////////////////////////////////////////////////////////////////
// ConfigParts building-blocks

// ConfigPartsIconLabel adds to config to create parts, of icon
// and label left-to right in a row, based on whether items are nil or empty
func (wb *PartsWidgetBase) ConfigPartsIconLabel(config *kit.TypeAndNameList, icnm string, txt string) (icIdx, lbIdx int) {
	wb.Parts.SetProp("overflow", "hidden") // no scrollbars!
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
		ic := wb.Parts.KnownChild(icIdx).(*Icon)
		if set, _ := ic.SetIcon(icnm); set || wb.NeedsFullReRender() {
			wb.StylePart(Node2D(ic))
		}
	}
	if lbIdx >= 0 {
		lbl := wb.Parts.KnownChild(lbIdx).(*Label)
		if lbl.Text != txt {
			wb.StylePart(Node2D(lbl))
			if icIdx >= 0 {
				wb.StylePart(wb.Parts.KnownChild(lbIdx - 1).(Node2D)) // also get the space
			}
			lbl.SetText(txt)
		}
	}
}

// PartsNeedUpdateIconLabel check if parts need to be updated -- for ConfigPartsIfNeeded
func (wb *PartsWidgetBase) PartsNeedUpdateIconLabel(icnm string, txt string) bool {
	if IconName(icnm).IsValid() {
		ick, ok := wb.Parts.ChildByName("icon", 0)
		if !ok {
			return true
		}
		ic := ick.(*Icon)
		if !ic.HasChildren() || ic.UniqueNm != icnm || wb.NeedsFullReRender() {
			return true
		}
	} else {
		_, ok := wb.Parts.ChildByName("icon", 0)
		if ok { // need to remove it
			return true
		}
	}
	if txt != "" {
		lblk, ok := wb.Parts.ChildByName("label", 2)
		if !ok {
			return true
		}
		lbl := lblk.(*Label)
		lbl.Sty.Font.Color = wb.Sty.Font.Color
		if lbl.Text != txt {
			return true
		}
	} else {
		_, ok := wb.Parts.ChildByName("label", 2)
		if ok {
			return true
		}
	}
	return false
}

// SetFullReRenderIconLabel sets the icon and label to be re-rendered, needed
// when styles change
func (wb *PartsWidgetBase) SetFullReRenderIconLabel() {
	if ick, ok := wb.Parts.ChildByName("icon", 0); ok {
		ic := ick.(*Icon)
		ic.SetFullReRender()
	}
	if lblk, ok := wb.Parts.ChildByName("label", 2); ok {
		lbl := lblk.(*Label)
		lbl.SetFullReRender()
	}
	wb.Parts.Style2DWidget() // restyle parent so parts inherit
}
