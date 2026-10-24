// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shapedjs

import (
	"reflect"
	"testing"

	"cogentcore.org/core/text/rich"
)

func TestParseUnicodeRange(t *testing.T) {
	got := parseUnicodeRange("U+0-FF, U+131, U+152-153, u+4??")
	want := [][2]rune{{0, 0xff}, {0x131, 0x131}, {0x152, 0x153}, {0x400, 0x4ff}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	for _, bad := range []string{"", "0-FF", "U+XYZ", "U+0-FF, junk"} {
		if got := parseUnicodeRange(bad); got != nil {
			t.Errorf("%q: got %v, want nil (all runes)", bad, got)
		}
	}
}

// TestTextUsesFaces tests which texts use the faces that have loaded.
func TestTextUsesFaces(t *testing.T) {
	plain := rich.NewStyle()
	mono := *plain
	mono.Family = rich.Monospace
	italic := *plain
	italic.Slant = rich.Italic
	custom := *plain
	custom.Family = rich.Custom
	latin := [][2]rune{{0, 0xff}}
	greek := [][2]rune{{0x370, 0x3ff}}
	tests := []struct {
		face webFace
		sty  *rich.Style
		text string
		want bool
	}{
		{webFace{family: "Noto Sans", ranges: latin}, plain, "abc", true},
		{webFace{family: "Noto Sans", ranges: greek}, plain, "abc", false},
		{webFace{family: "Noto Sans", ranges: greek}, plain, "a ψ", true},
		{webFace{family: "Noto Sans", ranges: latin}, &mono, "abc", false},
		{webFace{family: "Roboto Mono", ranges: latin}, &mono, "abc", true},
		{webFace{family: "Noto Sans", italic: true, ranges: latin}, plain, "abc", false},
		{webFace{family: "Noto Sans", italic: true, ranges: latin}, &italic, "abc", true},
		{webFace{family: "Noto Sans", ranges: latin}, &italic, "abc", true},
		{webFace{family: "Noto Sans"}, plain, "ψ", true},
		{webFace{family: "My Font", ranges: latin}, &custom, "abc", true},
		{webFace{family: "Other", ranges: latin}, &custom, "abc", false},
	}
	for i, test := range tests {
		tx := rich.NewText(test.sty, []rune(test.text))
		if got := textUsesFaces(tx, `"My Font", serif`, []webFace{test.face}); got != test.want {
			t.Errorf("%d: %+v %q: got %v, want %v", i, test.face, test.text, got, test.want)
		}
	}
}
