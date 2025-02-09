// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !js

package core

import (
	"fmt"
	"image"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/system"
	"golang.org/x/image/draw"
)

// doRender is the implementation of the main render pass on non-web platforms.
func (w *renderWindow) doRender(top *Stage) {
	drw := w.SystemWindow.Drawer()
	drw.Start()

	w.fillInsets()

	sm := &w.mains
	n := sm.stack.Len()

	// first, find the top-level window:
	winIndex := 0
	var winScene *Scene
	for i := n - 1; i >= 0; i-- {
		st := sm.stack.ValueByIndex(i)
		if st.Type == WindowStage {
			if DebugSettings.WindowRenderTrace {
				fmt.Println("GatherScenes: main Window:", st.String())
			}
			winScene = st.Scene
			winScene.RenderDraw(drw, draw.Src) // first window blits
			winIndex = i
			for _, w := range st.Scene.directRenders {
				w.RenderDraw(drw, draw.Over)
			}
			break
		}
	}

	// then add everyone above that
	for i := winIndex + 1; i < n; i++ {
		st := sm.stack.ValueByIndex(i)
		if st.Scrim && i == n-1 {
			clr := colors.Uniform(colors.ApplyOpacity(colors.ToUniform(colors.Scheme.Scrim), 0.5))
			drw.Copy(image.Point{}, clr, winScene.Geom.TotalBBox, draw.Over, system.Unchanged)
		}
		st.Scene.RenderDraw(drw, draw.Over)
		if DebugSettings.WindowRenderTrace {
			fmt.Println("GatherScenes: overlay Stage:", st.String())
		}
	}

	// then add the popups for the top main stage
	for _, kv := range top.popups.stack.Order {
		st := kv.Value
		st.Scene.RenderDraw(drw, draw.Over)
		if DebugSettings.WindowRenderTrace {
			fmt.Println("GatherScenes: popup:", st.String())
		}
	}
	top.Sprites.drawSprites(drw, winScene.SceneGeom.Pos)
	drw.End()
}
