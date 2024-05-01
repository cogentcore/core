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

	"cogentcore.org/core/core"
)

// SaveCSV writes a tensor to a comma-separated-values (CSV) file
// (where comma = any delimiter, specified in the delim arg).
// Outer-most dims are rows in the file, and inner-most is column --
// Reading just grabs all values and doesn't care about shape.
func SaveCSV(tsr Tensor, filename core.Filename, delim rune) error {
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
func OpenCSV(tsr Tensor, filename core.Filename, delim rune) error {
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
func WriteCSV(tsr Tensor, w io.Writer, delim rune) error {
	prec := -1
	if ps, ok := tsr.MetaData("precision"); ok {
		prec, _ = strconv.Atoi(ps)
	}
	cw := csv.NewWriter(w)
	if delim != 0 {
		cw.Comma = delim
	}
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
func ReadCSV(tsr Tensor, r io.Reader, delim rune) error {
	cr := csv.NewReader(r)
	if delim != 0 {
		cr.Comma = delim
	}
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
			tsr.SetString1D(idx, str)
			idx++
			if idx >= sz {
				goto done
			}
		}
	}
done:
	return nil
}
