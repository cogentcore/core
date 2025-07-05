// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package keymap

import (
	"strings"

	"cogentcore.org/core/events/key"
)

// MarkdownDoc generates a markdown table of all the key mappings
func (km *Maps) MarkdownDoc() string { //types:add
	mods := []string{"", "Shift", "Control", "Control+Shift", "Meta", "Meta+Shift", "Alt", "Alt+Shift", "Control+Alt", "Meta+Alt"}

	var b strings.Builder

	fmap := make([][][]string, len(*km)) // km, kf, ch
	for i := range *km {
		fmap[i] = make([][]string, FunctionsN)
	}

	// By function
	b.WriteString("### By function\n\n")

	b.WriteString("| Function                         ")
	for _, m := range *km {
		b.WriteString("| `" + m.Name + "` ")
	}
	b.WriteString("|\n")
	b.WriteString("| ---------------------------- ")
	for _, m := range *km {
		b.WriteString("| " + strings.Repeat("-", len(m.Name)+2) + " ")
	}
	b.WriteString("|\n")

	for kf := MoveUp; kf < FunctionsN; kf++ {
		b.WriteString("| " + kf.String() + " ")
		for mi, m := range *km {
			f := fmap[mi][kf]
			b.WriteString("| ")
			if len(f) > 0 {
				for fi, fs := range f {
					b.WriteString(fs)
					if fi < len(f)-1 {
						b.WriteString(", ")
					} else {
						b.WriteString(" ")
					}
				}
			} else {
				b.WriteString(strings.Repeat(" ", len(m.Name)+2) + " ")
			}
		}
		b.WriteString("|\n")
	}
	b.WriteString("\n\n")

	for _, md := range mods {
		if md == "" {
			b.WriteString("### No Modifiers\n\n")
		} else {
			b.WriteString("### " + md + "\n\n")
		}
		b.WriteString("| Key                          ")
		for _, m := range *km {
			b.WriteString("| `" + m.Name + "` ")
		}
		b.WriteString("|\n")
		b.WriteString("| ---------------------------- ")
		for _, m := range *km {
			b.WriteString("| " + strings.Repeat("-", len(m.Name)+2) + " ")
		}
		b.WriteString("|\n")

		for cd := key.CodeA; cd < key.CodesN; cd++ {
			ch := key.Chord(md + "+" + cd.String())
			if md == "" {
				ch = key.Chord(cd.String())
			}
			has := false
			for _, m := range *km {
				_, ok := m.Map[ch]
				if ok {
					has = true
					break
				}
			}
			if !has {
				continue
			}
			b.WriteString("| " + string(ch) + " ")
			for mi, m := range *km {
				kf, ok := m.Map[ch]
				if ok {
					b.WriteString("| " + kf.String() + " ")
					fmap[mi][kf] = append(fmap[mi][kf], string(ch))
				} else {
					b.WriteString("| " + strings.Repeat(" ", len(m.Name)+2) + " ")
				}
			}
			b.WriteString("|\n")
		}
		b.WriteString("\n\n")
	}

	return b.String()
}
