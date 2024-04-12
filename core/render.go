// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

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
	"cogentcore.org/core/math32"
	"cogentcore.org/core/profile"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/tree"
)

// Rendering logic:
//
// Key principles:
//
// * Async updates (animation, mouse events, etc) change state, _set only flags_
//   using thread-safe atomic bitflag operations. True async rendering
//   is really hard to get right, and requires tons of mutexes etc. Async updates
//	 must go through [WidgetBase.AsyncLock] and [WidgetBase.AsyncUnlock].
// * Synchronous, full-tree render updates do the layout, rendering,
//   at regular FPS (frames-per-second) rate -- nop unless flag set.
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
// Use NeedsRender() to drive the rendering update at next DoNeedsRender call.
//
// The initial configuration of a scene can skip calling
// Config and ApplyStyle because these will be called automatically
// during the Run() process for the Scene.
//
// For dynamic reconfiguration after initial display,
// ReConfg() is the key method, calling Config then
// ApplyStyle on the node and all of its children.
//
// For nodes with dynamic content that doesn't require styling or config,
// a simple NeedsRender call will drive re-rendering.
//
// Updating is _always_ driven top-down by RenderWin at FPS sampling rate,
// in the DoUpdate() call on the Scene.
// Three types of updates can be triggered, in order of least impact
// and highest frequency first:
// * ScNeedsRender: does NeedsRender on nodes.
// * ScNeedsLayout: does GetSize, DoLayout, then Render -- after Config.
//

// AsyncLock must be called before making any updates in a separate goroutine
// outside of the main configuration, rendering, and event handling structure.
// It must have a matching [WidgetBase.AsyncUnlock] after it.
func (wb *WidgetBase) AsyncLock() {
	rc := wb.Scene.RenderContext()
	if rc == nil {
		// if there is no render context, we are probably
		// being deleted, so we just block forever
		select {}
	}
	rc.Lock()
	if wb.This() == nil {
		rc.Unlock()
		select {}
	}
	wb.Scene.SetFlag(true, ScUpdating)
}

// AsyncUnlock must be called after making any updates in a separate goroutine
// outside of the main configuration, rendering, and event handling structure.
// It must have a matching [WidgetBase.AsyncLock] before it.
func (wb *WidgetBase) AsyncUnlock() {
	rc := wb.Scene.RenderContext()
	if rc == nil {
		return
	}
	rc.Unlock()
	wb.Scene.SetFlag(false, ScUpdating)
}

// NeedsRender specifies that the widget needs to be rendered.
func (wb *WidgetBase) NeedsRender() {
	if wb.Scene == nil {
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
		return p.Is(tree.Field)
	})
	if fi != nil && fi.Parent() != nil && fi.Parent().This() != nil {
		fi.Parent().(Widget).AsWidget().NeedsRender()
	}
}

// NeedsLayout specifies that the widget's scene needs to do a layout.
// This needs to be called after any changes that affect the structure
// and/or size of elements.
func (wb *WidgetBase) NeedsLayout() {
	if wb.Scene == nil {
		return
	}
	if DebugSettings.UpdateTrace {
		fmt.Println("\tDebugSettings.UpdateTrace: NeedsLayout:", wb)
	}
	wb.Scene.SetFlag(true, ScNeedsLayout)
}

// AddReRender adds given widget to be re-rendered next pass
func (sc *Scene) AddReRender(w Widget) {
	sc.ReRender = append(sc.ReRender, w)
}

// NeedsRebuild returns true if the RenderContext indicates
// a full rebuild is needed.
func (wb *WidgetBase) NeedsRebuild() bool {
	if wb.This() == nil || wb.Scene == nil || wb.Scene.Stage == nil {
		return false
	}
	rc := wb.Scene.RenderContext()
	if rc == nil {
		return false
	}
	return rc.HasFlag(RenderRebuild)
}

///////////////////////////////////////////////////////////////
// 	Config

// Config is the interface method called by Config that
// should be defined for each Widget type, which actually does
// the configuration work.
func (wb *WidgetBase) Config() {
	// this must be defined for each widget type
}

// ConfigParts initializes the parts of the widget if they
// are not already through [WidgetBase.NewParts], calls
// [tree.NodeBase.ConfigChildren] on those parts with the given config,
// calls the given after function if it is specified,
// and then handles necessary updating logic.
func (wb *WidgetBase) ConfigParts(config tree.Config, after ...func()) {
	parts := wb.NewParts()
	mods := parts.ConfigChildren(config)
	if len(after) > 0 {
		after[0]()
	}
	if !mods && !wb.NeedsRebuild() {
		return
	}
	parts.Update()
}

// ConfigTree calls Config on every Widget in the tree from me.
func (wb *WidgetBase) ConfigTree() {
	if wb.This() == nil {
		return
	}
	pr := profile.Start(wb.This().NodeType().ShortName())
	wb.WidgetWalkPre(func(wi Widget, wb *WidgetBase) bool {
		wi.Config()
		return tree.Continue
	})
	pr.End()
}

// Update does a general purpose update of the widget and everything
// below it by reconfiguring it, applying its styles, and indicating
// that it needs a new layout pass. It is the main way that end users
// should update widgets, and it should be called after making any
// changes to the core properties of a widget (for example, the text
// of a label, the icon of a button, or the slice of a table view).
//
// If you are calling this in a separate goroutine outside of the main
// configuration, rendering, and event handling structure, you need to
// call [WidgetBase.AsyncLock] and [WidgetBase.AsyncUnlock] before and
// after this, respectively.
func (wb *WidgetBase) Update() { //types:add
	if wb == nil || wb.This() == nil {
		return
	}
	if DebugSettings.UpdateTrace {
		fmt.Println("\tDebugSettings.UpdateTrace Update:", wb)
	}
	wb.WidgetWalkPre(func(wi Widget, wb *WidgetBase) bool {
		wi.Config()
		wi.ApplyStyle()
		return tree.Continue
	})
	wb.NeedsLayout()
}

// ApplyStyleTree calls ApplyStyle on every Widget in the tree from me.
// Called during FullRender
func (wb *WidgetBase) ApplyStyleTree() {
	if wb.This() == nil {
		return
	}
	pr := profile.Start(wb.This().NodeType().ShortName())
	wb.WidgetWalkPre(func(wi Widget, wb *WidgetBase) bool {
		wi.ApplyStyle()
		return tree.Continue
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
	sc.RenderWidget()
}

// DoNeedsRender calls Render on tree from me for nodes
// with NeedsRender flags set
func (wb *WidgetBase) DoNeedsRender() {
	if wb.This() == nil {
		return
	}
	pr := profile.Start(wb.This().NodeType().ShortName())
	wb.WidgetWalkPre(func(kwi Widget, kwb *WidgetBase) bool {
		if kwi.Is(NeedsRender) {
			kwi.RenderWidget()
			return tree.Break // done
		}
		if ly := AsLayout(kwi); ly != nil {
			for d := math32.X; d <= math32.Y; d++ {
				if ly.HasScroll[d] {
					ly.Scrolls[d].DoNeedsRender()
				}
			}
		}
		return tree.Continue
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

	rc := sc.RenderContext()

	if sc.ShowIter < SceneShowIters {
		if sc.ShowIter == 0 { // first time
			sc.EventMgr.GetShortcuts()
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

// PushBounds pushes our bounding box bounds onto the bounds stack
// if they are non-empty. This automatically limits our drawing to
// our own bounding box. This must be called as the first step in
// Render implementations. It returns whether the new bounds are
// empty or not; if they are empty, then don't render.
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
	pc.Defaults() // start with default values
	if DebugSettings.RenderTrace {
		fmt.Printf("Render: %v at %v\n", wb.Path(), wb.Geom.TotalBBox)
	}
	return true
}

// PopBounds pops our bounding box bounds. This is the last step
// in Render implementations after rendering children.
func (wb *WidgetBase) PopBounds() {
	if wb == nil || wb.This() == nil {
		return
	}
	pc := &wb.Scene.PaintContext

	isSelw := wb.Scene.SelectedWidget == wb.This()
	if wb.Scene.Is(ScRenderBBoxes) || isSelw {
		pos := math32.Vector2FromPoint(wb.Geom.TotalBBox.Min)
		sz := math32.Vector2FromPoint(wb.Geom.TotalBBox.Size())
		// node: we won't necc. get a push prior to next update, so saving these.
		pcsw := pc.StrokeStyle.Width
		pcsc := pc.StrokeStyle.Color
		pcfc := pc.FillStyle.Color
		pcop := pc.FillStyle.Opacity
		pc.StrokeStyle.Width.Dot(1)
		pc.StrokeStyle.Color = colors.C(hct.New(wb.Scene.RenderBBoxHue, 100, 50))
		pc.FillStyle.Color = nil
		if isSelw {
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

// Render is the method that widgets should implement to define their
// custom rendering steps. It should not be called outside of
// [Widget.RenderWidget], which also does other steps applicable
// for all widgets.
func (wb *WidgetBase) Render() {}

// RenderWidget renders the widget and any parts and children that it has.
// It does not render if the widget is invisible. It calls Widget.Render]
// for widget-specific rendering.
func (wb *WidgetBase) RenderWidget() {
	if wb.PushBounds() {
		wb.This().(Widget).Render()
		wb.RenderParts()
		wb.RenderChildren()
		wb.PopBounds()
	}
}

func (wb *WidgetBase) RenderParts() {
	if wb.Parts != nil {
		wb.Parts.RenderWidget()
	}
}

// RenderChildren renders all of the widget's children.
func (wb *WidgetBase) RenderChildren() {
	wb.WidgetKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
		kwi.RenderWidget()
		return tree.Continue
	})
}

////////////////////////////////////////////////////////////////////////////////
//  Standard Box Model rendering

// RenderBoxImpl implements the standard box model rendering -- assumes all
// paint params have already been set
func (wb *WidgetBase) RenderBoxImpl(pos math32.Vector2, sz math32.Vector2, bs styles.Border) {
	wb.Scene.PaintContext.DrawBorder(pos.X, pos.Y, sz.X, sz.Y, bs)
}

// RenderStandardBox renders the standard box model.
func (wb *WidgetBase) RenderStandardBox() {
	pos := wb.Geom.Pos.Total
	sz := wb.Geom.Size.Actual.Total
	wb.Scene.PaintContext.DrawStandardBox(&wb.Styles, pos, sz, wb.ParentActualBackground())
}

//////////////////////////////////////////////////////////////////
//		Widget position functions

// HasSc checks that the Sc Scene has been set.
// Called prior to using -- logs an error if not.
func (wb *WidgetBase) HasSc() bool {
	if wb.This() == nil || wb.Scene == nil {
		slog.Debug("core.WidgetBase: object or scene is nil")
		return false
	}
	return true
}

// PointToRelPos translates a point in Scene pixel coords
// into relative position within node, based on the Content BBox
func (wb *WidgetBase) PointToRelPos(pt image.Point) image.Point {
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
	pt.X = bb.Min.X + int(math32.Round(float32(sz.X)*x))
	pt.Y = bb.Min.Y + int(math32.Round(float32(sz.Y)*y))
	return pt
}

/////////////////////////////////////////////////////////////////////////////
//	Profiling and Benchmarking, controlled by hot-keys

// ProfileToggle turns profiling on or off, which does both
// targeted and global CPU and Memory profiling.
func ProfileToggle() { //types:add
	if profile.Profiling {
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

// StartTargProfile starts targeted profiling using the prof package.
func StartTargProfile() {
	fmt.Printf("Starting Targeted Profiling\n")
	profile.Reset()
	profile.Profiling = true
}

// EndTargProfile ends targeted profiling and prints report.
func EndTargProfile() {
	profile.Report(time.Millisecond)
	profile.Profiling = false
}

// ReportWinNodes reports the number of nodes in this scene
func (sc *Scene) ReportWinNodes() {
	nn := 0
	sc.WidgetWalkPre(func(wi Widget, wb *WidgetBase) bool {
		nn++
		return tree.Continue
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
		sc.RenderWidget()
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
		sc.RenderWidget()
	}
	td := time.Since(ts)
	fmt.Printf("Time for %v Re-Renders: %12.2f s\n", n, float64(td)/float64(time.Second))
	EndTargProfile()
}
