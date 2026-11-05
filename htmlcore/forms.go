// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"fmt"
	"log/slog"
	"net/url"
	"strings"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/text/textcore"
	"golang.org/x/net/html"
)

// The forms are submitted with the GET method,
// as browsers do, by opening the URL of their action with the values of
// their controls in the query, when a submit button is clicked or Enter
// is pressed in one of their text fields.

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
// if it was submitted by one.
func (c *Context) submitForm(form, submitter *html.Node, e events.Event) {
	if method := strings.ToLower(GetAttr(form, "method")); method != "" && method != "get" {
		slog.Error("htmlcore: only forms with the GET method can be submitted", "method", method)
		return
	}
	values := url.Values{}
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			if name := GetAttr(n, "name"); name != "" && !HasAttr(n, "disabled") {
				if v, ok := c.controlValue(n, submitter); ok {
					values.Add(name, v)
				}
			}
		}
		for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
			walk(ch)
		}
	}
	walk(form)
	action := GetAttr(form, "action")
	if submitter != nil && HasAttr(submitter, "formaction") {
		action = GetAttr(submitter, "formaction")
	}
	if action == "" {
		action = c.PageURL
	}
	u, err := parseRelativeURL(action, c.PageURL)
	if err != nil {
		slog.Error("htmlcore: invalid form action", "action", action, "err", err)
		return
	}
	u.RawQuery = values.Encode()
	c.OpenURL(u.String(), e)
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
