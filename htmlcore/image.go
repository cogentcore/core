// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"bytes"
	"image"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/core"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles/units"
	"golang.org/x/net/html"
)

// handleSVG handles an svg element, which is rendered by an SVG widget.
func handleSVG(ctx *Context) {
	var b strings.Builder
	if err := html.Render(&b, ctx.svgWithVars(ctx.Node)); err != nil {
		return
	}
	sv := New[core.SVG](ctx)
	errors.Log(sv.ReadString(b.String()))
	ctx.handleWidget(ctx.Node, sv)
}

// svgWithVars returns the given svg element as it is to be rendered: a
// copy of it with the var() functions in its attributes (as in fill and
// style, as in the icons of BBC News) replaced by the values of the custom
// properties of its elements, which the SVG reader does not know, and the
// attributes whose var() functions are invalid removed. It returns the
// element itself if it has no var() functions.
func (c *Context) svgWithVars(n *html.Node) *html.Node {
	has := false
	var check func(n *html.Node)
	check = func(n *html.Node) {
		for _, a := range n.Attr {
			if strings.Contains(a.Val, "var(") {
				has = true
				return
			}
		}
		for ch := n.FirstChild; ch != nil && !has; ch = ch.NextSibling {
			check(ch)
		}
	}
	check(n)
	if !has {
		return n
	}
	var clone func(n *html.Node) *html.Node
	clone = func(n *html.Node) *html.Node {
		cn := &html.Node{Type: n.Type, Data: n.Data, DataAtom: n.DataAtom, Namespace: n.Namespace}
		for _, a := range n.Attr {
			if strings.Contains(a.Val, "var(") {
				if a.Key == "style" { // each of its declarations
					var ds []string
					for _, d := range strings.Split(a.Val, ";") {
						if v, ok := c.substituteVars(n, d, nil); ok {
							ds = append(ds, v)
						}
					}
					a.Val = strings.Join(ds, ";")
				} else {
					v, ok := c.substituteVars(n, a.Val, nil)
					if !ok {
						continue
					}
					a.Val = v
				}
			}
			cn.Attr = append(cn.Attr, a)
		}
		for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
			cn.AppendChild(clone(ch))
		}
		return cn
	}
	return clone(n)
}

func handleImage(ctx *Context, tag string) core.Widget {
	if strings.TrimSpace(GetAttr(ctx.Node, "src")) == "" {
		return nil // an image without a source is not shown
	}
	n := ctx.Node
	src := GetAttr(n, "src")
	alt := GetAttr(n, "alt")
	isSVG := strings.HasSuffix(src, ".svg") || strings.HasPrefix(src, "data:image/svg")
	var newWidget core.Widget
	var resp *http.Response
	var err error
	// imageError handles an error getting the image, which is only logged
	// at the debug level, as browsers just do not show images that are
	// missing or not supported.
	imageError := func(err error) bool {
		if err == nil {
			return false
		}
		slog.Debug("error loading image", "url", src, "err", err)
		return true
	}
	var svgData []byte
	if !ctx.DelayedImageLoad && isSVG {
		resp, err = Get(ctx, src)
		if imageError(err) {
			return nil
		}
		defer resp.Body.Close()
	}
	var im image.Image
	if !ctx.DelayedImageLoad && !isSVG && ctx.ImageCache == nil {
		// An image is an svg by its type (not only the extension of its URL).
		resp, err = Get(ctx, src)
		if imageError(err) {
			return nil
		}
		data, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if imageError(err) {
			return nil
		}
		if strings.Contains(resp.Header.Get("Content-Type"), "svg") || looksLikeSVG(data) {
			isSVG, svgData = true, data
		} else if im, _, err = imagex.Read(bytes.NewReader(data)); imageError(err) {
			return nil
		}
	} else if !ctx.DelayedImageLoad && !isSVG {
		im, err = readImage(ctx, src)
		if imageError(err) {
			return nil
		}
	}

	// Can be either image or svg.
	var img *core.Image
	var svg *core.SVG
	if isSVG {
		svg = New[core.SVG](ctx)
		svg.SetTooltip(alt)
		if !ctx.DelayedImageLoad {
			data, err := svgData, error(nil)
			if data == nil {
				data, err = io.ReadAll(resp.Body)
			}
			if errors.Log(err) == nil {
				errors.Log(svg.Read(bytes.NewReader(data)))
				if w, h, ok := svgSize(data); ok { // its intrinsic size
					svg.SetProperty("svg-size", math32.Vec2(w, h))
				}
			}
		}
		newWidget = svg
	} else {
		img = New[core.Image](ctx)
		img.SetTooltip(alt)
		img.SetProperty("src", src) // for generatehtml
		img.SetImage(im)
		newWidget = img
	}

	if !ctx.DelayedImageLoad {
		return newWidget
	}
	go func() {
		if svg == nil {
			im, err := readImage(ctx, src)
			if errors.Log(err) != nil {
				return
			}
			img.AsyncLock()
			img.SetImage(im)
			img.Update()
			img.AsyncUnlock()
			return
		}
		resp, err := Get(ctx, src)
		if errors.Log(err) != nil {
			return
		}
		defer resp.Body.Close()
		svg.AsyncLock()
		errors.Log(svg.Read(resp.Body))
		svg.Update()
		svg.AsyncUnlock()
	}()
	return newWidget
}

// readImage returns the decoded image at the given URL, using
// [Context.ImageCache] if it is set. A decoding error is logged
// and gives a nil image without an error, as the image widget is
// still made then.
func readImage(ctx *Context, src string) (image.Image, error) {
	cache := ctx.ImageCache
	key := ""
	if cache != nil {
		u, err := parseRelativeURL(src, ctx.PageURL)
		if err != nil {
			return nil, err
		}
		key = u.String()
		if im := cache.Get(key); im != nil {
			return im, nil
		}
	}
	resp, err := Get(ctx, src)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	im, _, err := imagex.Read(resp.Body)
	if err != nil {
		// browsers just do not show images that are not supported
		slog.Debug("error loading image", "url", src, "err", err)
		return nil, nil
	}
	if cache != nil {
		cache.Add(key, im)
	}
	return im, nil
}

// looksLikeSVG returns whether the given image data is an SVG document.
func looksLikeSVG(data []byte) bool {
	head := data[:min(len(data), 512)]
	return bytes.Contains(head, []byte("<svg")) || (bytes.HasPrefix(bytes.TrimSpace(head), []byte("<?xml")) && bytes.Contains(data[:min(len(data), 4096)], []byte("<svg")))
}

// svgRootTag matches the start tag of the root svg element.
var svgRootTag = regexp.MustCompile(`(?is)<svg\b[^>]*>`)

// svgAttr matches an attribute of a start tag.
var svgAttr = regexp.MustCompile(`(?is)\s(width|height|viewBox)\s*=\s*["']([^"']*)["']`)

// svgSize returns the intrinsic size of the given SVG document, in px,
// from the width and height of its root element, or the size of its view
// box for those that are not set, as for the size of an img of it.
func svgSize(data []byte) (w, h float32, ok bool) {
	env := defaultLengthEnv() // only absolute lengths are used
	tag := svgRootTag.Find(data)
	if tag == nil {
		return 0, 0, false
	}
	var vw, vh float32
	for _, m := range svgAttr.FindAllSubmatch(tag, -1) {
		v := string(m[2])
		switch strings.ToLower(string(m[1])) {
		case "width":
			if l, lok := env.length(v, units.UnitPw); lok && l.Unit == units.UnitDp {
				w = l.Value
			}
		case "height":
			if l, lok := env.length(v, units.UnitPh); lok && l.Unit == units.UnitDp {
				h = l.Value
			}
		case "viewbox":
			vb := strings.Fields(strings.ReplaceAll(v, ",", " "))
			if len(vb) == 4 {
				x, _ := strconv.ParseFloat(vb[2], 32)
				y, _ := strconv.ParseFloat(vb[3], 32)
				vw, vh = float32(x), float32(y)
			}
		}
	}
	switch {
	case w > 0 && h == 0 && vw > 0:
		h = w * vh / vw
	case h > 0 && w == 0 && vh > 0:
		w = h * vw / vh
	case w == 0 && h == 0:
		w, h = vw, vh
	}
	return w, h, w > 0 && h > 0
}
