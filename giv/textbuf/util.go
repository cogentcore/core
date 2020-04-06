// Copyright (c) 2020, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textbuf

import (
	"bufio"
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/goki/ki/ints"
)

// BytesToLineStrings returns []string lines
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

// FileBytes returns the bytes of given file.
func FileBytes(fpath string) ([]byte, error) {
	fp, err := os.Open(fpath)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	txt, err := ioutil.ReadAll(fp)
	fp.Close()
	if err != nil {
		log.Println(err)
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
		el := ints.MinInt(edLn+1, nln-1)
		lns = lns[:el]
	}
	if preComments && stLn > 0 && stLn < nln {
		comLn, comSt, comEd := SupportedComments(fpath)
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
// and lines in given region of source (lines = 1 + End.Ln - Start.Ln)
func CountWordsLinesRegion(src [][]rune, reg Region) (words, lines int) {
	lns := len(src)
	mx := ints.MinInt(lns-1, reg.End.Ln)
	for ln := reg.Start.Ln; ln <= mx; ln++ {
		sln := src[ln]
		if ln == reg.Start.Ln {
			sln = sln[reg.Start.Ch:]
		} else if ln == reg.End.Ln {
			sln = sln[:reg.End.Ch]
		}
		flds := strings.Fields(string(sln))
		words += len(flds)
	}
	lines = 1 + (reg.End.Ln - reg.Start.Ln)
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
