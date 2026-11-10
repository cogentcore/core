// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package video implements a video player widget in Cogent Core.
package video

//go:generate core generate

import (
	"image"
	"sync"

	"cogentcore.org/core/core"
)

// Video represents a video playback widget without any controls.
// On desktop platforms, the playing itself is a [Playback], which other
// widgets can show too; on the web, the video is a video element of the
// browser, placed at the widget.
type Video struct { //types:add
	core.WidgetBase

	// Media is the video media.
	Media *Media

	// degrees of rotation to apply to the video images
	// 90 = left 90, -90 = right 90
	Rotation float32

	// Muted is whether the audio is not played.
	Muted bool

	// Loop is whether to play the video again from the start at its end.
	Loop bool

	// Controls is whether to show the playback controls of the browser,
	// on the web (where the video is a video element of the browser).
	Controls bool

	// playback is the current playing, if any.
	playback *Playback

	// mu protects frame and changed, which are set by the playback.
	mu sync.Mutex

	// frame is the last frame of the playback, to render.
	frame *image.RGBA

	// changed is whether frame has changed since it was last rendered.
	changed bool
}

func (v *Video) OnAdd() {
	v.WidgetBase.OnAdd()
	v.Scene.AddDirectRender(v)
}

func (v *Video) Destroy() {
	v.Stop()
	v.release()
	v.Scene.DeleteDirectRender(v)
	v.WidgetBase.Destroy()
}

// Open opens the video specified by the given filepath.
func (v *Video) Open(fpath string) error {
	media, err := OpenMedia(fpath, nil)
	if err != nil {
		return err
	}
	v.Media = media
	return nil
}
