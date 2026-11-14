// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"image"
	"strings"

	"cogentcore.org/core/core"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// The images of img elements whose sources change (as lazy loaders set
// them as they are shown) are set in their widgets, without reading
// anything again: the img elements are inline, so they were read again
// with the nearest elements that they are in that could be, which for the
// lazy images of nytimes.com, set at once, was its #app (4600 elements,
// for 0.9s).

// imageSwap is the new image of the widget of an img element whose source
// has changed (see [Context.imageChange]).
type imageSwap struct {
	n, img  *html.Node // the changed element, and its img
	w       *core.Image
	src     string
	density float32
	im      image.Image
}

// imageAttrs are the attributes of img and source elements that only
// choose their images.
var imageAttrs = map[string]bool{"src": true, "srcset": true, "sizes": true, "loading": true,
	"decoding": true, "fetchpriority": true, "media": true, "type": true}

// imageChange returns the new image of the img element whose image the
// change of the given element changes, if that is all that it changes:
// the attributes that choose the image (see [imageAttrs]) of an img, or of
// a source of its picture, with a widget of its own (see [handleImage]),
// that no style rule selects by those attributes. It does not get the
// image (see [imageSwap.load]).
func (c *Context) imageChange(n *html.Node, srcRules *int) (*imageSwap, bool) {
	if c.snaps == nil || c.DelayedImageLoad || n.Type != html.ElementNode || n.Namespace != "" {
		return nil, false
	}
	o, ok := c.snaps[n]
	if !ok || o.class != GetAttr(n, "class") || o.id != GetAttr(n, "id") || o.style != GetAttr(n, "style") {
		return nil, false
	}
	ov, nv := attrValues(o.others), attrValues(otherAttrs(n))
	for k, v := range ov {
		if w, ok := nv[k]; (!ok || w != v) && !imageAttrs[k] {
			return nil, false
		}
	}
	for k := range nv {
		if _, ok := ov[k]; !ok && !imageAttrs[k] {
			return nil, false
		}
	}
	img := n
	switch n.DataAtom {
	case atom.Img:
	case atom.Source:
		img = nil
		if p := n.Parent; p != nil && p.DataAtom == atom.Picture {
			for k := n.NextSibling; k != nil && img == nil; k = k.NextSibling {
				if k.Type == html.ElementNode && k.DataAtom == atom.Img {
					img = k
				}
			}
		}
		if img == nil {
			return nil, false
		}
	default:
		return nil, false
	}
	ws := c.elementWidgets[img]
	if len(ws) != 1 || ownNode(ws[0]) != img {
		return nil, false
	}
	w, ok := ws[0].(*core.Image)
	if !ok || parseFilter(c.declaration(img, "filter")) != nil {
		return nil, false
	}
	src, density := c.imageSource(img)
	if src == "" || strings.HasSuffix(src, ".svg") || strings.HasPrefix(src, "data:image/svg") {
		return nil, false // (no image, or an svg, which has another widget)
	}
	if *srcRules < 0 {
		*srcRules = 0
		for _, r := range c.rules {
			if strings.Contains(r.text, "[src") || strings.Contains(r.text, "[loading") || strings.Contains(r.text, "[sizes") {
				*srcRules = 1
				break
			}
		}
	}
	if *srcRules > 0 {
		return nil, false // (selected by them, which are not matched again)
	}
	return &imageSwap{n: n, img: img, w: w, src: src, density: density}, true
}

// load gets the image of the swap, and records the state of its element,
// as it has been styled for it.
func (sw *imageSwap) load(c *Context) {
	if w := sw.w; w.Image != nil {
		if s, _ := w.Property("src").(string); s == sw.src && w.Density == sw.density {
			sw.im = w.Image // (another attribute changed)
		}
	}
	if sw.im == nil {
		sw.im, _ = readImage(c, sw.src) // (none if it can not be gotten, as for an error)
	}
	if p := sw.n.Parent; p != nil {
		snapChildren(p, func(k *html.Node, s nodeSnap) {
			if k == sw.n {
				c.snaps[k] = s
			}
		})
	}
}

// apply sets the image of the swap in its widget, with the lock of the
// widgets of the page held.
func (sw *imageSwap) apply() {
	w := sw.w
	w.Density = sw.density
	w.SetProperty("src", sw.src)
	w.SetImage(sw.im)
	w.NeedsLayout()
}
