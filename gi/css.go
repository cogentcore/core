// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"log"

	"github.com/aymerick/douceur/css"
	"github.com/aymerick/douceur/parser"

	// 	"github.com/benbjohnson/css" // this was too low-level
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
)

// StyleSheet is a Node2D node that contains a stylesheet -- property values
// contained in this sheet can be transformed into ki.Props and set in CSS
// field of appropriate node
type StyleSheet struct {
	Node2DBase
	Sheet *css.Stylesheet
}

var KiT_StyleSheet = kit.Types.AddType(&StyleSheet{}, nil)

// AddNewStyleSheet adds a new CSS stylesheet to given parent node, with given name.
func AddNewStyleSheet(parent ki.Ki, name string) *StyleSheet {
	return parent.AddNewChild(KiT_StyleSheet, name).(*StyleSheet)
}

func (ss *StyleSheet) CopyFieldsFrom(frm interface{}) {
	fr, ok := frm.(*StyleSheet)
	if !ok {
		ki.GenCopyFieldsFrom(ss.This(), frm)
		return
	}
	ss.Node2DBase.CopyFieldsFrom(&fr.Node2DBase)
	// probably don't copy Sheet pointer..
}

// ParseString parses the string into a StyleSheet of rules, which can then be
// used for extracting properties
func (ss *StyleSheet) ParseString(str string) error {
	pss, err := parser.Parse(str)
	if err != nil {
		log.Printf("gist.StyleSheet ParseString parser error: %v\n", err)
		return err
	}
	ss.Sheet = pss
	return nil
}

// CSSProps returns the properties for each of the rules in this style sheet,
// suitable for setting the CSS value of a node -- returns nil if empty sheet
func (ss *StyleSheet) CSSProps() ki.Props {
	if ss.Sheet == nil {
		return nil
	}
	sz := len(ss.Sheet.Rules)
	if sz == 0 {
		return nil
	}
	pr := make(ki.Props, sz)
	for _, r := range ss.Sheet.Rules {
		if r.Kind == css.AtRule {
			continue // not supported
		}
		nd := len(r.Declarations)
		if nd == 0 {
			continue
		}
		for _, sel := range r.Selectors {
			sp := make(ki.Props, nd)
			for _, de := range r.Declarations {
				sp[de.Property] = de.Value
			}
			pr[sel] = sp
		}
	}
	return pr
}
