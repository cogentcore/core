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

	// restyle is whether its class or style changed too, for which it is
	// styled again (see Context.updateDecls)
	restyle bool
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
	if !ok || o.id != GetAttr(n, "id") {
		return nil, false
	}
	// (its class and style may change too, as lazy loaders mark the images
	// that they have loaded, for which it is restyled; see restyle)
	restyle := o.class != GetAttr(n, "class") || o.style != GetAttr(n, "style")
	ov, nv := attrValues(o.others), attrValues(otherAttrs(n))
	for k, v := range ov {
		if w, ok := nv[k]; (!ok || w != v) && !imageAttrs[k] && !c.inertAttr(k) {
			return nil, false
		}
	}
	for k := range nv {
		if _, ok := ov[k]; !ok && !imageAttrs[k] && !c.inertAttr(k) {
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
	return &imageSwap{n: n, img: img, w: w, src: src, density: density, restyle: restyle && n == img}, !restyle || n == img
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
	if !sw.restyle {
		c.resnap(sw.n) // (as updateDecls does for those that it styles again)
	}
}

// resnap records the state of the given element (see [nodeSnap]), as its
// declarations are for it.
func (c *Context) resnap(n *html.Node) {
	if p := n.Parent; p != nil {
		snapChildren(p, func(k *html.Node, s nodeSnap) {
			if k == n {
				c.snaps[k] = s
			}
		})
	}
}

// inertAttr returns whether a change of the given attribute changes nothing
// that is rendered: a custom one, with a hyphen (as data-*, and the
// pinger-seen that the analytics of nytimes.com add to the images that they
// have seen, for which they were read again; not those of the context, as
// for :hover), that no style rule selects elements by.
func (c *Context) inertAttr(name string) bool {
	if !strings.Contains(name, "-") || strings.HasPrefix(name, "data-htmlcore-") || strings.Contains(name, ":") ||
		name == "http-equiv" || name == "accept-charset" {
		return false
	}
	if n := len(c.rules) + len(c.pseudoRules); c.attrsSelected == nil || c.attrsRules != n {
		c.attrsSelected, c.attrsRules = map[string]bool{}, n
	}
	sel, ok := c.attrsSelected[name]
	if !ok {
		for _, r := range c.rules {
			if sel = hasAttrSelector(r.text, name); sel {
				break
			}
		}
		for _, r := range c.pseudoRules {
			if sel = sel || hasAttrSelector(r.text, name); sel {
				break
			}
		}
		c.attrsSelected[name] = sel
	}
	return !sel
}

// inertChange returns whether the given element, which has widgets of its
// own, has changed only in its attributes whose changes change nothing that
// is rendered (see [Context.inertAttr]). (The attributes of the elements
// without widgets, in runs of text, are in the markup of the text.)
func (c *Context) inertChange(n *html.Node) bool {
	if c.snaps == nil || n.Type != html.ElementNode {
		return false
	}
	if ws := c.elementWidgets[n]; len(ws) == 0 || ownNode(ws[0]) != n {
		return false
	}
	o, ok := c.snaps[n]
	if !ok || o.class != GetAttr(n, "class") || o.id != GetAttr(n, "id") || o.style != GetAttr(n, "style") {
		return false
	}
	others := otherAttrs(n)
	if others == o.others {
		return false // (a change of its states, as for :hover)
	}
	ov, nv := attrValues(o.others), attrValues(others)
	for k, v := range ov {
		if w, ok := nv[k]; (!ok || w != v) && !c.inertAttr(k) {
			return false
		}
	}
	for k := range nv {
		if _, ok := ov[k]; !ok && !c.inertAttr(k) {
			return false
		}
	}
	return true
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
