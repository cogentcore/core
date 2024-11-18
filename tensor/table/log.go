// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package table

import (
	"os"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/metadata"
	"cogentcore.org/core/tensor"
)

func setLogRow(dt *Table, row int) {
	metadata.SetTo(dt, "LogRow", row)
}

func logRow(dt *Table) int {
	return errors.Ignore1(metadata.GetFrom[int](dt, "LogRow"))
}

func setLogDelim(dt *Table, delim tensor.Delims) {
	metadata.SetTo(dt, "LogDelim", delim)
}

func logDelim(dt *Table) tensor.Delims {
	return errors.Ignore1(metadata.GetFrom[tensor.Delims](dt, "LogDelim"))
}

// OpenLog opens a log file for this table, which supports incremental
// output of table data as it is generated, using the standard [Table.SaveCSV]
// output formatting, using given delimiter between values on a line.
// Call [Table.WriteToLog] to write any new data rows to
// the open log file, and [Table.CloseLog] to close the file.
func (dt *Table) OpenLog(filename string, delim tensor.Delims) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	metadata.SetFile(dt, f)
	setLogDelim(dt, delim)
	setLogRow(dt, 0)
	return nil
}

var (
	ErrLogNoNewRows = errors.New("no new rows to write")
)

// WriteToLog writes any accumulated rows in the table to the file
// opened by [Table.OpenLog]. A Header row is written for the first output.
// If the current number of rows is less than the last number of rows,
// all of those rows are written under the assumption that the rows
// were reset via [Table.SetNumRows].
// Returns error for any failure, including [ErrLogNoNewRows] if
// no new rows are available to write.
func (dt *Table) WriteToLog() error {
	f := metadata.File(dt)
	if f == nil {
		return errors.New("tensor.Table.WriteToLog: log file was not opened")
	}
	delim := logDelim(dt)
	lrow := logRow(dt)
	nr := dt.NumRows()
	if nr == 0 || lrow == nr {
		return ErrLogNoNewRows
	}
	if lrow == 0 {
		dt.WriteCSVHeaders(f, delim)
	}
	sr := lrow
	if nr < lrow {
		sr = 0
	}
	for r := sr; r < nr; r++ {
		dt.WriteCSVRow(f, r, delim)
	}
	setLogRow(dt, nr)
	return nil
}

// CloseLog closes the log file opened by [Table.OpenLog].
func (dt *Table) CloseLog() {
	f := metadata.File(dt)
	f.Close()
}
