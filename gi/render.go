// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"
	"log"
	"log/slog"
	"os"
	"runtime"
	"runtime/pprof"
	"time"

	"cogentcore.org/core/cam/hct"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/ki"
	"cogentcore.org/core/mat32"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/prof"
	"cogentcore.org/core/styles"
)

// Rendering logic:
//
// Key principles:
//
// * Async updates (animation, mouse events, etc) change state, _set only flags_
//   using thread-safe atomic bitflag operations.  Actually rendering async (in V1)
//   is really hard to get right, and requires tons of mutexes etc.
// * Synchronous, full-tree render updates do the layout, rendering,
//   at regular FPS (frames-per-second) rate -- nop unless flag set.
// * Ki UpdateStart / End ensures that _only the highest changed node is flagged_,
//   while each individual state update uses the same Update wrapper calls locally,
//   so that rendering updates automatically happen at this highest common node.
// * UpdateStart starts naturally on the highest node driving a change, causing
//   a cascade of other UpdateStart on lower nodes, but the IsUpdating flag signals
//   that they are not the highest.  Only the highest calls UpdateEnd with true,
//   which is the point at which the change is flagged for render updating.
// * Thus, rendering updates skip any nodes with IsUpdating set, and are only
//   triggered at the highest UpdateEnd, so there shouldn't be conflicts
//   unless a node starts updating again before the next render hits.
//
// Three main steps:
// * Config: (re)configures widgets based on current params
//   typically by making Parts.  Always calls ApplyStyle.
// * Layout: does sizing and positioning on tree, arranging widgets.
//   Needed for whole tree after any Config changes anywhere
//   See layimpl.go for full details and code.
// * Render: just draws with current config, layout.
//
// ApplyStyle is always called after Config, and after any
// current state of the Widget changes via events, animations, etc
// (e.g., a Hover started or a Button is pushed down).
// These changes should be protected by UpdateStart / End,
// such that ApplyStyle is only ever called within that scope.
// Use UpdateEndRender(updt) to call SetNeedsRender on updt
// which sets the node NeedsRender and ScNeedsRender flags,
// to drive the rendering update at next DoNeedsRender call.
//
// The initial configuration of a scene can skip calling
// Config and ApplyStyle because these will be called automatically
// during the Run() process for the Scene.
//
// For dynamic reconfiguration after initial display,
// ReConfg() is the key method, calling Config then
// ApplyStyle on the node and all of its children.
//
// UpdateStart also sets the Scene-level ScUpdating flag, and
// UpdateEnd clears it, to prevent any Scene-level layout etc
// from happening while a widget is updating.
//
// For nodes with dynamic content that doesn't require styling or config,
// a simple SetNeedsRender call will drive re-rendering.
//
// Updating is _always_ driven top-down by RenderWin at FPS sampling rate,
// in the DoUpdate() call on the Scene.
// Three types of updates can be triggered, in order of least impact
// and highest frequency first:
// * ScNeedsRender: does NeedsRender on nodes.
// * ScNeedsLayout: does GetSize, DoLayout, then Render -- after Config.
//
// Event handling, styling, etc updates should:
// * Wrap with UpdateStart / End
// * End with: SetNeedsStyle(vp, updt) if needs style updates needed based
//   on state change, or SetNeedsRender(vp, updt)
// * Or, if Config-level changes are needed, the Config(vp) must call
//   SetNeedsLayout(vp, updt) to trigger vp Layout step after.
//
// The one mutex that is still needed is a RWMutex on the BBbox fields
// because they are read by the event manager (and potentially inside
// event handler code) which does not have any flag protection,
// and are also read in rendering and written in Layout.

/*
// UpdateStart sets the scene ScUpdating flag to prevent
// render updates during construction on a scene.
func (wb *WidgetBase) UpdateStart() bool {
	updt := wb.Node.UpdateStart()
	if updt && !wb.Is(ki.Field) && wb.Sc != nil {
		wb.Sc.SetFlag(true, ScUpdating)
		if DebugSettings.UpdateTrace {
			fmt.Println("DebugSettings.UpdateTrace Scene Start:", wb.Sc, "from widget:", wb)
		}
	}
	return updt
}

// UpdateEnd resets the scene ScUpdating flag
func (wb *WidgetBase) UpdateEnd(updt bool) {
	if updt && !wb.Is(ki.Field) && wb.Sc != nil {
		wb.Sc.SetFlag(false, ScUpdating)
		if DebugSettings.UpdateTrace {
			fmt.Println("DebugSettings.UpdateTrace Scene End:", wb.Sc, "from widget:", wb)
		}
	}
	wb.Node.UpdateEnd(updt)
}
*/

// UpdateStartAsync must be called for any asynchronous update
// that happens outside of the usual user event-driven, same-thread
// updates, or other updates that can happen during standard layout / rendering.
// It waits for any current Render update to finish, via RenderCtx().ReadLock().
// It must be paired with an UpdateEndAsync.
// These calls CANNOT be triggered during a standard render update,
// (whereas UpdateStart / End can be, and typically are)
// because it will cause a hang on the Read Lock which
// was already write locked at the start of the render.
func (wb *WidgetBase) UpdateStartAsync() bool {
	if wb.Scene == nil || wb.Scene.RenderCtx() == nil {
		return wb.Node.UpdateStart()
	}
	wb.Scene.RenderCtx().ReadLock()
	wb.Scene.SetFlag(true, ScUpdating)
	return wb.UpdateStart()
}

// UpdateEndAsync must be called after [UpdateStartAsync] for any
// asynchronous update that happens outside of the usual user event-driven,
// same-thread updates.
func (wb *WidgetBase) UpdateEndAsync(updt bool) {
	if wb.Scene == nil || wb.Scene.RenderCtx() == nil {
		wb.Node.UpdateEnd(updt)
		return
	}
	wb.Scene.SetFlag(false, ScUpdating)
	wb.Scene.RenderCtx().ReadUnlock()
	wb.UpdateEnd(updt)
}

// UpdateEndAsyncLayout should be called instead of [UpdateEndAsync]
// for any [UpdateStartAsync] / [UpdateEndAsync] block that needs
// a re-layout at the end.  Just does [SetNeedsLayoutUpdate] after UpdateEnd,
// and uses the cached wb.Sc pointer.
func (wb *WidgetBase) UpdateEndAsyncLayout(updt bool) {
	wb.UpdateEndAsync(updt)
	wb.SetNeedsLayout(updt)
}

// UpdateEndAsyncRender should be called instead of [UpdateEndAsync]
// for any [UpdateStartAsync] / [UpdateEndAsync] block that needs
// a re-render at the end.  Just does [SetNeedsRenderUpdate] after UpdateEnd,
// and uses the cached wb.Sc pointer.
func (wb *WidgetBase) UpdateEndAsyncRender(updt bool) {
	wb.UpdateEndAsync(updt)
	wb.SetNeedsRender(updt)
}

// SetNeedsRender sets the NeedsRender and Scene NeedsRender flags, if updt is true.
// See [UpdateEndRender] for convenience method.
// This should be called after widget state changes
// that don't need styling, e.g., in event handlers
// or other update code, _after_ calling UpdateEnd(updt) and passing
// that same updt flag from UpdateStart.
func (wb *WidgetBase) SetNeedsRender(updt bool) {
	if !updt || wb.Scene == nil {
		return
	}
	if DebugSettings.UpdateTrace {
		fmt.Println("\tDebugSettings.UpdateTrace: NeedsRender:", wb)
	}
	wb.SetFlag(true, NeedsRender)
	if wb.Scene != nil {
		wb.Scene.SetFlag(true, ScNeedsRender)
	}
	// parent of Parts needs to render if parent
	fi := wb.ParentWidgetIf(func(p *WidgetBase) bool {
		return p.Is(ki.Field)
	})
	if fi != nil && fi.Parent() != nil && fi.Parent().This() != nil {
		fi.Parent().(Widget).AsWidget().SetNeedsRender(true)
	}
}

// UpdateEndRender should be called instead of UpdateEnd
// for any [UpdateStart] / [UpdateEnd] block that needs a re-render
// at the end.  Just does [SetNeedsRenderUpdate] after UpdateEnd,
// and uses the cached wb.Sc pointer.
func (wb *WidgetBase) UpdateEndRender(updt bool) {
	if !updt {
		return
	}
	wb.UpdateEnd(updt)
	wb.SetNeedsRender(updt)
}

// AddReRender adds given widget to be re-rendered next pass
func (sc *Scene) AddReRender(w Widget) {
	sc.ReRender = append(sc.ReRender, w)
}

// note: this is replacement for "SetNeedsFullReRender()" call:

// SetNeedsLayoutUpdate sets the ScNeedsLayout flag
// if updt is true. See UpdateEndLayout for convenience method.
// This should be called after widget Config call
// _after_ calling UpdateEnd(updt) and passing
// that same updt flag from UpdateStart.
func (wb *WidgetBase) SetNeedsLayout(updt bool) {
	if !updt || wb.Scene == nil {
		return
	}
	if updt && DebugSettings.UpdateTrace {
		fmt.Println("\tDebugSettings.UpdateTrace: NeedsLayout:", wb)
	}
	wb.Scene.SetFlag(true, ScNeedsLayout)
}

// UpdateEndLayout should be called instead of UpdateEnd
// for any UpdateStart / UpdateEnd block that needs a re-layout
// at the end.  Just does SetNeedsLayout after UpdateEnd,
// and uses the cached wb.Sc pointer.
func (wb *WidgetBase) UpdateEndLayout(updt bool) {
	if !updt {
		return
	}
	wb.UpdateEnd(updt)
	wb.SetNeedsLayout(updt)
}

// NeedsRebuild returns true if the RenderContext indicates
// a full rebuild is needed.
func (wb *WidgetBase) NeedsRebuild() bool {
	if wb.This() == nil || wb.Scene == nil || wb.Scene.Stage == nil {
		return false
	}
	rc := wb.Scene.RenderCtx()
	if rc == nil {
		return false
	}
	return rc.HasFlag(RenderRebuild)
}

///////////////////////////////////////////////////////////////
// 	Config

// Config is the main wrapper configuration call, calling ConfigWidget
// which actually does the work. Use [WidgetBase.Update] to update styles too,
// which is typically needed once an item is displayed.
// Config by itself is sufficient during initial construction because
// everything will be automatically styled during initial display.
func (wb *WidgetBase) Config() {
	wi := wb.This().(Widget)
	updt := wi.UpdateStart()
	wi.ConfigWidget() // where everything actually happens
	wb.UpdateEnd(updt)
	wb.SetNeedsLayout(updt) // todo: switch to render here
}

// ConfigWidget is the interface method called by Config that
// should be defined for each Widget type, which actually does
// the configuration work.
func (wb *WidgetBase) ConfigWidget() {
	// this must be defined for each widget type
}

// ConfigParts initializes the parts of the widget if they
// are not already through [WidgetBase.NewParts], calls
// [ki.Node.ConfigChildren] on those parts with the given config,
// calls the given after function if it is specified,
// and then handles necessary updating logic.
func (wb *WidgetBase) ConfigParts(config ki.Config, after ...func()) {
	parts := wb.NewParts()
	mods, updt := parts.ConfigChildren(config)
	if len(after) > 0 {
		after[0]()
	}
	if !mods && !wb.NeedsRebuild() {
		return
	}
	parts.Update()
	parts.UpdateEnd(updt)
	wb.SetNeedsLayout(updt)
}

// ConfigTree calls Config on every Widget in the tree from me.
func (wb *WidgetBase) ConfigTree() {
	if wb.This() == nil {
		return
	}
	pr := prof.Start(wb.This().KiType().ShortName())
	wb.WidgetWalkPre(func(wi Widget, wb *WidgetBase) bool {
		wi.Config()
		return ki.Continue
	})
	pr.End()
}

// Update calls Config and then ApplyStyle
// on every Widget in the tree from me.
// This should be used after any structural changes
// to currently-displayed widgets.
// It wraps everything in UpdateStart / UpdateEndRender
// so node will render on next pass.
// Call SetNeedsLayout to also trigger a layout where needed.
func (wb *WidgetBase) Update() { //gti:add
	if wb == nil || wb.This() == nil || wb.Is(ki.Deleted) || wb.Is(ki.Destroyed) {
		return
	}
	updt := wb.UpdateStart()
	if DebugSettings.UpdateTrace {
		fmt.Println("\tDebugSettings.UpdateTrace Update:", wb, "updt:", updt)
	}
	wb.WidgetWalkPre(func(wi Widget, wb *WidgetBase) bool {
		wi.Config()
		wi.ApplyStyle()
		return ki.Continue
	})
	wb.UpdateEndRender(updt)
}

// ApplyStyleTree calls ApplyStyle on every Widget in the tree from me.
// Called during FullRender
func (wb *WidgetBase) ApplyStyleTree() {
	if wb.This() == nil {
		return
	}
	pr := prof.Start(wb.This().KiType().ShortName())
	wb.WidgetWalkPre(func(wi Widget, wb *WidgetBase) bool {
		wi.ApplyStyle()
		return ki.Continue
	})
	pr.End()
}

// LayoutScene does a layout of the scene: Size, Position
func (sc *Scene) LayoutScene() {
	if DebugSettings.LayoutTrace {
		fmt.Println("\n############################\nLayoutScene SizeUp start:", sc)
	}
	sc.SizeUp()
	sz := &sc.Geom.Size
	sz.Alloc.Total.SetPoint(sc.SceneGeom.Size)
	sz.SetContentFromTotal(&sz.Alloc)
	// sz.Actual = sz.Alloc // todo: is this needed??
	if DebugSettings.LayoutTrace {
		fmt.Println("\n############################\nSizeDown start:", sc)
	}
	maxIter := 3
	for iter := 0; iter < maxIter; iter++ { // 3  > 2; 4 same as 3
		redo := sc.SizeDown(iter)
		if redo && iter < maxIter-1 {
			if DebugSettings.LayoutTrace {
				fmt.Println("\n############################\nSizeDown redo:", sc, "iter:", iter+1)
			}
		} else {
			break
		}
	}
	if DebugSettings.LayoutTrace {
		fmt.Println("\n############################\nSizeFinal start:", sc)
	}
	sc.SizeFinal()
	if DebugSettings.LayoutTrace {
		fmt.Println("\n############################\nPosition start:", sc)
	}
	sc.Position()
	if DebugSettings.LayoutTrace {
		fmt.Println("\n############################\nScenePos start:", sc)
	}
	sc.ScenePos()
}

// LayoutRenderScene does a layout and render of the tree:
// GetSize, DoLayout, Render.  Needed after Config.
func (sc *Scene) LayoutRenderScene() {
	sc.LayoutScene()
	sc.Render()
}

// DoNeedsRender calls Render on tree from me for nodes
// with NeedsRender flags set
func (wb *WidgetBase) DoNeedsRender() {
	if wb.This() == nil {
		return
	}
	pr := prof.Start(wb.This().KiType().ShortName())
	wb.WidgetWalkPre(func(kwi Widget, kwb *WidgetBase) bool {
		if kwi.Is(NeedsRender) && !kwi.Is(ki.Updating) {
			kwi.SetFlag(false, NeedsRender)
			kwi.Render()
			return ki.Break // done
		}
		if ly := AsLayout(kwi); ly != nil {
			for d := mat32.X; d <= mat32.Y; d++ {
				if ly.HasScroll[d] {
					ly.Scrolls[d].DoNeedsRender()
				}
			}
		}
		return ki.Continue
	})
	pr.End()
}

//////////////////////////////////////////////////////////////////
//		Scene

var SceneShowIters = 2

// DoUpdate checks scene Needs flags to do whatever updating is required.
// returns false if already updating.
// This is the main update call made by the RenderWin at FPS frequency.
func (sc *Scene) DoUpdate() bool {
	if sc.Is(ScUpdating) {
		// fmt.Println("scene bail on updt")
		return false
	}
	sc.SetFlag(true, ScUpdating) // prevent rendering
	defer sc.SetFlag(false, ScUpdating)

	rc := sc.RenderCtx()

	if sc.ShowIter < SceneShowIters {
		if sc.ShowIter == 0 { // first time
			sc.EventMgr.GetPriorityWidgets()
		}
		sc.SetFlag(true, ScNeedsLayout)
		sc.ShowIter++
	}

	switch {
	case rc.HasFlag(RenderRebuild):
		sc.DoRebuild()
		sc.SetFlag(false, ScNeedsLayout, ScNeedsRender)
		sc.SetFlag(true, ScImageUpdated)
	case sc.LastRender.NeedsRestyle(rc):
		// fmt.Println("restyle")
		sc.ApplyStyleScene()
		sc.LayoutRenderScene()
		sc.SetFlag(false, ScNeedsLayout, ScNeedsRender)
		sc.SetFlag(true, ScImageUpdated)
		sc.LastRender.SaveRender(rc)
	case sc.Is(ScNeedsLayout):
		// fmt.Println("layout")
		sc.LayoutRenderScene()
		sc.SetFlag(false, ScNeedsLayout, ScNeedsRender)
		sc.SetFlag(true, ScImageUpdated)
	case sc.Is(ScNeedsRender):
		// fmt.Println("render")
		sc.DoNeedsRender()
		sc.SetFlag(false, ScNeedsRender)
		sc.SetFlag(true, ScImageUpdated)
	case len(sc.ReRender) > 0:
		// fmt.Println("re-render")
		for _, w := range sc.ReRender {
			w.SetFlag(true, ScNeedsRender)
		}
		sc.ReRender = nil
		sc.DoNeedsRender()
		sc.SetFlag(false, ScNeedsRender)
		sc.SetFlag(true, ScImageUpdated)

	default:
		return false
	}

	if sc.ShowIter == SceneShowIters { // end of first pass
		sc.ShowIter++
		if !sc.Is(ScPrefSizing) {
			sc.EventMgr.ActivateStartFocus()
		}
	}

	return true
}

// ConfigSceneWidgets calls Config on all widgets in the Scene,
// which will set NeedsLayout to drive subsequent layout and render.
// This is a top-level call, typically only done when the window
// is first drawn, once the full sizing information is available.
func (sc *Scene) ConfigSceneWidgets() {
	sc.SetFlag(true, ScUpdating) // prevent rendering
	defer sc.SetFlag(false, ScUpdating)

	sc.ConfigTree()
}

// ApplyStyleScene calls ApplyStyle on all widgets in the Scene,
// This is needed whenever the window geometry, DPI,
// etc is updated, which affects styling.
func (sc *Scene) ApplyStyleScene() {
	sc.SetFlag(true, ScUpdating) // prevent rendering
	defer sc.SetFlag(false, ScUpdating)

	sc.ApplyStyleTree()
	sc.SetFlag(true, ScNeedsLayout)
}

// DoRebuild does the full re-render and RenderContext Rebuild flag
// should be used by Widgets to rebuild things that are otherwise
// cached (e.g., Icon, TextCursor).
func (sc *Scene) DoRebuild() {
	sc.ConfigSceneWidgets()
	sc.ApplyStyleScene()
	sc.LayoutRenderScene()
}

// PrefSize computes the preferred size of the scene based on current contents.
// initSz is the initial size -- e.g., size of screen.
// Used for auto-sizing windows.
func (sc *Scene) PrefSize(initSz image.Point) image.Point {
	sc.SetFlag(true, ScUpdating) // prevent rendering
	defer sc.SetFlag(false, ScUpdating)

	sc.SetFlag(true, ScPrefSizing)
	sc.ConfigSceneWidgets()
	sc.ApplyStyleScene()
	sc.LayoutScene()
	sz := &sc.Geom.Size
	psz := sz.Actual.Total
	// fmt.Println("\npref size:", psz, "csz:", sz.Actual.Content, "internal:", sz.Internal, "space:", sc.Geom.Size.Space)
	sc.SetFlag(false, ScPrefSizing)
	sc.ShowIter = 0
	return psz.ToPointFloor()
}

//////////////////////////////////////////////////////////////////
//		Widget local rendering

// PushBounds pushes our bounding-box bounds onto the bounds stack if non-empty
// -- this limits our drawing to our own bounding box, automatically -- must
// be called as first step in Render returns whether the new bounds are
// empty or not -- if empty then don't render!
func (wb *WidgetBase) PushBounds() bool {
	if wb == nil || wb.This() == nil {
		return false
	}
	wb.SetFlag(false, NeedsRender)       // done!
	if !wb.This().(Widget).IsVisible() { // checks deleted etc
		return false
	}
	if wb.Geom.TotalBBox.Empty() {
		if DebugSettings.RenderTrace {
			fmt.Printf("Render empty bbox: %v at %v\n", wb.Path(), wb.Geom.TotalBBox)
		}
		return false
	}
	wb.Styles.ComputeActualBackground(wb.ParentActualBackground())
	pc := &wb.Scene.PaintContext
	pc.PushBounds(wb.Geom.TotalBBox)
	// rs.PushBounds(wb.Sc.Geom.TotalBBox)
	pc.Defaults() // start with default values
	if DebugSettings.RenderTrace {
		fmt.Printf("Render: %v at %v\n", wb.Path(), wb.Geom.TotalBBox)
	}
	return true
}

// PopBounds pops our bounding-box bounds -- last step in Render after
// rendering children
func (wb *WidgetBase) PopBounds() {
	if wb == nil || wb.This() == nil || wb.Is(ki.Deleted) {
		return
	}
	pc := &wb.Scene.PaintContext

	if wb.Scene.Is(ScRenderBBoxes) {
		pos := mat32.V2FromPoint(wb.Geom.TotalBBox.Min)
		sz := mat32.V2FromPoint(wb.Geom.TotalBBox.Size())
		// node: we won't necc. get a push prior to next update, so saving these.
		pcsw := pc.StrokeStyle.Width
		pcsc := pc.StrokeStyle.Color
		pcfc := pc.FillStyle.Color
		pcop := pc.FillStyle.Opacity
		pc.StrokeStyle.Width.Dot(1)
		pc.StrokeStyle.Color = colors.C(hct.New(wb.Scene.RenderBBoxHue, 100, 50))
		pc.FillStyle.Color = nil
		if wb.Scene.SelectedWidget != nil && wb.Scene.SelectedWidget.This() == wb.This() {
			fc := pc.StrokeStyle.Color
			pc.FillStyle.Color = fc
			pc.FillStyle.Opacity = 0.2
		}
		pc.DrawRectangle(pos.X, pos.Y, sz.X, sz.Y)
		pc.FillStrokeClear()
		// restore
		pc.FillStyle.Opacity = pcop
		pc.FillStyle.Color = pcfc
		pc.StrokeStyle.Width = pcsw
		pc.StrokeStyle.Color = pcsc

		wb.Scene.RenderBBoxHue += 10
		if wb.Scene.RenderBBoxHue > 360 {
			rmdr := (int(wb.Scene.RenderBBoxHue-360) + 1) % 9
			wb.Scene.RenderBBoxHue = float32(rmdr)
		}
	}

	pc.PopBounds()
}

// Render performs rendering on widget and parts, but not Children
// for the base type, which does not manage children (see Layout).
func (wb *WidgetBase) Render() {
	if wb.PushBounds() {
		wb.RenderParts()
		wb.PopBounds()
	}
}

func (wb *WidgetBase) RenderParts() {
	if wb.Parts == nil {
		return
	}
	wb.Parts.Render() // is a layout, will do all
}

// RenderChildren renders all of node's children.
func (wb *WidgetBase) RenderChildren() {
	wb.WidgetKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
		if !kwi.Is(ki.Updating) {
			kwi.Render()
		}
		return ki.Continue
	})
}

////////////////////////////////////////////////////////////////////////////////
//  Standard Box Model rendering

// RenderLock returns the locked [paint.Context] and [styles.Style] with StyMu locked.
// This should be called at start of widget-level rendering, and should always
// be associated with a corresponding [WidgetBase.RenderUnlock].
func (wb *WidgetBase) RenderLock() (*paint.Context, *styles.Style) {
	wb.StyMu.RLock()
	pc := &wb.Scene.PaintContext
	pc.Lock()
	return pc, &wb.Styles
}

// RenderUnlock unlocks the widget's associated [paint.Context] and StyMu.
// This should be called at the end of widget-level rendering, and should always
// be associated with a corresponding [WidgetBase.RenderLock].
func (wb *WidgetBase) RenderUnlock() {
	wb.Scene.PaintContext.Unlock()
	wb.StyMu.RUnlock()
}

// RenderBoxImpl implements the standard box model rendering -- assumes all
// paint params have already been set
func (wb *WidgetBase) RenderBoxImpl(pos mat32.Vec2, sz mat32.Vec2, bs styles.Border) {
	pc := &wb.Scene.PaintContext
	pc.DrawBorder(pos.X, pos.Y, sz.X, sz.Y, bs)
}

// RenderStdBox draws standard box using given style.
// paint.State and Style must already be locked at this point (RenderLock)
func (wb *WidgetBase) RenderStdBox(st *styles.Style) {
	wb.StyMu.RLock()
	defer wb.StyMu.RUnlock()

	pc := &wb.Scene.PaintContext

	pos := wb.Geom.Pos.Total
	sz := wb.Geom.Size.Actual.Total
	pc.DrawStdBox(st, pos, sz, wb.ParentActualBackground())
}

//////////////////////////////////////////////////////////////////
//		Widget position functions

// HasSc checks that the Sc Scene has been set.
// Called prior to using -- logs an error if not.
func (wb *WidgetBase) HasSc() bool {
	if wb.This() == nil || wb.Scene == nil {
		slog.Debug("gi.WidgetBase: object or scene is nil")
		return false
	}
	return true
}

// PointToRelPos translates a point in Scene pixel coords
// into relative position within node, based on the Content BBox
func (wb *WidgetBase) PointToRelPos(pt image.Point) image.Point {
	wb.BBoxMu.RLock()
	defer wb.BBoxMu.RUnlock()
	return pt.Sub(wb.Geom.ContentBBox.Min)
}

// WinBBox returns the RenderWin based bounding box for the widget
// by adding the Scene position to the ScBBox
func (wb *WidgetBase) WinBBox() image.Rectangle {
	if !wb.HasSc() {
		return wb.Geom.TotalBBox
	}
	return wb.Geom.TotalBBox.Add(wb.Scene.SceneGeom.Pos)
}

// WinPos returns the RenderWin based position within the
// bounding box of the widget, where the x, y coordinates
// are the proportion across the bounding box to use:
// 0 = left / top, 1 = right / bottom
func (wb *WidgetBase) WinPos(x, y float32) image.Point {
	bb := wb.WinBBox()
	sz := bb.Size()
	var pt image.Point
	pt.X = bb.Min.X + int(mat32.Round(float32(sz.X)*x))
	pt.Y = bb.Min.Y + int(mat32.Round(float32(sz.Y)*y))
	return pt
}

/////////////////////////////////////////////////////////////////////////////
//	Profiling and Benchmarking, controlled by hot-keys

// ProfileToggle turns profiling on or off, which does both
// targeted and global CPU and Memory profiling.
func ProfileToggle() {
	if prof.Profiling {
		EndTargProfile()
		EndCPUMemProfile()
	} else {
		StartTargProfile()
		StartCPUMemProfile()
	}
}

// StartCPUMemProfile starts the standard Go cpu and memory profiling.
func StartCPUMemProfile() {
	fmt.Println("Starting Std CPU / Mem Profiling")
	f, err := os.Create("cpu.prof")
	if err != nil {
		log.Fatal("could not create CPU profile: ", err)
	}
	if err := pprof.StartCPUProfile(f); err != nil {
		log.Fatal("could not start CPU profile: ", err)
	}
}

// EndCPUMemProfile ends the standard Go cpu and memory profiling.
func EndCPUMemProfile() {
	fmt.Println("Ending Std CPU / Mem Profiling")
	pprof.StopCPUProfile()
	f, err := os.Create("mem.prof")
	if err != nil {
		log.Fatal("could not create memory profile: ", err)
	}
	runtime.GC() // get up-to-date statistics
	if err := pprof.WriteHeapProfile(f); err != nil {
		log.Fatal("could not write memory profile: ", err)
	}
	f.Close()
}

// StartTargProfile starts targeted profiling using goki prof package.
func StartTargProfile() {
	fmt.Printf("Starting Targeted Profiling\n")
	prof.Reset()
	prof.Profiling = true
}

// EndTargProfile ends targeted profiling and prints report.
func EndTargProfile() {
	prof.Report(time.Millisecond)
	prof.Profiling = false
}

// ReportWinNodes reports the number of nodes in this scene
func (sc *Scene) ReportWinNodes() {
	nn := 0
	sc.WidgetWalkPre(func(wi Widget, wb *WidgetBase) bool {
		nn++
		return ki.Continue
	})
	fmt.Printf("Scene: %v has: %v nodes\n", sc.Name(), nn)
}

// BenchmarkFullRender runs benchmark of 50 full re-renders (full restyling, layout,
// and everything), reporting targeted profile results and generating standard
// Go cpu.prof and mem.prof outputs.
func (sc *Scene) BenchmarkFullRender() {
	fmt.Println("Starting BenchmarkFullRender")
	sc.ReportWinNodes()
	StartCPUMemProfile()
	StartTargProfile()
	ts := time.Now()
	n := 50
	for i := 0; i < n; i++ {
		sc.LayoutScene()
		sc.Render()
	}
	td := time.Since(ts)
	fmt.Printf("Time for %v Re-Renders: %12.2f s\n", n, float64(td)/float64(time.Second))
	EndTargProfile()
	EndCPUMemProfile()
}

// BenchmarkReRender runs benchmark of 50 re-render-only updates of display
// (just the raw rendering, no styling or layout), reporting targeted profile
// results and generating standard Go cpu.prof and mem.prof outputs.
func (sc *Scene) BenchmarkReRender() {
	fmt.Println("Starting BenchmarkReRender")
	sc.ReportWinNodes()
	StartTargProfile()
	ts := time.Now()
	n := 50
	for i := 0; i < n; i++ {
		sc.Render()
	}
	td := time.Since(ts)
	fmt.Printf("Time for %v Re-Renders: %12.2f s\n", n, float64(td)/float64(time.Second))
	EndTargProfile()
}
