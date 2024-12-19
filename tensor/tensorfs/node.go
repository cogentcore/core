// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tensorfs

import (
	"io/fs"
	"reflect"
	"time"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/fsx"
	"cogentcore.org/core/base/metadata"
	"cogentcore.org/core/tensor"
	"cogentcore.org/core/tensor/table"
)

// Node is the element type for the filesystem, which can represent either
// a [tensor] Value as a "file" equivalent, or a "directory" containing other Nodes.
// The [tensor.Tensor] can represent everything from a single scalar value up to
// n-dimensional collections of patterns, in a range of data types.
// Directories have an ordered map of nodes.
type Node struct {
	// Parent is the parent data directory.
	Parent *Node

	// name is the name of this node.  it is not a path.
	name string

	// modTime tracks time added to directory, used for ordering.
	modTime time.Time

	// Tensor is the tensor value for a file or leaf Node in the FS,
	// represented using the universal [tensor] data type of
	// [tensor.Tensor], which can represent anything from a scalar
	// to n-dimensional data, in a range of data types.
	Tensor tensor.Tensor

	// nodes is for directory nodes, with all the nodes in the directory.
	nodes *Nodes

	// DirTable is a summary [table.Table] with columns comprised of Value
	// nodes in the directory, which can be used for plotting or other operations.
	DirTable *table.Table
}

// newNode returns a new Node in given directory Node, which can be nil.
// If dir is not a directory, returns nil and an error.
// If an node already exists in dir with that name, that node is returned
// with an [fs.ErrExist] error, and the caller can decide how to proceed.
// The modTime is set to now. The name must be unique within parent.
func newNode(dir *Node, name string) (*Node, error) {
	if dir == nil {
		return &Node{name: name, modTime: time.Now()}, nil
	}
	if err := dir.mustDir("newNode", name); err != nil {
		return nil, err
	}
	if ex, ok := dir.nodes.AtTry(name); ok {
		return ex, fs.ErrExist
	}
	d := &Node{Parent: dir, name: name, modTime: time.Now()}
	dir.nodes.Add(name, d)
	return d, nil
}

// Value creates / returns a Node with given name as a [tensor.Tensor]
// of given data type and shape sizes, in given directory Node.
// If it already exists, it is returned as-is (no checking against the
// type or sizes provided, for efficiency -- if there is doubt, check!),
// otherwise a new tensor is created. It is fine to not pass any sizes and
// use `SetShapeSizes` method later to set the size.
func Value[T tensor.DataTypes](dir *Node, name string, sizes ...int) tensor.Values {
	it := dir.Node(name)
	if it != nil {
		return it.Tensor.(tensor.Values)
	}
	tsr := tensor.New[T](sizes...)
	metadata.SetName(tsr, name)
	nd, err := newNode(dir, name)
	if errors.Log(err) != nil {
		return nil
	}
	nd.Tensor = tsr
	return tsr
}

// NewValues makes new tensor Node value(s) (as a [tensor.Tensor])
// of given data type and shape sizes, in given directory.
// Any existing nodes with the same names are recycled without checking
// or updating the data type or sizes.
// See the [Value] documentation for more info.
func NewValues[T tensor.DataTypes](dir *Node, shape []int, names ...string) {
	for _, nm := range names {
		Value[T](dir, nm, shape...)
	}
}

// Scalar returns a scalar Node value (as a [tensor.Tensor])
// of given data type, in given directory and name.
// If it already exists, it is returned without checking against args,
// else a new one is made. See the [Value] documentation for more info.
func Scalar[T tensor.DataTypes](dir *Node, name string) tensor.Values {
	return Value[T](dir, name, 1)
}

// ValueType creates / returns a Node with given name as a [tensor.Tensor]
// of given data type specified as a reflect.Kind, with shape sizes,
// in given directory Node.
// Supported types are string, bool (for [Bool]), float32, float64, int, int32, and byte.
// If it already exists, it is returned as-is (no checking against the
// type or sizes provided, for efficiency -- if there is doubt, check!),
// otherwise a new tensor is created. It is fine to not pass any sizes and
// use `SetShapeSizes` method later to set the size.
func ValueType(dir *Node, name string, typ reflect.Kind, sizes ...int) tensor.Values {
	it := dir.Node(name)
	if it != nil {
		return it.Tensor.(tensor.Values)
	}
	tsr := tensor.NewOfType(typ, sizes...)
	metadata.SetName(tsr, name)
	nd, err := newNode(dir, name)
	if errors.Log(err) != nil {
		return nil
	}
	nd.Tensor = tsr
	return tsr
}

// NewForTensor creates a new Node node for given existing tensor with given name.
// If the name already exists, that Node is returned with [fs.ErrExists] error.
func NewForTensor(dir *Node, tsr tensor.Tensor, name string) (*Node, error) {
	nd, err := newNode(dir, name)
	if err != nil {
		return nd, err
	}
	nd.Tensor = tsr
	return nd, nil
}

// DirTable returns a [table.Table] with all of the tensor values under
// the given directory, with columns as the Tensor values elements in the directory
// and any subdirectories, using given filter function.
// This is a convenient mechanism for creating a plot of all the data
// in a given directory.
// If such was previously constructed, it is returned from "DirTable"
// where it is stored for later use.
// Row count is updated to current max row.
// Set DirTable = nil to regenerate.
func DirTable(dir *Node, fun func(node *Node) bool) *table.Table {
	nds := dir.NodesFunc(fun)
	if dir.DirTable != nil {
		if dir.DirTable.NumColumns() == len(nds) {
			dir.DirTable.SetNumRowsToMax()
			return dir.DirTable
		}
	}
	dt := table.New(fsx.DirAndFile(string(dir.Path())))
	for _, it := range nds {
		tsr := it.Tensor
		rows := tsr.DimSize(0)
		if dt.Columns.Rows < rows {
			dt.Columns.Rows = rows
			dt.SetNumRows(dt.Columns.Rows)
		}
		nm := it.name
		if it.Parent != dir {
			nm = fsx.DirAndFile(string(it.Path()))
		}
		dt.AddColumn(nm, tsr.AsValues())
	}
	dir.DirTable = dt
	return dt
}

// DirFromTable sets tensor values under given directory node to the
// columns of the given [table.Table]. Also sets the DirTable to this table.
func DirFromTable(dir *Node, dt *table.Table) {
	for i, cl := range dt.Columns.Values {
		nm := dt.Columns.Keys[i]
		nd, err := newNode(dir, nm)
		if err == nil || err == fs.ErrExist {
			nd.Tensor = cl
		}
	}
	dir.DirTable = dt
}

// Float64 creates / returns a Node with given name as a [tensor.Float64]
// for given shape sizes, in given directory [Node].
// See [Values] function for more info.
func (dir *Node) Float64(name string, sizes ...int) *tensor.Float64 {
	return Value[float64](dir, name, sizes...).(*tensor.Float64)
}

// Float32 creates / returns a Node with given name as a [tensor.Float32]
// for given shape sizes, in given directory [Node].
// See [Values] function for more info.
func (dir *Node) Float32(name string, sizes ...int) *tensor.Float32 {
	return Value[float32](dir, name, sizes...).(*tensor.Float32)
}

// Int creates / returns a Node with given name as a [tensor.Int]
// for given shape sizes, in given directory [Node].
// See [Values] function for more info.
func (dir *Node) Int(name string, sizes ...int) *tensor.Int {
	return Value[int](dir, name, sizes...).(*tensor.Int)
}

// StringValue creates / returns a Node with given name as a [tensor.String]
// for given shape sizes, in given directory [Node].
// See [Values] function for more info.
func (dir *Node) StringValue(name string, sizes ...int) *tensor.String {
	return Value[string](dir, name, sizes...).(*tensor.String)
}
