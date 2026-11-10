// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package video

import (
	"fmt"
	"syscall/js"
)

// On the web, a [Video] is a video element of the browser, which decodes
// and plays it (with its audio), placed over the canvas at the widget by
// its render source (see render_js.go).

// elements are the video elements of the videos. A video keeps its element
// while it exists, as the element has the state of the playing, and the
// composer removes the element of a source that is not drawn.
var elements = map[*Video]js.Value{}

// element returns the video element of the video, making it the first time,
// with the source of its media.
func (v *Video) element() js.Value {
	e, ok := elements[v]
	if !ok {
		e = js.Global().Get("document").Call("createElement", "video")
		e.Set("playsInline", true)
		e.Set("preload", "auto")
		e.Get("style").Set("objectFit", "contain")
		e.Call("addEventListener", "loadedmetadata", js.FuncOf(func(this js.Value, args []js.Value) any {
			go v.metadataLoaded(e.Get("videoWidth").Int(), e.Get("videoHeight").Int())
			return nil
		}))
		elements[v] = e
	}
	v.setAttributes(e)
	return e
}

// setAttributes sets the attributes of the video element from the video.
func (v *Video) setAttributes(e js.Value) {
	if v.Media != nil && e.Get("dataset").Get("src").String() != v.Media.Path {
		e.Get("dataset").Set("src", v.Media.Path)
		e.Set("src", v.Media.Path)
	}
	e.Set("muted", v.Muted)
	e.Set("loop", v.Loop)
	e.Set("controls", v.Controls)
	if e.Get("poster").String() != v.Poster {
		e.Set("poster", v.Poster)
	}
	// The canvas gets the events of the widget, unless the controls of
	// the browser are shown.
	pe := "none"
	if v.Controls {
		pe = "auto"
	}
	st := e.Get("style")
	st.Set("pointerEvents", pe)
	tr := ""
	if v.Rotation != 0 {
		tr = fmt.Sprintf("rotate(%gdeg)", -v.Rotation)
	}
	st.Set("transform", tr)
}

// metadataLoaded sets the size of the media once the browser knows it,
// and updates the video, whose size can depend on it (as in a web page).
func (v *Video) metadataLoaded(width, height int) {
	if v.Media == nil || !v.AsyncLockIfShown() {
		return
	}
	defer v.AsyncUnlock()
	v.Media.Width, v.Media.Height = width, height
	v.Update()
}

// release removes the video element of the video.
func (v *Video) release() {
	if e, ok := elements[v]; ok {
		e.Call("pause")
		e.Call("remove")
		delete(elements, v)
	}
}

// Play starts playing the video from the start. The size is that of the
// widget on the web, where the browser scales the video.
func (v *Video) Play(width, height float32) error {
	if v.Media == nil {
		return fmt.Errorf("video: no media to play")
	}
	e := v.element()
	e.Set("currentTime", 0)
	v.play(e)
	v.NeedsRender() // to place the element
	return nil
}

// play plays the video element, logging the error if the browser does
// not play it (as for its autoplay policy, without a user gesture, for
// a video that is not muted).
func (v *Video) play(e js.Value) {
	p := e.Call("play")
	if p.Truthy() && p.Get("catch").Truthy() {
		p.Call("catch", js.FuncOf(func(this js.Value, args []js.Value) any {
			if len(args) > 0 {
				fmt.Println("video: the browser did not play", v.Media.Path+":", args[0].Call("toString").String())
			}
			return nil
		}))
	}
}

// Pause pauses the playing, if it is playing.
func (v *Video) Pause() {
	if e, ok := elements[v]; ok {
		e.Call("pause")
	}
}

// Resume resumes the playing, if it is paused.
func (v *Video) Resume() {
	if e, ok := elements[v]; ok && e.Get("paused").Bool() {
		v.play(e)
	}
}

// Stop stops the playing, if any.
func (v *Video) Stop() {
	if e, ok := elements[v]; ok {
		e.Call("pause")
		e.Set("currentTime", 0)
	}
}

// IsPlaying returns whether the video is playing, and not paused.
func (v *Video) IsPlaying() bool {
	e, ok := elements[v]
	return ok && !e.Get("paused").Bool() && !e.Get("ended").Bool()
}
