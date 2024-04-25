// Copyright (c) 2024, The Cogent Core Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package table

/*
import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"io/fs"
	"log"
	"math"
	"os"
	"strconv"
	"strings"

	"cogentcore.org/core/core"
	"cogentcore.org/core/errors"
	"cogentcore.org/core/tensor"
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

const (
	//	Headers is passed to CSV methods for the headers arg, to use headers
	Headers = true

	// NoHeaders is passed to CSV methods for the headers arg, to not use headers
	NoHeaders = false
)

// SaveCSV writes a table to a comma-separated-values (CSV) file
// (where comma = any delimiter, specified in the delim arg).
// If headers = true then generate C++ emergent-tyle column headers.
// These headers have full configuration information for the tensor
// columns.  Otherwise, only the data is written.
func (dt *Table) SaveCSV(filename core.Filename, delim Delims, headers bool) error { //types:add
	fp, err := os.Create(string(filename))
	defer fp.Close()
	if err != nil {
		log.Println(err)
		return err
	}
	bw := bufio.NewWriter(fp)
	err = dt.WriteCSV(bw, delim, headers)
	bw.Flush()
	return err
}

// SaveCSV writes a table idx view to a comma-separated-values (CSV) file
// (where comma = any delimiter, specified in the delim arg).
// If headers = true then generate C++ emergent-tyle column headers.
// These headers have full configuration information for the tensor
// columns.  Otherwise, only the data is written.
func (ix *IndexView) SaveCSV(filename core.Filename, delim Delims, headers bool) error { //types:add
	fp, err := os.Create(string(filename))
	defer fp.Close()
	if err != nil {
		log.Println(err)
		return err
	}
	bw := bufio.NewWriter(fp)
	err = ix.WriteCSV(bw, delim, headers)
	bw.Flush()
	return err
}

// OpenCSV reads a table from a comma-separated-values (CSV) file
// (where comma = any delimiter, specified in the delim arg),
// using the Go standard encoding/csv reader conforming to the official CSV standard.
// If the table does not currently have any columns, the first row of the file
// is assumed to be headers, and columns are constructed therefrom.
// The C++ emergent column headers are parsed -- these have full configuration
// information for tensor dimensionality.
// If the table DOES have existing columns, then those are used robustly
// for whatever information fits from each row of the file.
func (dt *Table) OpenCSV(filename core.Filename, delim Delims) error { //types:add
	fp, err := os.Open(string(filename))
	if err != nil {
		return errors.Log(err)
	}
	defer fp.Close()
	return dt.ReadCSV(bufio.NewReader(fp), delim)
}

// OpenFS is the version of [Table.OpenCSV] that uses an [fs.FS] filesystem.
func (dt *Table) OpenFS(fsys fs.FS, filename string, delim Delims) error {
	fp, err := fsys.Open(filename)
	if err != nil {
		return errors.Log(err)
	}
	defer fp.Close()
	return dt.ReadCSV(bufio.NewReader(fp), delim)
}

// OpenCSV reads a table idx view from a comma-separated-values (CSV) file
// (where comma = any delimiter, specified in the delim arg),
// using the Go standard encoding/csv reader conforming to the official CSV standard.
// If the table does not currently have any columns, the first row of the file
// is assumed to be headers, and columns are constructed therefrom.
// The C++ emergent column headers are parsed -- these have full configuration
// information for tensor dimensionality.
// If the table DOES have existing columns, then those are used robustly
// for whatever information fits from each row of the file.
func (ix *IndexView) OpenCSV(filename core.Filename, delim Delims) error { //types:add
	err := ix.Table.OpenCSV(filename, delim)
	ix.Sequential()
	return err
}

// OpenFS is the version of [IndexView.OpenCSV] that uses an [fs.FS] filesystem.
func (ix *IndexView) OpenFS(fsys fs.FS, filename string, delim Delims) error {
	err := ix.Table.OpenFS(fsys, filename, delim)
	ix.Sequential()
	return err
}

// ReadCSV reads a table from a comma-separated-values (CSV) file
// (where comma = any delimiter, specified in the delim arg),
// using the Go standard encoding/csv reader conforming to the official CSV standard.
// If the table does not currently have any columns, the first row of the file
// is assumed to be headers, and columns are constructed therefrom.
// The C++ emergent column headers are parsed -- these have full configuration
// information for tensor dimensionality.
// If the table DOES have existing columns, then those are used robustly
// for whatever information fits from each row of the file.
func (dt *Table) ReadCSV(r io.Reader, delim Delims) error {
	cr := csv.NewReader(r)
	cr.Comma = delim.Rune()
	rec, err := cr.ReadAll() // todo: lazy, avoid resizing
	if err != nil || len(rec) == 0 {
		return err
	}
	rows := len(rec)
	// cols := len(rec[0])
	strow := 0
	if dt.NumCols() == 0 || DetectEmerHeaders(rec[0]) {
		sc, err := SchemaFromHeaders(rec[0], rec)
		if err != nil {
			log.Println(err.Error())
			return err
		}
		strow++
		rows--
		dt.SetFromSchema(sc, rows)
	}
	dt.SetNumRows(rows)
	for ri := 0; ri < rows; ri++ {
		dt.ReadCSVRow(rec[ri+strow], ri)
	}
	return nil
}

// ReadCSVRow reads a record of CSV data into given row in table
func (dt *Table) ReadCSVRow(rec []string, row int) {
	tc := dt.NumCols()
	ci := 0
	if rec[0] == "_D:" { // emergent data row
		ci++
	}
	nan := math.NaN()
	for j := 0; j < tc; j++ {
		tsr := dt.Cols[j]
		_, csz := tsr.RowCellSize()
		stoff := row * csz
		for cc := 0; cc < csz; cc++ {
			str := rec[ci]
			if !tsr.IsString() {
				if str == "" || str == "NaN" || str == "-NaN" || str == "Inf" || str == "-Inf" {
					tsr.SetNull1D(stoff+cc, true) // empty = missing
					tsr.SetFloat1D(stoff+cc, nan)
				} else {
					tsr.SetString1D(stoff+cc, str)
				}
			} else {
				tsr.SetString1D(stoff+cc, str)
			}
			ci++
			if ci >= len(rec) {
				return
			}
		}
	}
}

// SchemaFromHeaders attempts to configure a Table Schema based on the headers
// for non-Emergent headers, data is examined to
func SchemaFromHeaders(hdrs []string, rec [][]string) (Schema, error) {
	if DetectEmerHeaders(hdrs) {
		return SchemaFromEmerHeaders(hdrs)
	}
	return SchemaFromPlainHeaders(hdrs, rec)
}

// DetectEmerHeaders looks for emergent header special characters -- returns true if found
func DetectEmerHeaders(hdrs []string) bool {
	for _, hd := range hdrs {
		if hd == "" {
			continue
		}
		if hd == "_H:" {
			return true
		}
		if _, ok := EmerHdrCharToType[hd[0]]; !ok { // all must be emer
			return false
		}
	}
	return true
}

// SchemaFromEmerHeaders attempts to configure a Table Schema based on emergent DataTable headers
func SchemaFromEmerHeaders(hdrs []string) (Schema, error) {
	sc := Schema{}
	for _, hd := range hdrs {
		if hd == "" || hd == "_H:" {
			continue
		}
		var typ tensor.Type
		typ, hd = EmerColType(hd)
		dimst := strings.Index(hd, "]<")
		if dimst > 0 {
			dims := hd[dimst+2 : len(hd)-1]
			lbst := strings.Index(hd, "[")
			hd = hd[:lbst]
			csh := ShapeFromString(dims)
			// new tensor starting
			sc = append(sc, Column{Name: hd, Type: tensor.Type(typ), CellShape: csh})
			continue
		}
		dimst = strings.Index(hd, "[")
		if dimst > 0 {
			continue
		}
		sc = append(sc, Column{Name: hd, Type: tensor.Type(typ), CellShape: nil})
	}
	return sc, nil
}

var EmerHdrCharToType = map[byte]tensor.Type{
	'$': tensor.STRING,
	'%': tensor.FLOAT32,
	'#': tensor.FLOAT64,
	'|': tensor.INT64,
	'@': tensor.UINT8,
	'&': tensor.STRING,
	'^': tensor.BOOL,
}

var EmerHdrTypeToChar map[tensor.Type]byte

func init() {
	EmerHdrTypeToChar = make(map[tensor.Type]byte)
	for k, v := range EmerHdrCharToType {
		if k != '&' {
			EmerHdrTypeToChar[v] = k
		}
	}
	EmerHdrTypeToChar[tensor.INT8] = '@'
	EmerHdrTypeToChar[tensor.INT16] = '|'
	EmerHdrTypeToChar[tensor.UINT16] = '|'
	EmerHdrTypeToChar[tensor.INT32] = '|'
	EmerHdrTypeToChar[tensor.UINT32] = '|'
	EmerHdrTypeToChar[tensor.UINT64] = '|'
	EmerHdrTypeToChar[tensor.INT] = '|'
}

// EmerColType parses the column header for type information using the emergent naming convention
func EmerColType(nm string) (tensor.Type, string) {
	typ, ok := EmerHdrCharToType[nm[0]]
	if ok {
		nm = nm[1:]
	} else {
		typ = tensor.STRING // most general, default
	}
	return typ, nm
}

// ShapeFromString parses string representation of shape as N:d,d,..
func ShapeFromString(dims string) []int {
	clni := strings.Index(dims, ":")
	nd, _ := strconv.Atoi(dims[:clni])
	sh := make([]int, nd)
	ci := clni + 1
	for i := 0; i < nd; i++ {
		dstr := ""
		if i < nd-1 {
			nci := strings.Index(dims[ci:], ",")
			dstr = dims[ci : ci+nci]
			ci += nci + 1
		} else {
			dstr = dims[ci:]
		}
		d, _ := strconv.Atoi(dstr)
		sh[i] = d
	}
	return sh
}

// SchemaFromPlainHeaders configures a Table Schema based on plain headers.
// All columns are of type String and must be converted later to numerical types
// as appropriate.
func SchemaFromPlainHeaders(hdrs []string, rec [][]string) (Schema, error) {
	sc := Schema{}
	nr := len(rec)
	for ci, hd := range hdrs {
		if hd == "" {
			hd = fmt.Sprintf("col_%d", ci)
		}
		dt := tensor.STRING
		nmatch := 0
		for ri := 1; ri < nr; ri++ {
			rv := rec[ri][ci]
			if rv == "" {
				continue
			}
			cdt := InferDataType(rv)
			switch {
			case cdt == tensor.STRING: // definitive
				dt = cdt
				break
			case dt == cdt && (nmatch > 1 || ri == nr-1): // good enough
				break
			case dt == cdt: // gather more info
				nmatch++
			case dt == tensor.STRING: // always upgrade from string default
				nmatch = 0
				dt = cdt
			case dt == tensor.INT64 && cdt == tensor.FLOAT64: // upgrade
				nmatch = 0
				dt = cdt
			}
		}
		sc = append(sc, Column{Name: hd, Type: dt, CellShape: nil})
	}
	return sc, nil
}

// InferDataType returns the inferred data type for the given string
// only deals with float64, int, and string types
func InferDataType(str string) tensor.Type {
	if strings.Contains(str, ".") {
		_, err := strconv.ParseFloat(str, 64)
		if err == nil {
			return tensor.FLOAT64
		}
	}
	_, err := strconv.ParseInt(str, 10, 64)
	if err == nil {
		return tensor.INT64
	}
	// try float again just in case..
	_, err = strconv.ParseFloat(str, 64)
	if err == nil {
		return tensor.FLOAT64
	}
	return tensor.STRING
}

//////////////////////////////////////////////////////////////////////////
// WriteCSV

// WriteCSV writes a table to a comma-separated-values (CSV) file
// (where comma = any delimiter, specified in the delim arg).
// If headers = true then generate C++ emergent-style column headers.
// These headers have full configuration information for the tensor
// columns.  Otherwise, only the data is written.
func (dt *Table) WriteCSV(w io.Writer, delim Delims, headers bool) error {
	ncol := 0
	var err error
	if headers {
		ncol, err = dt.WriteCSVHeaders(w, delim)
		if err != nil {
			log.Println(err)
			return err
		}
	}
	cw := csv.NewWriter(w)
	cw.Comma = delim.Rune()
	for ri := 0; ri < dt.Rows; ri++ {
		err = dt.WriteCSVRowWriter(cw, ri, ncol)
		if err != nil {
			log.Println(err)
			return err
		}
	}
	cw.Flush()
	return nil
}

// WriteCSV writes only rows in table idx view to a comma-separated-values (CSV) file
// (where comma = any delimiter, specified in the delim arg).
// If headers = true then generate C++ emergent-style column headers.
// These headers have full configuration information for the tensor
// columns.  Otherwise, only the data is written.
func (ix *IndexView) WriteCSV(w io.Writer, delim Delims, headers bool) error {
	ncol := 0
	var err error
	if headers {
		ncol, err = ix.Table.WriteCSVHeaders(w, delim)
		if err != nil {
			log.Println(err)
			return err
		}
	}
	cw := csv.NewWriter(w)
	cw.Comma = delim.Rune()
	nrow := ix.Len()
	for ri := 0; ri < nrow; ri++ {
		err = ix.Table.WriteCSVRowWriter(cw, ix.Indexes[ri], ncol)
		if err != nil {
			log.Println(err)
			return err
		}
	}
	cw.Flush()
	return nil
}

// WriteCSVHeaders writes headers to a comma-separated-values (CSV) file
// (where comma = any delimiter, specified in the delim arg).
// Returns number of columns in header
func (dt *Table) WriteCSVHeaders(w io.Writer, delim Delims) (int, error) {
	cw := csv.NewWriter(w)
	cw.Comma = delim.Rune()
	hdrs := dt.EmerHeaders()
	nc := len(hdrs)
	err := cw.Write(hdrs)
	if err != nil {
		return nc, err
	}
	cw.Flush()
	return nc, nil
}

// WriteCSVRow writes given row to a comma-separated-values (CSV) file
// (where comma = any delimiter, specified in the delim arg)
func (dt *Table) WriteCSVRow(w io.Writer, row int, delim Delims) error {
	cw := csv.NewWriter(w)
	cw.Comma = delim.Rune()
	err := dt.WriteCSVRowWriter(cw, row, 0)
	cw.Flush()
	return err
}

// WriteCSVRowWriter uses csv.Writer to write one row
func (dt *Table) WriteCSVRowWriter(cw *csv.Writer, row int, ncol int) error {
	prec := -1
	if ps, ok := dt.MetaData["precision"]; ok {
		prec, _ = strconv.Atoi(ps)
	}
	var rec []string
	if ncol > 0 {
		rec = make([]string, 0, ncol)
	} else {
		rec = make([]string, 0)
	}
	rc := 0
	for i := range dt.Cols {
		tsr := dt.Cols[i]
		nd := tsr.NumDims()
		if nd == 1 {
			vl := ""
			if prec <= 0 || tsr.IsString() {
				vl = tsr.StringValue1D(row)
			} else {
				vl = strconv.FormatFloat(tsr.Float1D(row), 'g', prec, 64)
			}
			if len(rec) <= rc {
				rec = append(rec, vl)
			} else {
				rec[rc] = vl
			}
			rc++
		} else {
			csh := tensor.NewShape(tsr.Shapes()[1:], nil, nil) // cell shape
			tc := csh.Len()
			for ti := 0; ti < tc; ti++ {
				vl := ""
				if prec <= 0 || tsr.IsString() {
					vl = tsr.StringValue1D(row*tc + ti)
				} else {
					vl = strconv.FormatFloat(tsr.Float1D(row*tc+ti), 'g', prec, 64)
				}
				if len(rec) <= rc {
					rec = append(rec, vl)
				} else {
					rec[rc] = vl
				}
				rc++
			}
		}
	}
	err := cw.Write(rec)
	return err
}

// EmerHeaders generates emergent DataTable header strings from the table.
// These have full information about type and tensor cell dimensionality.
func (dt *Table) EmerHeaders() []string {
	hdrs := []string{}
	for i := range dt.Cols {
		tsr := dt.Cols[i]
		nm := dt.ColNames[i]
		nm = string([]byte{EmerHdrTypeToChar[tsr.DataType()]}) + nm
		if tsr.NumDims() == 1 {
			hdrs = append(hdrs, nm)
		} else {
			csh := tensor.NewShape(tsr.Shapes()[1:], nil, nil) // cell shape
			tc := csh.Len()
			nd := csh.NumDims()
			fnm := nm + fmt.Sprintf("[%v:", nd)
			dn := fmt.Sprintf("<%v:", nd)
			ffnm := fnm
			for di := 0; di < nd; di++ {
				ffnm += "0"
				dn += fmt.Sprintf("%v", csh.Dim(di))
				if di < nd-1 {
					ffnm += ","
					dn += ","
				}
			}
			ffnm += "]" + dn + ">"
			hdrs = append(hdrs, ffnm)
			for ti := 1; ti < tc; ti++ {
				idx := csh.Index(ti)
				ffnm := fnm
				for di := 0; di < nd; di++ {
					ffnm += fmt.Sprintf("%v", idx[di])
					if di < nd-1 {
						ffnm += ","
					}
				}
				ffnm += "]"
				hdrs = append(hdrs, ffnm)
			}
		}
	}
	return hdrs
}

*/
