// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"strings"

	"cogentcore.org/core/core"
	"golang.org/x/net/html"
)

// The video element (see handleVideo) is a replaced element: a box of the
// size of its video, or of its poster image, which is 300 by 150 without
// either, as in browsers. Its children (its source and track elements, and
// the content for browsers without video) are not shown.
//
// On the web (js), the box is a [video.Video], which is a video element of
// the browser, with its attributes (autoplay, muted, loop, controls and
// poster). Elsewhere, it is a [core.Image] of the poster (if any), with
// the URL of the video in its "video-src" property (see [VideoSourceOf]),
// as htmlcore does not decode video itself there (which needs FFmpeg):
// an app that plays videos (as a web browser) shows the frames of a
// [video.Playback] in the image, from a widget handler (see
// [Context.WidgetHandlers]).

// VideoDefaultWidth and VideoDefaultHeight are the size in CSS pixels of
// a video element without a width or height, before its size is known,
// as in browsers.
const (
	VideoDefaultWidth  = 300
	VideoDefaultHeight = 150
)

// VideoSource returns the absolute URL of the video of the given video
// element: its src, or else the src of its first source element with one,
// or "" for none.
func (c *Context) VideoSource(n *html.Node) string {
	src := strings.TrimSpace(GetAttr(n, "src"))
	for ch := n.FirstChild; src == "" && ch != nil; ch = ch.NextSibling {
		if ch.Type == html.ElementNode && ch.Data == "source" {
			src = strings.TrimSpace(GetAttr(ch, "src"))
		}
	}
	if src == "" {
		return ""
	}
	if u, err := ParseRelativeURL(src, c.PageURL); err == nil && c.PageURL != "" {
		return u.String()
	}
	return src
}

// VideoSourceOf returns the URL of the video of the given widget, if it is
// the image of a video element (see [Context.VideoSource]), or "".
func VideoSourceOf(w core.Widget) string {
	src, _ := w.AsWidget().Property("video-src").(string)
	return src
}
