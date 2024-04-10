// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlview

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
	"cogentcore.org/core/core"
	"cogentcore.org/core/errors"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/states"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/texteditor"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/views"
	"cogentcore.org/core/xio/images"
	"golang.org/x/net/html"
)

// ElementHandlers is a map of handler functions for each HTML element
// type (eg: "button", "input", "p"). It is empty by default, but can be
// used by anyone in need of behavior different than the default behavior
// defined in [HandleElement] (for example, for custom elements).
// If the handler for an element returns false, then the default behavior
// for an element is used.
var ElementHandlers = map[string]func(ctx *Context) bool{}

// New adds a new widget of the given type to the context parent.
// It automatically calls [Context.Config] on the resulting widget.
func New[T core.Widget](ctx *Context) T {
	parent := ctx.Parent()
	w := tree.New[T](parent)
	ctx.Config(w)
	return w
}

// NewValue adds a new [views.Value] with the given value to the
// context parent. It automatically calls [Context.Config] on
// the resulting value widget.
func NewValue(ctx *Context, val any) views.Value {
	parent := ctx.Parent()
	v := views.NewValue(parent, val)
	ctx.Config(v.AsWidget())
	return v
}

// HandleElement calls the handler in [ElementHandlers] associated with the current node
// using the given context. If there is no handler associated with it, it uses default
// hardcoded configuration code.
func HandleElement(ctx *Context) {
	tag := ctx.Node.Data
	h, ok := ElementHandlers[tag]
	if ok {
		if h(ctx) {
			return
		}
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
		// TODO(kai/htmlview): maybe handle preload
		if rel == "preload" {
			return
		}
		// TODO(kai/htmlview): support links other than stylesheets
		if rel != "stylesheet" {
			return
		}
		resp, err := Get(ctx, GetAttr(ctx.Node, "href"))
		if errors.Log(err) != nil {
			return
		}
		defer resp.Body.Close()
		b, err := io.ReadAll(resp.Body)
		if errors.Log(err) != nil {
			return
		}
		ctx.AddStyle(string(b))
	case "style":
		ctx.AddStyle(ExtractText(ctx))
	case "body", "main", "div", "section", "nav", "footer", "header", "ol", "ul":
		ctx.NewParent = New[*core.Frame](ctx)
	case "button":
		New[*core.Button](ctx).SetText(ExtractText(ctx))
	case "h1":
		HandleLabel(ctx).SetType(core.LabelHeadlineLarge)
	case "h2":
		HandleLabel(ctx).SetType(core.LabelHeadlineSmall)
	case "h3":
		HandleLabel(ctx).SetType(core.LabelTitleLarge)
	case "h4":
		HandleLabel(ctx).SetType(core.LabelTitleMedium)
	case "h5":
		HandleLabel(ctx).SetType(core.LabelTitleSmall)
	case "h6":
		HandleLabel(ctx).SetType(core.LabelLabelSmall)
	case "p":
		HandleLabel(ctx)
	case "pre":
		hasCode := ctx.Node.FirstChild != nil && ctx.Node.FirstChild.Data == "code"
		HandleLabel(ctx).Style(func(s *styles.Style) {
			s.Text.WhiteSpace = styles.WhiteSpacePreWrap
			if hasCode {
				s.Background = colors.C(colors.Scheme.SurfaceContainer)
				s.Border.Radius = styles.BorderRadiusMedium
			}
		})
	case "li":
		// if we have a p as our first or second child, which is typical
		// for markdown-generated HTML, we use it directly for data extraction
		// to prevent double elements and unnecessary line breaks.
		if ctx.Node.FirstChild != nil && ctx.Node.FirstChild.Data == "p" {
			ctx.Node = ctx.Node.FirstChild
		} else if ctx.Node.FirstChild != nil && ctx.Node.FirstChild.NextSibling != nil && ctx.Node.FirstChild.NextSibling.Data == "p" {
			ctx.Node = ctx.Node.FirstChild.NextSibling
		}

		label := HandleLabel(ctx)
		start := ""
		if pw, ok := label.Parent().(core.Widget); ok {
			switch pw.Property("tag") {
			case "ol":
				number := 0
				for _, k := range *pw.Children() {
					// we only consider labels for the number (frames may be
					// added for nested lists, interfering with the number)
					if _, ok := k.(*core.Label); ok {
						number++
					}
				}
				start = strconv.Itoa(number) + ". "
			case "ul":
				// TODO(kai/htmlview): have different bullets for different depths
				start = "• "
			}
		}
		label.SetText(start + label.Text)
	case "img":
		img := New[*core.Image](ctx)
		n := ctx.Node
		go func() {
			src := GetAttr(n, "src")
			resp, err := Get(ctx, src)
			if errors.Log(err) != nil {
				return
			}
			defer resp.Body.Close()
			if strings.Contains(resp.Header.Get("Content-Type"), "svg") {
				// TODO(kai/htmlview): support svg
			} else {
				im, _, err := images.Read(resp.Body)
				if err != nil {
					slog.Error("error loading image", "url", src, "err", err)
					return
				}
				img.SetImage(im)
				img.Update()
			}
		}()
	case "input":
		ityp := GetAttr(ctx.Node, "type")
		val := GetAttr(ctx.Node, "value")
		switch ityp {
		case "number":
			fval := float32(errors.Log1(strconv.ParseFloat(val, 32)))
			New[*core.Spinner](ctx).SetValue(fval)
		case "checkbox":
			New[*core.Switch](ctx).SetType(core.SwitchCheckbox).
				SetState(HasAttr(ctx.Node, "checked"), states.Checked)
		case "radio":
			New[*core.Switch](ctx).SetType(core.SwitchRadioButton).
				SetState(HasAttr(ctx.Node, "checked"), states.Checked)
		case "range":
			fval := float32(errors.Log1(strconv.ParseFloat(val, 32)))
			New[*core.Slider](ctx).SetValue(fval)
		case "button", "submit":
			New[*core.Button](ctx).SetText(val)
		case "color":
			NewValue(ctx, colors.Black).SetValue(val)
		case "datetime":
			NewValue(ctx, time.Now()).SetValue(val)
		case "file":
			NewValue(ctx, core.Filename("")).SetValue(val)
		default:
			New[*core.TextField](ctx).SetText(val)
		}
	case "textarea":
		buf := texteditor.NewBuffer()
		buf.SetText([]byte(ExtractText(ctx)))
		New[*texteditor.Editor](ctx).SetBuffer(buf)
	default:
		ctx.NewParent = ctx.Parent()
	}
}

// HandleLabel creates a new label from the given information, setting the text and
// the label click function so that URLs are opened according to [Context.OpenURL].
func HandleLabel(ctx *Context) *core.Label {
	lb := New[*core.Label](ctx).SetText(ExtractText(ctx))
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
func HandleLabelTag(ctx *Context) *core.Label {
	start, end := NodeString(ctx.Node)
	str := start + ExtractText(ctx) + end
	lb := New[*core.Label](ctx).SetText(str)
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
