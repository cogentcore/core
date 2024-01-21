// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package coredom

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/giv"
	"cogentcore.org/core/grows/images"
	"cogentcore.org/core/grr"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/ki"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/states"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/texteditor"
	"golang.org/x/net/html"
)

// ElementHandlers is a map of handler functions for each HTML element
// type (eg: "button", "input", "p"). It is empty by default, but can be
// used by anyone in need of behavior different than the default behavior
// defined in [HandleElement] (for example, for custom elements).
var ElementHandlers = map[string]func(ctx *Context){}

// New adds a new widget of the given type to the context parent.
// It automatically calls [Context.Config] on the resulting widget.
func New[T gi.Widget](ctx *Context) T {
	par := ctx.Parent()
	w := ki.New[T](par)
	ctx.Config(w)
	return w
}

// NewValue adds a new [giv.Value] with the given value to the
// context parent. It automatically calls [Context.Config] on
// the resulting value widget.
func NewValue(ctx *Context, val any) giv.Value {
	par := ctx.Parent()
	v := giv.NewValue(par, val)
	ctx.Config(v.AsWidget())
	return v
}

// HandleELement calls the handler in [ElementHandlers] associated with the current node
// using the given context. If there is no handler associated with it, it uses default
// hardcoded configuration code.
func HandleElement(ctx *Context) {
	tag := ctx.Node.Data
	h, ok := ElementHandlers[tag]
	if ok {
		h(ctx)
		return
	}

	if slices.Contains(TextTags, tag) {
		HandleLabelTag(ctx)
		return
	}

	switch tag {
	case "script", "title", "meta":
		// we don't render anything
	case "link":
		rel := GetAttr(ctx.Node, "rel")
		// TODO(kai/coredom): maybe handle preload
		if rel == "preload" {
			return
		}
		// TODO(kai/coredom): support links other than stylesheets
		if rel != "stylesheet" {
			return
		}
		resp, err := Get(ctx, GetAttr(ctx.Node, "href"))
		if grr.Log(err) != nil {
			return
		}
		defer resp.Body.Close()
		b, err := io.ReadAll(resp.Body)
		if grr.Log(err) != nil {
			return
		}
		ctx.AddStyle(string(b))
	case "style":
		ctx.AddStyle(ExtractText(ctx))
	case "body", "main", "div", "section", "nav", "footer", "header":
		f := New[*gi.Frame](ctx)
		ctx.NewParent = f
	case "button":
		New[*gi.Button](ctx).SetText(ExtractText(ctx))
	case "h1":
		HandleLabel(ctx).SetType(gi.LabelHeadlineLarge)
	case "h2":
		HandleLabel(ctx).SetType(gi.LabelHeadlineSmall)
	case "h3":
		HandleLabel(ctx).SetType(gi.LabelTitleLarge)
	case "h4":
		HandleLabel(ctx).SetType(gi.LabelTitleMedium)
	case "h5":
		HandleLabel(ctx).SetType(gi.LabelTitleSmall)
	case "h6":
		HandleLabel(ctx).SetType(gi.LabelLabelSmall)
	case "p":
		HandleLabel(ctx)
	case "pre":
		HandleLabel(ctx).Style(func(s *styles.Style) {
			s.Text.WhiteSpace = styles.WhiteSpacePre
		})
	case "ol", "ul":
		// if we are already in a treeview, we just return in the last item in it
		// (which is the list item we are contained in), which fixes the associativity
		// of nested list items and prevents the created of duplicated tree view items.
		if ptv, ok := ctx.Parent().(*giv.TreeView); ok {
			w := ki.LastChild(ptv).(gi.Widget)
			ctx.NewParent = w
			return
		}
		tv := New[*giv.TreeView](ctx).SetText("").SetIcon(icons.None)
		ctx.NewParent = tv
		return
	case "li":
		ntv := New[*giv.TreeView](ctx)
		ftxt := ""
		if ptv := giv.AsTreeView(ctx.Parent()); ptv != nil {
			if ptv.Prop("tag") == "ol" {
				ftxt = strconv.Itoa(ntv.IndexInParent()+1) + ". " // start at 1
			} else {
				// TODO(kai/coredom): have different bullets for different depths
				ftxt = "• "
			}
		}

		// if we have a p as our first or second child, which is typical
		// for markdown-generated HTML, we use it directly for data extraction
		// to prevent double elements and unnecessary line breaks.
		if ctx.Node.FirstChild != nil && ctx.Node.FirstChild.Data == "p" {
			ctx.Node = ctx.Node.FirstChild
		} else if ctx.Node.FirstChild != nil && ctx.Node.FirstChild.NextSibling != nil && ctx.Node.FirstChild.NextSibling.Data == "p" {
			ctx.Node = ctx.Node.FirstChild.NextSibling
		}

		etxt := ExtractText(ctx)
		ntv.SetText(ftxt + etxt)
		ntv.OnWidgetAdded(func(w gi.Widget) {
			switch w := w.(type) {
			case *gi.Label:
				if w.PathFrom(ntv) == "parts/label" {
					w.HandleLabelClick(func(tl *paint.TextLink) {
						ctx.OpenURL(tl.URL)
					})
				}
			}
		})
	case "img":
		img := New[*gi.Image](ctx)
		n := ctx.Node
		go func() {
			src := GetAttr(n, "src")
			resp, err := Get(ctx, src)
			if grr.Log(err) != nil {
				return
			}
			defer resp.Body.Close()
			if strings.Contains(resp.Header.Get("Content-Type"), "svg") {
				// TODO(kai/coredom): support svg
			} else {
				im, _, err := images.Read(resp.Body)
				if err != nil {
					slog.Error("error loading image", "url", src, "err", err)
					return
				}
				img.Filename = gi.Filename(src)
				img.SetImage(im)
				img.Update()
			}
		}()
	case "input":
		ityp := GetAttr(ctx.Node, "type")
		val := GetAttr(ctx.Node, "value")
		switch ityp {
		case "number":
			fval := float32(grr.Log1(strconv.ParseFloat(val, 32)))
			New[*gi.Spinner](ctx).SetValue(fval)
		case "checkbox":
			New[*gi.Switch](ctx).SetType(gi.SwitchCheckbox).
				SetState(HasAttr(ctx.Node, "checked"), states.Checked)
		case "radio":
			New[*gi.Switch](ctx).SetType(gi.SwitchRadioButton).
				SetState(HasAttr(ctx.Node, "checked"), states.Checked)
		case "range":
			fval := float32(grr.Log1(strconv.ParseFloat(val, 32)))
			New[*gi.Slider](ctx).SetValue(fval)
		case "button", "submit":
			New[*gi.Button](ctx).SetText(val)
		case "color":
			NewValue(ctx, colors.Black).SetValue(val)
		case "datetime":
			NewValue(ctx, time.Now()).SetValue(val)
		case "file":
			NewValue(ctx, gi.Filename("")).SetValue(val)
		default:
			New[*gi.TextField](ctx).SetText(val)
		}
	case "textarea":
		buf := texteditor.NewBuf()
		buf.SetText([]byte(ExtractText(ctx)))
		New[*texteditor.Editor](ctx).SetBuf(buf)
	default:
		ctx.NewParent = ctx.Parent()
	}
}

// HandleLabel creates a new label from the given information, setting the text and
// the label click function so that URLs are opened according to [Context.OpenURL].
func HandleLabel(ctx *Context) *gi.Label {
	lb := New[*gi.Label](ctx).SetText(ExtractText(ctx))
	lb.HandleLabelClick(func(tl *paint.TextLink) {
		ctx.OpenURL(tl.URL)
	})
	return lb
}

// HandleLabelTag creates a new label from the given information, setting the text and
// the label click function so that URLs are opened according to [Context.OpenURL]. Also,
// it wraps the label text with the [NodeString] of the given node, meaning that it
// should be used for standalone elements that are meant to only exist in labels
// (eg: a, span, b, code, etc).
func HandleLabelTag(ctx *Context) *gi.Label {
	start, end := NodeString(ctx.Node)
	str := start + ExtractText(ctx) + end
	lb := New[*gi.Label](ctx).SetText(str)
	lb.HandleLabelClick(func(tl *paint.TextLink) {
		ctx.OpenURL(tl.URL)
	})
	return lb
}

// GetAttr gets the given attribute from the given node, returning ""
// if the attribute is not found.
func GetAttr(n *html.Node, attr string) string {
	res := ""
	for _, a := range n.Attr {
		if a.Key == attr {
			res = a.Val
		}
	}
	return res
}

// HasAttr returns whether the given node has the given attribute defined.
func HasAttr(n *html.Node, attr string) bool {
	return slices.ContainsFunc(n.Attr, func(a html.Attribute) bool {
		return a.Key == attr
	})
}

// Get is a helper function that calls [http.Get] with the given URL, parsed
// relative to the page URL of the given context. It also checks the status
// code of the response and closes the response body and returns an error if
// it is not [http.StatusOK]. If the error is nil, then the response body is
// not closed and must be closed by the caller.
func Get(ctx *Context, url string) (*http.Response, error) {
	u, err := ParseRelativeURL(url, ctx.PageURL)
	if err != nil {
		return nil, err
	}
	resp, err := http.Get(u.String())
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return resp, fmt.Errorf("got error status %q (code %d)", resp.Status, resp.StatusCode)
	}
	return resp, nil
}
