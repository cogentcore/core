// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rich

import (
	"image/color"
	"testing"

	"cogentcore.org/core/base/runes"
	"cogentcore.org/core/colors"
	"github.com/stretchr/testify/assert"
)

func TestColors(t *testing.T) {
	c := color.RGBA{22, 55, 77, 255}
	r := ColorToRune(c)
	rc := ColorFromRune(r)
	assert.Equal(t, c, rc)
}

func TestStyle(t *testing.T) {
	s := NewStyle()
	s.Family = Maths
	s.Special = Math
	s.SetLink("https://example.com/readme.md")
	s.SetBackground(colors.Blue)

	sr := RuneFromSpecial(s.Special)
	ss := RuneToSpecial(sr)
	assert.Equal(t, s.Special, ss)

	rs := s.ToRunes()

	assert.Equal(t, 33, len(rs))
	assert.Equal(t, 1, s.Decoration.NumColors())

	ns := &Style{}
	ns.FromRunes(rs)

	assert.Equal(t, s, ns)
}

func TestText(t *testing.T) {
	src := "The lazy fox typed in some familiar text"
	sr := []rune(src)
	sp := Text{}
	plain := NewStyle()
	ital := NewStyle().SetSlant(Italic)
	ital.SetStrokeColor(colors.Red)
	boldBig := NewStyle().SetWeight(Bold).SetSize(1.5)
	sp.Add(plain, sr[:4])
	sp.Add(ital, sr[4:8])
	fam := []rune("familiar")
	ix := runes.Index(sr, fam)
	sp.Add(plain, sr[8:ix])
	sp.Add(boldBig, sr[ix:ix+8])
	sp.Add(plain, sr[ix+8:])

	str := sp.String()
	trg := `[]: The 
[italic stroke-color]: lazy
[]:  fox typed in some 
[1.50x bold]: familiar
[]:  text
`
	assert.Equal(t, trg, str)

	os := sp.Join()
	assert.Equal(t, src, string(os))

	for i := range fam {
		assert.Equal(t, fam[i], sp.At(ix+i))
	}

	// spl := tx.Split()
	// for i := range spl {
	// 	fmt.Println(string(spl[i]))
	// }
}
