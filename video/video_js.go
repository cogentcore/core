// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package video

import (
	"fmt"
	"math"
	"syscall/js"
	"time"
)

// On the web, a [Video] is a video element of the browser, which decodes
// and plays it (with its audio), placed over the canvas at the widget by
// its render source (see render_js.go).

// elements are the video elements of the videos. A video keeps its element
// while it exists, as the element has the state of the playing, and the
// composer removes the element of a source that is not drawn.
var elements = map[*Video]js.Value{}

// applied are the attributes last set on the elements of the videos from
// their fields, which are only set again when the fields change, so that
// the changes of the user (as muting with the controls of the browser)
// are kept.
var applied = map[*Video]attributes{}

// attributes are the attributes of a video element from its [Video].
type attributes struct {
	muted, loop, controls bool
	volume                float32
}

// controls are the controls of a video, which are those of the browser
// on the web (see [Video.Controls]).
type controls struct{}

// initControls does nothing on the web, where the controls are those of
// the browser.
func (v *Video) initControls() {}

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
			go v.metadataLoaded(e.Get("videoWidth").Int(), e.Get("videoHeight").Int(), e.Get("duration").Float())
			return nil
		}))
		// the changes of the volume by the user (with the controls)
		e.Call("addEventListener", "volumechange", js.FuncOf(func(this js.Value, args []js.Value) any {
			v.Muted, v.volume = e.Get("muted").Bool(), float32(e.Get("volume").Float())
			applied[v] = v.attributes()
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
	if a := v.attributes(); a != applied[v] {
		applied[v] = a
		e.Set("muted", a.muted)
		e.Set("loop", a.loop)
		e.Set("controls", a.controls)
		e.Set("volume", a.volume)
	}
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
func (v *Video) metadataLoaded(width, height int, duration float64) {
	if v.Media == nil {
		return
	}
	// (set even before the video is shown, which then styles it with it;
	// the web is single-threaded)
	v.Media.Width, v.Media.Height = width, height
	if duration > 0 && !math.IsInf(duration, 0) {
		v.Media.Duration = time.Duration(duration * float64(time.Second))
	}
	if v.AsyncLockIfShown() {
		v.Update()
		v.AsyncUnlock()
	}
}

// attributes returns the attributes of the video element from the video.
func (v *Video) attributes() attributes {
	return attributes{muted: v.Muted, loop: v.Loop, controls: v.Controls, volume: v.Volume()}
}

// release removes the video element of the video.
func (v *Video) release() {
	if e, ok := elements[v]; ok {
		e.Call("pause")
		e.Call("remove")
		delete(elements, v)
		delete(applied, v)
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

// play plays the video element. Browsers only play a video with sound
// after a user gesture (their autoplay policy), so if the browser does not
// play it for that (NotAllowedError), it is played muted, with the controls
// of the browser shown, with which the user can turn the sound on.
func (v *Video) play(e js.Value) {
	p := e.Call("play")
	if !p.Truthy() || !p.Get("catch").Truthy() {
		return
	}
	p.Call("catch", js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) == 0 {
			return nil
		}
		if args[0].Get("name").String() == "NotAllowedError" && !e.Get("muted").Bool() {
			v.Muted, v.Controls = true, true
			v.setAttributes(e)
			e.Call("play")
			return nil
		}
		fmt.Println("video: the browser did not play", v.Media.Path+":", args[0].Call("toString").String())
		return nil
	}))
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

// isPaused returns whether the video is paused (not ended).
func (v *Video) isPaused() bool {
	e, ok := elements[v]
	return ok && e.Get("paused").Bool() && !e.Get("ended").Bool() && e.Get("currentTime").Float() > 0
}

// Seek plays the video from the given time, staying paused if it is.
func (v *Video) Seek(t time.Duration) {
	if e, ok := elements[v]; ok {
		e.Set("currentTime", max(t, 0).Seconds())
	}
}

// CurrentTime returns the current time of the video.
func (v *Video) CurrentTime() time.Duration {
	if e, ok := elements[v]; ok {
		return time.Duration(e.Get("currentTime").Float() * float64(time.Second))
	}
	return 0
}

// Duration returns the duration of the video, once the browser knows it.
func (v *Video) Duration() time.Duration {
	if v.Media == nil {
		return 0
	}
	return v.Media.Duration
}

// SetMuted sets whether the audio is not heard.
func (v *Video) SetMuted(muted bool) {
	v.Muted = muted
	if e, ok := elements[v]; ok {
		v.setAttributes(e)
	}
}

// Volume returns the volume of the audio, from 0 to 1.
func (v *Video) Volume() float32 {
	if v.volume == 0 {
		return 1
	}
	return v.volume
}

// SetVolume sets the volume of the audio, from 0 to 1.
func (v *Video) SetVolume(volume float32) {
	v.volume = max(min(volume, 1), 1e-6)
	if e, ok := elements[v]; ok {
		v.setAttributes(e)
	}
}
