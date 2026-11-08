// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"bytes"
	"fmt"
	"log/slog"
	"mime/multipart"
	"net/url"
	"strconv"
	"strings"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/text/textcore"
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
	if tf, ok := w.(*core.TextField); ok {
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
			checked := HasAttr(n, "checked")
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
		if ed, ok := w.(*textcore.Editor); ok {
			return ed.Lines.String(), true
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
	ctx.handleWidget(ctx.Node, ch)
}

// handleButton handles a button element: a block of its content,
// styled by CSS, that is clickable.
func handleButton(ctx *Context) {
	w := New[core.Frame](ctx)
	ctx.NewParent = w
	ctx.handleWidget(ctx.Node, w)
	w.Styler(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Clickable)
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
		s.SetAbilities(true, abilities.Clickable)
		s.Cursor = cursors.Pointer
	})
	tx := core.NewText(w).SetText(html.EscapeString(label))
	ctx.handleWidget(nil, tx)
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
			newWidget = New[core.Spinner](ctx).SetValue(fval)
		case "checkbox", "radio":
			// (SetState returns the WidgetBase, which is not the widget)
			sw := New[core.Switch](ctx).SetType(core.SwitchCheckbox)
			if ityp == "radio" {
				sw.SetType(core.SwitchRadioButton)
			}
			sw.SetState(HasAttr(ctx.Node, "checked"), states.Checked)
			newWidget = sw
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
	}
	return newWidget
}
