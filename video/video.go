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
// The playing itself is a [Playback], which other widgets can show too.
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
	v.Scene.DeleteDirectRender(v)
	v.WidgetBase.Destroy()
}

// CurrentFrame returns the current frame (for rendering),
// and whether it is unchanged since it was last returned.
func (v *Video) CurrentFrame() (*image.RGBA, bool) {
	if !v.IsVisible() {
		return nil, true
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	unchanged := !v.changed
	v.changed = false
	return v.frame, unchanged
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

// Play starts playing the video from the start at the specified size.
// Values of 0 indicate to use the inherent size of the video for that
// dimension (keeping its aspect ratio if only one is 0).
func (v *Video) Play(width, height float32) error {
	v.Stop()
	v.playback = &Playback{Media: v.Media, Width: int(width), Height: int(height), Muted: v.Muted, Loop: v.Loop,
		OnFrame: func(f Frame) {
			v.mu.Lock()
			v.frame, v.changed = f.Image, true
			v.mu.Unlock()
			if v.AsyncLockIfShown() {
				v.NeedsRender()
				v.AsyncUnlock()
			}
		}}
	return v.playback.Play()
}

// Pause pauses the playing, if it is playing.
func (v *Video) Pause() {
	if v.playback != nil {
		v.playback.Pause()
	}
}

// Resume resumes the playing, if it is paused.
func (v *Video) Resume() {
	if v.playback != nil {
		v.playback.Resume()
	}
}

// Stop stops the playing, if any.
func (v *Video) Stop() {
	if v.playback != nil {
		v.playback.Stop()
	}
}

// IsPlaying returns whether the video is playing, and not paused.
func (v *Video) IsPlaying() bool {
	return v.playback != nil && v.playback.IsPlaying()
}
