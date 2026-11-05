// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package content

import (
	"strconv"
	"strings"

	"cogentcore.org/core/core"
	"cogentcore.org/core/tree"
)

// headingOf returns the heading level (1 to 6) and text of the given
// widget, if it is a heading: a [core.Text] with a tag of h1 to h6, or a
// widget with such a tag with its text in it (as in the browser style of
// htmlcore), which is the widget of the heading (as for scrolling to it).
func headingOf(w core.Widget) (level int, text string, ok bool) {
	tag, _ := w.AsWidget().Property("tag").(string)
	if len(tag) != 2 || tag[0] != 'h' {
		return 0, "", false
	}
	level, err := strconv.Atoi(tag[1:])
	if err != nil || level < 1 || level > 6 {
		return 0, "", false
	}
	if tx, ok := w.(*core.Text); ok {
		return level, tx.Text, true
	}
	var texts []string
	w.AsWidget().WidgetWalkDown(func(cw core.Widget, cwb *core.WidgetBase) bool {
		if tx, ok := cw.(*core.Text); ok && strings.TrimSpace(tx.Text) != "" {
			texts = append(texts, strings.TrimSpace(tx.Text))
		}
		return tree.Continue
	})
	if len(texts) == 0 {
		return 0, "", false
	}
	return level, strings.Join(texts, " "), true
}

// paragraphText returns the text of the given widget if it is a paragraph:
// a [core.Text] with a tag of p, or a widget with that tag with a text in it.
func paragraphText(w core.Widget) (string, bool) {
	if w.AsWidget().Property("tag") != "p" {
		return "", false
	}
	if tx, ok := w.(*core.Text); ok {
		return tx.Text, true
	}
	text := ""
	w.AsWidget().WidgetWalkDown(func(cw core.Widget, cwb *core.WidgetBase) bool {
		if tx, ok := cw.(*core.Text); ok {
			text = tx.Text
			return tree.Break
		}
		return tree.Continue
	})
	return text, text != ""
}
