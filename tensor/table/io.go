// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package table

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"io/fs"
	"log"
	"log/slog"
	"math"
	"os"
	"reflect"
	"strconv"
	"strings"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/fsx"
	"cogentcore.org/core/tensor"
)

const (
	//	Headers is passed to CSV methods for the headers arg, to use headers
	// that capture full type and tensor shape information.
	Headers = true

	// NoHeaders is passed to CSV methods for the headers arg, to not use headers
	NoHeaders = false
)

// SaveCSV writes a table to a comma-separated-values (CSV) file
// (where comma = any delimiter, specified in the delim arg).
// If headers = true then generate column headers that capture the type
// and tensor cell geometry of the columns, enabling full reloading
// of exactly the same table format and data (recommended).
// Otherwise, only the data is written.
func (dt *Table) SaveCSV(filename fsx.Filename, delim tensor.Delims, headers bool) error { //types:add
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

// OpenCSV reads a table from a comma-separated-values (CSV) file
// (where comma = any delimiter, specified in the delim arg),
// using the Go standard encoding/csv reader conforming to the official CSV standard.
// If the table does not currently have any columns, the first row of the file
// is assumed to be headers, and columns are constructed therefrom.
// If the file was saved from table with headers, then these have full configuration
// information for tensor type and dimensionality.
// If the table DOES have existing columns, then those are used robustly
// for whatever information fits from each row of the file.
func (dt *Table) OpenCSV(filename fsx.Filename, delim tensor.Delims) error { //types:add
	fp, err := os.Open(string(filename))
	if err != nil {
		return errors.Log(err)
	}
	defer fp.Close()
	return dt.ReadCSV(bufio.NewReader(fp), delim)
}

// OpenFS is the version of [Table.OpenCSV] that uses an [fs.FS] filesystem.
func (dt *Table) OpenFS(fsys fs.FS, filename string, delim tensor.Delims) error {
	fp, err := fsys.Open(filename)
	if err != nil {
		return errors.Log(err)
	}
	defer fp.Close()
	return dt.ReadCSV(bufio.NewReader(fp), delim)
}

// ReadCSV reads a table from a comma-separated-values (CSV) file
// (where comma = any delimiter, specified in the delim arg),
// using the Go standard encoding/csv reader conforming to the official CSV standard.
// If the table does not currently have any columns, the first row of the file
// is assumed to be headers, and columns are constructed therefrom.
// If the file was saved from table with headers, then these have full configuration
// information for tensor type and dimensionality.
// If the table DOES have existing columns, then those are used robustly
// for whatever information fits from each row of the file.
func (dt *Table) ReadCSV(r io.Reader, delim tensor.Delims) error {
	dt.Sequential()
	cr := csv.NewReader(r)
	cr.Comma = delim.Rune()
	rec, err := cr.ReadAll() // todo: lazy, avoid resizing
	if err != nil || len(rec) == 0 {
		return err
	}
	rows := len(rec)
	strow := 0
	if dt.NumColumns() == 0 || DetectTableHeaders(rec[0]) {
		dt.DeleteAll()
		err := ConfigFromHeaders(dt, rec[0], rec)
		if err != nil {
			log.Println(err.Error())
			return err
		}
		strow++
		rows--
	}
	dt.SetNumRows(rows)
	for ri := 0; ri < rows; ri++ {
		dt.ReadCSVRow(rec[ri+strow], ri)
	}
	return nil
}

// ReadCSVRow reads a record of CSV data into given row in table
func (dt *Table) ReadCSVRow(rec []string, row int) {
	ci := 0
	if rec[0] == "_D:" { // data row
		ci++
	}
	nan := math.NaN()
	for _, tsr := range dt.Columns {
		_, csz := tsr.Shape().RowCellSize()
		stoff := row * csz
		for cc := 0; cc < csz; cc++ {
			str := rec[ci]
			if !tsr.IsString() {
				if str == "" || str == "NaN" || str == "-NaN" || str == "Inf" || str == "-Inf" {
					tsr.SetFloat1D(nan, stoff+cc)
				} else {
					tsr.SetString1D(strings.TrimSpace(str), stoff+cc)
				}
			} else {
				tsr.SetString1D(strings.TrimSpace(str), stoff+cc)
			}
			ci++
			if ci >= len(rec) {
				return
			}
		}
	}
}

// ConfigFromHeaders attempts to configure Table based on the headers.
// for non-table headers, data is examined to determine types.
func ConfigFromHeaders(dt *Table, hdrs []string, rec [][]string) error {
	if DetectTableHeaders(hdrs) {
		return ConfigFromTableHeaders(dt, hdrs)
	}
	return ConfigFromDataValues(dt, hdrs, rec)
}

// DetectTableHeaders looks for special header characters -- returns true if found
func DetectTableHeaders(hdrs []string) bool {
	for _, hd := range hdrs {
		hd = strings.TrimSpace(hd)
		if hd == "" {
			continue
		}
		if hd == "_H:" {
			return true
		}
		if _, ok := TableHeaderToType[hd[0]]; !ok { // all must be table
			return false
		}
	}
	return true
}

// ConfigFromTableHeaders attempts to configure a Table based on special table headers
func ConfigFromTableHeaders(dt *Table, hdrs []string) error {
	for _, hd := range hdrs {
		hd = strings.TrimSpace(hd)
		if hd == "" || hd == "_H:" {
			continue
		}
		typ, hd := TableColumnType(hd)
		dimst := strings.Index(hd, "]<")
		if dimst > 0 {
			dims := hd[dimst+2 : len(hd)-1]
			lbst := strings.Index(hd, "[")
			hd = hd[:lbst]
			csh := ShapeFromString(dims)
			// new tensor starting
			dt.AddColumnOfType(hd, typ, csh...)
			continue
		}
		dimst = strings.Index(hd, "[")
		if dimst > 0 {
			continue
		}
		dt.AddColumnOfType(hd, typ)
	}
	return nil
}

// TableHeaderToType maps special header characters to data type
var TableHeaderToType = map[byte]reflect.Kind{
	'$': reflect.String,
	'%': reflect.Float32,
	'#': reflect.Float64,
	'|': reflect.Int,
	'^': reflect.Bool,
}

// TableHeaderChar returns the special header character based on given data type
func TableHeaderChar(typ reflect.Kind) byte {
	switch {
	case typ == reflect.Bool:
		return '^'
	case typ == reflect.Float32:
		return '%'
	case typ == reflect.Float64:
		return '#'
	case typ >= reflect.Int && typ <= reflect.Uintptr:
		return '|'
	default:
		return '$'
	}
}

// TableColumnType parses the column header for special table type information
func TableColumnType(nm string) (reflect.Kind, string) {
	typ, ok := TableHeaderToType[nm[0]]
	if ok {
		nm = nm[1:]
	} else {
		typ = reflect.String // most general, default
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

// ConfigFromDataValues configures a Table based on data types inferred
// from the string representation of given records, using header names if present.
func ConfigFromDataValues(dt *Table, hdrs []string, rec [][]string) error {
	nr := len(rec)
	for ci, hd := range hdrs {
		hd = strings.TrimSpace(hd)
		if hd == "" {
			hd = fmt.Sprintf("col_%d", ci)
		}
		nmatch := 0
		typ := reflect.String
		for ri := 1; ri < nr; ri++ {
			rv := rec[ri][ci]
			if rv == "" {
				continue
			}
			ctyp := InferDataType(rv)
			switch {
			case ctyp == reflect.String: // definitive
				typ = ctyp
				break
			case typ == ctyp && (nmatch > 1 || ri == nr-1): // good enough
				break
			case typ == ctyp: // gather more info
				nmatch++
			case typ == reflect.String: // always upgrade from string default
				nmatch = 0
				typ = ctyp
			case typ == reflect.Int && ctyp == reflect.Float64: // upgrade
				nmatch = 0
				typ = ctyp
			}
		}
		dt.AddColumnOfType(hd, typ)
	}
	return nil
}

// InferDataType returns the inferred data type for the given string
// only deals with float64, int, and string types
func InferDataType(str string) reflect.Kind {
	if strings.Contains(str, ".") {
		_, err := strconv.ParseFloat(str, 64)
		if err == nil {
			return reflect.Float64
		}
	}
	_, err := strconv.ParseInt(str, 10, 64)
	if err == nil {
		return reflect.Int
	}
	// try float again just in case..
	_, err = strconv.ParseFloat(str, 64)
	if err == nil {
		return reflect.Float64
	}
	return reflect.String
}

//////////////////////////////////////////////////////////////////////////
// WriteCSV

// WriteCSV writes only rows in table idx view to a comma-separated-values (CSV) file
// (where comma = any delimiter, specified in the delim arg).
// If headers = true then generate column headers that capture the type
// and tensor cell geometry of the columns, enabling full reloading
// of exactly the same table format and data (recommended).
// Otherwise, only the data is written.
func (dt *Table) WriteCSV(w io.Writer, delim tensor.Delims, headers bool) error {
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
	nrow := dt.NumRows()
	for ri := range nrow {
		ix := dt.RowIndex(ri)
		err = dt.WriteCSVRowWriter(cw, ix, ncol)
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
func (dt *Table) WriteCSVHeaders(w io.Writer, delim tensor.Delims) (int, error) {
	cw := csv.NewWriter(w)
	cw.Comma = delim.Rune()
	hdrs := dt.TableHeaders()
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
func (dt *Table) WriteCSVRow(w io.Writer, row int, delim tensor.Delims) error {
	cw := csv.NewWriter(w)
	cw.Comma = delim.Rune()
	err := dt.WriteCSVRowWriter(cw, row, 0)
	cw.Flush()
	return err
}

// WriteCSVRowWriter uses csv.Writer to write one row
func (dt *Table) WriteCSVRowWriter(cw *csv.Writer, row int, ncol int) error {
	prec := -1
	if ps, err := tensor.Precision(dt.Meta); err == nil {
		prec = ps
	}
	var rec []string
	if ncol > 0 {
		rec = make([]string, 0, ncol)
	} else {
		rec = make([]string, 0)
	}
	rc := 0
	for _, tsr := range dt.Columns {
		nd := tsr.NumDims()
		if nd == 1 {
			vl := ""
			if prec <= 0 || tsr.IsString() {
				vl = tsr.String1D(row)
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
			csh := tensor.NewShape(tsr.ShapeSizes()[1:]...) // cell shape
			tc := csh.Len()
			for ti := 0; ti < tc; ti++ {
				vl := ""
				if prec <= 0 || tsr.IsString() {
					vl = tsr.String1D(row*tc + ti)
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

// TableHeaders generates special header strings from the table
// with full information about type and tensor cell dimensionality.
func (dt *Table) TableHeaders() []string {
	hdrs := []string{}
	for i, nm := range dt.Names {
		tsr := dt.Columns[i]
		nm = string([]byte{TableHeaderChar(tsr.DataType())}) + nm
		if tsr.NumDims() == 1 {
			hdrs = append(hdrs, nm)
		} else {
			csh := tensor.NewShape(tsr.ShapeSizes()[1:]...) // cell shape
			tc := csh.Len()
			nd := csh.NumDims()
			fnm := nm + fmt.Sprintf("[%v:", nd)
			dn := fmt.Sprintf("<%v:", nd)
			ffnm := fnm
			for di := 0; di < nd; di++ {
				ffnm += "0"
				dn += fmt.Sprintf("%v", csh.DimSize(di))
				if di < nd-1 {
					ffnm += ","
					dn += ","
				}
			}
			ffnm += "]" + dn + ">"
			hdrs = append(hdrs, ffnm)
			for ti := 1; ti < tc; ti++ {
				idx := csh.IndexFrom1D(ti)
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

// CleanCatTSV cleans a TSV file formed by concatenating multiple files together.
// Removes redundant headers and then sorts by given set of columns.
func CleanCatTSV(filename string, sorts ...string) error {
	str, err := os.ReadFile(filename)
	if err != nil {
		slog.Error(err.Error())
		return err
	}
	lns := strings.Split(string(str), "\n")
	if len(lns) == 0 {
		return nil
	}
	hdr := lns[0]
	f, err := os.Create(filename)
	if err != nil {
		slog.Error(err.Error())
		return err
	}
	for i, ln := range lns {
		if i > 0 && ln == hdr {
			continue
		}
		io.WriteString(f, ln)
		io.WriteString(f, "\n")
	}
	f.Close()
	dt := New()
	err = dt.OpenCSV(fsx.Filename(filename), tensor.Detect)
	if err != nil {
		slog.Error(err.Error())
		return err
	}
	dt.SortColumns(tensor.Ascending, tensor.StableSort, sorts...)
	st := dt.New()
	err = st.SaveCSV(fsx.Filename(filename), tensor.Tab, true)
	if err != nil {
		slog.Error(err.Error())
	}
	return err
}
