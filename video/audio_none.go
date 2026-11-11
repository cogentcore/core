// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js || novideo

package video

// On the web, the browser plays the audio of a video element (see
// [Video]), so a [Playback] has no audio, nor without video (novideo).

func audioAvailable() bool { return false }

type audio struct{}

func playAudio(samples <-chan [2]float64, paused bool) *audio { return &audio{} }

func (a *audio) setPaused(paused bool) {}
