// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tensor

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"

	"cogentcore.org/core/base/fsx"
	"cogentcore.org/core/base/metadata"
	"cogentcore.org/core/base/reflectx"
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
func SetPrecision(obj any, prec int) {
	metadata.SetTo(obj, "Precision", prec)
}

// Precision gets the "precision" metadata value that determines
// the precision to use in writing floating point numbers to files.
// returns an error if not set.
func Precision(obj any) (int, error) {
	return metadata.GetFrom[int](obj, "Precision")
}

// SaveCSV writes a tensor to a comma-separated-values (CSV) file
// (where comma = any delimiter, specified in the delim arg).
// Outer-most dims are rows in the file, and inner-most is column --
// Reading just grabs all values and doesn't care about shape.
func SaveCSV(tsr Tensor, filename fsx.Filename, delim Delims) error {
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
func OpenCSV(tsr Tensor, filename fsx.Filename, delim Delims) error {
	fp, err := os.Open(string(filename))
	defer fp.Close()
	if err != nil {
		log.Println(err)
		return err
	}
	return ReadCSV(tsr, fp, delim)
}

//////// WriteCSV

// WriteCSV writes a tensor to a comma-separated-values (CSV) file
// (where comma = any delimiter, specified in the delim arg).
// Outer-most dims are rows in the file, and inner-most is column --
// Reading just grabs all values and doesn't care about shape.
func WriteCSV(tsr Tensor, w io.Writer, delim Delims) error {
	prec := -1
	if ps, err := Precision(tsr); err == nil {
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

// padToLength returns the given string with added spaces
// to pad out to target length. at least 1 space will be added
func padToLength(str string, tlen int) string {
	slen := len(str)
	if slen < tlen-1 {
		return str + strings.Repeat(" ", tlen-slen)
	}
	return str + " "
}

// prepadToLength returns the given string with added spaces
// to pad out to target length at start (for numbers).
// at least 1 space will be added
func prepadToLength(str string, tlen int) string {
	slen := len(str)
	if slen < tlen-1 {
		return strings.Repeat(" ", tlen-slen-1) + str + " "
	}
	return str + " "
}

// MaxPrintLineWidth is the maximum line width in characters
// to generate for tensor Sprintf function.
var MaxPrintLineWidth = 80

// Sprintf returns a string representation of the given tensor,
// with a maximum length of as given: output is terminated
// when it exceeds that length. If maxLen = 0, [MaxSprintLength] is used.
// The format is the per-element format string.
// If empty it uses general %g for number or %s for string.
func Sprintf(format string, tsr Tensor, maxLen int) string {
	if maxLen == 0 {
		maxLen = MaxSprintLength
	}
	defFmt := format == ""
	if defFmt {
		switch {
		case tsr.IsString():
			format = "%s"
		case reflectx.KindIsInt(tsr.DataType()):
			format = "%.10g"
		default:
			format = "%.10g"
		}
	}
	nd := tsr.NumDims()
	if nd == 1 && tsr.DimSize(0) == 1 { // scalar special case
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
	onedRow := false
	shp := tsr.Shape()
	rowShape, colShape, _, colIdxs := Projection2DDimShapes(shp, onedRow)
	rows, cols, _, _ := Projection2DShape(shp, onedRow)

	rowWd := len(rowShape.String()) + 1
	legend := ""
	if nd > 2 {
		leg := bytes.Repeat([]byte("r "), nd)
		for _, i := range colIdxs {
			leg[2*i] = 'c'
		}
		legend = "[" + string(leg[:len(leg)-1]) + "]"
	}
	rowWd = max(rowWd, len(legend)+1)
	hdrWd := len(colShape.String()) + 1
	colWd := mxwd + 1

	var b strings.Builder
	b.WriteString(tsr.Label())
	noidx := false
	if tsr.NumDims() == 1 {
		b.WriteString(" ")
		rowWd = len(tsr.Label()) + 1
		noidx = true
	} else {
		b.WriteString("\n")
	}
	if !noidx && nd > 1 && cols > 1 {
		colWd = max(colWd, hdrWd)
		b.WriteString(padToLength(legend, rowWd))
		totWd := rowWd
		for c := 0; c < cols; c++ {
			_, cc := Projection2DCoords(shp, onedRow, 0, c)
			s := prepadToLength(fmt.Sprintf("%v", cc), colWd)
			if totWd+len(s) > MaxPrintLineWidth {
				b.WriteString("\n" + strings.Repeat(" ", rowWd))
				totWd = rowWd
			}
			b.WriteString(s)
			totWd += len(s)
		}
		b.WriteString("\n")
	}
	ctr := 0
	for r := range rows {
		rc, _ := Projection2DCoords(shp, onedRow, r, 0)
		if !noidx {
			b.WriteString(padToLength(fmt.Sprintf("%v", rc), rowWd))
		}
		ri := r
		totWd := rowWd
		for c := 0; c < cols; c++ {
			s := ""
			if tsr.IsString() {
				s = padToLength(fmt.Sprintf(format, Projection2DString(tsr, onedRow, ri, c)), colWd)
			} else {
				s = prepadToLength(fmt.Sprintf(format, Projection2DValue(tsr, onedRow, ri, c)), colWd)
			}
			if totWd+len(s) > MaxPrintLineWidth {
				b.WriteString("\n" + strings.Repeat(" ", rowWd))
				totWd = rowWd
			}
			b.WriteString(s)
			totWd += len(s)
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
