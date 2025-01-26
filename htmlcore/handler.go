// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

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
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/texteditor"
	"cogentcore.org/core/tree"
	"golang.org/x/net/html"
)

// New adds a new widget of the given type to the context parent.
// It automatically calls [Context.config] on the resulting widget.
func New[T tree.NodeValue](ctx *Context) *T {
	parent := ctx.Parent()
	w := tree.New[T](parent)
	ctx.config(any(w).(core.Widget)) // TODO: better htmlcore structure with new config paradigm?
	return w
}

// handleElement calls the handler in [Context.ElementHandlers] associated with the current node
// using the given context. If there is no handler associated with it, it uses default
// hardcoded configuration code.
func handleElement(ctx *Context) {
	tag := ctx.Node.Data
	h, ok := ctx.ElementHandlers[tag]
	if ok {
		if h(ctx) {
			return
		}
	}

	if slices.Contains(textTags, tag) {
		handleTextTag(ctx)
		return
	}

	switch tag {
	case "script", "title", "meta":
		// we don't render anything
	case "link":
		rel := GetAttr(ctx.Node, "rel")
		// TODO(kai/htmlcore): maybe handle preload
		if rel == "preload" {
			return
		}
		// TODO(kai/htmlcore): support links other than stylesheets
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
		ctx.addStyle(string(b))
	case "style":
		ctx.addStyle(ExtractText(ctx))
	case "body", "main", "div", "section", "nav", "footer", "header", "ol", "ul", "blockquote":
		w := New[core.Frame](ctx)
		ctx.NewParent = w
		if tag == "body" {
			w.Styler(func(s *styles.Style) {
				s.Grow.Set(1, 1)
			})
		}
		if tag == "ol" || tag == "ul" {
			w.Styler(func(s *styles.Style) {
				s.Grow.Set(1, 0)
			})
		}
	case "button":
		New[core.Button](ctx).SetText(ExtractText(ctx))
	case "h1":
		handleText(ctx).SetType(core.TextDisplaySmall)
	case "h2":
		handleText(ctx).SetType(core.TextHeadlineMedium)
	case "h3":
		handleText(ctx).SetType(core.TextTitleLarge)
	case "h4":
		handleText(ctx).SetType(core.TextTitleMedium)
	case "h5":
		handleText(ctx).SetType(core.TextTitleSmall)
	case "h6":
		handleText(ctx).SetType(core.TextLabelSmall)
	case "p":
		handleText(ctx)
	case "pre":
		hasCode := ctx.Node.FirstChild != nil && ctx.Node.FirstChild.Data == "code"
		if hasCode {
			ed := New[texteditor.Editor](ctx)
			ctx.Node = ctx.Node.FirstChild // go to the code element
			lang := getLanguage(GetAttr(ctx.Node, "class"))
			if lang != "" {
				ed.Buffer.SetFileExt(lang)
			}
			ed.Buffer.SetString(ExtractText(ctx))
			if BindTextEditor != nil && (lang == "Go" || lang == "Goal") {
				ed.Buffer.SpacesToTabs(0, ed.Buffer.NumLines()) // Go uses tabs
				parent := core.NewFrame(ed.Parent)
				parent.Styler(func(s *styles.Style) {
					s.Direction = styles.Column
					s.Grow.Set(1, 0)
				})
				// we inherit our Grow.Y from our first child so that
				// elements that want to grow can do so
				parent.SetOnChildAdded(func(n tree.Node) {
					if _, ok := n.(*core.Body); ok { // Body should not grow
						return
					}
					wb := core.AsWidget(n)
					if wb.IndexInParent() != 0 {
						return
					}
					wb.FinalStyler(func(s *styles.Style) {
						parent.Styles.Grow.Y = s.Grow.Y
					})
				})
				BindTextEditor(ed, parent, lang)
			} else {
				ed.SetReadOnly(true)
				ed.Buffer.Options.LineNumbers = false
				ed.Styler(func(s *styles.Style) {
					s.Border.Width.Zero()
					s.MaxBorder.Width.Zero()
					s.StateLayer = 0
					s.Background = colors.Scheme.SurfaceContainer
				})
			}
		} else {
			handleText(ctx).Styler(func(s *styles.Style) {
				s.Text.WhiteSpace = styles.WhiteSpacePreWrap
			})
		}
	case "li":
		// if we have a p as our first or second child, which is typical
		// for markdown-generated HTML, we use it directly for data extraction
		// to prevent double elements and unnecessary line breaks.
		if ctx.Node.FirstChild != nil && ctx.Node.FirstChild.Data == "p" {
			ctx.Node = ctx.Node.FirstChild
		} else if ctx.Node.FirstChild != nil && ctx.Node.FirstChild.NextSibling != nil && ctx.Node.FirstChild.NextSibling.Data == "p" {
			ctx.Node = ctx.Node.FirstChild.NextSibling
		}

		text := handleText(ctx)
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
				// TODO(kai/htmlcore): have different bullets for different depths
				start = "• "
			}
		}
		text.SetText(start + text.Text)
	case "img":
		img := New[core.Image](ctx)
		n := ctx.Node
		img.SetTooltip(GetAttr(n, "alt"))
		go func() {
			src := GetAttr(n, "src")
			resp, err := Get(ctx, src)
			if errors.Log(err) != nil {
				return
			}
			defer resp.Body.Close()
			if strings.Contains(resp.Header.Get("Content-Type"), "svg") {
				// TODO(kai/htmlcore): support svg
			} else {
				im, _, err := imagex.Read(resp.Body)
				if err != nil {
					slog.Error("error loading image", "url", src, "err", err)
					return
				}
				img.AsyncLock()
				img.SetImage(im)
				img.Update()
				img.AsyncUnlock()
			}
		}()
	case "input":
		ityp := GetAttr(ctx.Node, "type")
		val := GetAttr(ctx.Node, "value")
		switch ityp {
		case "number":
			fval := float32(errors.Log1(strconv.ParseFloat(val, 32)))
			New[core.Spinner](ctx).SetValue(fval)
		case "checkbox":
			New[core.Switch](ctx).SetType(core.SwitchCheckbox).
				SetState(HasAttr(ctx.Node, "checked"), states.Checked)
		case "radio":
			New[core.Switch](ctx).SetType(core.SwitchRadioButton).
				SetState(HasAttr(ctx.Node, "checked"), states.Checked)
		case "range":
			fval := float32(errors.Log1(strconv.ParseFloat(val, 32)))
			New[core.Slider](ctx).SetValue(fval)
		case "button", "submit":
			New[core.Button](ctx).SetText(val)
		case "color":
			core.Bind(val, New[core.ColorButton](ctx))
		case "datetime":
			core.Bind(val, New[core.TimeInput](ctx))
		case "file":
			core.Bind(val, New[core.FileButton](ctx))
		default:
			New[core.TextField](ctx).SetText(val)
		}
	case "textarea":
		buf := texteditor.NewBuffer()
		buf.SetText([]byte(ExtractText(ctx)))
		New[texteditor.Editor](ctx).SetBuffer(buf)
	default:
		ctx.NewParent = ctx.Parent()
	}
}

func textStyler(s *styles.Style) {
	s.Margin.SetVertical(units.Em(core.ConstantSpacing(0.25)))
	// TODO: it would be ideal for htmlcore to automatically save a scale factor
	// in general and for each domain, that is applied only to page content
	// scale := float32(1.2)
	// s.Font.Size.Value *= scale
	// s.Text.LineHeight.Value *= scale
	// s.Text.LetterSpacing.Value *= scale
}

// handleText creates a new [core.Text] from the given information, setting the text and
// the text click function so that URLs are opened according to [Context.OpenURL].
func handleText(ctx *Context) *core.Text {
	tx := New[core.Text](ctx).SetText(ExtractText(ctx))
	tx.Styler(textStyler)
	tx.HandleTextClick(func(tl *paint.TextLink) {
		ctx.OpenURL(tl.URL)
	})
	return tx
}

// handleTextTag creates a new [core.Text] from the given information, setting the text and
// the text click function so that URLs are opened according to [Context.OpenURL]. Also,
// it wraps the text with the [nodeString] of the given node, meaning that it
// should be used for standalone elements that are meant to only exist in text
// (eg: a, span, b, code, etc).
func handleTextTag(ctx *Context) *core.Text {
	start, end := nodeString(ctx.Node)
	str := start + ExtractText(ctx) + end
	tx := New[core.Text](ctx).SetText(str)
	tx.Styler(textStyler)
	tx.HandleTextClick(func(tl *paint.TextLink) {
		ctx.OpenURL(tl.URL)
	})
	return tx
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

// getLanguage returns the 'x' in a `language-x` class from the given
// string of class(es).
func getLanguage(class string) string {
	fields := strings.Fields(class)
	for _, field := range fields {
		if strings.HasPrefix(field, "language-") {
			return strings.TrimPrefix(field, "language-")
		}
	}
	return ""
}

// Get is a helper function that calls [Context.GetURL] with the given URL, parsed
// relative to the page URL of the given context. It also checks the status
// code of the response and closes the response body and returns an error if
// it is not [http.StatusOK]. If the error is nil, then the response body is
// not closed and must be closed by the caller.
func Get(ctx *Context, url string) (*http.Response, error) {
	u, err := parseRelativeURL(url, ctx.PageURL)
	if err != nil {
		return nil, err
	}
	resp, err := ctx.GetURL(u.String())
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return resp, fmt.Errorf("got error status %q (code %d)", resp.Status, resp.StatusCode)
	}
	return resp, nil
}

// BindTextEditor is a function set to [cogentcore.org/core/yaegicore.BindTextEditor]
// when importing yaegicore, which provides interactive editing functionality for Go
// code blocks in text editors.
var BindTextEditor func(ed *texteditor.Editor, parent *core.Frame, language string)
