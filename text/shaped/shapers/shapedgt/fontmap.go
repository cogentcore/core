// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shapedgt

import (
	"fmt"
	"slices"
	"strings"

	"github.com/cogentcore/typesetting/font"
	"github.com/cogentcore/typesetting/font/opentype"
	"github.com/cogentcore/typesetting/fontscan"
	"github.com/cogentcore/typesetting/language"
)

// maxFaceSets is the number of queries and scripts that a [fontMap] keeps
// the faces of runes for before it starts over.
const maxFaceSets = 64

// fontMap is a [fontscan.FontMap] that keeps the faces of runes per query
// and script, and does not set the query or script that it already has,
// which the shaper sets for every span and the segmenter for every script
// run. The FontMap keeps the candidate fonts of its recent queries and
// scripts itself, which used to be built again on every face cache miss
// after a new query, as text alternates between families.
type fontMap struct {
	*fontscan.FontMap

	// query and script are the last ones set, when hasQuery and hasScript.
	query     fontscan.Query
	script    language.Script
	hasQuery  bool
	hasScript bool

	// faces are the faces of runes resolved for the query and script, when
	// they are set; faceSets has those of each query and script. The FontMap
	// itself caches faces by a hash of the query families and the rune, which
	// costs more than shaping the rune when the query changes span by span.
	faces    map[rune]*font.Face
	faceSets map[string]map[rune]*font.Face

	// variants are the faces of variable fonts with the weight of a query
	// (as the variable web fonts of pages, of which the FontMap gives the
	// default instance), which are faces of their own, as the runs that
	// are shaped with them keep them for rendering.
	variants map[faceVariant]*font.Face
}

// faceVariant is the key of a variant of a face of a variable font.
type faceVariant struct {
	face   *font.Face
	weight float32
}

// wghtTag is the tag of the weight axis of variable fonts.
var wghtTag = opentype.MustNewTag("wght")

// variant returns the face with the weight of the query, for a variable
// font with a weight axis, or the face itself.
func (fm *fontMap) variant(face *font.Face) *font.Face {
	if face == nil || !fm.hasQuery || fm.query.Aspect.Weight == 0 {
		return face
	}
	key := faceVariant{face, float32(fm.query.Aspect.Weight)}
	if v, ok := fm.variants[key]; ok {
		return v
	}
	v := font.NewFace(face.Font)
	v.SetVariations([]font.Variation{{Tag: wghtTag, Value: key.weight}})
	if len(v.Coords()) == 0 { // not a variable font
		v = face
	}
	if fm.variants == nil || len(fm.variants) >= 256 {
		fm.variants = map[faceVariant]*font.Face{}
	}
	fm.variants[key] = v
	return v
}

// ResolveFace returns the face of the rune for the query and script.
func (fm *fontMap) ResolveFace(r rune) *font.Face {
	// Without both set, the FontMap resolves it as it is.
	if fm.faces == nil {
		if !fm.hasQuery || !fm.hasScript {
			return fm.FontMap.ResolveFace(r)
		}
		fm.useFaceSet()
	}

	// The face of a rune is resolved once.
	if face, ok := fm.faces[r]; ok {
		return face
	}
	face := fm.variant(fm.FontMap.ResolveFace(r))
	fm.faces[r] = face
	return face
}

// useFaceSet sets faces to the set for the current query and script.
func (fm *fontMap) useFaceSet() {
	key := fmt.Sprintf("%s\x00%v\x00%v", strings.Join(fm.query.Families, "\x00"), fm.query.Aspect, fm.script)
	if fm.faceSets == nil || len(fm.faceSets) >= maxFaceSets {
		fm.faceSets = make(map[string]map[rune]*font.Face)
	}
	fm.faces = fm.faceSets[key]
	if fm.faces == nil {
		fm.faces = make(map[rune]*font.Face)
		fm.faceSets[key] = fm.faces
	}
}

// SetQuery sets the query of the font map unless it already has it.
func (fm *fontMap) SetQuery(query fontscan.Query) {
	if fm.hasQuery && query.Aspect == fm.query.Aspect && slices.Equal(query.Families, fm.query.Families) {
		return
	}
	fm.query = query
	fm.hasQuery = true
	fm.faces = nil
	fm.FontMap.SetQuery(query)
}

// SetScript sets the script of the font map unless it already has it.
func (fm *fontMap) SetScript(script language.Script) {
	if fm.hasScript && script == fm.script {
		return
	}
	fm.script = script
	fm.hasScript = true
	fm.faces = nil
	fm.FontMap.SetScript(script)
}

// forget forgets the last query and script, so that the next ones are always
// set, and the faces of runes, as when other code may have set the query or
// script of the FontMap, or added fonts to it.
func (fm *fontMap) forget() {
	fm.hasQuery = false
	fm.hasScript = false
	fm.faces = nil
	fm.faceSets = nil
}
