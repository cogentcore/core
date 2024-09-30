// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tensor

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"

	"cogentcore.org/core/base/metadata"
	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/core"
	"cogentcore.org/core/math32"
)

// Delim are standard CSV delimiter options (Tab, Comma, Space)
type Delims int32 //enums:enum

const (
	// Tab is the tab rune delimiter, for TSV tab separated values
	Tab Delims = iota

	// Comma is the comma rune delimiter, for CSV comma separated values
	Comma

	// Space is the space rune delimiter, for SSV space separated value
	Space

	// Detect is used during reading a file -- reads the first line and detects tabs or commas
	Detect
)

func (dl Delims) Rune() rune {
	switch dl {
	case Tab:
		return '\t'
	case Comma:
		return ','
	case Space:
		return ' '
	}
	return '\t'
}

// SetPrecision sets the "precision" metadata value that determines
// the precision to use in writing floating point numbers to files.
func SetPrecision(md metadata.Data, prec int) {
	md.Set("precision", prec)
}

// Precision gets the "precision" metadata value that determines
// the precision to use in writing floating point numbers to files.
// returns an error if not set.
func Precision(md metadata.Data) (int, error) {
	return metadata.Get[int](md, "precision")
}

// SaveCSV writes a tensor to a comma-separated-values (CSV) file
// (where comma = any delimiter, specified in the delim arg).
// Outer-most dims are rows in the file, and inner-most is column --
// Reading just grabs all values and doesn't care about shape.
func SaveCSV(tsr Tensor, filename core.Filename, delim Delims) error {
	fp, err := os.Create(string(filename))
	defer fp.Close()
	if err != nil {
		log.Println(err)
		return err
	}
	WriteCSV(tsr, fp, delim)
	return nil
}

// OpenCSV reads a tensor from a comma-separated-values (CSV) file
// (where comma = any delimiter, specified in the delim arg),
// using the Go standard encoding/csv reader conforming
// to the official CSV standard.
// Reads all values and assigns as many as fit.
func OpenCSV(tsr Tensor, filename core.Filename, delim Delims) error {
	fp, err := os.Open(string(filename))
	defer fp.Close()
	if err != nil {
		log.Println(err)
		return err
	}
	return ReadCSV(tsr, fp, delim)
}

//////////////////////////////////////////////////////////////////////////
// WriteCSV

// WriteCSV writes a tensor to a comma-separated-values (CSV) file
// (where comma = any delimiter, specified in the delim arg).
// Outer-most dims are rows in the file, and inner-most is column --
// Reading just grabs all values and doesn't care about shape.
func WriteCSV(tsr Tensor, w io.Writer, delim Delims) error {
	prec := -1
	if ps, err := Precision(*tsr.Metadata()); err == nil {
		prec = ps
	}
	cw := csv.NewWriter(w)
	cw.Comma = delim.Rune()
	nrow := tsr.DimSize(0)
	nin := tsr.Len() / nrow
	rec := make([]string, nin)
	str := tsr.IsString()
	for ri := 0; ri < nrow; ri++ {
		for ci := 0; ci < nin; ci++ {
			idx := ri*nin + ci
			if str {
				rec[ci] = tsr.String1D(idx)
			} else {
				rec[ci] = strconv.FormatFloat(tsr.Float1D(idx), 'g', prec, 64)
			}
		}
		err := cw.Write(rec)
		if err != nil {
			log.Println(err)
			return err
		}
	}
	cw.Flush()
	return nil
}

// ReadCSV reads a tensor from a comma-separated-values (CSV) file
// (where comma = any delimiter, specified in the delim arg),
// using the Go standard encoding/csv reader conforming
// to the official CSV standard.
// Reads all values and assigns as many as fit.
func ReadCSV(tsr Tensor, r io.Reader, delim Delims) error {
	cr := csv.NewReader(r)
	cr.Comma = delim.Rune()
	rec, err := cr.ReadAll() // todo: lazy, avoid resizing
	if err != nil || len(rec) == 0 {
		return err
	}
	rows := len(rec)
	cols := len(rec[0])
	sz := tsr.Len()
	idx := 0
	for ri := 0; ri < rows; ri++ {
		for ci := 0; ci < cols; ci++ {
			str := rec[ri][ci]
			tsr.SetString1D(str, idx)
			idx++
			if idx >= sz {
				goto done
			}
		}
	}
done:
	return nil
}

func label(nm string, sh *Shape) string {
	if nm != "" {
		nm += " " + sh.String()
	} else {
		nm = sh.String()
	}
	return nm
}

// Sprintf returns a string representation of the given tensor,
// with a maximum length of as given: output is terminated
// when it exceeds that length. If maxLen = 0, [MaxSprintLength] is used.
// The format is the per-element format string, which should include
// any delimiter or spacing between elements (which will apply to last
// element too).  If empty it uses compact defaults for the data type.
func Sprintf(tsr Tensor, maxLen int, format string) string {
	if maxLen == 0 {
		maxLen = MaxSprintLength
	}
	colWd := 1 // column width in tabs
	defFmt := format == ""
	isint := false
	if defFmt {
		switch {
		case tsr.IsString():
			format = "%15s\t"
		case reflectx.KindIsInt(tsr.DataType()):
			isint = true
			format = "%7g\t"
		default:
			format = "%7.3g\t"
		}
	}
	if tsr.NumDims() == 1 && tsr.DimSize(0) == 1 { // scalar special case
		if tsr.IsString() {
			return fmt.Sprintf(format, tsr.String1D(0))
		} else {
			return fmt.Sprintf(format, tsr.Float1D(0))
		}
	}
	mxwd := 0
	n := min(tsr.Len(), maxLen)
	for i := range n {
		s := ""
		if tsr.IsString() {
			s = fmt.Sprintf(format, tsr.String1D(i))
		} else {
			s = fmt.Sprintf(format, tsr.Float1D(i))
		}
		if len(s) > mxwd {
			mxwd = len(s)
		}
	}
	colWd = int(math32.IntMultipleGE(float32(mxwd), 8)) / 8
	if colWd > 1 && !tsr.IsString() && defFmt { // should be 2
		if isint {
			format = "%15g\t"
		} else {
			format = "%15.7g\t"
		}
	}
	sh := tsr.Shape()
	oddRow := false
	rows, cols, _, _ := Projection2DShape(sh, oddRow)
	var b strings.Builder
	b.WriteString(tsr.Label())
	noidx := false
	if tsr.NumDims() == 1 && tsr.Len() < 8 {
		b.WriteString(" ")
		noidx = true
	} else {
		b.WriteString("\n")
	}
	if !noidx && tsr.NumDims() > 1 && cols > 1 {
		b.WriteString("\t")
		for c := 0; c < cols; c++ {
			_, cc := Projection2DCoords(sh, oddRow, 0, c)
			b.WriteString(fmt.Sprintf("%v:\t", cc))
			if colWd > 1 {
				b.WriteString(strings.Repeat("\t", colWd-1))
			}
		}
		b.WriteString("\n")
	}
	// todo: could do something better for large numbers of columns..
	ctr := 0
	for r := range rows {
		rc, _ := Projection2DCoords(sh, oddRow, r, 0)
		if !noidx {
			b.WriteString(fmt.Sprintf("%v:\t", rc))
		}
		ri := r
		for c := 0; c < cols; c++ {
			s := ""
			if tsr.IsString() {
				s = fmt.Sprintf(format, Projection2DString(tsr, oddRow, ri, c))
			} else {
				s = fmt.Sprintf(format, Projection2DValue(tsr, oddRow, ri, c))
			}
			b.WriteString(s)
			nt := int(math32.IntMultipleGE(float32(len(s)), 8)) / 8
			if nt < colWd {
				b.WriteString(strings.Repeat("\t", nt-colWd))
			}
		}
		b.WriteString("\n")
		ctr += cols
		if ctr > maxLen {
			b.WriteString("...\n")
			break
		}
	}
	return b.String()
}
