// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tensor

import (
	"encoding/csv"
	"io"
	"log"
	"os"
	"strconv"

	"cogentcore.org/core/base/metadata"
	"cogentcore.org/core/core"
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

// GetPrecision gets the "precision" metadata value that determines
// the precision to use in writing floating point numbers to files.
// returns an error if not set.
func GetPrecision(md metadata.Data) (int, error) {
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
	if ps, err := GetPrecision(*tsr.Metadata()); err == nil {
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
