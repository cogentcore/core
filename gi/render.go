// Copyright (c) 2023, The GoKi Authors. All rights reserved.
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

	"goki.dev/girl/paint"
	"goki.dev/girl/styles"
	"goki.dev/ki/v2"
	"goki.dev/mat32/v2"
	"goki.dev/prof/v2"
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
// * Layout: does GetSize, DoLayout on tree, arranging widgets.
//   Needed for whole tree after any Config changes anywhere
//   (could contain at RenderAnchor nodes).
// * Render: just draws with current config, layout.
//
// ApplyStyle is always called after Config, and after any
// current state of the Widget changes via events, animations, etc
// (e.g., a Hover started or a Button is pushed down).
// These changes should be protected by UpdateStart / End,
// such that ApplyStyle is only ever called within that scope.
// After the UpdateEnd(updt), call SetNeedsRender(vp, updt)
// which sets the node NeedsRender and ScNeedsRender flags,
// to drive the rendering update at next DoNeedsRender call.
//
// Because Render checks for Is(Updating) flag, and doesn't render
// if so, it should never be the case that a node is being modified
// and rendered at the same time, avoiding need for mutexes.
//
// For nodes with dynamic content that doesn't require styling or config
// a simple SetNeedsRender call will drive re-rendering. UpdateSig does this.
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
//
// ki Signals in general should not be used

// note: SetNeedsRender() is now SetNeedsRender()

// UpdateStart sets the scene ScUpdating flag to prevent
// render updates during construction on a scene.
func (wb *WidgetBase) UpdateStart() bool {
	updt := wb.Node.UpdateStart()
	if updt && wb.Sc != nil {
		wb.Sc.SetFlag(true, ScUpdating)
	}
	return updt
}

// UpdateEnd resets the scene ScUpdating flag
func (wb *WidgetBase) UpdateEnd(updt bool) {
	if updt && wb.Sc != nil {
		wb.Sc.SetFlag(false, ScUpdating)
	}
	wb.Node.UpdateEnd(updt)
}

// SetNeedsRender sets the NeedsRender and Scene NeedsRender flags,
// triggering a render of this widget on the next window update.
// Also sets a Field Parent NeedsRender too.
func (wb *WidgetBase) SetNeedsRender() {
	wb.SetNeedsRenderUpdate(wb.Sc, true)
}

// SetNeedsRenderUpdate sets the NeedsRender and Scene
// NeedsRender flags, if updt is true.
// See UpdateEndRender for convenience method.
// This should be called after widget state changes
// that don't need styling, e.g., in event handlers
// or other update code, _after_ calling UpdateEnd(updt) and passing
// that same updt flag from UpdateStart.
func (wb *WidgetBase) SetNeedsRenderUpdate(sc *Scene, updt bool) {
	if !updt || sc == nil {
		return
	}
	if UpdateTrace {
		fmt.Println("UpdateTrace: NeedsRender:", wb)
	}
	wb.SetFlag(true, NeedsRender)
	if sc != nil {
		sc.SetFlag(true, ScNeedsRender)
	}
	// parent of Parts needs to render if parent
	fi, _ := wb.ParentWidgetIf(func(p *WidgetBase) bool {
		return p.Is(ki.Field)
	})
	if fi != nil && fi.Parent() != nil {
		fi.Parent().This().SetFlag(true, NeedsRender)
	}
}

// UpdateEndRender should be called instead of UpdateEnd
// for any UpdateStart / UpdateEnd block that needs a re-render
// at the end.  Just does SetNeedsRender after UpdateEnd,
// and uses the cached wb.Sc pointer.
func (wb *WidgetBase) UpdateEndRender(updt bool) {
	if !updt {
		return
	}
	wb.UpdateEnd(updt)
	wb.SetNeedsRenderUpdate(wb.Sc, updt)
}

// note: this is replacement for "SetNeedsFullReRender()" call:

// SetNeedsLayout sets the ScNeedsLayout flag.
// Use this when a change definitely requires a new Layout.
func (wb *WidgetBase) SetNeedsLayout() {
	wb.SetNeedsLayoutUpdate(wb.Sc, true)
}

// SetNeedsLayoutUpdate sets the ScNeedsLayout flag
// if updt is true. See UpdateEndLayout for convenience method.
// This should be called after widget Config call
// _after_ calling UpdateEnd(updt) and passing
// that same updt flag from UpdateStart.
func (wb *WidgetBase) SetNeedsLayoutUpdate(sc *Scene, updt bool) {
	if !updt || sc == nil {
		return
	}
	if updt && UpdateTrace {
		fmt.Println("UpdateTrace: NeedsLayout:", wb)
	}
	sc.SetFlag(true, ScNeedsLayout)
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
	wb.SetNeedsLayoutUpdate(wb.Sc, updt)
}

// NeedsRebuild returns true if the RenderContext indicates
// a full rebuild is needed.
func (wb *WidgetBase) NeedsRebuild() bool {
	if wb.This() == nil || wb.Sc == nil || wb.Sc.Stage == nil {
		return false
	}
	rc := wb.Sc.RenderCtx()
	if rc == nil {
		return false
	}
	return rc.HasFlag(RenderRebuild)
}

///////////////////////////////////////////////////////////////
// 	Config

// Config is the main wrapper configuration call, calling ConfigWidget
// which actually does the work. Use ReConfig to update styles too,
// which is typically needed once an item is displayed.
// Config by itself is sufficient during initial construction because
// everything will be automatically styled during initial display.
func (wb *WidgetBase) Config(sc *Scene) {
	if wb.This() == nil {
		slog.Error("nil this in config")
		return
	}
	wi := wb.This().(Widget)
	updt := wi.UpdateStart()
	wb.Sc = sc
	wi.ConfigWidget(sc) // where everything actually happens
	wb.UpdateEnd(updt)
	wb.SetNeedsLayoutUpdate(sc, updt)
}

// ConfigWidget is the interface method called by Config that
// should be defined for each Widget type, which actually does
// the configuration work.
func (wb *WidgetBase) ConfigWidget(sc *Scene) {
	// this must be defined for each widget type
}

// ConfigPartsImpl initializes the parts of the widget if they
// are not already through [WidgetBase.NewParts], calls
// [ki.Node.ConfigChildren] on those parts with the given config,
// and then handles necessary updating logic with the given scene.
func (wb *WidgetBase) ConfigPartsImpl(sc *Scene, config ki.Config, lay Layouts) {
	parts := wb.NewParts(lay)
	mods, updt := parts.ConfigChildren(config)
	if !mods && !wb.NeedsRebuild() {
		parts.UpdateEnd(updt)
		return
	}
	parts.UpdateEnd(updt)
	wb.SetNeedsLayoutUpdate(sc, updt)
}

// ConfigTree calls Config on every Widget in the tree from me.
func (wb *WidgetBase) ConfigTree(sc *Scene) {
	if wb.This() == nil {
		return
	}
	pr := prof.Start("Widget.ConfigTree." + wb.KiType().Name)
	wb.WalkPre(func(k ki.Ki) bool {
		wi, w := AsWidget(k)
		if w == nil || w.Is(ki.Deleted) || w.Is(ki.Destroyed) {
			return ki.Break
		}
		wi.Config(sc)
		return ki.Continue
	})
	pr.End()
}

// ReConfig is a convenience method for reconfiguring a widget after
// changes have been made, calling Config then ApplyStyle.
// The plain Config method is used during initial configuration.
// It is called by the Scene and caches the Sc pointer.
func (wb *WidgetBase) ReConfig() {
	if !wb.HasSc() {
		return
	}
	wi := wb.This().(Widget)
	wi.Config(wb.Sc)
	wi.ApplyStyle(wb.Sc)
}

// ReConfigTree calls Config and then ApplyStyle
// on every Widget in the tree from me.
// This should be used for currently-displayed widgets.
func (wb *WidgetBase) ReConfigTree(sc *Scene) {
	if wb.This() == nil {
		return
	}
	pr := prof.Start("Widget.ConfigTree." + wb.KiType().Name)
	wb.WalkPre(func(k ki.Ki) bool {
		wi, w := AsWidget(k)
		if w == nil || w.Is(ki.Deleted) || w.Is(ki.Destroyed) {
			return ki.Break
		}
		wi.Config(sc) // sets sc if not
		wi.ApplyStyle(sc)
		return ki.Continue
	})
	pr.End()
}

// ApplyStyleTree calls ApplyStyle on every Widget in the tree from me.
// Called during FullRender
func (wb *WidgetBase) ApplyStyleTree(sc *Scene) {
	if wb.This() == nil {
		return
	}
	pr := prof.Start("Widget.ApplyStyleTree." + wb.KiType().Name)
	wb.WalkPre(func(k ki.Ki) bool {
		wi, w := AsWidget(k)
		if w == nil || w.Is(ki.Deleted) || w.Is(ki.Destroyed) {
			return ki.Break
		}
		wi.ApplyStyle(sc)
		return ki.Continue
	})
	pr.End()
}

// GetSizeTree does the sizing as a depth-first pass from me,
// needed for Layout stack.
func (wb *WidgetBase) GetSizeTree(sc *Scene, iter int) {
	if wb.This() == nil {
		return
	}
	pr := prof.Start("Widget.GetSizeTree." + wb.KiType().Name)
	wb.WalkPost(func(k ki.Ki) bool { // tests whether to process node
		_, w := AsWidget(k)
		if w == nil || w.Is(ki.Deleted) || w.Is(ki.Destroyed) {
			return ki.Break
		}
		return ki.Continue
	},
		func(k ki.Ki) bool { // this one does the work
			wi, w := AsWidget(k)
			if w == nil || w.Is(ki.Deleted) || w.Is(ki.Destroyed) {
				return ki.Break
			}
			wi.GetSize(sc, iter)
			return ki.Continue
		})
	pr.End()
}

// DoLayoutTree does layout pass for tree from me.
// Each node iterates over children for maximum control,
// Starting with parent ScBBox.
// Handles multiple iterations if needed.
func (wb *WidgetBase) DoLayoutTree(sc *Scene) {
	if wb.This() == nil {
		return
	}
	pr := prof.Start("WidgetBase.DoLayoutTree." + wb.KiType().Name)
	parBBox := image.Rectangle{}
	pwi, _ := AsWidget(wb.Par)
	if pwi != nil {
		parBBox = pwi.ChildrenBBoxes(sc)
	} else {
		parBBox = image.Rectangle{Max: sc.Geom.Size} // sc.Pixels.Bounds()
		// fmt.Println("parBBox:", parBBox)
	}
	wi := wb.This().(Widget)
	if wi == nil {
		return
	}
	redo := wi.DoLayout(sc, parBBox, 0) // important to use interface version to get interface!
	if redo {
		if LayoutTrace {
			fmt.Printf("Layout: ----------  Redo: %v ----------- \n", wi.Path())
		}
		la := wb.LayState.Alloc
		wb.GetSizeTree(sc, 1)
		wb.LayState.Alloc = la
		wi.DoLayout(sc, parBBox, 1) // todo: multiple iters?
	}
	pr.End()
}

// LayoutScene does a layout of the tree: GetSize, DoLayout.
func (sc *Scene) LayoutScene() {
	sc.GetSizeTree(sc, 0)
	sc.LayState.Alloc.Size = mat32.NewVec2FmPoint(sc.Geom.Size)
	sc.DoLayoutTree(sc)
}

// LayoutRenderScene does a layout and render of the tree:
// GetSize, DoLayout, Render.  Needed after Config.
func (sc *Scene) LayoutRenderScene() {
	sc.LayoutScene()
	sc.Render(sc)
}

// DoNeedsRender calls Render on tree from me for nodes
// with NeedsRender flags set
func (wb *WidgetBase) DoNeedsRender(sc *Scene) {
	if wb.This() == nil {
		return
	}
	pr := prof.Start("Widget.DoNeedsRender." + wb.KiType().Name)
	wb.WalkPre(func(k ki.Ki) bool {
		wi, w := AsWidget(k)
		if w == nil || w.Is(ki.Deleted) || w.Is(ki.Destroyed) {
			return ki.Break
		}
		if w.Is(NeedsRender) && !w.Is(ki.Updating) {
			w.SetFlag(false, NeedsRender)
			wi.Render(sc)
			return ki.Break // done
		}
		return ki.Continue
	})
	pr.End()
}

//////////////////////////////////////////////////////////////////
//		Scene

// DoUpdate checks scene Needs flags to do whatever updating is required.
// returns false if already updating.
// This is the main update call made by the RenderWin at FPS frequency.
func (sc *Scene) DoUpdate() bool {
	if sc.Is(ScUpdating) {
		fmt.Println("scene bail on updt")
		return false
	}
	sc.SetFlag(true, ScUpdating) // prevent rendering
	defer sc.SetFlag(false, ScUpdating)

	rc := sc.RenderCtx()
	if rc == nil {
		slog.Error("scene render context is nil", "scene", sc.Nm)
		return true
	}

	switch {
	case rc.HasFlag(RenderRebuild):
		// fmt.Println("rebuild")
		sc.DoRebuild()
		sc.SetFlag(false, ScNeedsLayout, ScNeedsRender)
		sc.SetFlag(true, ScImageUpdated)
	case sc.LastRender.NeedsRestyle(rc):
		// fmt.Println("scene restyle")
		sc.Fill() // full redraw
		sc.ApplyStyleScene()
		sc.LayoutRenderScene()
		sc.SetFlag(false, ScNeedsLayout, ScNeedsRender)
		sc.SetFlag(true, ScImageUpdated)
		sc.LastRender.SaveRender(rc)
	case sc.Is(ScNeedsLayout):
		// fmt.Println("scene layout start")
		sc.Fill() // full redraw
		sc.LayoutRenderScene()
		sc.SetFlag(false, ScNeedsLayout, ScNeedsRender)
		sc.SetFlag(true, ScImageUpdated)
		// fmt.Println("scene layout done")
	case sc.Is(ScNeedsRender):
		// fmt.Println("scene render start")
		sc.DoNeedsRender(sc)
		sc.SetFlag(false, ScNeedsRender)
		sc.SetFlag(true, ScImageUpdated)
		// fmt.Println("scene render done")
	default:
		return false
	}
	return true
}

// ConfigScene calls Config on all widgets in the Scene,
// which will set NeedsLayout to drive subsequent layout and render.
// This is a top-level call, typically only done when the window
// is first drawn, once the full sizing information is available.
func (sc *Scene) ConfigScene() {
	sc.SetFlag(true, ScUpdating) // prevent rendering
	defer sc.SetFlag(false, ScUpdating)

	sc.ConfigTree(sc)
}

// ApplyStyleScene calls ApplyStyle on all widgets in the Scene,
// This is needed whenever the window geometry, DPI,
// etc is updated, which affects styling.
func (sc *Scene) ApplyStyleScene() {
	sc.SetFlag(true, ScUpdating) // prevent rendering
	defer sc.SetFlag(false, ScUpdating)

	sc.ApplyStyleTree(sc)
	sc.SetFlag(true, ScNeedsLayout)
}

// DoRebuild does the full re-render and RenderContext Rebuild flag
// should be used by Widgets to rebuild things that are otherwise
// cached (e.g., Icon, TextCursor).
func (sc *Scene) DoRebuild() {
	sc.Fill()         // full redraw
	ld := sc.LayState // save our current layout data
	sc.ConfigScene()
	sc.ApplyStyleScene()
	sc.LayState = ld
	sc.LayoutRenderScene()
}

// Fill fills the scene with BgColor (default transparent)
// which is the starting base level for rendering.
// Typically the root Frame fills its background with color
// but it can e.g., leave corners transparent for popups etc.
func (sc *Scene) Fill() {
	rs := &sc.RenderState
	rs.Lock()
	rs.Paint.FillBox(rs, mat32.Vec2Zero, mat32.NewVec2FmPoint(sc.Geom.Size), &sc.BgColor)
	rs.Unlock()
}

// PrefSize computes the preferred size of the scene based on current contents.
// initSz is the initial size -- e.g., size of screen.
// Used for auto-sizing windows.
func (sc *Scene) PrefSize(initSz image.Point) image.Point {
	sc.SetFlag(true, ScUpdating) // prevent rendering
	defer sc.SetFlag(false, ScUpdating)

	sc.SetFlag(true, ScPrefSizing)
	sc.ConfigScene()

	sc.Geom.Size = initSz
	for i := 0; i < 2; i++ {
		sc.ApplyStyleTree(sc) // sufficient to get sizes
		sc.LayState.Alloc.Size.SetPoint(initSz)
		sc.GetSizeTree(sc, 0) // collect sizes
		sc.LayState.Alloc.Size.SetPoint(initSz)
		sc.DoLayoutTree(sc)
	}

	sc.SetFlag(false, ScPrefSizing)

	vpsz := sc.LayState.Size.Pref.ToPoint()
	// also take into account min size pref
	stw := int(sc.Styles.MinWidth.Dots)
	sth := int(sc.Styles.MinHeight.Dots)
	// fmt.Printf("dlg stw %v sth %v dpi %v vpsz: %v\n", stw, sth, dlg.Sty.UnContext.DPI, vpsz)
	vpsz.X = max(vpsz.X, stw)
	vpsz.Y = max(vpsz.Y, sth)
	return vpsz
}

//////////////////////////////////////////////////////////////////
//		Widget local rendering

// PushBounds pushes our bounding-box bounds onto the bounds stack if non-empty
// -- this limits our drawing to our own bounding box, automatically -- must
// be called as first step in Render returns whether the new bounds are
// empty or not -- if empty then don't render!
func (wb *WidgetBase) PushBounds(sc *Scene) bool {
	if wb == nil || wb.This() == nil {
		return false
	}
	if !wb.This().(Widget).IsVisible() {
		return false
	}
	if wb.ScBBox.Empty() {
		return false
	}
	rs := &sc.RenderState
	rs.PushBounds(wb.ScBBox)
	if RenderTrace {
		fmt.Printf("Render: %v at %v\n", wb.Path(), wb.ScBBox)
	}
	return true
}

// PopBounds pops our bounding-box bounds -- last step in Render after
// rendering children
func (wb *WidgetBase) PopBounds(sc *Scene) {
	if wb.Is(ki.Deleted) || wb.Is(ki.Destroyed) || wb.This() == nil {
		return
	}
	rs := &sc.RenderState
	rs.PopBounds()
}

func (wb *WidgetBase) Render(sc *Scene) {
	if wb.PushBounds(sc) {
		wb.RenderParts(sc)
		wb.RenderChildren(sc)
		wb.PopBounds(sc)
	}
}

func (wb *WidgetBase) RenderParts(sc *Scene) {
	if wb.Parts == nil {
		return
	}
	wb.Parts.Render(sc) // is a layout, will do all
}

// RenderChildren renders all of node's children,
// This is the default call at end of Render()
func (wb *WidgetBase) RenderChildren(sc *Scene) {
	for _, kid := range wb.Kids {
		wi, w := AsWidget(kid)
		if w == nil || w.Is(ki.Deleted) || w.Is(ki.Destroyed) || w.Is(ki.Updating) {
			continue
		}
		wi.Render(sc)
	}
}

/* todo: anything needed here?

// ReRenderTree does a re-render of the tree -- after it has already been
// initialized and styled -- redoes the full stack
func (wb *WidgetBase) ReRenderTree() {
	parBBox := image.Rectangle{}
	pni, _ := KiToWidget(wb.Par)
	if pni != nil {
		parBBox = pni.ChildrenBBoxes(vp)
	}
	delta := wb.LayState.Alloc.Pos.Sub(wb.LayState.Alloc.PosOrig)
	wb.LayState.Alloc.Pos = wb.LayState.Alloc.PosOrig
	ld := wb.LayState // save our current layout data
	updt := wb.UpdateStart()
	wb.ConfigTree()
	wb.ApplyStyleTree()
	wb.GetSizeTree(0)
	wb.LayState = ld // restore
	wb.DoLayoutTree()
	if !delta.IsNil() {
		wb.LayoutScroll(delta.ToPointFloor(), parBBox)
	}
	wb.RenderTree()
	wb.UpdateEndNoSig(updt)
}
*/

////////////////////////////////////////////////////////////////////////////////
//  Standard Box Model rendering

// RenderLock returns the locked paint.State, Paint, and Style with StyMu locked.
// This should be called at start of widget-level rendering.
func (wb *WidgetBase) RenderLock(sc *Scene) (*paint.State, *paint.Paint, *styles.Style) {
	wb.StyMu.RLock()
	rs := &sc.RenderState
	rs.Lock()
	return rs, &rs.Paint, &wb.Styles
}

// RenderUnlock unlocks paint.State and style
func (wb *WidgetBase) RenderUnlock(rs *paint.State) {
	rs.Unlock()
	wb.StyMu.RUnlock()
}

// RenderBoxImpl implements the standard box model rendering -- assumes all
// paint params have already been set
func (wb *WidgetBase) RenderBoxImpl(sc *Scene, pos mat32.Vec2, sz mat32.Vec2, bs styles.Border) {
	rs := &sc.RenderState
	pc := &rs.Paint
	pc.DrawBox(rs, pos, sz, bs)
}

// RenderStdBox draws standard box using given style.
// paint.State and Style must already be locked at this point (RenderLock)
func (wb *WidgetBase) RenderStdBox(sc *Scene, st *styles.Style) {
	// SidesTODO: this is a pretty critical function, so a good place to look if things aren't working
	wb.StyMu.RLock()
	defer wb.StyMu.RUnlock()

	rs := &sc.RenderState
	pc := &rs.Paint

	csp := wb.ParentBackgroundColor()
	pc.DrawStdBox(rs, st, wb.LayState.Alloc.Pos, wb.LayState.Alloc.Size, &csp)
}

/////////////////////////////////////////////////////////////////////////////
//                   Profiling and Benchmarking, controlled by hot-keys

// ProfileToggle turns profiling on or off
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
	sc.WalkPre(func(k ki.Ki) bool {
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
		sc.DoLayoutTree(sc)
		sc.Render(sc)
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
		sc.Render(sc)
	}
	td := time.Since(ts)
	fmt.Printf("Time for %v Re-Renders: %12.2f s\n", n, float64(td)/float64(time.Second))
	EndTargProfile()
}
