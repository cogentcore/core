// Copyright (c) 2020, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"image"
	"sync"

	termbox "github.com/nsf/termbox-go"
)

// Term represents the terminal display -- has all drawing routines
// and all display data.  See Tail for two diff display modes.
type Term struct {

	// size of terminal
	Size image.Point `desc:"size of terminal"`

	// number of fixed (non-scrolling) columns on left
	FixCols int `desc:"number of fixed (non-scrolling) columns on left"`

	// starting column index -- relative to FixCols
	ColSt int `desc:"starting column index -- relative to FixCols"`

	// starting row index -- for !Tail mode
	RowSt int `desc:"starting row index -- for !Tail mode"`

	// row from end -- for Tail mode
	RowFromEnd int `desc:"row from end (relative to RowsPer) -- for Tail mode"`

	// starting index into files (if too many to display)
	FileSt int `desc:"starting index into files (if too many to display)"`

	// number of files to display (if too many to display)
	NFiles int `desc:"number of files to display (if too many to display)"`

	// minimum number of lines per file
	MinLines int `desc:"minimum number of lines per file"`

	// maximum column width (1/4 of term width)
	MaxWd int `desc:"maximum column width (1/4 of term width)"`

	// max number of rows across all files
	MaxRows int `desc:"max number of rows across all files"`

	// number of Y rows per file total: Size.Y / len(TheFiles)
	YPer int `desc:"number of Y rows per file total: Size.Y / len(TheFiles)"`

	// rows of data per file (subtracting header, filename)
	RowsPer int `desc:"rows of data per file (subtracting header, filename)"`

	// if true, print filename
	ShowFName bool `desc:"if true, print filename"`

	// if true, display is synchronized by the last row for each file, and otherwise it is synchronized by the starting row.  Tail also checks for file updates
	Tail bool `desc:"if true, display is synchronized by the last row for each file, and otherwise it is synchronized by the starting row.  Tail also checks for file updates"`

	// display column numbers instead of names
	ColNums bool `desc:"display column numbers instead of names"`

	// draw mutex
	Mu sync.Mutex `desc:"draw mutex"`
}

// TheTerm is the terminal instance
var TheTerm Term

// Draw draws the current terminal display
func (tm *Term) Draw() error {
	tm.Mu.Lock()
	defer tm.Mu.Unlock()

	err := termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
	if err != nil {
		return err
	}

	w, h := termbox.Size()
	tm.Size.X = w
	tm.Size.Y = h
	tm.MaxWd = tm.Size.X / 4

	if tm.MinLines == 0 {
		tm.MinLines = min(5, tm.Size.Y-1)
	}

	nf := len(TheFiles)
	if nf == 0 {
		return fmt.Errorf("No files")
	}
	ysz := tm.Size.Y - 1 // status line
	tm.YPer = ysz / nf
	tm.NFiles = nf

	if tm.YPer < tm.MinLines {
		tm.NFiles = ysz / tm.MinLines
		tm.YPer = tm.MinLines
	}
	if tm.NFiles+tm.FileSt > nf {
		tm.FileSt = max(0, nf-tm.NFiles)
	}

	tm.RowsPer = tm.YPer - 1
	if tm.ShowFName {
		tm.RowsPer--
	}
	sty := 0
	mxrows := 0
	for fi := 0; fi < tm.NFiles; fi++ {
		ffi := tm.FileSt + fi
		if ffi >= nf {
			break
		}
		fl := TheFiles[ffi]
		tm.DrawFile(fl, sty)
		sty += tm.YPer
		mxrows = max(mxrows, fl.Rows)
	}
	tm.MaxRows = mxrows

	tm.StatusLine()

	termbox.Flush()
	return nil
}

// StatusLine renders the status line at bottom
func (tm *Term) StatusLine() {
	pos := tm.RowSt
	if tm.Tail {
		pos = tm.RowFromEnd
	}
	stat := fmt.Sprintf("Tail: %v\tPos: %d\tMaxRows: %d\tNFile: %d\tFileSt: %d\t h = help [spc,n,p,r,f,l,b,w,s,t,a,e,v,u,m,l,c,q]      ", tm.Tail, pos, tm.MaxRows, len(TheFiles), tm.FileSt)
	tm.DrawString(0, tm.Size.Y-1, stat, len(stat), termbox.AttrReverse, termbox.AttrReverse)
}

// NextPage moves down a page
func (tm *Term) NextPage() error {
	if tm.Tail {
		mn := min(-(tm.MaxRows - tm.RowsPer), 0)
		tm.RowFromEnd = min(tm.RowFromEnd+tm.RowsPer, 0)
		tm.RowFromEnd = max(tm.RowFromEnd, mn)
	} else {
		tm.RowSt = min(tm.RowSt+tm.RowsPer, tm.MaxRows-tm.RowsPer)
		tm.RowSt = max(tm.RowSt, 0)
	}
	return tm.Draw()
}

// PrevPage moves up a page
func (tm *Term) PrevPage() error {
	if tm.Tail {
		mn := min(-(tm.MaxRows - tm.RowsPer), 0)
		tm.RowFromEnd = min(tm.RowFromEnd-tm.RowsPer, 0)
		tm.RowFromEnd = max(tm.RowFromEnd, mn)
	} else {
		tm.RowSt = max(tm.RowSt-tm.RowsPer, 0)
		tm.RowSt = min(tm.RowSt, tm.MaxRows-tm.RowsPer)
	}
	return tm.Draw()
}

// NextLine moves down a page
func (tm *Term) NextLine() error {
	if tm.Tail {
		mn := min(-(tm.MaxRows - tm.RowsPer), 0)
		tm.RowFromEnd = min(tm.RowFromEnd+1, 0)
		tm.RowFromEnd = max(tm.RowFromEnd, mn)
	} else {
		tm.RowSt = min(tm.RowSt+1, tm.MaxRows-tm.RowsPer)
		tm.RowSt = max(tm.RowSt, 0)
	}
	return tm.Draw()
}

// PrevLine moves up a page
func (tm *Term) PrevLine() error {
	if tm.Tail {
		mn := min(-(tm.MaxRows - tm.RowsPer), 0)
		tm.RowFromEnd = min(tm.RowFromEnd-1, 0)
		tm.RowFromEnd = max(tm.RowFromEnd, mn)
	} else {
		tm.RowSt = max(tm.RowSt-1, 0)
		tm.RowSt = min(tm.RowSt, tm.MaxRows-tm.RowsPer)
	}
	return tm.Draw()
}

// Top moves to starting row = 0
func (tm *Term) Top() error {
	mn := min(-(tm.MaxRows - tm.RowsPer), 0)
	tm.RowFromEnd = mn
	tm.RowSt = 0
	return tm.Draw()
}

// End moves row start to last position in longest file
func (tm *Term) End() error {
	mx := max(tm.MaxRows-tm.RowsPer, 0)
	tm.RowFromEnd = 0
	tm.RowSt = mx
	return tm.Draw()
}

// ScrollRight scrolls columns to right
func (tm *Term) ScrollRight() error {
	tm.ColSt++ // no obvious max
	return tm.Draw()
}

// ScrollLeft scrolls columns to left
func (tm *Term) ScrollLeft() error {
	tm.ColSt = max(tm.ColSt-1, 0)
	return tm.Draw()
}

// FixRight increases number of fixed columns
func (tm *Term) FixRight() error {
	tm.FixCols++ // no obvious max
	return tm.Draw()
}

// FixLeft decreases number of fixed columns
func (tm *Term) FixLeft() error {
	tm.FixCols = max(tm.FixCols-1, 0)
	return tm.Draw()
}

// FilesNext moves down in list of files to display
func (tm *Term) FilesNext() error {
	nf := len(TheFiles)
	tm.FileSt = min(tm.FileSt+1, nf-tm.NFiles)
	tm.FileSt = max(tm.FileSt, 0)
	return tm.Draw()
}

// FilesPrev moves up in list of files to display
func (tm *Term) FilesPrev() error {
	nf := len(TheFiles)
	tm.FileSt = max(tm.FileSt-1, 0)
	tm.FileSt = min(tm.FileSt, nf-tm.NFiles)
	return tm.Draw()
}

// MoreMinLines increases minimum number of lines per file
func (tm *Term) MoreMinLines() error {
	tm.MinLines++
	return tm.Draw()
}

// LessMinLines decreases minimum number of lines per file
func (tm *Term) LessMinLines() error {
	tm.MinLines--
	tm.MinLines = max(3, tm.MinLines)
	return tm.Draw()
}

// ToggleNames toggles whether file names are shown
func (tm *Term) ToggleNames() error {
	tm.ShowFName = !tm.ShowFName
	return tm.Draw()
}

// ToggleTail toggles Tail mode
func (tm *Term) ToggleTail() error {
	tm.Tail = !tm.Tail
	return tm.Draw()
}

// ToggleColNums toggles ColNums mode
func (tm *Term) ToggleColNums() error {
	tm.ColNums = !tm.ColNums
	return tm.Draw()
}

// TailCheck does tail update check -- returns true if updated
func (tm *Term) TailCheck() bool {
	if !tm.Tail {
		return false
	}
	tm.Mu.Lock()
	update := TheFiles.CheckUpdates()
	tm.Mu.Unlock()
	if !update {
		return false
	}
	tm.Draw()
	return true
}

// DrawFile draws one file, starting at given y offset
func (tm *Term) DrawFile(fl *File, sty int) {
	tdo := (fl.Rows - tm.RowsPer) + tm.RowFromEnd // tail data offset for this file
	tdo = max(0, tdo)
	rst := min(tm.RowSt, fl.Rows-tm.RowsPer)
	rst = max(0, rst)
	stx := 0
	for ci, hs := range fl.Heads {
		if !(ci < tm.FixCols || ci >= tm.FixCols+tm.ColSt) {
			continue
		}
		my := sty
		if tm.ShowFName {
			tm.DrawString(0, my, fl.FName, tm.Size.X, termbox.AttrReverse, termbox.AttrReverse)
			my++
		}
		wmax := min(fl.Widths[ci], tm.MaxWd)
		if tm.ColNums {
			hs = fmt.Sprintf("%d", ci)
		}
		tm.DrawString(stx, my, hs, wmax, termbox.AttrReverse, termbox.AttrReverse)
		if ci == tm.FixCols-1 {
			tm.DrawString(stx+wmax+1, my, "|", 1, termbox.AttrReverse, termbox.AttrReverse)
		}
		my++
		for ri := 0; ri < tm.RowsPer; ri++ {
			var di int
			if tm.Tail {
				di = tdo + ri
			} else {
				di = rst + ri
			}
			if di >= len(fl.Data) || di < 0 {
				continue
			}
			dr := fl.Data[di]
			if ci >= len(dr) {
				break
			}
			ds := dr[ci]
			tm.DrawString(stx, my+ri, ds, wmax, termbox.ColorDefault, termbox.ColorDefault)
			if ci == tm.FixCols-1 {
				tm.DrawString(stx+wmax+1, my+ri, "|", 1, termbox.AttrReverse, termbox.AttrReverse)
			}
		}
		stx += wmax + 1
		if ci == tm.FixCols-1 {
			stx += 2
		}
		if stx >= tm.Size.X {
			break
		}
	}
}

// DrawStringDef draws string at given position, using default colors
func (tm *Term) DrawStringDef(x, y int, s string) {
	tm.DrawString(x, y, s, tm.Size.X, termbox.ColorDefault, termbox.ColorDefault)
}

// DrawString draws string at given position, using given attributes
func (tm *Term) DrawString(x, y int, s string, maxlen int, fg, bg termbox.Attribute) {
	if y >= tm.Size.Y || y < 0 {
		return
	}
	for i, r := range s {
		if i >= maxlen {
			break
		}
		xp := x + i
		if xp >= tm.Size.X || xp < 0 {
			continue
		}
		termbox.SetCell(xp, y, r, fg, bg)
	}
}
