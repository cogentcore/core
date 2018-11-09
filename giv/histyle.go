// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"strings"

	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/styles"
	"github.com/goki/gi"
	"github.com/goki/ki/kit"
)

// Trilean value for StyleEntry value inheritance.
type Trilean uint8

const (
	TrPass Trilean = iota
	TrYes
	TrNo

	TrileanN
)

// func (t Trilean) String() string {
// 	switch t {
// 	case TrYes:
// 		return "Yes"
// 	case TrNo:
// 		return "No"
// 	default:
// 		return "Pass"
// 	}
// }

func (t Trilean) Prefix(s string) string {
	if t == TrYes {
		return s
	} else if t == TrNo {
		return "no" + s
	}
	return ""
}

//go:generate stringer -type=Trilean

var KiT_Trilean = kit.Enums.AddEnumAltLower(TrileanN, false, nil, "Tr")

func (ev Trilean) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *Trilean) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

// HiStyleEntry is one value in the map of highilight style values
type HiStyleEntry struct {
	Color      gi.Color
	Background gi.Color
	Border     gi.Color

	Bold      Trilean
	Italic    Trilean
	Underline Trilean
	NoInherit bool
}

// FromChroma copies styles from chroma
func (he *HiStyleEntry) FromChroma(ce chroma.StyleEntry) {
	he.Color.SetString(ce.Colour.String(), nil)
	he.Background.SetString(ce.Background.String(), nil)
	he.Border.SetString(ce.Border.String(), nil)
	he.Bold = Trilean(ce.Bold)
	he.Italic = Trilean(ce.Italic)
	he.Underline = Trilean(ce.Underline)
	he.NoInherit = ce.NoInherit
}

func (s HiStyleEntry) String() string {
	out := []string{}
	if s.Bold != TrPass {
		out = append(out, s.Bold.Prefix("bold"))
	}
	if s.Italic != TrPass {
		out = append(out, s.Italic.Prefix("italic"))
	}
	if s.Underline != TrPass {
		out = append(out, s.Underline.Prefix("underline"))
	}
	if s.NoInherit {
		out = append(out, "noinherit")
	}
	if !s.Color.IsNil() {
		out = append(out, s.Color.String())
	}
	if !s.Background.IsNil() {
		out = append(out, "bg:"+s.Background.String())
	}
	if !s.Border.IsNil() {
		out = append(out, "border:"+s.Border.String())
	}
	return strings.Join(out, " ")
}

// HiStyle is a full style map of styles for different token tag values
type HiStyle map[chroma.TokenType]*HiStyleEntry

// FromChroma copies styles from chroma
func (hs *HiStyle) FromChroma(cs *chroma.Style) {
	if *hs == nil {
		*hs = make(HiStyle, 20)
	}
	for tag, _ := range chroma.StandardTypes {
		if cs.Has(tag) {
			ce := cs.Get(tag)
			he := &HiStyleEntry{}
			he.FromChroma(ce)
			(*hs)[tag] = he
		}
	}
}

// Open hi style from a JSON-formatted file.
func (hs *HiStyle) OpenJSON(filename string) error {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		// PromptDialog(nil, "File Not Found", err.Error(), true, false, nil, nil, nil)
		log.Println(err)
		return err
	}
	return json.Unmarshal(b, hs)
}

// Save hi style to a JSON-formatted file.
func (hs *HiStyle) SaveJSON(filename string) error {
	b, err := json.MarshalIndent(hs, "", "  ")
	if err != nil {
		log.Println(err) // unlikely
		return err
	}
	err = ioutil.WriteFile(filename, b, 0644)
	if err != nil {
		// PromptDialog(nil, "Could not Save to File", err.Error(), true, false, nil, nil, nil)
		log.Println(err)
	}
	return err
}

// HiStyles is a collection of styles
type HiStyles map[string]*HiStyle

// StdHiStyles are the styles from chroma package
var StdHiStyles HiStyles

// CustomHiStyles are user's special styles
var CustomHiStyles HiStyles

// AvailHiStyles are all highlighting styles
var AvailHiStyles HiStyles

// FromChroma copies styles from chroma
func (hs *HiStyles) FromChroma(cs map[string]*chroma.Style) {
	if *hs == nil {
		*hs = make(HiStyles, len(cs))
	}
	for nm, cse := range cs {
		hse := &HiStyle{}
		hse.FromChroma(cse)
		(*hs)[nm] = hse
	}
}

// CopyFrom copies styles from another collection
func (hs *HiStyles) CopyFrom(os HiStyles) {
	if *hs == nil {
		*hs = make(HiStyles, len(os))
	}
	for nm, cse := range os {
		(*hs)[nm] = cse
	}
}

// Open hi styles from a JSON-formatted file.
func (hs *HiStyles) OpenJSON(filename string) error {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		// PromptDialog(nil, "File Not Found", err.Error(), true, false, nil, nil, nil)
		log.Println(err)
		return err
	}
	return json.Unmarshal(b, hs)
}

// Save hi styles to a JSON-formatted file.
func (hs *HiStyles) SaveJSON(filename string) error {
	b, err := json.MarshalIndent(hs, "", "  ")
	if err != nil {
		log.Println(err) // unlikely
		return err
	}
	err = ioutil.WriteFile(filename, b, 0644)
	if err != nil {
		// PromptDialog(nil, "Could not Save to File", err.Error(), true, false, nil, nil, nil)
		log.Println(err)
	}
	return err
}

// Names outputs names of styles in collection
func (hs *HiStyles) Names() []string {
	nms := make([]string, len(*hs))
	idx := 0
	for nm, _ := range *hs {
		nms[idx] = nm
		idx++
	}
	return nms
}

// HiStyleNames are all the names of all the available highlighting styles
var HiStyleNames []string

func InitHiStyles() {
	StdHiStyles.FromChroma(styles.Registry)
	// allow custom apps, e.g., gide to load special prefs
	AvailHiStyles.CopyFrom(StdHiStyles)
	HiStyleNames = AvailHiStyles.Names()
}
