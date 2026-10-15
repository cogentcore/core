// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shapedgt

import (
	"slices"

	"github.com/go-text/typesetting/fontscan"
	"github.com/go-text/typesetting/language"
)

// fontMap is a [fontscan.FontMap] that keeps its candidate fonts when it is
// given the query or script that it already has. The FontMap itself discards
// them on every SetQuery and SetScript, which the shaper calls for every span
// and the segmenter for every script run, and then rebuilds them on the next
// face cache miss, which is costly.
type fontMap struct {
	*fontscan.FontMap

	// query and script are the last ones set, when hasQuery and hasScript.
	query     fontscan.Query
	script    language.Script
	hasQuery  bool
	hasScript bool
}

// SetQuery sets the query of the font map unless it already has it.
func (fm *fontMap) SetQuery(query fontscan.Query) {
	if fm.hasQuery && query.Aspect == fm.query.Aspect && slices.Equal(query.Families, fm.query.Families) {
		return
	}
	fm.query = query
	fm.hasQuery = true
	fm.FontMap.SetQuery(query)
}

// SetScript sets the script of the font map unless it already has it.
func (fm *fontMap) SetScript(script language.Script) {
	if fm.hasScript && script == fm.script {
		return
	}
	fm.script = script
	fm.hasScript = true
	fm.FontMap.SetScript(script)
}

// forget forgets the last query and script, so that the next ones are always
// set, as when other code may have set the query or script of the FontMap.
func (fm *fontMap) forget() {
	fm.hasQuery = false
	fm.hasScript = false
}
