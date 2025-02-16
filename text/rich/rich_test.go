// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rich

import (
	"image/color"
	"testing"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/text/runes"
	"cogentcore.org/core/text/textpos"
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
	s.SetBackground(colors.Blue)

	sr := RuneFromSpecial(s.Special)
	ss := RuneToSpecial(sr)
	assert.Equal(t, s.Special, ss)

	rs := s.ToRunes()

	assert.Equal(t, 3, len(rs))
	assert.Equal(t, 1, s.Decoration.NumColors())

	ns := &Style{}
	ns.FromRunes(rs)

	assert.Equal(t, s, ns)
}

func TestText(t *testing.T) {
	src := "The lazy fox typed in some familiar text"
	sr := []rune(src)
	tx := Text{}
	plain := NewStyle()
	ital := NewStyle().SetSlant(Italic)
	ital.SetStrokeColor(colors.Red)
	boldBig := NewStyle().SetWeight(Bold).SetSize(1.5)
	tx.AddSpan(plain, sr[:4])
	tx.AddSpan(ital, sr[4:8])
	fam := []rune("familiar")
	ix := runes.Index(sr, fam)
	tx.AddSpan(plain, sr[8:ix])
	tx.AddSpan(boldBig, sr[ix:ix+8])
	tx.AddSpan(plain, sr[ix+8:])

	str := tx.String()
	trg := `[]: "The "
[italic stroke-color]: "lazy"
[]: " fox typed in some "
[1.50x bold]: "familiar"
[]: " text"
`
	assert.Equal(t, trg, str)

	os := tx.Join()
	assert.Equal(t, src, string(os))

	for i := range src {
		assert.Equal(t, rune(src[i]), tx.At(i))
	}

	tx.SplitSpan(12)
	trg = `[]: "The "
[italic stroke-color]: "lazy"
[]: " fox"
[]: " typed in some "
[1.50x bold]: "familiar"
[]: " text"
`
	// fmt.Println(tx)
	assert.Equal(t, trg, tx.String())

	// spl := tx.Split()
	// for i := range spl {
	// 	fmt.Println(string(spl[i]))
	// }
}

func TestLink(t *testing.T) {
	src := "Pre link link text post link"
	tx := Text{}
	plain := NewStyle()
	ital := NewStyle().SetSlant(Italic)
	ital.SetStrokeColor(colors.Red)
	boldBig := NewStyle().SetWeight(Bold).SetSize(1.5)
	tx.AddSpan(plain, []rune("Pre link "))
	tx.AddLink(ital, "https://example.com", "link text")
	tx.AddSpan(boldBig, []rune(" post link"))

	str := tx.String()
	trg := `[]: "Pre link "
[italic link [https://example.com] stroke-color]: "link text"
[{End Special}]: ""
[1.50x bold]: " post link"
`
	assert.Equal(t, trg, str)

	os := tx.Join()
	assert.Equal(t, src, string(os))

	for i := range src {
		assert.Equal(t, rune(src[i]), tx.At(i))
	}

	lks := tx.GetLinks()
	assert.Equal(t, 1, len(lks))
	assert.Equal(t, textpos.Range{9, 18}, lks[0].Range)
	assert.Equal(t, "link text", lks[0].Label)
	assert.Equal(t, "https://example.com", lks[0].URL)
}
