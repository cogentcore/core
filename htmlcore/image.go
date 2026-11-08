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
	"sort"
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
	if err := html.Render(&b, ctx.svgForRender(ctx.Node)); err != nil {
		return
	}
	sv := New[core.SVG](ctx)
	errors.Log(sv.ReadString(b.String()))
	ctx.handleWidget(ctx.Node, sv)
}

// svgProperties are the CSS properties of SVG elements that the SVG
// reader gets from their style attribute, which are given to it from the
// cascade of the page (as the rules of its style sheets for SVG icons).
var svgProperties = map[string]bool{
	"fill": true, "fill-opacity": true, "fill-rule": true, "stroke": true, "stroke-width": true,
	"stroke-opacity": true, "stroke-linecap": true, "stroke-linejoin": true, "stroke-dasharray": true,
	"stroke-dashoffset": true, "stroke-miterlimit": true, "opacity": true, "display": true,
	"color": true, "clip-path": true, "clip-rule": true, "mask": true, "font-size": true,
	"font-family": true, "font-weight": true, "font-style": true, "text-anchor": true,
	"stop-color": true, "stop-opacity": true, "vector-effect": true, "transform": true,
	"visibility": true,
}

// svgRootSkip are the svgProperties that are not given for the svg element
// itself, which are those of its widget.
var svgRootSkip = map[string]bool{"display": true, "visibility": true, "opacity": true, "transform": true}

// svgForRender returns the given svg element as it is to be rendered: a
// copy of it whose elements have the values of the SVG properties of the
// cascade of the page (as of its style sheets, which the SVG reader does
// not know) in their style attribute, with the var() functions in their
// attributes (as in fill, as in the icons of BBC News) replaced by the
// values of the custom properties of their elements, and the attributes
// whose var() functions are invalid removed.
func (c *Context) svgForRender(n *html.Node) *html.Node {
	var clone func(n *html.Node) *html.Node
	clone = func(n *html.Node) *html.Node {
		cn := &html.Node{Type: n.Type, Data: n.Data, DataAtom: n.DataAtom, Namespace: n.Namespace}
		style := c.svgStyle(n, n == c.Node || n.Data == "svg" && n.Parent != nil && n.Parent.Namespace != "svg")
		for _, a := range n.Attr {
			if a.Key == "style" && n.Type == html.ElementNode {
				continue // in style
			}
			if _, ok := style[a.Key]; ok {
				continue // the style has precedence over presentation attributes
			}
			if strings.Contains(a.Val, "var(") {
				v, ok := c.substituteVars(n, a.Val, nil)
				if !ok {
					continue
				}
				a.Val = v
			}
			cn.Attr = append(cn.Attr, a)
		}
		if len(style) > 0 {
			props := make([]string, 0, len(style))
			for p := range style {
				props = append(props, p)
			}
			sort.Strings(props)
			var sb strings.Builder
			for i, p := range props {
				if i > 0 {
					sb.WriteString("; ")
				}
				sb.WriteString(p + ": " + style[p])
			}
			cn.Attr = append(cn.Attr, html.Attribute{Key: "style", Val: sb.String()})
		}
		for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
			cn.AppendChild(clone(ch))
		}
		return cn
	}
	return clone(n)
}

// svgStyle returns the values of the SVG properties of the given element
// in the cascade of the page (its style sheets and style attribute, with
// var() resolved, not the defaults of the browser), as the SVG reader
// understands them; for the root svg element, without those of its widget.
func (c *Context) svgStyle(n *html.Node, root bool) map[string]string {
	if n.Type != html.ElementNode {
		return nil
	}
	var style map[string]string
	for _, d := range c.declarations(n) { // in the order of the cascade: the last one wins
		if d.origin == originUserAgent || !svgProperties[d.Property] || (root && svgRootSkip[d.Property]) {
			continue
		}
		if style == nil {
			style = map[string]string{}
		}
		v := strings.TrimSpace(d.Value)
		if d.invalid || v == "" || strings.Contains(v, "var(") {
			delete(style, d.Property)
			continue
		}
		switch d.Property {
		case "visibility": // hidden as not displayed (the SVG reader has no visibility)
			if strings.EqualFold(v, "hidden") || strings.EqualFold(v, "collapse") {
				style["display"] = "none"
			}
			continue
		case "transform": // CSS units, which SVG transforms do not have
			v = cssTransformUnits.ReplaceAllString(v, "$1")
		}
		style[d.Property] = v
	}
	return style
}

// cssTransformUnits matches the units of the numbers of CSS transforms that
// are the user units of SVG transforms (px, and deg for angles).
var cssTransformUnits = regexp.MustCompile(`(-?[0-9.]+)(px|deg)\b`)

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
			if logGetError(err) != nil {
				return
			}
			img.AsyncLock()
			img.SetImage(im)
			img.Update()
			img.AsyncUnlock()
			return
		}
		resp, err := Get(ctx, src)
		if logGetError(err) != nil {
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
