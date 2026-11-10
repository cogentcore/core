// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !js

package htmlcore

import (
	"image"
	"log/slog"
	"strings"

	"cogentcore.org/core/core"
	"cogentcore.org/core/styles"
)

// handleVideo handles a video element: an image of its poster, of the
// default size without one, with the URL of the video in its "video-src"
// property, for an app to play it (see video.go).
func handleVideo(ctx *Context) core.Widget {
	img := New[core.Image](ctx)
	// the video is contained in its box, keeping its aspect ratio, as in
	// browsers (before the styler of its CSS, which can set object-fit)
	img.Styler(func(s *styles.Style) { s.ObjectFit = styles.FitContain })
	img.NaturalDp = true // in CSS px (see core.Image.NaturalDp)
	img.SetProperty("video-src", ctx.VideoSource(ctx.Node))
	img.SetImage(image.NewRGBA(image.Rect(0, 0, VideoDefaultWidth, VideoDefaultHeight)))
	poster := strings.TrimSpace(GetAttr(ctx.Node, "poster"))
	if poster == "" {
		return img
	}
	load := func() image.Image {
		im, err := readImage(ctx, poster)
		if err != nil || im == nil {
			slog.Debug("error loading the poster of a video", "url", poster, "err", err)
			return nil
		}
		return im
	}
	if !ctx.DelayedImageLoad {
		if im := load(); im != nil {
			img.SetImage(im)
		}
		return img
	}
	go func() {
		if im := load(); im != nil && img.AsyncLockIfShown() {
			img.SetImage(im)
			img.Update()
			img.AsyncUnlock()
		}
	}()
	return img
}
