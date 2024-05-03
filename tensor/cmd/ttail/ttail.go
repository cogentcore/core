// Copyright (c) 2020, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/nsf/termbox-go"
)

func main() {
	err := termbox.Init()
	if err != nil {
		log.Println(err)
		panic(err)
	}
	defer termbox.Close()

	TheFiles.Open(os.Args[1:])

	nf := len(TheFiles)
	if nf == 0 {
		fmt.Printf("usage: etail <filename>...  (space separated)\n")
		return
	}

	if nf > 1 {
		TheTerm.ShowFName = true
	}

	err = TheTerm.ToggleTail() // start in tail mode
	if err != nil {
		log.Println(err)
		panic(err)
	}

	Tailer := time.NewTicker(time.Duration(500) * time.Millisecond)
	go func() {
		for {
			<-Tailer.C
			TheTerm.TailCheck()
		}
	}()

loop:
	for {
		switch ev := termbox.PollEvent(); ev.Type {
		case termbox.EventKey:
			switch {
			case ev.Key == termbox.KeyEsc || ev.Ch == 'Q' || ev.Ch == 'q':
				break loop
			case ev.Ch == ' ' || ev.Ch == 'n' || ev.Ch == 'N' || ev.Key == termbox.KeyPgdn || ev.Key == termbox.KeySpace:
				TheTerm.NextPage()
			case ev.Ch == 'p' || ev.Ch == 'P' || ev.Key == termbox.KeyPgup:
				TheTerm.PrevPage()
			case ev.Key == termbox.KeyArrowDown:
				TheTerm.NextLine()
			case ev.Key == termbox.KeyArrowUp:
				TheTerm.PrevLine()
			case ev.Ch == 'f' || ev.Ch == 'F' || ev.Key == termbox.KeyArrowRight:
				TheTerm.ScrollRight()
			case ev.Ch == 'b' || ev.Ch == 'B' || ev.Key == termbox.KeyArrowLeft:
				TheTerm.ScrollLeft()
			case ev.Ch == 'a' || ev.Ch == 'A' || ev.Key == termbox.KeyHome:
				TheTerm.Top()
			case ev.Ch == 'e' || ev.Ch == 'E' || ev.Key == termbox.KeyEnd:
				TheTerm.End()
			case ev.Ch == 'w' || ev.Ch == 'W':
				TheTerm.FixRight()
			case ev.Ch == 's' || ev.Ch == 'S':
				TheTerm.FixLeft()
			case ev.Ch == 'v' || ev.Ch == 'V':
				TheTerm.FilesNext()
			case ev.Ch == 'u' || ev.Ch == 'U':
				TheTerm.FilesPrev()
			case ev.Ch == 'm' || ev.Ch == 'M':
				TheTerm.MoreMinLines()
			case ev.Ch == 'l' || ev.Ch == 'L':
				TheTerm.LessMinLines()
			case ev.Ch == 'd' || ev.Ch == 'D':
				TheTerm.ToggleNames()
			case ev.Ch == 't' || ev.Ch == 'T':
				TheTerm.ToggleTail()
			case ev.Ch == 'c' || ev.Ch == 'C':
				TheTerm.ToggleColNums()
			case ev.Ch == 'h' || ev.Ch == 'H':
				TheTerm.Help()
			}
		case termbox.EventResize:
			TheTerm.Draw()
		}
	}
}
