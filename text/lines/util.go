// Copyright (c) 2020, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lines

import (
	"bufio"
	"bytes"
	"io"
	"log/slog"
	"os"
	"strings"

	"cogentcore.org/core/base/indent"
	"cogentcore.org/core/text/parse"
	"cogentcore.org/core/text/parse/lexer"
	"cogentcore.org/core/text/runes"
	"cogentcore.org/core/text/textpos"
	"cogentcore.org/core/text/token"
)

// BytesToLineStrings returns []string lines from []byte input.
// If addNewLn is true, each string line has a \n appended at end.
func BytesToLineStrings(txt []byte, addNewLn bool) []string {
	lns := bytes.Split(txt, []byte("\n"))
	nl := len(lns)
	if nl == 0 {
		return nil
	}
	str := make([]string, nl)
	for i, l := range lns {
		str[i] = string(l)
		if addNewLn {
			str[i] += "\n"
		}
	}
	return str
}

// StringLinesToByteLines returns [][]byte lines from []string lines
func StringLinesToByteLines(str []string) [][]byte {
	nl := len(str)
	bl := make([][]byte, nl)
	for i, s := range str {
		bl[i] = []byte(s)
	}
	return bl
}

// FileBytes returns the bytes of given file.
func FileBytes(fpath string) ([]byte, error) {
	fp, err := os.Open(fpath)
	if err != nil {
		slog.Error(err.Error())
		return nil, err
	}
	txt, err := io.ReadAll(fp)
	fp.Close()
	if err != nil {
		slog.Error(err.Error())
		return nil, err
	}
	return txt, nil
}

// FileRegionBytes returns the bytes of given file within given
// start / end lines, either of which might be 0 (in which case full file
// is returned).
// If preComments is true, it also automatically includes any comments
// that might exist just prior to the start line if stLn is > 0, going back
// a maximum of lnBack lines.
func FileRegionBytes(fpath string, stLn, edLn int, preComments bool, lnBack int) []byte {
	txt, err := FileBytes(fpath)
	if err != nil {
		return nil
	}
	if stLn == 0 && edLn == 0 {
		return txt
	}
	lns := bytes.Split(txt, []byte("\n"))
	nln := len(lns)

	if edLn > 0 && edLn > stLn && edLn < nln {
		el := min(edLn+1, nln-1)
		lns = lns[:el]
	}
	if preComments && stLn > 0 && stLn < nln {
		comLn, comSt, comEd := KnownComments(fpath)
		stLn = PreCommentStart(lns, stLn, comLn, comSt, comEd, lnBack)
	}

	if stLn > 0 && stLn < len(lns) {
		lns = lns[stLn:]
	}
	txt = bytes.Join(lns, []byte("\n"))
	txt = append(txt, '\n')
	return txt
}

// PreCommentStart returns the starting line for comment line(s) that just
// precede the given stLn line number within the given lines of bytes,
// using the given line-level and block start / end comment chars.
// returns stLn if nothing found.  Only looks back a total of lnBack lines.
func PreCommentStart(lns [][]byte, stLn int, comLn, comSt, comEd string, lnBack int) int {
	comLnb := []byte(strings.TrimSpace(comLn))
	comStb := []byte(strings.TrimSpace(comSt))
	comEdb := []byte(strings.TrimSpace(comEd))
	nback := 0
	gotEd := false
	for i := stLn - 1; i >= 0; i-- {
		l := lns[i]
		fl := bytes.Fields(l)
		if len(fl) == 0 {
			stLn = i + 1
			break
		}
		if !gotEd {
			for _, ff := range fl {
				if bytes.Equal(ff, comEdb) {
					gotEd = true
					break
				}
			}
			if gotEd {
				continue
			}
		}
		if bytes.Equal(fl[0], comStb) {
			stLn = i
			break
		}
		if !bytes.Equal(fl[0], comLnb) && !gotEd {
			stLn = i + 1
			break
		}
		nback++
		if nback > lnBack {
			stLn = i
			break
		}
	}
	return stLn
}

// CountWordsLinesRegion counts the number of words (aka Fields, space-separated strings)
// and lines in given region of source (lines = 1 + End.Line - Start.Line)
func CountWordsLinesRegion(src [][]rune, reg textpos.Region) (words, lines int) {
	lns := len(src)
	mx := min(lns-1, reg.End.Line)
	for ln := reg.Start.Line; ln <= mx; ln++ {
		sln := src[ln]
		if ln == reg.Start.Line {
			sln = sln[reg.Start.Char:]
		} else if ln == reg.End.Line {
			sln = sln[:reg.End.Char]
		}
		flds := strings.Fields(string(sln))
		words += len(flds)
	}
	lines = 1 + (reg.End.Line - reg.Start.Line)
	return
}

// CountWordsLines counts the number of words (aka Fields, space-separated strings)
// and lines given io.Reader input
func CountWordsLines(reader io.Reader) (words, lines int) {
	scan := bufio.NewScanner(reader)
	for scan.Scan() {
		flds := bytes.Fields(scan.Bytes())
		words += len(flds)
		lines++
	}
	return
}

////////   Indenting

// see parse/lexer/indent.go for support functions

// indentLine indents line by given number of tab stops, using tabs or spaces,
// for given tab size (if using spaces) -- either inserts or deletes to reach target.
// Returns edit record for any change.
func (ls *Lines) indentLine(ln, ind int) *textpos.Edit {
	tabSz := ls.Settings.TabSize
	ichr := indent.Tab
	if ls.Settings.SpaceIndent {
		ichr = indent.Space
	}
	curind, _ := lexer.LineIndent(ls.lines[ln], tabSz)
	if ind > curind {
		txt := runes.SetFromBytes([]rune{}, indent.Bytes(ichr, ind-curind, tabSz))
		return ls.insertText(textpos.Pos{Line: ln}, txt)
	} else if ind < curind {
		spos := indent.Len(ichr, ind, tabSz)
		cpos := indent.Len(ichr, curind, tabSz)
		return ls.deleteText(textpos.Pos{Line: ln, Char: spos}, textpos.Pos{Line: ln, Char: cpos})
	}
	return nil
}

// autoIndent indents given line to the level of the prior line, adjusted
// appropriately if the current line starts with one of the given un-indent
// strings, or the prior line ends with one of the given indent strings.
// Returns any edit that took place (could be nil), along with the auto-indented
// level and character position for the indent of the current line.
func (ls *Lines) autoIndent(ln int) (tbe *textpos.Edit, indLev, chPos int) {
	tabSz := ls.Settings.TabSize
	lp, _ := parse.LanguageSupport.Properties(ls.parseState.Known)
	var pInd, delInd int
	if lp != nil && lp.Lang != nil {
		pInd, delInd, _, _ = lp.Lang.IndentLine(&ls.parseState, ls.lines, ls.hiTags, ln, tabSz)
	} else {
		pInd, delInd, _, _ = lexer.BracketIndentLine(ls.lines, ls.hiTags, ln, tabSz)
	}
	ichr := ls.Settings.IndentChar()
	indLev = max(pInd+delInd, 0)
	chPos = indent.Len(ichr, indLev, tabSz)
	tbe = ls.indentLine(ln, indLev)
	return
}

// autoIndentRegion does auto-indent over given region; end is *exclusive*
func (ls *Lines) autoIndentRegion(start, end int) {
	end = min(ls.numLines(), end)
	for ln := start; ln < end; ln++ {
		ls.autoIndent(ln)
	}
}

// commentStart returns the char index where the comment
// starts on given line, -1 if no comment.
func (ls *Lines) commentStart(ln int) int {
	if !ls.isValidLine(ln) {
		return -1
	}
	comst, _ := ls.Settings.CommentStrings()
	if comst == "" {
		return -1
	}
	return runes.Index(ls.lines[ln], []rune(comst))
}

// inComment returns true if the given text position is within
// a commented region.
func (ls *Lines) inComment(pos textpos.Pos) bool {
	if ls.inTokenSubCat(pos, token.Comment) {
		return true
	}
	cs := ls.commentStart(pos.Line)
	if cs < 0 {
		return false
	}
	return pos.Char > cs
}

// lineCommented returns true if the given line is a full-comment
// line (i.e., starts with a comment).
func (ls *Lines) lineCommented(ln int) bool {
	if !ls.isValidLine(ln) {
		return false
	}
	tags := ls.hiTags[ln]
	if len(tags) == 0 {
		return false
	}
	return tags[0].Token.Token.InCat(token.Comment)
}

// commentRegion inserts comment marker on given lines; end is *exclusive*.
func (ls *Lines) commentRegion(start, end int) {
	tabSz := ls.Settings.TabSize
	ch := 0
	ind, _ := lexer.LineIndent(ls.lines[start], tabSz)
	if ind > 0 {
		if ls.Settings.SpaceIndent {
			ch = ls.Settings.TabSize * ind
		} else {
			ch = ind
		}
	}

	comst, comed := ls.Settings.CommentStrings()
	if comst == "" {
		// log.Printf("text.Lines: attempt to comment region without any comment syntax defined")
		comst = "// "
		return
	}

	eln := min(ls.numLines(), end)
	ncom := 0
	nln := eln - start
	for ln := start; ln < eln; ln++ {
		if ls.lineCommented(ln) {
			ncom++
		}
	}
	trgln := max(nln-2, 1)
	doCom := true
	if ncom >= trgln {
		doCom = false
	}
	rcomst := []rune(comst)
	rcomed := []rune(comed)

	for ln := start; ln < eln; ln++ {
		if doCom {
			ipos, ok := ls.validCharPos(textpos.Pos{Line: ln, Char: ch})
			if ok {
				ls.insertText(ipos, rcomst)
				if comed != "" {
					lln := len(ls.lines[ln]) // automatically ok
					ls.insertText(textpos.Pos{Line: ln, Char: lln}, rcomed)
				}
			}
		} else {
			idx := ls.commentStart(ln)
			if idx >= 0 {
				ls.deleteText(textpos.Pos{Line: ln, Char: idx}, textpos.Pos{Line: ln, Char: idx + len(comst)})
			}
			if comed != "" {
				idx := runes.IndexFold(ls.lines[ln], []rune(comed))
				if idx >= 0 {
					ls.deleteText(textpos.Pos{Line: ln, Char: idx}, textpos.Pos{Line: ln, Char: idx + len(comed)})
				}
			}
		}
	}
}

// joinParaLines merges sequences of lines with hard returns forming paragraphs,
// separated by blank lines, into a single line per paragraph,
// within the given line regions; endLine is *inclusive*.
func (ls *Lines) joinParaLines(startLine, endLine int) {
	// current end of region being joined == last blank line
	curEd := endLine
	for ln := endLine; ln >= startLine; ln-- { // reverse order
		lr := ls.lines[ln]
		lrt := runes.TrimSpace(lr)
		if len(lrt) == 0 || ln == startLine {
			if ln < curEd-1 {
				stp := textpos.Pos{Line: ln + 1}
				if ln == startLine {
					stp.Line--
				}
				ep := textpos.Pos{Line: curEd - 1}
				if curEd == endLine {
					ep.Line = curEd
				}
				eln := ls.lines[ep.Line]
				ep.Char = len(eln)
				trt := runes.Join(ls.lines[stp.Line:ep.Line+1], []rune(" "))
				ls.replaceText(stp, ep, stp, string(trt), ReplaceNoMatchCase)
			}
			curEd = ln
		}
	}
}

// tabsToSpacesLine replaces tabs with spaces in the given line.
func (ls *Lines) tabsToSpacesLine(ln int) {
	tabSz := ls.Settings.TabSize

	lr := ls.lines[ln]
	st := textpos.Pos{Line: ln}
	ed := textpos.Pos{Line: ln}
	i := 0
	for {
		if i >= len(lr) {
			break
		}
		r := lr[i]
		if r == '\t' {
			po := i % tabSz
			nspc := tabSz - po
			st.Char = i
			ed.Char = i + 1
			ls.replaceText(st, ed, st, indent.Spaces(1, nspc), ReplaceNoMatchCase)
			i += nspc
			lr = ls.lines[ln]
		} else {
			i++
		}
	}
}

// tabsToSpaces replaces tabs with spaces over given region; end is *exclusive*.
func (ls *Lines) tabsToSpaces(start, end int) {
	end = min(ls.numLines(), end)
	for ln := start; ln < end; ln++ {
		ls.tabsToSpacesLine(ln)
	}
}

// spacesToTabsLine replaces spaces with tabs in the given line.
func (ls *Lines) spacesToTabsLine(ln int) {
	tabSz := ls.Settings.TabSize

	lr := ls.lines[ln]
	st := textpos.Pos{Line: ln}
	ed := textpos.Pos{Line: ln}
	i := 0
	nspc := 0
	for {
		if i >= len(lr) {
			break
		}
		r := lr[i]
		if r == ' ' {
			nspc++
			if nspc == tabSz {
				st.Char = i - (tabSz - 1)
				ed.Char = i + 1
				ls.replaceText(st, ed, st, "\t", ReplaceNoMatchCase)
				i -= tabSz - 1
				lr = ls.lines[ln]
				nspc = 0
			} else {
				i++
			}
		} else {
			nspc = 0
			i++
		}
	}
}

// spacesToTabs replaces tabs with spaces over given region; end is *exclusive*
func (ls *Lines) spacesToTabs(start, end int) {
	end = min(ls.numLines(), end)
	for ln := start; ln < end; ln++ {
		ls.spacesToTabsLine(ln)
	}
}
