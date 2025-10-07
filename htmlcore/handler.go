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
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/text/lines"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/text"
	"cogentcore.org/core/text/textcore"
	"cogentcore.org/core/tree"
	"golang.org/x/net/html"
)

// New adds a new widget of the given type to the context parent.
// It automatically calls [Context.config] on the resulting widget.
func New[T tree.NodeValue](ctx *Context) *T {
	parent := ctx.Parent()
	w := tree.New[T](parent)
	ctx.config(any(w).(core.Widget))
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

	pid := ""
	pstyle := ""
	if ctx.BlockParent != nil { // these attributes get put on a block parent element
		pstyle = GetAttr(ctx.Node.Parent, "style")
		pid = GetAttr(ctx.Node.Parent, "id")
	}
	var newWidget core.Widget

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
		newWidget = w
		ctx.NewParent = w
		switch tag {
		case "body":
			w.Styler(func(s *styles.Style) {
				s.Grow.Set(1, 1)
			})
		case "ol", "ul":
			ld := listDepth(ctx, w.This.(core.Widget)) + 1
			w.SetProperty("listDepth", ld)
			if tag == "ol" {
				w.SetProperty("listCount", 0)
			}
			w.Styler(func(s *styles.Style) {
				s.Grow.Set(1, 0)
				s.Padding.Left.Ch(core.ConstantSpacing(float32(4 * ld)))
			})
		case "div":
			w.Styler(func(s *styles.Style) {
				s.Grow.Set(1, 1)
				s.Overflow.Y = styles.OverflowAuto
			})
		case "blockquote":
			w.Styler(func(s *styles.Style) { // todo: need a better marker
				s.Grow.Set(1, 0)
				s.Background = colors.Scheme.SurfaceContainer
			})
		}
	case "button":
		newWidget = New[core.Button](ctx).SetText(ExtractText(ctx))
	case "h1":
		newWidget = handleText(ctx, tag).SetType(core.TextDisplaySmall)
	case "h2":
		newWidget = handleText(ctx, tag).SetType(core.TextHeadlineMedium)
	case "h3":
		newWidget = handleText(ctx, tag).SetType(core.TextTitleLarge)
	case "h4":
		newWidget = handleText(ctx, tag).SetType(core.TextTitleMedium)
	case "h5":
		newWidget = handleText(ctx, tag).SetType(core.TextTitleSmall)
	case "h6":
		newWidget = handleText(ctx, tag).SetType(core.TextLabelSmall)
	case "p":
		newWidget = handleText(ctx, tag)
	case "pre":
		hasCode := ctx.Node.FirstChild != nil && ctx.Node.FirstChild.Data == "code"
		if hasCode {
			codeEl := ctx.Node.FirstChild
			lang := GetLanguage(GetAttr(codeEl, "class"))
			id := GetAttr(codeEl, "id")
			var ed *textcore.Editor
			ed = New[textcore.Editor](ctx)
			if id != "" {
				ed.SetName(id)
			}
			newWidget = ed
			ctx.Node = codeEl
			if lang != "" {
				ed.Lines.SetFileExt(lang)
			}
			ed.Lines.SetString(ExtractText(ctx))
			ed.SetReadOnly(true)
			ed.Lines.Settings.LineNumbers = false
			ed.Styler(func(s *styles.Style) {
				s.Border.Width.Zero()
				s.MaxBorder.Width.Zero()
				s.StateLayer = 0
				s.Background = colors.Scheme.SurfaceContainer
			})
		} else {
			newWidget = handleText(ctx, tag)
			newWidget.AsWidget().Styler(func(s *styles.Style) {
				s.Text.WhiteSpace = text.WhiteSpacePreWrap
			})
		}
	case "li":
		// if we have a p as our first or second child, which is typical
		// for markdown-generated HTML, we use it directly for data extraction
		// to prevent double elements and unnecessary line breaks.
		hasPChild := false
		if ctx.Node.FirstChild != nil && ctx.Node.FirstChild.Data == "p" {
			ctx.Node = ctx.Node.FirstChild
			hasPChild = true
		} else if ctx.Node.FirstChild != nil && ctx.Node.FirstChild.NextSibling != nil && ctx.Node.FirstChild.NextSibling.Data == "p" {
			ctx.Node = ctx.Node.FirstChild.NextSibling
		}
		text, sublist := handleTextExclude(ctx, "ol", "ul") // exclude other lists
		newWidget = text
		start := ""
		if pw, ok := text.Parent.(core.Widget); ok {
			pwt := pw.AsTree()
			switch pwt.Property("tag") {
			case "ol":
				number := pwt.Property("listCount").(int) + 1
				pwt.SetProperty("listCount", number)
				start = strconv.Itoa(number) + ". "
			case "ul":
				ld := pwt.Property("listDepth").(int)
				if ld%2 == 1 {
					start = "• "
				} else {
					start = "◦ "
				}
			}
		}
		text.SetText(start + text.Text)

		if hasPChild { // handle potential additional <p> blocks that should be indented
			cnode := ctx.Node
			// ctx.BlockParent = text.Parent.(core.Widget)
			for cnode.NextSibling != nil {
				cnode = cnode.NextSibling
				ctx.Node = cnode
				if cnode.Data != "p" {
					continue
				}
				txt, psub := handleTextExclude(ctx, "ol", "ul")
				txt.SetText(txt.Text)
				ctx.handleWidget(cnode, txt)
				if psub != nil {
					if psub != sublist {
						readHTMLNode(ctx, ctx.Parent(), psub)
					}
					break
				}
			}
		}
		if sublist != nil {
			readHTMLNode(ctx, ctx.Parent(), sublist)
		}
	case "img":
		n := ctx.Node
		src := GetAttr(n, "src")
		alt := GetAttr(n, "alt")
		if pstyle != "" {
			ctx.setStyleAttr(n, pstyle)
		}
		// Can be either image or svg.
		var img *core.Image
		var svg *core.SVG
		if strings.HasSuffix(src, ".svg") {
			svg = New[core.SVG](ctx)
			svg.SetTooltip(alt)
			if pid != "" {
				svg.SetName(pid)
				svg.SetProperty("id", pid)
			}
			newWidget = svg
		} else {
			img = New[core.Image](ctx)
			img.SetTooltip(alt)
			if pid != "" {
				img.SetName(pid)
				img.SetProperty("id", pid)
			}
			newWidget = img
		}

		go func() {
			resp, err := Get(ctx, src)
			if errors.Log(err) != nil {
				return
			}
			defer resp.Body.Close()
			if svg != nil {
				svg.AsyncLock()
				errors.Log(svg.Read(resp.Body))
				svg.Update()
				svg.AsyncUnlock()
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
			newWidget = New[core.Spinner](ctx).SetValue(fval)
		case "checkbox":
			newWidget = New[core.Switch](ctx).SetType(core.SwitchCheckbox).
				SetState(HasAttr(ctx.Node, "checked"), states.Checked)
		case "radio":
			newWidget = New[core.Switch](ctx).SetType(core.SwitchRadioButton).
				SetState(HasAttr(ctx.Node, "checked"), states.Checked)
		case "range":
			fval := float32(errors.Log1(strconv.ParseFloat(val, 32)))
			newWidget = New[core.Slider](ctx).SetValue(fval)
		case "button", "submit":
			newWidget = New[core.Button](ctx).SetText(val)
		case "color":
			newWidget = core.Bind(val, New[core.ColorButton](ctx))
		case "datetime":
			newWidget = core.Bind(val, New[core.TimeInput](ctx))
		case "file":
			newWidget = core.Bind(val, New[core.FileButton](ctx))
		default:
			newWidget = New[core.TextField](ctx).SetText(val)
		}
	case "textarea":
		buf := lines.NewLines()
		buf.SetText([]byte(ExtractText(ctx)))
		newWidget = New[textcore.Editor](ctx).SetLines(buf)
	case "table":
		w := New[core.Frame](ctx)
		newWidget = w
		ctx.NewParent = w
		ctx.TableParent = w
		ctx.firstRow = true
		w.SetProperty("cols", 0)
		w.Styler(func(s *styles.Style) {
			s.Display = styles.Grid
			s.Overflow.X = styles.OverflowAuto
			s.Grow.Set(1, 0)
			s.Columns = w.Property("cols").(int)
			s.Gap.X.Dp(core.ConstantSpacing(6))
			s.Justify.Content = styles.Center
		})
	case "th", "td":
		if ctx.TableParent != nil && ctx.firstRow {
			cols := ctx.TableParent.Property("cols").(int)
			cols++
			ctx.TableParent.SetProperty("cols", cols)
		}
		tx := handleText(ctx, tag)
		if tx.Parent == nil { // if empty we need a real empty text to keep structure
			tx = New[core.Text](ctx)
		}
		newWidget = tx
		// fmt.Println(tag, "val:", tx.Text)
		if tag == "th" {
			tx.Styler(func(s *styles.Style) {
				s.Font.Weight = rich.Bold
				s.Border.Width.Bottom.Dp(2)
				s.Margin.Bottom.Dp(6)
				s.Margin.Top.Dp(6)
			})
		} else {
			tx.Styler(func(s *styles.Style) {
				s.Margin.Bottom.Dp(6)
				s.Margin.Top.Dp(6)
			})
		}
	case "thead", "tbody":
		ctx.NewParent = ctx.TableParent
	case "tr":
		if ctx.TableParent != nil && ctx.firstRow && ctx.TableParent.NumChildren() > 0 {
			ctx.firstRow = false
		}
		ctx.NewParent = ctx.TableParent
	default:
		ctx.NewParent = ctx.Parent()
	}
	if newWidget != nil {
		ctx.handleWidget(ctx.Node, newWidget)
	}
}

// handleText creates a new [core.Text] from the given information, setting the text and
// the text click function so that URLs are opened according to [Context.OpenURL].
func handleText(ctx *Context, tag string) *core.Text {
	tx, _ := handleTextExclude(ctx)
	return tx
}

// handleTextExclude creates a new [core.Text] from the given information, setting the text and
// the text click function so that URLs are opened according to [Context.OpenURL].
// excludeSubs is a list of sub-node types to exclude in processing this element.
// If one of those types is encountered, it is returned.
func handleTextExclude(ctx *Context, excludeSubs ...string) (*core.Text, *html.Node) {
	et, excl := ExtractTextExclude(ctx, excludeSubs...)
	if et == "" {
		// Empty text elements do not render, so we just return a fake one (to avoid panics).
		return core.NewText(), excl
	}
	tx := New[core.Text](ctx).SetText(et)
	tx.HandleTextClick(func(tl *rich.Hyperlink) {
		ctx.OpenURL(tl.URL)
	})
	return tx, excl
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
	tx.HandleTextClick(func(tl *rich.Hyperlink) {
		ctx.OpenURL(tl.URL)
	})
	return tx
}

// listDepth returns the depth of list elements ("ol", "ul") above
// the given widget. 0 = none, 1 = 1 etc.
func listDepth(ctx *Context, w core.Widget) int {
	ld := 0
	pw, ok := w.AsTree().Parent.(core.Widget)
	for ok {
		ptag := pw.AsTree().Property("tag")
		if ptag == "ol" || ptag == "ul" {
			ld++
			pw, ok = pw.AsTree().Parent.(core.Widget)
		} else {
			break
		}
	}
	return ld
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

// GetLanguage returns the 'x' in a `language-x` class from the given
// string of class(es).
func GetLanguage(class string) string {
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
