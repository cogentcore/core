// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package video

import (
	_ "embed"
	"image"
	"syscall/js"
	"testing"
	"time"

	"cogentcore.org/core/system/composer"
	"cogentcore.org/core/tree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests run in a browser (see goscript_test.go), where a [Video] is
// a video element of the browser.

// shortMP4 is 1 second of 64x48 video at 10 frames per second, with audio.
//
//go:embed testdata/short.mp4
var shortMP4 []byte

// blobURL returns a blob URL of the given data, as a file that the browser
// can load (the tests have no server of files).
func blobURL(data []byte, typ string) string {
	arr := js.Global().Get("Uint8Array").New(len(data))
	js.CopyBytesToJS(arr, data)
	opts := js.Global().Get("Object").New()
	opts.Set("type", typ)
	blob := js.Global().Get("Blob").New([]any{arr}, opts)
	return js.Global().Get("URL").Call("createObjectURL", blob).String()
}

// newVideo returns a new video of the short test video.
func newVideo(t *testing.T) *Video {
	m, err := OpenMedia(blobURL(shortMP4, "video/mp4"), nil)
	require.NoError(t, err)
	v := tree.New[Video]()
	v.Media = m
	v.Muted = true // (browsers only play videos with audio after a user gesture)
	t.Cleanup(v.release)
	return v
}

// waitFor waits up to 5 seconds for the given condition, failing otherwise.
func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	for range 100 {
		if cond() {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func currentTime(v *Video) float64 {
	return elements[v].Get("currentTime").Float()
}

func TestVideoPlay(t *testing.T) {
	v := newVideo(t)
	require.NoError(t, v.Play(0, 0))
	waitFor(t, "playing", func() bool { return currentTime(v) > 0.2 })
	assert.True(t, v.IsPlaying())
	e := elements[v]
	assert.Equal(t, 64, e.Get("videoWidth").Int())
	assert.Equal(t, 48, e.Get("videoHeight").Int())
	assert.True(t, e.Get("muted").Bool())
	assert.False(t, e.Get("controls").Bool())

	// Pause, Resume and Stop.
	v.Pause()
	assert.False(t, v.IsPlaying())
	paused := currentTime(v)
	time.Sleep(200 * time.Millisecond)
	assert.Equal(t, paused, currentTime(v), "no time passes while paused")
	v.Resume()
	waitFor(t, "resumed", func() bool { return currentTime(v) > paused+0.1 })
	v.Stop()
	assert.False(t, v.IsPlaying())
	assert.Equal(t, 0.0, currentTime(v))

	// It ends after its second, unless it loops.
	require.NoError(t, v.Play(0, 0))
	waitFor(t, "ended", func() bool { return e.Get("ended").Bool() })
	assert.False(t, v.IsPlaying())
	v.Loop = true
	require.NoError(t, v.Play(0, 0))
	time.Sleep(1500 * time.Millisecond)
	assert.True(t, v.IsPlaying(), "playing again")
	assert.Less(t, currentTime(v), 0.9)
	v.Stop()
}

func TestVideoElementPlaced(t *testing.T) {
	// The render source places the video element at the content of the
	// widget, clipped to its visible part, with the composer.
	v := newVideo(t)
	v.Controls = true
	cw := composer.NewComposerWeb()
	rect := image.Rect(10, 20, 330, 260)
	draw := func(visible image.Rectangle) js.Value {
		cw.Start()
		cw.Add(&videoSource{v: v, rect: rect, visible: visible}, v.This)
		cw.Compose()
		return elements[v]
	}
	e := draw(rect)
	assert.True(t, e.Get("isConnected").Bool())
	st := e.Get("style")
	assert.Equal(t, "10px", st.Get("left").String())
	assert.Equal(t, "20px", st.Get("top").String())
	assert.Equal(t, "320px", st.Get("width").String())
	assert.Equal(t, "240px", st.Get("height").String())
	assert.Equal(t, "", st.Get("clipPath").String())
	assert.True(t, e.Get("controls").Bool())
	assert.Equal(t, "auto", st.Get("pointerEvents").String(), "the controls get the events")

	// Scrolled half out of its parent, and then out of view.
	e = draw(image.Rect(10, 140, 330, 260))
	// (inset(120px 0px 0px 0px), which Chrome shortens)
	assert.Equal(t, "inset(120px 0px 0px)", e.Get("style").Get("clipPath").String())
	e = draw(image.Rectangle{})
	assert.Equal(t, "inset(50%)", e.Get("style").Get("clipPath").String())
	assert.True(t, e.Get("isConnected").Bool(), "kept, so that it keeps playing")

	// It is removed with the video.
	v.release()
	assert.False(t, e.Get("isConnected").Bool())
	assert.NotContains(t, elements, v)
}

func TestVideoAutoplayBlocked(t *testing.T) {
	// A video with sound plays muted, with the controls of the browser to
	// turn it on, when the browser does not play it without a user gesture.
	v := newVideo(t)
	v.Muted = false
	require.NoError(t, v.Play(0, 0))
	waitFor(t, "playing", func() bool { return currentTime(v) > 0.1 })
	e := elements[v]
	if !e.Get("muted").Bool() {
		t.Skip("this browser plays videos with sound without a user gesture")
	}
	assert.True(t, e.Get("controls").Bool())
	assert.Equal(t, "auto", e.Get("style").Get("pointerEvents").String())

	// it stays muted as it is drawn again (the attributes are only set
	// when the fields of the video change)
	v.setAttributes(e)
	assert.True(t, e.Get("muted").Bool())
	assert.True(t, v.Muted)
}

func TestVideoSeekVolume(t *testing.T) {
	v := newVideo(t)
	require.NoError(t, v.Play(0, 0))
	waitFor(t, "the duration", func() bool { return v.Duration() > 0 })
	assert.InDelta(t, 1.0, v.Duration().Seconds(), 0.05)
	v.Pause()
	v.Seek(600 * time.Millisecond)
	waitFor(t, "the seek", func() bool { return v.CurrentTime() >= 590*time.Millisecond })
	assert.False(t, v.IsPlaying())
	v.SkipBy(-10 * time.Second)
	waitFor(t, "the skip", func() bool { return v.CurrentTime() == 0 })

	e := elements[v]
	v.SetVolume(0.5)
	assert.InDelta(t, 0.5, e.Get("volume").Float(), 0.01)
	v.SetMuted(false)
	assert.False(t, e.Get("muted").Bool())

	// a change of the volume by the user (with the controls) is kept
	e.Set("volume", 0.25)
	waitFor(t, "the volume change", func() bool { return v.Volume() < 0.3 })
	v.setAttributes(e)
	assert.InDelta(t, 0.25, e.Get("volume").Float(), 0.01)
}
