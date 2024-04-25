// Copyright (c) 2024, The Cogent Core Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package table

// Column specifies everything about a column -- can be used for constructing tables
type Column struct {

	// name of column -- must be unique for a table
	Name string

	// data type, using etensor types which are isomorphic with arrow.Type
	// Type tensor.Type

	// shape of a single cell in the column (i.e., without the row dimension) -- for scalars this is nil -- tensor column will add the outer row dimension to this shape
	CellShape []int

	// names of the dimensions within the CellShape -- 'Row' will be added to outer dimension
	DimNames []string
}

// Schema specifies all of the columns of a table, sufficient to create the table.
// It is just a slice list of Columns
type Schema []Column
