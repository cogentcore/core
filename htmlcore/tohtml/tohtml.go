// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package tohtml converts Cogent Core widgets to HTML, which the core
// tool uses to pre-render apps for a preview while the app loads and for
// SEO purposes. Importing it sets [core.GenerateHTML] to [Generate],
// which the app calls when it runs with the -generatehtml argument.
// The core tool links it into the app for that run, so apps do not need
// to import it themselves.
package tohtml

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/core"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/tree"
)

// Widget converts the given widget and all of its children to HTML.
// This is not guaranteed to be perfect HTML, and it should not be used as a
// replacement for a Cogent Core app. However, it is good enough to be used as
// a preview or for SEO purposes (see [Generate]). The following
// [tree.NodeBase.Properties] of widgets are used:
//   - "tag": the HTML element name.
//   - "id": the HTML id, which is otherwise only set for headings.
//   - "class": the HTML class.
//   - "href" and "src": the corresponding HTML attributes.
//   - "data-*": the data attributes of the element (as htmlcore sets them).
//   - "html": HTML that is used for the widget and its children instead.
func Widget(w core.Widget) ([]byte, error) {
	b := &bytes.Buffer{}
	e := xml.NewEncoder(b)
	err := toHTML(w, e, b)
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// htmlElementNames is a map from widget [types.Type.IDName]s to HTML element
// names for cases in which those differ.
var htmlElementNames = map[string]string{
	"body":       "main", // we are typically placed in a different outer body
	"frame":      "div",
	"text":       "p",
	"image":      "img",
	"icon":       "svg",
	"space":      "div",
	"separator":  "hr",
	"text-field": "input",
	"spinner":    "input",
	"slider":     "input",
	"meter":      "progress",
	"chooser":    "div", // a select element can only contain option elements
	"switches":   "div",
	"switch":     "input",
	"splits":     "div",
	"tabs":       "div",
	"tab":        "button",
	"tree":       "div",
	"page":       "main",
}

func addAttr(se *xml.StartElement, name, value string) {
	if value == "" {
		return
	}
	se.Attr = append(se.Attr, xml.Attr{Name: xml.Name{Local: name}, Value: value})
}

// toHTML is the recursive implementation of [Widget].
func toHTML(w core.Widget, e *xml.Encoder, b *bytes.Buffer) error {
	wb := w.AsWidget()
	if h, ok := wb.Property("html").(string); ok {
		// Custom HTML for this widget and its children (see [Widget]).
		err := e.Flush()
		if err != nil {
			return err
		}
		b.WriteString(h)
		return nil
	}
	// Text without a tag of its own in a widget with a tag (as the text of an
	// element in the browser style of htmlcore) is just its content, and an
	// inline container of it is just its children.
	_, hasTag := wb.Property("tag").(string)
	if tx, ok := w.(*core.Text); ok && !hasTag {
		if pw := core.AsWidget(wb.Parent); pw != nil {
			if _, ok := pw.Property("tag").(string); ok || strings.HasPrefix(pw.Name, "inline-container") {
				if err := e.Flush(); err != nil {
					return err
				}
				// Text of a fixed width (as the marker of a list item) keeps it.
				if width, mx := tx.Styles.Min.X, tx.Styles.Max.X; width.Value > 0 && mx.Value == width.Value && mx.Unit == width.Unit {
					fmt.Fprintf(b, `<span style="display:inline-block;flex:none;width:%s">`, styles.ToCSSLength(&tx.Styles, width))
					b.WriteString(textParagraphReplacer.Replace(tx.Text))
					b.WriteString("</span>")
					return nil
				}
				b.WriteString(textParagraphReplacer.Replace(tx.Text))
				return nil
			}
		}
	}
	if _, ok := w.(*core.Frame); ok && !hasTag && strings.HasPrefix(wb.Name, "inline-container") {
		var err error
		wb.ForWidgetChildren(func(i int, cw core.Widget, cwb *core.WidgetBase) bool {
			err = toHTML(cw, e, b)
			return err == nil
		})
		return err
	}
	se := &xml.StartElement{}
	typ := wb.NodeType()
	idName := typ.IDName
	se.Name.Local = idName
	if tag, ok := wb.Property("tag").(string); ok {
		se.Name.Local = tag
	}
	if se.Name.Local == "tree" { // trees not supported yet
		return nil
	}
	if en, ok := htmlElementNames[se.Name.Local]; ok {
		se.Name.Local = en
	}

	switch typ.Name {
	case "cogentcore.org/cogent/canvas.Canvas", "cogentcore.org/cogent/code.Code":
		se.Name.Local = "div"
	case "cogentcore.org/core/textcore.Editor":
		se.Name.Local = "textarea"
	}
	if se.Name.Local == "textarea" {
		wb.Styles.Min.X.Pw(95)
	}

	// Only explicit ids and heading ids (used as link targets) are meaningful;
	// automatic widget names are not unique in the resulting HTML.
	if id, ok := wb.Property("id").(string); ok {
		addAttr(se, "id", id)
	} else if isHTMLHeading(se.Name.Local) {
		addAttr(se, "id", wb.Name)
	}
	switch se.Name.Local {
	case "table": // styled as a real table, not a grid
	case "img":
		src, _ := wb.Property("src").(string)
		addAttr(se, "src", src)
		addAttr(se, "alt", wb.Tooltip)
		addAttr(se, "style", imageCSS(&wb.Styles))
	default:
		// An element of only inline text (as a paragraph in the browser
		// style of htmlcore) has no layout of its own, so its text is inline.
		// A container of floats has block layout, as floats have no effect
		// in a flex layout.
		cssID := idName
		floats := hasFloats(wb)
		if inlineOnly(wb) || floats {
			cssID = "text"
		}
		css := styles.ToCSS(&wb.Styles, cssID, se.Name.Local) + growCSS(wb)
		if floats {
			css += ";display:flow-root" // a block that contains its floats
		}
		if _, ok := w.(*core.Button); ok && se.Name.Local == "a" {
			css += ";text-decoration:none" // link button
		}
		// A widget of an inline element (as a span with a width) is a box.
		if inlineElements[se.Name.Local] {
			css += ";display:inline-block"
		}
		addAttr(se, "style", strings.TrimPrefix(css, ";"))
	}
	if class, ok := wb.Property("class").(string); ok {
		addAttr(se, "class", class)
	}
	if href, ok := wb.Property("href").(string); ok {
		addAttr(se, "href", href)
	}
	addDataAttrs(se, wb)

	handleChildren := true

	switch w := w.(type) {
	case *core.TextField:
		addAttr(se, "type", "text")
		addAttr(se, "value", w.Text())
		handleChildren = false
	case *core.Spinner:
		addAttr(se, "type", "number")
		addAttr(se, "value", fmt.Sprintf("%g", w.Value))
		handleChildren = false
	case *core.Slider:
		addAttr(se, "type", "range")
		addAttr(se, "value", fmt.Sprintf("%g", w.Value))
		handleChildren = false
	case *core.Switch:
		addAttr(se, "type", "checkbox")
		addAttr(se, "value", strconv.FormatBool(w.IsChecked()))
	case *core.Collapser:
		se.Name.Local = "details"
		if w.Open {
			addAttr(se, "open", "open")
		}
		handleChildren = false
	}
	if se.Name.Local == "textarea" {
		addAttr(se, "rows", "10")
		addAttr(se, "cols", "30")
	}

	err := e.EncodeToken(*se)
	if err != nil {
		return err
	}
	err = e.Flush()
	if err != nil {
		return err
	}

	switch w := w.(type) {
	case *core.Text:
		// We don't want any escaping of HTML-formatted text, so we write directly.
		// Paragraphs within the text (such as in a figure caption) are rendered
		// inline in a Text, and are not allowed within a <p>, so we remove them.
		b.WriteString(textParagraphReplacer.Replace(w.Text))
	case *core.Icon:
		// TODO: just remove the width and height attributes from the source SVGs?
		icon := strings.ReplaceAll(string(w.Icon), ` width="48" height="48"`, "")
		b.WriteString(icon)
	case *core.SVG:
		w.SVG.PhysicalWidth = wb.Styles.Min.X
		w.SVG.PhysicalHeight = wb.Styles.Min.Y
		err := w.SVG.WriteXML(b, false)
		if err != nil {
			return err
		}
	}
	if se.Name.Local == "textarea" && idName == "editor" {
		b.WriteString(reflectx.Underlying(reflect.ValueOf(w)).FieldByName("Lines").Interface().(fmt.Stringer).String())
	}

	switch w := w.(type) {
	case *core.Collapser:
		err = collapserToHTML(w, e, b)
		if err != nil {
			return err
		}
	}
	if se.Name.Local == "table" {
		err = tableToHTML(wb, e, b)
		if err != nil {
			return err
		}
		handleChildren = false
	}

	if handleChildren {
		wb.ForWidgetChildren(func(i int, cw core.Widget, cwb *core.WidgetBase) bool {
			if idName == "switch" && cwb.Name == "stack" {
				return tree.Continue
			}
			err = toHTML(cw, e, b)
			if err != nil {
				return tree.Break
			}
			return tree.Continue
		})
		if err != nil {
			return err
		}
	}
	err = e.EncodeToken(xml.EndElement{Name: se.Name})
	if err != nil {
		return err
	}
	return e.Flush()
}

// hasFloats returns whether any of the children of the widget float.
func hasFloats(wb *core.WidgetBase) bool {
	floats := false
	wb.ForWidgetChildren(func(i int, cw core.Widget, cwb *core.WidgetBase) bool {
		floats = cwb.Styles.Float != styles.FloatNone
		return !floats
	})
	return floats
}

// inlineElements are the HTML elements that are inline by default, which
// are boxes ("inline-block") when they are widgets of their own.
var inlineElements = map[string]bool{"span": true, "a": true, "b": true, "i": true, "em": true,
	"strong": true, "code": true, "small": true, "label": true, "mark": true, "sub": true, "sup": true}

// inlineOnly returns whether all of the children of the widget are inline
// text: [core.Text] without a tag of its own and inline containers of it,
// which [toHTML] writes as the content of the element.
func inlineOnly(wb *core.WidgetBase) bool {
	if _, ok := wb.Property("tag").(string); !ok || wb.NumChildren() == 0 {
		return false
	}
	only := true
	wb.ForWidgetChildren(func(i int, cw core.Widget, cwb *core.WidgetBase) bool {
		_, tagged := cwb.Property("tag").(string)
		_, isText := cw.(*core.Text)
		inline := !tagged && (isText || strings.HasPrefix(cwb.Name, "inline-container"))
		if !inline {
			only = false
			return tree.Break
		}
		return tree.Continue
	})
	return only
}

// addDataAttrs adds the data attributes of the widget, which are its
// string properties with a data- prefix, in sorted order.
func addDataAttrs(se *xml.StartElement, wb *core.WidgetBase) {
	var names []string
	for k, v := range wb.Properties {
		if _, ok := v.(string); ok && strings.HasPrefix(k, "data-") {
			names = append(names, k)
		}
	}
	slices.Sort(names)
	for _, k := range names {
		addAttr(se, k, wb.Properties[k].(string))
	}
}

// textParagraphReplacer removes paragraph tags within [core.Text.Text] in [Widget].
var textParagraphReplacer = strings.NewReplacer("<p>", "", "</p>", "")

// isHTMLHeading returns whether the given HTML element name is a heading (h1-h6).
func isHTMLHeading(name string) bool {
	return len(name) == 2 && name[0] == 'h' && name[1] >= '1' && name[1] <= '6'
}

// imageCSS returns the CSS style for an image, which only needs its size.
func imageCSS(s *styles.Style) string {
	parts := []string{}
	if s.Min.X.Value > 0 {
		parts = append(parts, "width:"+s.Min.X.StringCSSDp())
	}
	if s.Min.Y.Value > 0 {
		parts = append(parts, "height:"+s.Min.Y.StringCSSDp())
	}
	if s.Max.X.Value > 0 {
		parts = append(parts, "max-width:"+s.Max.X.StringCSSDp())
	}
	if s.Max.Y.Value > 0 {
		parts = append(parts, "max-height:"+s.Max.Y.StringCSSDp())
	}
	if s.ObjectFit != styles.FitFill {
		parts = append(parts, "object-fit:"+s.ObjectFit.String())
	}
	return strings.Join(parts, ";")
}

// tableToHTML writes the children of the given table frame, which are laid
// out in a grid, as the rows of an HTML table.
func tableToHTML(wb *core.WidgetBase, e *xml.Encoder, b *bytes.Buffer) error {
	cols := max(wb.Styles.Columns, 1)
	tr := xml.StartElement{Name: xml.Name{Local: "tr"}}
	var err error
	wb.ForWidgetChildren(func(i int, cw core.Widget, cwb *core.WidgetBase) bool {
		if i%cols == 0 {
			if i > 0 {
				err = e.EncodeToken(tr.End())
				if err != nil {
					return tree.Break
				}
			}
			err = e.EncodeToken(tr)
			if err != nil {
				return tree.Break
			}
		}
		err = toHTML(cw, e, b)
		if err != nil {
			return tree.Break
		}
		return tree.Continue
	})
	if err != nil {
		return err
	}
	if wb.NumChildren() > 0 {
		err = e.EncodeToken(tr.End())
	}
	return err
}

// collapserToHTML writes the children of the given collapser as the
// <summary> and contents of a <details> element, excluding the toggle switch.
func collapserToHTML(cl *core.Collapser, e *xml.Encoder, b *bytes.Buffer) error {
	sum := xml.StartElement{Name: xml.Name{Local: "summary"}}
	err := e.EncodeToken(sum)
	if err != nil {
		return err
	}
	cl.Summary.ForWidgetChildren(func(i int, cw core.Widget, cwb *core.WidgetBase) bool {
		if _, ok := cw.(*core.Switch); ok {
			return tree.Continue
		}
		err = toHTML(cw, e, b)
		if err != nil {
			return tree.Break
		}
		return tree.Continue
	})
	if err != nil {
		return err
	}
	err = e.EncodeToken(sum.End())
	if err != nil {
		return err
	}
	cl.Details.ForWidgetChildren(func(i int, cw core.Widget, cwb *core.WidgetBase) bool {
		err = toHTML(cw, e, b)
		if err != nil {
			return tree.Break
		}
		return tree.Continue
	})
	return err
}

// growCSS returns additional CSS for the grow factors of the given widget
// relative to the layout direction of its parent, since [styles.ToCSS]
// does not have access to the parent.
func growCSS(wb *core.WidgetBase) string {
	if wb.Parent == nil {
		return ""
	}
	pw := core.AsWidget(wb.Parent)
	if pw == nil || pw.Styles.Display != styles.Flex {
		return ""
	}
	main, cross := wb.Styles.Grow.Y, wb.Styles.Grow.X
	if pw.Styles.Direction == styles.Row {
		main, cross = cross, main
	}
	css := fmt.Sprintf(";flex-grow:%g", main)
	switch {
	case wb.Styles.Align.Self != styles.Auto: // as for auto margins in htmlcore
		css += ";align-self:" + wb.Styles.Align.Self.String()
	case cross > 0:
		css += ";align-self:stretch"
	}
	return css
}
