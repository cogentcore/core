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

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/texteditor"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/views"
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
	ctx.Config(w) // TODO(config): better htmlview structure with new config paradigm?
	return w
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
		HandleTextTag(ctx)
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
		if tag == "body" {
			ctx.NewParent.AsWidget().Styler(func(s *styles.Style) {
				s.Grow.Set(1, 1)
			})
		}
	case "button":
		New[*core.Button](ctx).SetText(ExtractText(ctx))
	case "h1":
		HandleText(ctx).SetType(core.TextHeadlineLarge)
	case "h2":
		HandleText(ctx).SetType(core.TextHeadlineSmall)
	case "h3":
		HandleText(ctx).SetType(core.TextTitleLarge)
	case "h4":
		HandleText(ctx).SetType(core.TextTitleMedium)
	case "h5":
		HandleText(ctx).SetType(core.TextTitleSmall)
	case "h6":
		HandleText(ctx).SetType(core.TextLabelSmall)
	case "p":
		HandleText(ctx)
	case "pre":
		hasCode := ctx.Node.FirstChild != nil && ctx.Node.FirstChild.Data == "code"
		HandleText(ctx).Styler(func(s *styles.Style) {
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

		text := HandleText(ctx)
		start := ""
		if pw, ok := text.Parent.(core.Widget); ok {
			switch pw.AsTree().Property("tag") {
			case "ol":
				number := 0
				for _, k := range pw.AsTree().Children {
					// we only consider text for the number (frames may be
					// added for nested lists, interfering with the number)
					if _, ok := k.(*core.Text); ok {
						number++
					}
				}
				start = strconv.Itoa(number) + ". "
			case "ul":
				// TODO(kai/htmlview): have different bullets for different depths
				start = "• "
			}
		}
		text.SetText(start + text.Text)
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
				im, _, err := imagex.Read(resp.Body)
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
			core.Bind(val, New[*views.ColorButton](ctx))
		case "datetime":
			core.Bind(val, New[*views.TimeInput](ctx))
		case "file":
			core.Bind(val, New[*views.FileButton](ctx))
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

// HandleText creates a new [core.Text] from the given information, setting the text and
// the text click function so that URLs are opened according to [Context.OpenURL].
func HandleText(ctx *Context) *core.Text {
	lb := New[*core.Text](ctx).SetText(ExtractText(ctx))
	lb.HandleTextClick(func(tl *paint.TextLink) {
		ctx.OpenURL(tl.URL)
	})
	return lb
}

// HandleTextTag creates a new [core.Text] from the given information, setting the text and
// the text click function so that URLs are opened according to [Context.OpenURL]. Also,
// it wraps the text with the [NodeString] of the given node, meaning that it
// should be used for standalone elements that are meant to only exist in text
// (eg: a, span, b, code, etc).
func HandleTextTag(ctx *Context) *core.Text {
	start, end := NodeString(ctx.Node)
	str := start + ExtractText(ctx) + end
	lb := New[*core.Text](ctx).SetText(str)
	lb.HandleTextClick(func(tl *paint.TextLink) {
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
