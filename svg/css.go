// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"log"

	"github.com/aymerick/douceur/css"
	"github.com/aymerick/douceur/parser"
)

// StyleSheet is a node that contains a stylesheet -- property values
// contained in this sheet can be transformed into tree.Properties and set in CSS
// field of appropriate node.
type StyleSheet struct {
	NodeBase
	Sheet *css.Stylesheet `copier:"-"`
}

func (ss *StyleSheet) SVGName() string {
	return "css"
}

// ParseString parses the string into a StyleSheet of rules, which can then be
// used for extracting properties
func (ss *StyleSheet) ParseString(str string) error {
	pss, err := parser.Parse(str)
	if err != nil {
		log.Printf("styles.StyleSheet ParseString parser error: %v\n", err)
		return err
	}
	ss.Sheet = pss
	return nil
}

// CSSProperties returns the properties for each of the rules in this style sheet,
// suitable for setting the CSS value of a node -- returns nil if empty sheet
func (ss *StyleSheet) CSSProperties() map[string]any {
	if ss.Sheet == nil {
		return nil
	}
	sz := len(ss.Sheet.Rules)
	if sz == 0 {
		return nil
	}
	pr := map[string]any{}
	for _, r := range ss.Sheet.Rules {
		if r.Kind == css.AtRule {
			continue // not supported
		}
		nd := len(r.Declarations)
		if nd == 0 {
			continue
		}
		for _, sel := range r.Selectors {
			sp := map[string]any{}
			for _, de := range r.Declarations {
				sp[de.Property] = de.Value
			}
			pr[sel] = sp
		}
	}
	return pr
}

//////// MetaData

// MetaData is used for holding meta data info
type MetaData struct {
	NodeBase
	MetaData string
}

func (g *MetaData) SVGName() string {
	return "metadata"
}

func (g *MetaData) EnforceSVGName() bool {
	return false
}
