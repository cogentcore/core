// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"time"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/profile"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/cam/hct"
	"cogentcore.org/core/events"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint/render"
	_ "cogentcore.org/core/paint/renderers" // installs default renderer
	"cogentcore.org/core/styles"
	"cogentcore.org/core/system"
	"cogentcore.org/core/tree"
)

// AsyncLock must be called before making any updates in a separate goroutine
// outside of the main configuration, rendering, and event handling structure.
// It must have a matching [WidgetBase.AsyncUnlock] after it.
//
// If the widget has been deleted, or if the [Scene] has been shown but the render
// context is not available, then this will block forever. Enable
// [DebugSettingsData.UpdateTrace] in [DebugSettings] to see when that happens.
// If the scene has not been shown yet and the render context is nil, it will wait
// until the scene is shown before trying again.
func (wb *WidgetBase) AsyncLock() {
	rc := wb.Scene.renderContext()
	if rc == nil {
		if wb.Scene.hasFlag(sceneHasShown) {
			// If the scene has been shown but there is no render context,
			// we are probably being deleted, so we just block forever.
			if DebugSettings.UpdateTrace {
				fmt.Println("AsyncLock: scene shown but no render context; blocking forever:", wb)
			}
			select {}
		}
		// Otherwise, if we haven't been shown yet, we just wait until we are
		// and then try again.
		if DebugSettings.UpdateTrace {
			fmt.Println("AsyncLock: waiting for scene to be shown:", wb)
		}
		onShow := make(chan struct{})
		wb.OnShow(func(e events.Event) {
			onShow <- struct{}{}
		})
		<-onShow
		wb.AsyncLock() // try again
		return
	}
	rc.Lock()
	if wb.This == nil {
		rc.Unlock()
		if DebugSettings.UpdateTrace {
			fmt.Println("AsyncLock: widget deleted; blocking forever:", wb)
		}
		select {}
	}
	wb.Scene.setFlag(true, sceneUpdating)
}

// AsyncUnlock must be called after making any updates in a separate goroutine
// outside of the main configuration, rendering, and event handling structure.
// It must have a matching [WidgetBase.AsyncLock] before it.
func (wb *WidgetBase) AsyncUnlock() {
	rc := wb.Scene.renderContext()
	if rc == nil {
		return
	}
	if wb.Scene != nil {
		wb.Scene.setFlag(false, sceneUpdating)
	}
	rc.Unlock()
}

// NeedsRender specifies that the widget needs to be rendered.
func (wb *WidgetBase) NeedsRender() {
	if DebugSettings.UpdateTrace {
		fmt.Println("\tDebugSettings.UpdateTrace: NeedsRender:", wb)
	}
	wb.setFlag(true, widgetNeedsRender)
	if wb.Scene != nil {
		wb.Scene.setFlag(true, sceneNeedsRender)
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
		wb.Scene.setFlag(true, sceneNeedsLayout)
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
	sz.Alloc.Total.SetPoint(sc.SceneGeom.Size)
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

func (sc *Scene) Render() {
	if TheApp.Platform() == system.Web {
		sc.Painter.Fill.Color = colors.Uniform(colors.Transparent)
		sc.Painter.Clear()
	}
	sc.RenderStandardBox()
}

// doNeedsRender calls Render on tree from me for nodes
// with NeedsRender flags set
func (wb *WidgetBase) doNeedsRender() {
	if wb.This == nil {
		return
	}
	wb.WidgetWalkDown(func(cw Widget, cwb *WidgetBase) bool {
		if cwb.hasFlag(widgetNeedsRender) {
			cw.RenderWidget()
			return tree.Break // don't go any deeper
		}
		if ly := AsFrame(cw); ly != nil {
			for d := math32.X; d <= math32.Y; d++ {
				if ly.HasScroll[d] && ly.Scrolls[d] != nil {
					ly.Scrolls[d].doNeedsRender()
				}
			}
		}
		return tree.Continue
	})
}

////////  Scene

var sceneShowIters = 2

// doUpdate checks scene Needs flags to do whatever updating is required.
// returns false if already updating.
// This is the main update call made by the RenderWindow at FPS frequency.
func (sc *Scene) doUpdate() bool {
	if sc.hasFlag(sceneUpdating) {
		return false
	}
	sc.setFlag(true, sceneUpdating) // prevent rendering
	defer func() { sc.setFlag(false, sceneUpdating) }()

	sc.runAnimations()
	rc := sc.renderContext()

	if sc.showIter < sceneShowIters {
		sc.setFlag(true, sceneNeedsLayout)
		sc.showIter++
	}

	switch {
	case rc.rebuild:
		// pr := profile.Start("rebuild")
		sc.doRebuild()
		sc.setFlag(false, sceneNeedsLayout, sceneNeedsRender)
		sc.setFlag(true, sceneImageUpdated)
		// pr.End()
	case sc.lastRender.needsRestyle(rc):
		// pr := profile.Start("restyle")
		sc.applyStyleScene()
		sc.layoutRenderScene()
		sc.setFlag(false, sceneNeedsLayout, sceneNeedsRender)
		sc.setFlag(true, sceneImageUpdated)
		sc.lastRender.saveRender(rc)
		// pr.End()
	case sc.hasFlag(sceneNeedsLayout):
		// pr := profile.Start("layout")
		sc.layoutRenderScene()
		sc.setFlag(false, sceneNeedsLayout, sceneNeedsRender)
		sc.setFlag(true, sceneImageUpdated)
		// pr.End()
	case sc.hasFlag(sceneNeedsRender):
		// pr := profile.Start("render")
		sc.doNeedsRender()
		sc.setFlag(false, sceneNeedsRender)
		sc.setFlag(true, sceneImageUpdated)
		// pr.End()
	default:
		return false
	}

	if sc.showIter == sceneShowIters { // end of first pass
		sc.showIter++ // just go 1 past the iters cutoff
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
	sc.setFlag(true, sceneUpdating) // prevent rendering
	defer func() { sc.setFlag(false, sceneUpdating) }()

	sc.UpdateTree()
}

// applyStyleScene calls ApplyStyle on all widgets in the Scene,
// This is needed whenever the window geometry, DPI,
// etc is updated, which affects styling.
func (sc *Scene) applyStyleScene() {
	sc.setFlag(true, sceneUpdating) // prevent rendering
	defer func() { sc.setFlag(false, sceneUpdating) }()

	sc.StyleTree()
	if sc.Painter.Paint != nil {
		sc.Painter.Paint.UnitContext = sc.Styles.UnitContext
	}
	sc.setFlag(true, sceneNeedsLayout)
}

// doRebuild does the full re-render and RenderContext Rebuild flag
// should be used by Widgets to rebuild things that are otherwise
// cached (e.g., Icon, TextCursor).
func (sc *Scene) doRebuild() {
	sc.Stage.Sprites.reset()
	sc.updateScene()
	sc.applyStyleScene()
	sc.layoutRenderScene()
}

// contentSize computes the size of the scene based on current content.
// initSz is the initial size, e.g., size of screen.
// Used for auto-sizing windows when created, and in [Scene.ResizeToContent].
func (sc *Scene) contentSize(initSz image.Point) image.Point {
	sc.setFlag(true, sceneUpdating) // prevent rendering
	defer func() { sc.setFlag(false, sceneUpdating) }()

	sc.setFlag(true, sceneContentSizing)
	sc.updateScene()
	sc.applyStyleScene()
	sc.layoutScene()
	sz := &sc.Geom.Size
	psz := sz.Actual.Total
	sc.setFlag(false, sceneContentSizing)
	sc.showIter = 0
	return psz.ToPointFloor()
}

//////// Widget local rendering

// StartRender starts the rendering process in the Painter, if the
// widget is visible, otherwise it returns false.
// It pushes our context and bounds onto the render stack.
// This must be called as the first step in [Widget.RenderWidget] implementations.
func (wb *WidgetBase) StartRender() bool {
	if wb == nil || wb.This == nil {
		return false
	}
	wb.setFlag(false, widgetNeedsRender) // done!
	if !wb.IsVisible() {
		return false
	}
	wb.Styles.ComputeActualBackground(wb.parentActualBackground())
	pc := &wb.Scene.Painter
	if pc.State == nil {
		return false
	}
	pc.PushContext(nil, render.NewBoundsRect(wb.Geom.TotalBBox, wb.Styles.Border.Radius.Dots()))
	pc.Paint.Defaults() // start with default style values
	if DebugSettings.RenderTrace {
		fmt.Printf("Render: %v at %v\n", wb.Path(), wb.Geom.TotalBBox)
	}
	return true
}

// EndRender is the last step in [Widget.RenderWidget] implementations after
// rendering children. It pops our state off of the render stack.
func (wb *WidgetBase) EndRender() {
	if wb == nil || wb.This == nil {
		return
	}
	pc := &wb.Scene.Painter

	isSelw := wb.Scene.selectedWidget == wb.This
	if wb.Scene.renderBBoxes || isSelw {
		pos := math32.FromPoint(wb.Geom.TotalBBox.Min)
		sz := math32.FromPoint(wb.Geom.TotalBBox.Size())
		// node: we won't necc. get a push prior to next update, so saving these.
		pcsw := pc.Stroke.Width
		pcsc := pc.Stroke.Color
		pcfc := pc.Fill.Color
		pcop := pc.Fill.Opacity
		pc.Stroke.Width.Dot(1)
		pc.Stroke.Color = colors.Uniform(hct.New(wb.Scene.renderBBoxHue, 100, 50))
		pc.Fill.Color = nil
		if isSelw {
			fc := pc.Stroke.Color
			pc.Fill.Color = fc
			pc.Fill.Opacity = 0.2
		}
		pc.Rectangle(pos.X, pos.Y, sz.X, sz.Y)
		pc.Draw()
		// restore
		pc.Fill.Opacity = pcop
		pc.Fill.Color = pcfc
		pc.Stroke.Width = pcsw
		pc.Stroke.Color = pcsc

		wb.Scene.renderBBoxHue += 10
		if wb.Scene.renderBBoxHue > 360 {
			rmdr := (int(wb.Scene.renderBBoxHue-360) + 1) % 9
			wb.Scene.renderBBoxHue = float32(rmdr)
		}
	}

	pc.PopContext()
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
	if wb.StartRender() {
		wb.This.(Widget).Render()
		wb.renderChildren()
		wb.renderParts()
		wb.EndRender()
	}
}

func (wb *WidgetBase) renderParts() {
	if wb.Parts != nil {
		wb.Parts.RenderWidget()
	}
}

// renderChildren renders all of the widget's children.
func (wb *WidgetBase) renderChildren() {
	wb.ForWidgetChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
		cw.RenderWidget()
		return tree.Continue
	})
}

////////  Defer

// Defer adds a function to [WidgetBase.Deferred] that will be called after the next
// [Scene] update/render, including on the initial Scene render. After the function
// is called, it is removed and not called again. In the function, sending events
// etc will work as expected.
func (wb *WidgetBase) Defer(fun func()) {
	wb.Deferred = append(wb.Deferred, fun)
	if wb.Scene != nil {
		wb.Scene.setFlag(true, sceneHasDeferred)
	}
}

// runDeferred runs deferred functions on all widgets in the scene.
func (sc *Scene) runDeferred() {
	sc.WidgetWalkDown(func(cw Widget, cwb *WidgetBase) bool {
		for _, f := range cwb.Deferred {
			f()
		}
		cwb.Deferred = nil
		return tree.Continue
	})
}

// DeferShown adds a [WidgetBase.Defer] function to call [WidgetBase.Shown]
// and activate [WidgetBase.StartFocus]. For example, this is called in [Tabs]
// and [Pages] when a tab/page is newly shown, so that elements can perform
// [WidgetBase.OnShow] updating as needed.
func (wb *WidgetBase) DeferShown() {
	wb.Defer(func() {
		wb.Shown()
	})
}

// Shown sends [events.Show] to all widgets from this one down. Also see
// [WidgetBase.DeferShown].
func (wb *WidgetBase) Shown() {
	wb.WidgetWalkDown(func(cw Widget, cwb *WidgetBase) bool {
		cwb.Send(events.Show)
		return tree.Continue
	})
	wb.Events().activateStartFocus()
}

////////  Standard Box Model rendering

// RenderBoxGeom renders a box with the given geometry.
func (wb *WidgetBase) RenderBoxGeom(pos math32.Vector2, sz math32.Vector2, bs styles.Border) {
	wb.Scene.Painter.Border(pos.X, pos.Y, sz.X, sz.Y, bs)
}

// RenderStandardBox renders the standard box model, using Actual size.
func (wb *WidgetBase) RenderStandardBox() {
	pos := wb.Geom.Pos.Total
	sz := wb.Geom.Size.Actual.Total
	wb.Scene.Painter.StandardBox(&wb.Styles, pos, sz, wb.parentActualBackground())
}

// RenderAllocBox renders the standard box model using Alloc size, instead of Actual.
func (wb *WidgetBase) RenderAllocBox() {
	pos := wb.Geom.Pos.Total
	sz := wb.Geom.Size.Alloc.Total
	wb.Scene.Painter.StandardBox(&wb.Styles, pos, sz, wb.parentActualBackground())
}

////////	Widget position functions

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
		return bb.Add(wb.Scene.SceneGeom.Pos)
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

//////// Profiling and Benchmarking, controlled by settings app bar

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

var (
	// cpuProfileDir is the directory where the profile started
	cpuProfileDir string

	// cpuProfileFile is the file created by [startCPUMemoryProfile],
	// which needs to be stored so that it can be closed in [endCPUMemoryProfile].
	cpuProfileFile *os.File
)

// startCPUMemoryProfile starts the standard Go cpu and memory profiling.
func startCPUMemoryProfile() {
	cpuProfileDir, _ = os.Getwd()
	cpufnm := filepath.Join(cpuProfileDir, "cpu.prof")
	fmt.Println("Starting standard cpu and memory profiling to:", cpufnm)
	f, err := os.Create(cpufnm)
	if errors.Log(err) == nil {
		cpuProfileFile = f
		errors.Log(pprof.StartCPUProfile(f))
	}
}

// endCPUMemoryProfile ends the standard Go cpu and memory profiling.
func endCPUMemoryProfile() {
	memfnm := filepath.Join(cpuProfileDir, "mem.prof")
	fmt.Println("Ending standard cpu and memory profiling to:", memfnm)
	pprof.StopCPUProfile()
	errors.Log(cpuProfileFile.Close())
	f, err := os.Create(memfnm)
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
