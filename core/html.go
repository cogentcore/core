// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"bytes"
	"encoding/xml"

	"cogentcore.org/core/styles"
	"cogentcore.org/core/tree"
)

// ToHTML converts the given widget and all of its children to HTML.
// This is not guaranteed to be perfect HTML, and it should not be used as a
// replacement for a Cogent Core app. However, it is good enough to be used as
// a preview or for SEO purposes (see generatehtml.go).
func ToHTML(w Widget) ([]byte, error) {
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
	"body":      "main", // we are typically placed in a different outer body
	"frame":     "div",
	"text":      "p",
	"image":     "img",
	"icon":      "svg",
	"space":     "div",
	"separator": "hr",
}

func addAttr(se *xml.StartElement, name, value string) {
	se.Attr = append(se.Attr, xml.Attr{Name: xml.Name{Local: name}, Value: value})
}

// toHTML is the recursive implementation of [ToHTML].
func toHTML(w Widget, e *xml.Encoder, b *bytes.Buffer) error {
	wb := w.AsWidget()
	se := &xml.StartElement{}
	se.Name.Local = wb.NodeType().IDName
	if en, ok := htmlElementNames[se.Name.Local]; ok {
		se.Name.Local = en
	}

	addAttr(se, "id", wb.Name)
	addAttr(se, "style", styles.ToCSS(&wb.Styles))

	// rv := reflect.ValueOf(w)
	// uv := reflectx.Underlying(rv)

	err := e.EncodeToken(*se)
	if err != nil {
		return err
	}
	err = e.Flush()
	if err != nil {
		return err
	}

	if text, ok := w.(*Text); ok {
		// We don't want any escaping of HTML-formatted text, so we write directly.
		b.WriteString(text.Text)
	}

	wb.ForWidgetChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
		err = toHTML(cw, e, b)
		if err != nil {
			return tree.Break
		}
		return tree.Continue
	})
	if err != nil {
		return err
	}
	err = e.EncodeToken(xml.EndElement{se.Name})
	if err != nil {
		return err
	}
	return e.Flush()
}
