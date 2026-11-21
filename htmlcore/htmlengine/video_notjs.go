// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !js

package htmlengine

import (
	"image"
	"log/slog"
	"strings"

	"cogentcore.org/core/core"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/types"
)

// VideoBox is the box of a video element on desktop platforms (see
// video.go), of the size of its video (see [resolver.videoSize]), with an
// [core.Image] in it ([VideoBox.Image]) that shows its poster, and the
// frames of the video, which an app that plays videos shows in it. The
// app can add controls over it (as video.Controls), for which the box is
// positioned (relative), and is rendered as a whole when they change
// ([VideoBox.RenderGroup]), as they are partly transparent over the image.
type VideoBox struct {
	core.Frame

	// Image shows the poster of the video, and its frames (nil for Audio).
	Image *core.Image

	// Audio is whether it is the box of an audio element with controls,
	// a bar (see handleAudio).
	Audio bool
}

// (the type is registered when the package is initialized, as typegen
// does, not when the first one is made, as when a page is read in another
// goroutine, which would change the types while others read them)
var _ = types.AddType(&types.Type{Name: "cogentcore.org/core/htmlcore/htmlengine.VideoBox", IDName: "video-box",
	Doc: "VideoBox is the box of a video element on desktop platforms.", Embeds: []types.Field{{Name: "Frame"}}, Instance: &VideoBox{}})

func (vb *VideoBox) Init() {
	vb.Frame.Init()
	vb.Styler(func(s *styles.Style) {
		s.Position = styles.PositionRelative // (the containing block of its image)
		s.Padding.Zero()
		s.Gap.Zero()
	})
}

// RenderGroup returns true: the box is rendered as a whole when any widget
// in it needs to be rendered (see [VideoBox]).
func (vb *VideoBox) RenderGroup() bool { return true }

// SetVideoSize sets the size of the video, which is the size of the box,
// or of its aspect when only its width or its height is set by its CSS
// (as the size of a video once it is opened, which updates the box).
func (vb *VideoBox) SetVideoSize(width, height int) {
	vb.SetProperty("video-size", math32.Vec2(float32(width), float32(height)))
}

// handleAudio handles an audio element with controls: a [VideoBox] that
// is Audio, without an image, with the URL of the audio in its "video-src"
// property, for an app to play it with its controls (see video.go).
func handleAudio(ctx *Context) core.Widget {
	vb := New[VideoBox](ctx)
	vb.Audio = true
	vb.SetProperty("audio", true)
	vb.SetProperty("video-src", ctx.VideoSource(ctx.Node))
	vb.SetVideoSize(0, 0) // (sized as audio, see [resolver.audioSize])
	return vb
}

// handleVideo handles a video element: a [VideoBox] with an image of its
// poster, of the default size without one, with the URL of the video in
// its "video-src" property, for an app to play it (see video.go).
func handleVideo(ctx *Context) core.Widget {
	vb := New[VideoBox](ctx)
	vb.SetProperty("video-src", ctx.VideoSource(ctx.Node))
	vb.SetVideoSize(0, 0) // the default size
	img := core.NewImage(vb)
	vb.Image = img
	img.Styler(func(s *styles.Style) {
		// it fills the box, with the video contained in it, keeping its
		// aspect ratio, as in browsers
		s.Position = styles.PositionAbsolute
		s.Inset.Set(units.Zero())
		s.InsetSet.Set(true)
		s.ObjectFit = styles.FitContain
		s.Min.Zero()
	})
	img.NaturalDp = true // in CSS px (see core.Image.NaturalDp)
	poster := strings.TrimSpace(GetAttr(ctx.Node, "poster"))
	if poster == "" {
		return vb
	}
	load := func() image.Image {
		im, err := readImage(ctx, poster)
		if err != nil || im == nil {
			slog.Debug("error loading the poster of a video", "url", poster, "err", err)
			return nil
		}
		return im
	}
	setPoster := func(im image.Image) {
		img.SetImage(im)
		sz := im.Bounds().Size()
		vb.SetVideoSize(sz.X, sz.Y) // until the video is opened
	}
	if !ctx.DelayedImageLoad {
		if im := load(); im != nil {
			setPoster(im)
		}
		return vb
	}
	go func() {
		if im := load(); im != nil && vb.AsyncLockIfShown() {
			setPoster(im)
			vb.Update()
			vb.AsyncUnlock()
		}
	}()
	return vb
}
