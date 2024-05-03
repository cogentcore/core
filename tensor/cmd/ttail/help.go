// Copyright (c) 2020, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import "github.com/nsf/termbox-go"

func (tm *Term) Help() {
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
	ln := 0
	tm.DrawStringDef(0, ln, "Key(s)  Function")
	ln++
	tm.DrawStringDef(0, ln, "--------------------------------------------------------------")
	ln++
	tm.DrawStringDef(0, ln, "spc,n   page down")
	ln++
	tm.DrawStringDef(0, ln, "p       page up")
	ln++
	tm.DrawStringDef(0, ln, "f       scroll right-hand panel to the right")
	ln++
	tm.DrawStringDef(0, ln, "b       scroll right-hand panel to the left")
	ln++
	tm.DrawStringDef(0, ln, "w       widen the left-hand panel of columns")
	ln++
	tm.DrawStringDef(0, ln, "s       shrink the left-hand panel of columns")
	ln++
	tm.DrawStringDef(0, ln, "t       toggle tail-mode (auto updating as file grows) on/off")
	ln++
	tm.DrawStringDef(0, ln, "a       jump to top")
	ln++
	tm.DrawStringDef(0, ln, "e       jump to end")
	ln++
	tm.DrawStringDef(0, ln, "v       rotate down through the list of files (if not all displayed)")
	ln++
	tm.DrawStringDef(0, ln, "u       rotate up through the list of files (if not all displayed)")
	ln++
	tm.DrawStringDef(0, ln, "m       more minimum lines per file -- increase amount shown of each file")
	ln++
	tm.DrawStringDef(0, ln, "l       less minimum lines per file -- decrease amount shown of each file")
	ln++
	tm.DrawStringDef(0, ln, "d       toggle display of file names")
	ln++
	tm.DrawStringDef(0, ln, "c       toggle display of column numbers instead of names")
	ln++
	tm.DrawStringDef(0, ln, "q       quit")
	ln++
	termbox.Flush()
}
