// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package css

// FontProperties are the properties of fonts, which are compiled first
// (with line-height, which the font shorthand sets).
var FontProperties = map[string]bool{
	"font": true, "font-family": true, "font-size": true, "font-weight": true,
	"font-style": true, "line-height": true,
}

// WebkitAliases are the properties whose -webkit- prefixed names are
// aliases of them in browsers, which pages may set only with the prefix.
var WebkitAliases = map[string]bool{
	"transform": true, "transform-origin": true, "box-sizing": true, "box-shadow": true,
	"border-radius": true, "flex": true, "flex-direction": true, "flex-wrap": true, "flex-flow": true,
	"flex-grow": true, "flex-shrink": true, "flex-basis": true, "order": true, "align-items": true,
	"align-self": true, "justify-content": true, "user-select": true, "appearance": true,
}

// KeywordLonghands are the longhands of the shorthands, which their
// CSS-wide keywords (inherit, initial) apply to (the others are applied
// to the style whole, see styles.Style.FromProperty).
var KeywordLonghands = map[string][]string{
	"font":   {"font-style", "font-weight", "font-size", "line-height", "font-family"},
	"margin": {"margin-top", "margin-right", "margin-bottom", "margin-left"},
}

// InheritedProperties are the properties that are inherited in CSS, for
// which unset is inherit (and initial for the others).
var InheritedProperties = map[string]bool{
	"color": true, "font": true, "font-family": true, "font-size": true, "font-style": true,
	"font-weight": true, "font-variant": true, "line-height": true, "letter-spacing": true,
	"word-spacing": true, "text-align": true, "text-indent": true, "text-transform": true,
	"text-shadow": true, "white-space": true, "visibility": true, "cursor": true, "direction": true, "pointer-events": true,
	"list-style": true, "list-style-type": true, "list-style-position": true, "quotes": true,
	"word-break": true, "overflow-wrap": true, "hyphens": true, "tab-size": true,
	"border-collapse": true, "border-spacing": true, "caption-side": true, "empty-cells": true,
}

// LogicalProperties are the physical properties for the logical
// properties of CSS, for horizontal, left to right text.
var LogicalProperties = map[string]string{
	"inline-size": "width", "block-size": "height",
	"min-inline-size": "min-width", "max-inline-size": "max-width",
	"min-block-size": "min-height", "max-block-size": "max-height",
	"margin-inline-start": "margin-left", "margin-inline-end": "margin-right",
	"margin-block-start": "margin-top", "margin-block-end": "margin-bottom",
	"padding-inline-start": "padding-left", "padding-inline-end": "padding-right",
	"padding-block-start": "padding-top", "padding-block-end": "padding-bottom",
	"border-inline-start": "border-left", "border-inline-end": "border-right",
	"border-block-start": "border-top", "border-block-end": "border-bottom",
	"border-inline-start-width": "border-left-width", "border-inline-end-width": "border-right-width",
	"border-block-start-width": "border-top-width", "border-block-end-width": "border-bottom-width",
	"border-inline-start-color": "border-left-color", "border-inline-end-color": "border-right-color",
	"border-block-start-color": "border-top-color", "border-block-end-color": "border-bottom-color",
	"inset-inline-start": "left", "inset-inline-end": "right",
	"inset-block-start": "top", "inset-block-end": "bottom",
	"border-start-start-radius": "border-top-left-radius", "border-start-end-radius": "border-top-right-radius",
	"border-end-start-radius": "border-bottom-left-radius", "border-end-end-radius": "border-bottom-right-radius",
}

var BorderStyleNames = map[string]bool{"none": true, "hidden": true, "solid": true, "dotted": true, "dashed": true,
	"double": true, "groove": true, "ridge": true, "inset": true, "outset": true}
