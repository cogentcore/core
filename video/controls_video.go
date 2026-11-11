// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package video

import (
	"time"

	"cogentcore.org/core/tree"
)

// The controls of a [Video] are [Controls], over the bottom of the video.
// The region of the video in the scene is transparent (see [Video.Render]):
// on desktop platforms, the video is drawn by its render source after the
// scene, which draws the region of the controls of the scene again over it;
// on the web, the video element of the browser is under the canvas of the
// scene. The video is rendered as a whole when its controls change
// ([Video.RenderGroup]), so that the transparent bar is drawn over the
// cleared region, not over its previous rendering.

// RenderGroup returns whether the video is rendered as a whole when any
// widget in it needs to be rendered: with its controls.
func (v *Video) RenderGroup() bool {
	return v.Controls
}

// initControls adds the controls to the video if [Video.Controls].
func (v *Video) initControls() {
	v.Maker(func(p *tree.Plan) {
		if !v.Controls {
			v.controls = nil
			return
		}
		tree.AddAt(p, "controls", func(c *Controls) {
			v.controls = c
			c.SetPlayer(v, v)
		})
	})
}

// updatePosition shows the given current time in the controls.
func (v *Video) updatePosition(t time.Duration) {
	if c := v.controls; c != nil {
		c.SetTime(t)
	}
}
