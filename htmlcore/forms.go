// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"bytes"
	"fmt"
	"image/color"
	"log/slog"
	"mime/multipart"
	"net/url"
	"strconv"
	"strings"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/text"
	"cogentcore.org/core/tree"
	"golang.org/x/net/html"
)

// The forms are submitted as browsers do, when a submit button is clicked
// or Enter is pressed in one of their text fields: with the GET method by
// opening the URL of their action with the values of their controls in the
// query, and with the POST method by [Context.PostURL].

// ancestorForm returns the form element that the given node is in.
func ancestorForm(n *html.Node) *html.Node {
	for p := n.Parent; p != nil; p = p.Parent {
		if p.Type == html.ElementNode && p.Data == "form" {
			return p
		}
	}
	return nil
}

// isSubmitter returns whether the given element submits its form.
func isSubmitter(n *html.Node) bool {
	typ := strings.ToLower(GetAttr(n, "type"))
	switch n.Data {
	case "button":
		return typ == "" || typ == "submit"
	case "input":
		return typ == "submit" || typ == "image"
	}
	return false
}

// handleFormWidget records the widget of the given node, and makes it
// submit its form, for the controls of forms.
func (c *Context) handleFormWidget(n *html.Node, w core.Widget) {
	if n == nil || n.Type != html.ElementNode {
		return
	}
	switch n.Data {
	case "input", "button", "textarea", "select":
	default:
		return
	}
	if c.nodeWidgets == nil {
		c.nodeWidgets = map[*html.Node]core.Widget{}
	}
	c.nodeWidgets[n] = w
	if sw, ok := w.(*core.Switch); ok {
		// the element after it, which shows the hover of a transparent
		// control over it (see [Context.handleHoverTarget])
		for k := n.NextSibling; k != nil; k = k.NextSibling {
			if k.Type == html.ElementNode {
				if c.hoverTargets == nil {
					c.hoverTargets = map[*html.Node]*core.Switch{}
				}
				c.hoverTargets[k] = sw
				break
			}
		}
	}
	if sw, ok := w.(*core.Switch); ok && strings.EqualFold(GetAttr(n, "type"), "radio") && GetAttr(n, "name") != "" {
		// a radio button that is checked unchecks the others of its group
		// (with its name, in its form or outside of forms)
		sw.OnChange(func(e events.Event) {
			if !sw.IsChecked() {
				return
			}
			for on, ow := range c.nodeWidgets {
				osw, ok := ow.(*core.Switch)
				if !ok || on == n || !strings.EqualFold(GetAttr(on, "type"), "radio") ||
					GetAttr(on, "name") != GetAttr(n, "name") || ancestorForm(on) != ancestorForm(n) {
					continue
				}
				if osw.IsChecked() {
					osw.SetChecked(false)
					osw.NeedsRender()
				}
			}
		})
	}
	form := ancestorForm(n)
	if form == nil {
		return
	}
	if isSubmitter(n) {
		w.AsWidget().OnClick(func(e events.Event) {
			e.SetHandled()
			c.submitForm(form, n, e)
		})
		return
	}
	if tf, ok := w.(*core.TextField); ok && n.Data != "textarea" { // enter submits, but not in a textarea
		tf.OnFirst(events.KeyChord, func(e events.Event) {
			if kf := keymap.Of(e.KeyChord()); kf == keymap.Enter || kf == keymap.Accept {
				e.SetHandled()
				c.submitForm(form, nil, e)
			}
		})
	}
}

// submitForm submits the given form, with the given submit button,
// if it was submitted by one: with the GET method, by opening the URL of
// its action with the values of its controls in the query, and with the
// POST method, by [Context.PostURL] with them in the body.
func (c *Context) submitForm(form, submitter *html.Node, e events.Event) {
	// the method, action and encoding, which the submit button can override
	attr := func(name string) string {
		if submitter != nil && HasAttr(submitter, "form"+name) {
			return GetAttr(submitter, "form"+name)
		}
		return GetAttr(form, name)
	}
	method := strings.ToLower(attr("method"))
	post := method == "post"
	if method != "" && method != "get" && !post || post && c.PostURL == nil {
		slog.Error("htmlcore: forms with this method can not be submitted", "method", method)
		return
	}
	// the values of the controls, in order
	var names, values []string
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			if name := GetAttr(n, "name"); name != "" && !HasAttr(n, "disabled") {
				if v, ok := c.controlValue(n, submitter); ok {
					names = append(names, name)
					values = append(values, v)
				}
			}
		}
		for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
			walk(ch)
		}
	}
	walk(form)
	// the URL of the action
	action := attr("action")
	if action == "" {
		action = c.PageURL
	}
	u, err := parseRelativeURL(action, c.PageURL)
	if err != nil {
		slog.Error("htmlcore: invalid form action", "action", action, "err", err)
		return
	}
	if post {
		contentType, body := encodeForm(strings.ToLower(attr("enctype")), names, values)
		c.PostURL(u.String(), contentType, body, e)
		return
	}
	u.RawQuery = urlEncode(names, values)
	c.OpenURL(u.String(), e)
}

// urlEncode returns the given names and values encoded as for a query,
// in their order.
func urlEncode(names, values []string) string {
	var b strings.Builder
	for i, name := range names {
		if i > 0 {
			b.WriteByte('&')
		}
		b.WriteString(url.QueryEscape(name) + "=" + url.QueryEscape(values[i]))
	}
	return b.String()
}

// encodeForm returns the content type and body of a POST request for the
// given names and values of the controls of a form, with the given
// encoding (its enctype attribute).
func encodeForm(enctype string, names, values []string) (string, []byte) {
	switch enctype {
	case "multipart/form-data":
		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		for i, name := range names {
			w.WriteField(name, values[i])
		}
		w.Close()
		return w.FormDataContentType(), b.Bytes()
	case "text/plain":
		var b strings.Builder
		for i, name := range names {
			b.WriteString(name + "=" + values[i] + "\r\n")
		}
		return "text/plain", []byte(b.String())
	}
	return "application/x-www-form-urlencoded", []byte(urlEncode(names, values))
}

// controlValue returns the value of the given form control element,
// and whether it has one to submit.
func (c *Context) controlValue(n, submitter *html.Node) (string, bool) {
	w := c.nodeWidgets[n]
	switch n.Data {
	case "input":
		typ := strings.ToLower(GetAttr(n, "type"))
		switch typ {
		case "submit", "image", "button", "reset":
			if n != submitter {
				return "", false
			}
			return GetAttr(n, "value"), true
		case "checkbox", "radio":
			checked := c.isChecked(n)
			if w != nil {
				checked = w.AsWidget().StateIs(states.Checked)
			}
			if !checked {
				return "", false
			}
			if v := GetAttr(n, "value"); v != "" {
				return v, true
			}
			return "on", true
		}
		if tf, ok := w.(*core.TextField); ok {
			return tf.Text(), true
		}
		return GetAttr(n, "value"), true
	case "button":
		if n != submitter {
			return "", false
		}
		return GetAttr(n, "value"), true
	case "textarea":
		if tf, ok := w.(*core.TextField); ok {
			return tf.Text(), true
		}
		return nodeText(n), true
	case "select":
		if ch, ok := w.(*core.Chooser); ok && ch.CurrentIndex >= 0 {
			return fmt.Sprint(ch.CurrentItem.Value), true
		}
		for o := range optionNodes(n) {
			if HasAttr(o, "selected") {
				return optionValue(o), true
			}
		}
		for o := range optionNodes(n) {
			return optionValue(o), true
		}
	}
	return "", false
}

// optionNodes returns the option elements of a select element.
func optionNodes(sel *html.Node) func(yield func(*html.Node) bool) {
	return func(yield func(*html.Node) bool) {
		var walk func(n *html.Node) bool
		walk = func(n *html.Node) bool {
			for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
				if ch.Type == html.ElementNode && ch.Data == "option" {
					if !yield(ch) {
						return false
					}
				} else if !walk(ch) {
					return false
				}
			}
			return true
		}
		walk(sel)
	}
}

// optionValue returns the value of an option element.
func optionValue(o *html.Node) string {
	if HasAttr(o, "value") {
		return GetAttr(o, "value")
	}
	return strings.TrimSpace(nodeText(o))
}

// handleSelect handles a select element: a chooser of its options,
// which are not read as content.
func handleSelect(ctx *Context) {
	ch := New[core.Chooser](ctx)
	current := -1
	for o := range optionNodes(ctx.Node) {
		if HasAttr(o, "selected") && current < 0 {
			current = len(ch.Items)
		}
		ch.Items = append(ch.Items, core.ChooserItem{Value: optionValue(o), Text: strings.TrimSpace(nodeText(o))})
	}
	ch.SetCurrentIndex(max(current, 0))
	// with no appearance, the page draws its own arrow (as a background)
	if a := ctx.declaration(ctx.Node, "appearance"); a == "none" || ctx.declaration(ctx.Node, "-webkit-appearance") == "none" {
		ch.SetIndicator(icons.None)
	}
	alignDeclared := ctx.declaration(ctx.Node, "text-align") != ""
	ctx.handleWidget(ctx.Node, ch)
	// as in browsers, the text of the current option is at the start, on
	// one line with the arrow at the end, in a select of any display (as
	// the navigation select of pkg.go.dev, a block of 100% width)
	ch.FinalStyler(func(s *styles.Style) {
		s.Display = styles.Flex
		s.Direction = styles.Row
		s.Wrap = false
		s.Align.Items = styles.Center
		s.Justify.Content = styles.Start
		if !alignDeclared {
			s.Text.Align = text.Start
		}
	})
	ch.Maker(func(p *tree.Plan) {
		tree.AddInit(p, "text", func(w *core.Text) {
			w.FinalStyler(func(s *styles.Style) {
				s.Grow.X = 1
				s.Text.Align = ch.Styles.Text.Align
			})
		})
	})
}

// handleButton handles a button element: a block of its content,
// styled by CSS, that is clickable.
func handleButton(ctx *Context) {
	w := New[core.Frame](ctx)
	ctx.NewParent = w
	ctx.handleWidget(ctx.Node, w)
	w.Styler(func(s *styles.Style) {
		// hoverable for the state layer of buttons when the mouse is over
		// them (as the darkening of buttons in browsers, by their :hover)
		s.SetAbilities(true, abilities.Clickable, abilities.Hoverable)
		s.Cursor = cursors.Pointer
	})
}

// handleInputButton handles an input element of the given type that is
// a button (submit, button or reset): a block of its label, as a button.
func handleInputButton(ctx *Context, typ string) {
	label := GetAttr(ctx.Node, "value")
	if !HasAttr(ctx.Node, "value") {
		label = map[string]string{"submit": "Submit", "reset": "Reset"}[typ]
	}
	w := New[core.Frame](ctx)
	ctx.handleWidget(ctx.Node, w)
	w.Styler(func(s *styles.Style) {
		// hoverable for the state layer of buttons when the mouse is over
		// them (as the darkening of buttons in browsers, by their :hover)
		s.SetAbilities(true, abilities.Clickable, abilities.Hoverable)
		s.Cursor = cursors.Pointer
	})
	tx := core.NewText(w).SetText(html.EscapeString(label))
	ctx.handleWidget(nil, tx)
	buttonText(tx) // the button gets the clicks on its label
}

// handleInput handles an input element.
func handleInput(ctx *Context) core.Widget {
	var newWidget core.Widget
	{
		ityp := GetAttr(ctx.Node, "type")
		val := GetAttr(ctx.Node, "value")
		switch ityp {
		case "number":
			fval := float32(errors.Log1(strconv.ParseFloat(val, 32)))
			sp := New[core.Spinner](ctx).SetValue(fval)
			// a field of its value, as in browsers, which step it with
			// the arrow keys (without the buttons of a spinner)
			sp.SetType(core.TextFieldOutlined)
			sp.SetLeadingIcon(icons.None).SetTrailingIcon(icons.None)
			cssFieldBorder(&sp.TextField)
			fieldWidth(ctx.Node, &sp.TextField)
			newWidget = sp
		case "checkbox", "radio":
			// (SetState returns the WidgetBase, which is not the widget)
			sw := New[core.Switch](ctx).SetType(core.SwitchCheckbox)
			if ityp == "radio" {
				sw.SetType(core.SwitchRadioButton)
			}
			sw.SetState(ctx.isChecked(ctx.Node), states.Checked)
			newWidget = sw
		case "range":
			fval := float32(errors.Log1(strconv.ParseFloat(val, 32)))
			newWidget = New[core.Slider](ctx).SetValue(fval)
		case "button", "submit":
			newWidget = New[core.Button](ctx).SetText(val)
		case "color":
			newWidget = colorInput(ctx, val)
		case "datetime":
			newWidget = core.Bind(val, New[core.TimeInput](ctx))
		case "file":
			newWidget = core.Bind(val, New[core.FileButton](ctx))
		default:
			// outlined, so that its borders are those of its CSS, on all
			// sides (filled ones have only a line at the bottom)
			tf := New[core.TextField](ctx).SetText(val).SetType(core.TextFieldOutlined)
			tf.SetPlaceholder(GetAttr(ctx.Node, "placeholder"))
			cssFieldBorder(tf)
			fieldWidth(ctx.Node, tf)
			tf.FinalStyler(func(s *styles.Style) { s.SetTextWrap(false) }) // an input is one line

			ctx.inputIcons(ctx.Node, tf)
			newWidget = tf
		}
	}
	return newWidget
}

// fieldWidth sets the width of the given text field of the given input or
// textarea element to that of browsers when its CSS gives it none: of its
// size (or cols for a textarea) in characters, 20 by default, of the
// average width of a character of its font (0.55em, or 0.6em for the
// monospace font of a textarea, with its scrollbar), and the height of a
// textarea to its rows of text (2 by default). It does not
// grow (as a core text field does), as in browsers, where a text input was
// as wide as it could be, up to 40ch, in a row of controls (wider than the
// 156px of Chrome).
func fieldWidth(n *html.Node, tf *core.TextField) {
	attr := "size"
	if n.Data == "textarea" {
		attr = "cols"
	}
	chars := 20
	if v, err := strconv.Atoi(strings.TrimSpace(GetAttr(n, attr))); err == nil && v > 0 {
		chars = v
	}
	rows := 0
	if n.Data == "textarea" {
		rows = 2
		if v, err := strconv.Atoi(strings.TrimSpace(GetAttr(n, "rows"))); err == nil && v > 0 {
			rows = v
		}
	}
	w := units.Em(float32(chars) * 0.55)
	if rows > 0 {
		w = units.Em(float32(chars)*0.6 + 1.15) // and its scrollbar of 15px
	}
	// (before the styler of its CSS, which sets the width that it gives)
	tf.Styler(func(s *styles.Style) {
		if rows > 0 {
			s.Min.Y = units.Em(float32(rows) * 1.2)
		}
		s.Min.X, s.Max.X = w, w
		s.Grow.X = 0
	})
	// one that its CSS makes grow (as by flex-grow: 1, as the search field
	// of pkg.go.dev, which fills the header) is not limited to it
	tf.FinalStyler(func(s *styles.Style) {
		if s.Grow.X > 0 && s.Max.X.Unit == w.Unit && s.Max.X.Value == w.Value {
			s.Max.X = units.Value{}
		}
	})
}

// colorInput returns the widget of an input of type color with the given
// value: a swatch of the color with a border, of the size of that of
// browsers (50 by 27px in Chrome), which opens a color picker when it is
// clicked (it was a large button of Cogent Core, with a label and icon).
func colorInput(ctx *Context, val string) core.Widget {
	cb := core.Bind(val, New[core.ColorButton](ctx))
	cb.SetText("").SetIcon(icons.None)
	// (before the styler of its CSS, which sets the sizes that it gives)
	cb.Styler(func(s *styles.Style) {
		s.Min.Set(units.Dp(48), units.Dp(25)) // (the content box, in CSS px)
		s.Max = s.Min
		s.Margin.Zero()
		s.Padding.Zero()
		s.BoxShadow, s.MaxBoxShadow = nil, nil // (the room of its shadow is around it)
		s.Border.Width.Set(units.Dp(1))
		s.Border.Color.Set(colors.Uniform(color.RGBA{0x76, 0x76, 0x76, 0xff}))
		s.Border.Radius.Set(units.Dp(3))
		s.MaxBorder = s.Border
	})
	return cb
}

// inputIcons makes the icons that are positioned over the given text input
// element the icons of its text field: elements after it with no content
// that are absolutely positioned and have an SVG mask or background image
// (as the search icon of the search field of Wikipedia, at its start). It
// is the leading icon for one at the left (or else the trailing one, for
// one at the right), with the padding of the field on its side as the
// space before the icon. The elements are not rendered.
func (c *Context) inputIcons(n *html.Node, tf *core.TextField) {
	for s := n.NextSibling; s != nil; s = s.NextSibling {
		if s.Type == html.TextNode && strings.TrimSpace(s.Data) == "" || s.Type == html.CommentNode {
			continue
		}
		icon, side, ok := c.iconElement(s)
		if !ok {
			return
		}
		if c.iconNodes == nil {
			c.iconNodes = map[*html.Node]bool{}
		}
		c.iconNodes[s] = true
		inset := strings.TrimSpace(c.declaration(s, side))
		env := c.readEnv()
		pad, hasPad := env.length(inset, units.UnitPw)
		if side == "left" {
			tf.SetLeadingIcon(icon)
		} else {
			tf.SetTrailingIcon(icon)
		}
		tf.FinalStyler(func(st *styles.Style) {
			if !hasPad {
				return
			}
			if side == "left" {
				st.Padding.Left = pad
			} else {
				st.Padding.Right = pad
			}
		})
	}
}

// iconElement returns the icon of the given element if it is an icon over
// a text input (see [Context.inputIcons]), and the side of the input that
// it is at ("left" or "right").
func (c *Context) iconElement(n *html.Node) (icon icons.Icon, side string, ok bool) {
	if n.Type != html.ElementNode || c.declaration(n, "position") != "absolute" || c.displayOf(n) == "none" {
		return
	}
	for k := n.FirstChild; k != nil; k = k.NextSibling {
		if k.Type != html.TextNode || strings.TrimSpace(k.Data) != "" {
			return // it has content
		}
	}
	mask := true
	u := maskURL(c.declaration(n, "mask-image"))
	if u == "" {
		u = maskURL(c.declaration(n, "-webkit-mask-image"))
	}
	if u == "" {
		u = maskURL(c.declaration(n, "mask"))
	}
	if u == "" {
		u, mask = maskURL(c.declaration(n, "background-image")), false
	}
	if u == "" {
		return
	}
	src := c.maskSourceOf(u)
	if src == nil || len(src.svg) == 0 {
		return
	}
	svg := string(src.svg)
	if mask { // the shape of a mask has the color of the text
		for _, black := range []string{`fill="#000"`, `fill="#000000"`, `fill="black"`} {
			svg = strings.ReplaceAll(svg, black, `fill="currentColor"`)
		}
	}
	side = "left"
	if c.declaration(n, "left") == "" && c.declaration(n, "right") != "" {
		side = "right"
	}
	return icons.Icon(svg), side, true
}

// isChecked returns whether the given checkbox or radio input element is
// checked: from [Context.Checked] if it is set, or else its attribute.
func (c *Context) isChecked(n *html.Node) bool {
	if c.Checked != nil {
		return c.Checked(n)
	}
	return HasAttr(n, "checked")
}

// setCheckedAttrs sets the checked attributes of the checkbox and radio
// inputs of the given document to their state from [Context.Checked], if
// it is set, for the :checked selectors of the style sheets (the selector
// library matches the attribute), and returns a function that restores
// them.
func (c *Context) setCheckedAttrs(root *html.Node) func() {
	if c.Checked == nil && c.Hovered == nil {
		return func() {}
	}
	var changed []*html.Node
	var hovered []*html.Node
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if c.Hovered != nil && n.Type == html.ElementNode && c.Hovered(n) {
			n.Attr = append(n.Attr, html.Attribute{Key: hoverAttr}) // for :hover
			hovered = append(hovered, n)
		}
		if c.Checked != nil && n.Type == html.ElementNode && n.Data == "input" {
			if typ := strings.ToLower(GetAttr(n, "type")); typ == "checkbox" || typ == "radio" {
				if c.Checked(n) != HasAttr(n, "checked") {
					toggleAttr(n, "checked")
					changed = append(changed, n)
				}
			}
		}
		for k := n.FirstChild; k != nil; k = k.NextSibling {
			walk(k)
		}
	}
	walk(root)
	return func() {
		for _, n := range changed {
			toggleAttr(n, "checked")
		}
		for _, n := range hovered {
			toggleAttr(n, hoverAttr)
		}
	}
}

// toggleAttr removes the given attribute of the given element if it has
// it, or else adds it (with no value).
func toggleAttr(n *html.Node, key string) {
	for i, a := range n.Attr {
		if a.Key == key && a.Namespace == "" {
			n.Attr = append(n.Attr[:i], n.Attr[i+1:]...)
			return
		}
	}
	n.Attr = append(n.Attr, html.Attribute{Key: key})
}

// handleHoverTarget makes the widget of the given element show the hover
// of the control before it, if that is transparent: a checkbox or radio
// button over the element that shows it, as the hidden checkboxes of the
// menu buttons of Wikipedia (over their labels) and its radio buttons
// (over their icons), which browsers style with :hover of the control.
func (c *Context) handleHoverTarget(n *html.Node, w core.Widget) {
	sw := c.hoverTargets[n]
	if sw == nil {
		return
	}
	delete(c.hoverTargets, n)
	target := w.AsWidget()
	hover := func(on bool) {
		if sw.Styles.Opacity > 0 || target.StateIs(states.Hovered) == on {
			return // a visible control shows its own hover
		}
		target.SetState(on, states.Hovered)
		target.Restyle()
	}
	sw.On(events.MouseEnter, func(e events.Event) { hover(true) })
	sw.On(events.MouseLeave, func(e events.Event) { hover(false) })
}

// cssFieldBorder makes the border of the given text field of an input (or
// textarea) that of its CSS when it has the focus too, with no space for
// the thicker border of an outlined field with the focus (which made it
// higher than its CSS height, as for the search box of duckduckgo.com).
func cssFieldBorder(tf *core.TextField) {
	tf.FinalStyler(func(s *styles.Style) {
		s.MaxBorder = s.Border
	})
}
