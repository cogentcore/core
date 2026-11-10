// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !js

package video

import "image"

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

// release releases the resources of the video (none on desktop).
func (v *Video) release() {}
