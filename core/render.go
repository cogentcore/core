// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"fmt"
	"image"
	"os"
	"runtime"
	"runtime/pprof"
	"time"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/profile"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/cam/hct"
	"cogentcore.org/core/math32"
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
//   See layout.go for full details and code.
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
// Update() is the key method, calling Config then
// ApplyStyle on the node and all of its children.
//
// For nodes with dynamic content that doesn't require styling or config,
// a simple NeedsRender call will drive re-rendering.
//
// Updating is _always_ driven top-down by RenderWindow at FPS sampling rate,
// in the DoUpdate() call on the Scene.
// Three types of updates can be triggered, in order of least impact
// and highest frequency first:
// * ScNeedsRender: does NeedsRender on nodes.
// * ScNeedsLayout: does GetSize, DoLayout, then Render -- after Config.
//

// Async calls the given function after [WidgetBase.AsyncLock] and before
// [WidgetBase.AsyncUnlock]. It should be used when making any updates in
// a separate goroutine outside of the main configuration, rendering, and
// event handling structure. If those updates are not contained within a
// single function, you should call [WidgetBase.AsyncLock] and
// [WidgetBase.AsyncUnlock] directly instead.
func (wb *WidgetBase) Async(f func()) {
	wb.AsyncLock()
	f()
	wb.AsyncUnlock()
}

// AsyncLock must be called before making any updates in a separate goroutine
// outside of the main configuration, rendering, and event handling structure.
// It must have a matching [WidgetBase.AsyncUnlock] after it. Also see
// [WidgetBase.Async].
func (wb *WidgetBase) AsyncLock() {
	rc := wb.Scene.renderContext()
	if rc == nil {
		// if there is no render context, we are probably
		// being deleted, so we just block forever
		select {}
	}
	rc.lock()
	if wb.This == nil {
		rc.unlock()
		select {}
	}
	wb.Scene.updating = true
}

// AsyncUnlock must be called after making any updates in a separate goroutine
// outside of the main configuration, rendering, and event handling structure.
// It must have a matching [WidgetBase.AsyncLock] before it. Also see
// [WidgetBase.Async].
func (wb *WidgetBase) AsyncUnlock() {
	rc := wb.Scene.renderContext()
	if rc == nil {
		return
	}
	rc.unlock()
	if wb.Scene != nil {
		wb.Scene.updating = false
	}
}

// NeedsRender specifies that the widget needs to be rendered.
func (wb *WidgetBase) NeedsRender() {
	if DebugSettings.UpdateTrace {
		fmt.Println("\tDebugSettings.UpdateTrace: NeedsRender:", wb)
	}
	wb.needsRender = true
	if wb.Scene != nil {
		wb.Scene.sceneNeedsRender = true
	}
}

// NeedsLayout specifies that the widget's scene needs to do a layout.
// This needs to be called after any changes that affect the structure
// and/or size of elements.
func (wb *WidgetBase) NeedsLayout() {
	if DebugSettings.UpdateTrace {
		fmt.Println("\tDebugSettings.UpdateTrace: NeedsLayout:", wb)
	}
	if wb.Scene != nil {
		wb.Scene.needsLayout = true
	}
}

// NeedsRebuild returns whether the [renderContext] indicates
// a full rebuild is needed. This is typically used to detect
// when the settings have been changed, such as when the color
// scheme or zoom is changed.
func (wb *WidgetBase) NeedsRebuild() bool {
	if wb.This == nil || wb.Scene == nil || wb.Scene.Stage == nil {
		return false
	}
	rc := wb.Scene.renderContext()
	if rc == nil {
		return false
	}
	return rc.rebuild
}

// layoutScene does a layout of the scene: Size, Position
func (sc *Scene) layoutScene() {
	if DebugSettings.LayoutTrace {
		fmt.Println("\n############################\nLayoutScene SizeUp start:", sc)
	}
	sc.SizeUp()
	sz := &sc.Geom.Size
	sz.Alloc.Total.SetPoint(sc.sceneGeom.Size)
	sz.setContentFromTotal(&sz.Alloc)
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
	sc.ApplyScenePos()
}

// layoutRenderScene does a layout and render of the tree:
// GetSize, DoLayout, Render.  Needed after Config.
func (sc *Scene) layoutRenderScene() {
	sc.layoutScene()
	sc.RenderWidget()
}

// doNeedsRender calls Render on tree from me for nodes
// with NeedsRender flags set
func (wb *WidgetBase) doNeedsRender() {
	if wb.This == nil {
		return
	}
	wb.WidgetWalkDown(func(w Widget, cwb *WidgetBase) bool {
		if cwb.needsRender {
			w.RenderWidget()
			return tree.Break // don't go any deeper
		}
		if ly := AsFrame(w); ly != nil {
			for d := math32.X; d <= math32.Y; d++ {
				if ly.HasScroll[d] && ly.scrolls[d] != nil {
					ly.scrolls[d].doNeedsRender()
				}
			}
		}
		return tree.Continue
	})
}

//////////////////////////////////////////////////////////////////
//		Scene

var sceneShowIters = 2

// doUpdate checks scene Needs flags to do whatever updating is required.
// returns false if already updating.
// This is the main update call made by the RenderWindow at FPS frequency.
func (sc *Scene) doUpdate() bool {
	if sc.updating {
		return false
	}
	sc.updating = true // prevent rendering
	defer func() { sc.updating = false }()

	rc := sc.renderContext()

	if sc.showIter < sceneShowIters {
		sc.needsLayout = true
		sc.showIter++
	}

	switch {
	case rc.rebuild:
		pr := profile.Start("rebuild")
		sc.doRebuild()
		sc.needsLayout = false
		sc.sceneNeedsRender = false
		sc.imageUpdated = true
		pr.End()
	case sc.lastRender.needsRestyle(rc):
		pr := profile.Start("restyle")
		sc.applyStyleScene()
		sc.layoutRenderScene()
		sc.needsLayout = false
		sc.sceneNeedsRender = false
		sc.imageUpdated = true
		sc.lastRender.saveRender(rc)
		pr.End()
	case sc.needsLayout:
		pr := profile.Start("layout")
		sc.layoutRenderScene()
		sc.needsLayout = false
		sc.sceneNeedsRender = false
		sc.imageUpdated = true
		pr.End()
	case sc.sceneNeedsRender:
		pr := profile.Start("render")
		sc.doNeedsRender()
		sc.sceneNeedsRender = false
		sc.imageUpdated = true
		pr.End()
	default:
		return false
	}

	if sc.showIter == sceneShowIters { // end of first pass
		sc.showIter++
		if !sc.prefSizing {
			sc.Events.activateStartFocus()
		}
	}

	return true
}

// updateScene calls UpdateTree on the Scene, which calls
// UpdateWidget on all widgets in the Scene.  This will set
// NeedsLayout to drive subsequent layout and render.
// This is a top-level call, typically only done when the window
// is first drawn or resized, or during rebuild,
// once the full sizing information is available.
func (sc *Scene) updateScene() {
	sc.updating = true // prevent rendering
	defer func() { sc.updating = false }()

	sc.UpdateTree()
}

// applyStyleScene calls ApplyStyle on all widgets in the Scene,
// This is needed whenever the window geometry, DPI,
// etc is updated, which affects styling.
func (sc *Scene) applyStyleScene() {
	sc.updating = true // prevent rendering
	defer func() { sc.updating = false }()

	sc.StyleTree()
	sc.needsLayout = true
}

// doRebuild does the full re-render and RenderContext Rebuild flag
// should be used by Widgets to rebuild things that are otherwise
// cached (e.g., Icon, TextCursor).
func (sc *Scene) doRebuild() {
	sc.updateScene()
	sc.applyStyleScene()
	sc.layoutRenderScene()
}

// prefSize computes the preferred size of the scene based on current contents.
// initSz is the initial size -- e.g., size of screen.
// Used for auto-sizing windows.
func (sc *Scene) prefSize(initSz image.Point) image.Point {
	sc.updating = true // prevent rendering
	defer func() { sc.updating = false }()

	sc.prefSizing = true
	sc.updateScene()
	sc.applyStyleScene()
	sc.layoutScene()
	sz := &sc.Geom.Size
	psz := sz.Actual.Total
	sc.prefSizing = false
	sc.showIter = 0
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
	if wb == nil || wb.This == nil {
		return false
	}
	wb.needsRender = false // done!
	if !wb.IsVisible() {   // checks deleted etc
		return false
	}
	if wb.Geom.TotalBBox.Empty() {
		if DebugSettings.RenderTrace {
			fmt.Printf("Render empty bbox: %v at %v\n", wb.Path(), wb.Geom.TotalBBox)
		}
		return false
	}
	wb.Styles.ComputeActualBackground(wb.parentActualBackground())
	pc := &wb.Scene.PaintContext
	if pc.State == nil || pc.Image == nil {
		return false
	}
	if len(pc.BoundsStack) == 0 && wb.Parent != nil {
		wb.firstRender = true
		// push our parent's bounds if we are the first to render
		pw := wb.parentWidget()
		pc.PushBoundsGeom(pw.Geom.TotalBBox, pw.Styles.Border.Radius.Dots())
	} else {
		wb.firstRender = false
	}
	pc.PushBoundsGeom(wb.Geom.TotalBBox, wb.Styles.Border.Radius.Dots())
	pc.Defaults() // start with default values
	if DebugSettings.RenderTrace {
		fmt.Printf("Render: %v at %v\n", wb.Path(), wb.Geom.TotalBBox)
	}
	return true
}

// PopBounds pops our bounding box bounds. This is the last step
// in Render implementations after rendering children.
func (wb *WidgetBase) PopBounds() {
	if wb == nil || wb.This == nil {
		return
	}
	pc := &wb.Scene.PaintContext

	isSelw := wb.Scene.selectedWidget == wb.This
	if wb.Scene.renderBBoxes || isSelw {
		pos := math32.Vector2FromPoint(wb.Geom.TotalBBox.Min)
		sz := math32.Vector2FromPoint(wb.Geom.TotalBBox.Size())
		// node: we won't necc. get a push prior to next update, so saving these.
		pcsw := pc.StrokeStyle.Width
		pcsc := pc.StrokeStyle.Color
		pcfc := pc.FillStyle.Color
		pcop := pc.FillStyle.Opacity
		pc.StrokeStyle.Width.Dot(1)
		pc.StrokeStyle.Color = colors.Uniform(hct.New(wb.Scene.renderBBoxHue, 100, 50))
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

		wb.Scene.renderBBoxHue += 10
		if wb.Scene.renderBBoxHue > 360 {
			rmdr := (int(wb.Scene.renderBBoxHue-360) + 1) % 9
			wb.Scene.renderBBoxHue = float32(rmdr)
		}
	}

	pc.PopBounds()
	if wb.firstRender {
		pc.PopBounds()
		wb.firstRender = false
	}
}

// Render is the method that widgets should implement to define their
// custom rendering steps. It should not typically be called outside of
// [Widget.RenderWidget], which also does other steps applicable
// for all widgets. The base [WidgetBase.Render] implementation
// renders the standard box model.
func (wb *WidgetBase) Render() {
	wb.RenderStandardBox()
}

// RenderWidget renders the widget and any parts and children that it has.
// It does not render if the widget is invisible. It calls Widget.Render]
// for widget-specific rendering.
func (wb *WidgetBase) RenderWidget() {
	if wb.PushBounds() {
		wb.This.(Widget).Render()
		wb.renderParts()
		wb.renderChildren()
		wb.PopBounds()
	}
}

func (wb *WidgetBase) renderParts() {
	if wb.Parts != nil {
		wb.Parts.RenderWidget()
	}
}

// renderChildren renders all of the widget's children.
func (wb *WidgetBase) renderChildren() {
	wb.WidgetKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
		kwi.RenderWidget()
		return tree.Continue
	})
}

////////////////////////////////////////////////////////////////////////////////
//  Standard Box Model rendering

// RenderBoxGeom renders a box with the given geometry.
func (wb *WidgetBase) RenderBoxGeom(pos math32.Vector2, sz math32.Vector2, bs styles.Border) {
	wb.Scene.PaintContext.DrawBorder(pos.X, pos.Y, sz.X, sz.Y, bs)
}

// RenderStandardBox renders the standard box model.
func (wb *WidgetBase) RenderStandardBox() {
	pos := wb.Geom.Pos.Total
	sz := wb.Geom.Size.Actual.Total
	wb.Scene.PaintContext.DrawStandardBox(&wb.Styles, pos, sz, wb.parentActualBackground())
}

//////////////////////////////////////////////////////////////////
//		Widget position functions

// PointToRelPos translates a point in Scene pixel coords
// into relative position within node, based on the Content BBox
func (wb *WidgetBase) PointToRelPos(pt image.Point) image.Point {
	return pt.Sub(wb.Geom.ContentBBox.Min)
}

// winBBox returns the RenderWindow based bounding box for the widget
// by adding the Scene position to the ScBBox
func (wb *WidgetBase) winBBox() image.Rectangle {
	bb := wb.Geom.TotalBBox
	if wb.Scene != nil {
		return bb.Add(wb.Scene.sceneGeom.Pos)
	}
	return bb
}

// winPos returns the RenderWindow based position within the
// bounding box of the widget, where the x, y coordinates
// are the proportion across the bounding box to use:
// 0 = left / top, 1 = right / bottom
func (wb *WidgetBase) winPos(x, y float32) image.Point {
	bb := wb.winBBox()
	sz := bb.Size()
	var pt image.Point
	pt.X = bb.Min.X + int(math32.Round(float32(sz.X)*x))
	pt.Y = bb.Min.Y + int(math32.Round(float32(sz.Y)*y))
	return pt
}

// Profiling and Benchmarking, controlled by settings app bar:

// ProfileToggle turns profiling on or off, which does both
// targeted profiling and global CPU and memory profiling.
func ProfileToggle() { //types:add
	if profile.Profiling {
		endTargetedProfile()
		endCPUMemoryProfile()
	} else {
		startTargetedProfile()
		startCPUMemoryProfile()
	}
}

// cpuProfileFile is the file created by [startCPUMemoryProfile],
// which needs to be stored so that it can be closed in [endCPUMemoryProfile].
var cpuProfileFile *os.File

// startCPUMemoryProfile starts the standard Go cpu and memory profiling.
func startCPUMemoryProfile() {
	fmt.Println("Starting standard cpu and memory profiling")
	f, err := os.Create("cpu.prof")
	if errors.Log(err) == nil {
		cpuProfileFile = f
		errors.Log(pprof.StartCPUProfile(f))
	}
}

// endCPUMemoryProfile ends the standard Go cpu and memory profiling.
func endCPUMemoryProfile() {
	fmt.Println("Ending standard cpu and memory profiling")
	pprof.StopCPUProfile()
	errors.Log(cpuProfileFile.Close())
	f, err := os.Create("mem.prof")
	if errors.Log(err) == nil {
		runtime.GC() // get up-to-date statistics
		errors.Log(pprof.WriteHeapProfile(f))
		errors.Log(f.Close())
	}
}

// startTargetedProfile starts targeted profiling using the [profile] package.
func startTargetedProfile() {
	fmt.Println("Starting targeted profiling")
	profile.Reset()
	profile.Profiling = true
}

// endTargetedProfile ends targeted profiling and prints the report.
func endTargetedProfile() {
	profile.Report(time.Millisecond)
	profile.Profiling = false
}
