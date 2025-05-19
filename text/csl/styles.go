// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package csl

import "cogentcore.org/core/text/rich"

// DefaultStyle is the default citation and reference formatting style.
var DefaultStyle = APA

// Styles are CSL citation and reference formatting styles.
type Styles int32 //enums:enum

const (
	APA Styles = iota
)

// CiteStyles are different types of citation styles that are supported by
// some formatting [Styles].
type CiteStyles int32 //enums:enum

const (
	// Parenthetical means that the citation is placed within parentheses.
	// This is default for most styles. In the APA style for example, it
	// adds a comma before the year, e.g., "(Smith, 1989)".
	// Note that the parentheses or other outer bracket syntax are NOT
	// generated directly, because often multiple are included together
	// in the same group.
	Parenthetical CiteStyles = iota

	// Narrative is an active, "inline" form of citation where the cited
	// content is used as the subject of a sentence. In the APA style this
	// puts the year in parentheses, e.g., "Smith (1989) invented the..."
	// In this case the parentheses are generated.
	Narrative
)

// Ref generates the reference text for given item,
// according to the given style.
func Ref(s Styles, it *Item) rich.Text {
	switch s {
	case APA:
		return RefAPA(it)
	}
	return nil
}

// Refs returns a list of references and matching items
// according to the given [Styles] style.
func Refs(s Styles, kl *KeyList) ([]rich.Text, []*Item) {
	switch s {
	case APA:
		return RefsAPA(kl)
	}
	return nil, nil
}

// RefsDefault returns a list of references and matching items
// according to the [DefaultStyle].
func RefsDefault(kl *KeyList) ([]rich.Text, []*Item) {
	return Refs(DefaultStyle, kl)
}

// Cite generates the citation text for given item,
// according to the given overall style an citation style.
func Cite(s Styles, cs CiteStyles, it *Item) string {
	switch s {
	case APA:
		return CiteAPA(cs, it)
	}
	return ""
}

// CiteDefault generates the citation text for given item,
// according to the [DefaultStyle] overall style, and given [CiteStyles].
func CiteDefault(cs CiteStyles, it *Item) string {
	return Cite(DefaultStyle, cs, it)
}
