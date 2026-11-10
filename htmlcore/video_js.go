// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js

package htmlcore

import (
	"strings"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/video"
)

// handleVideo handles a video element: a [video.Video], which is a video
// element of the browser on the web, with the attributes of the element
// (see video.go).
func handleVideo(ctx *Context) core.Widget {
	n := ctx.Node
	v := New[video.Video](ctx)
	if src := ctx.VideoSource(n); src != "" {
		v.Media, _ = video.OpenMedia(src, nil) // (no error on the web)
	}
	v.Muted = HasAttr(n, "muted")
	v.Loop = HasAttr(n, "loop")
	v.Controls = HasAttr(n, "controls")
	if poster := strings.TrimSpace(GetAttr(n, "poster")); poster != "" {
		if u, err := ParseRelativeURL(poster, ctx.PageURL); err == nil && ctx.PageURL != "" {
			poster = u.String()
		}
		v.Poster = poster
	}
	// its size is that of its video once the browser knows it (see
	// [resolver.videoSize]), which updates the video then
	v.SetProperty("video-size", math32.Vector2{})
	v.Styler(func(s *styles.Style) {
		if v.Media != nil {
			v.SetProperty("video-size", math32.Vec2(float32(v.Media.Width), float32(v.Media.Height)))
		}
	})
	if v.Media != nil && HasAttr(n, "autoplay") {
		errors.Log(v.Play(0, 0))
	}
	return v
}
