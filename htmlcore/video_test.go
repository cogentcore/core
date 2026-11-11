// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !js

package htmlcore

import (
	"bytes"
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/math32"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/html"
)

// TestVideoBox tests that a video element is a [VideoBox] with an image of
// its poster, of the size of the poster, or of 300 by 150 without one, as a
// replaced element (with its aspect for a width), with the URL of its video.
func TestVideoBox(t *testing.T) {
	src := `<div style="width: 800px"><video id="a" src="a.mp4"><p>no video</p></video><br>` +
		`<video id="b" width="600"><source src="b.webm" type="video/webm"><source src="b.mp4"></video><br>` +
		`<video id="c" style="width: 100%"></video><br>` +
		`<video id="d" poster="` + pngDataURL(40, 20) + `" src="https://example.com/d.mp4"></video></div>`
	widgetsByID(t, src, func(ws map[string]*core.WidgetBase, dp float32) {
		for _, id := range []string{"a", "b", "c", "d"} {
			require.NotNil(t, ws[id], id)
			vb, ok := ws[id].This.(*VideoBox)
			require.True(t, ok, id)
			require.NotNil(t, vb.Image, id)
		}
		// (within the rounding and spacing of an img of the same size)
		size := func(id string) math32.Vector2 { return ws[id].Geom.Size.Actual.Total }
		assert.InDelta(t, 300*dp, size("a").X, 5)
		assert.InDelta(t, 150*dp, size("a").Y, 5)
		assert.Equal(t, 1, ws["a"].NumChildren(), "its children are not shown, only its image")
		// the aspect ratio of the video (the default one, until it is known)
		assert.InDelta(t, 600*dp, size("b").X, 5)
		assert.InDelta(t, 300*dp, size("b").Y, 5)
		assert.InDelta(t, 800*dp, size("c").X, 5)
		assert.InDelta(t, 400*dp, size("c").Y, 5)
		// the size of its poster, which its image fills
		assert.InDelta(t, 40*dp, size("d").X, 5)
		assert.InDelta(t, 20*dp, size("d").Y, 5)
		d := ws["d"].This.(*VideoBox)
		assert.Equal(t, d.Geom.Size.Actual.Content, d.Image.Geom.Size.Actual.Total)

		assert.Equal(t, "a.mp4", VideoSourceOf(ws["a"].This.(core.Widget)))
		assert.Equal(t, "b.webm", VideoSourceOf(ws["b"].This.(core.Widget)), "its first source")
		assert.Equal(t, "", VideoSourceOf(ws["c"].This.(core.Widget)))
		assert.Equal(t, "https://example.com/d.mp4", VideoSourceOf(ws["d"].This.(core.Widget)))
	})
}

func TestVideoSource(t *testing.T) {
	ctx := NewContext()
	ctx.PageURL = "https://example.com/dir/page.html"
	doc, err := html.Parse(bytes.NewBufferString(`<video><source src="v.mp4"></video><video src="/top.mp4"></video>`))
	require.NoError(t, err)
	var videos []*html.Node
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "video" {
			videos = append(videos, n)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	require.Len(t, videos, 2)
	assert.Equal(t, "https://example.com/dir/v.mp4", ctx.VideoSource(videos[0]))
	assert.Equal(t, "https://example.com/top.mp4", ctx.VideoSource(videos[1]))
}
