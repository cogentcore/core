// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package csl

import "cogentcore.org/core/text/rich"

// DefaultStyle is the default citation style.
var DefaultStyle = APA

// Styles are CSL reference styles.
type Styles int32 //enums:enum

const (
	APA Styles = iota
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
// according to the given style.
func Cite(s Styles, it *Item) string {
	switch s {
	case APA:
		return CiteAPA(it)
	}
	return ""
}

// CiteDefault generates the citation text for given item,
// according to the [DefaultStyle] style.
func CiteDefault(it *Item) string {
	return Cite(DefaultStyle, it)
}
