// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tensorfs

import (
	"fmt"
	"io/fs"
	"path"
	"slices"
	"sort"
	"strings"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/keylist"
	"cogentcore.org/core/tensor"
)

// Dir is a map of directory entry names to Nodes.
// It retains the order that nodes were added in, which is
// the natural order nodes are processed in.
type Dir = keylist.List[string, *Node]

// NewDir returns a new tensorfs directory with the given name.
// If parent != nil and a directory, this dir is added to it.
// If the parent already has an node of that name, it is returned,
// with an [fs.ErrExist] error.
// If the name is empty, then it is set to "root", the root directory.
// Note that "/" is not allowed for the root directory in Go [fs].
// If no parent (i.e., a new root) and CurRoot is nil, then it is set
// to this.
func NewDir(name string, parent ...*Node) (*Node, error) {
	if name == "" {
		name = "root"
	}
	var par *Node
	if len(parent) == 1 {
		par = parent[0]
	}
	dir, err := newNode(par, name)
	if dir != nil && dir.Dir == nil {
		dir.Dir = &Dir{}
	}
	return dir, err
}

// Mkdir creates a new directory under given dir with the specified name.
// Returns an error if this dir node is not a directory.
// Returns existing directory and [fs.ErrExist] if a node
// with the same name already exists.
// See [Node.RecycleDir] for a version with no error return
// that is preferable when expecting an existing directory.
func (dir *Node) Mkdir(name string) (*Node, error) {
	if err := dir.mustDir("Mkdir", name); err != nil {
		return nil, err
	}
	return NewDir(name, dir)
}

// RecycleDir creates a new directory under given dir with the specified name
// if it doesn't already exist, otherwise returns the existing one.
// Path / slash separators can be used to make a path of multiple directories.
// It logs an error and returns nil if this dir node is not a directory.
func (dir *Node) RecycleDir(name string) *Node {
	if err := dir.mustDir("RecycleDir", name); errors.Log(err) != nil {
		return nil
	}
	if len(name) == 0 {
		return dir
	}
	path := strings.Split(name, "/")
	if cd := dir.Dir.At(path[0]); cd != nil {
		if len(path) > 1 {
			return cd.RecycleDir(strings.Join(path[1:], "/"))
		}
		return cd
	}
	nd, _ := NewDir(name, dir)
	if len(path) > 1 {
		return nd.RecycleDir(strings.Join(path[1:], "/"))
	}
	return nd
}

// Node returns a Node in given directory by name.
// This is for fast access and direct usage of known
// nodes, and it will panic if this node is not a directory.
// Returns nil if no node of given name exists.
func (dir *Node) Node(name string) *Node {
	return dir.Dir.At(name)
}

// Value returns the [tensor.Tensor] value for given node
// within this directory. This will panic if node is not
// found, and will return nil if it is not a Value
// (i.e., it is a directory).
func (dir *Node) Value(name string) tensor.Tensor {
	return dir.Dir.At(name).Tensor
}

// Nodes returns a slice of Nodes in given directory by names variadic list.
// If list is empty, then all nodes in the directory are returned.
// returned error reports any nodes not found, or if not a directory.
func (dir *Node) Nodes(names ...string) ([]*Node, error) {
	if err := dir.mustDir("Nodes", ""); err != nil {
		return nil, err
	}
	var nds []*Node
	if len(names) == 0 {
		for _, it := range dir.Dir.Values {
			nds = append(nds, it)
		}
		return nds, nil
	}
	var errs []error
	for _, nm := range names {
		dt := dir.Dir.At(nm)
		if dt != nil {
			nds = append(nds, dt)
		} else {
			err := fmt.Errorf("tensorfs Dir %q node not found: %q", dir.Path(), nm)
			errs = append(errs, err)
		}
	}
	return nds, errors.Join(errs...)
}

// Values returns a slice of tensor values in the given directory,
// by names variadic list. If list is empty, then all value nodes
// in the directory are returned.
// returned error reports any nodes not found, or if not a directory.
func (dir *Node) Values(names ...string) ([]tensor.Tensor, error) {
	if err := dir.mustDir("Values", ""); err != nil {
		return nil, err
	}
	var nds []tensor.Tensor
	if len(names) == 0 {
		for _, it := range dir.Dir.Values {
			if it.Tensor != nil {
				nds = append(nds, it.Tensor)
			}
		}
		return nds, nil
	}
	var errs []error
	for _, nm := range names {
		it := dir.Dir.At(nm)
		if it != nil && it.Tensor != nil {
			nds = append(nds, it.Tensor)
		} else {
			err := fmt.Errorf("tensorfs Dir %q node not found: %q", dir.Path(), nm)
			errs = append(errs, err)
		}
	}
	return nds, errors.Join(errs...)
}

// ValuesFunc returns all tensor Values under given directory,
// filtered by given function, in directory order (e.g., order added),
// recursively descending into directories to return a flat list of
// the entire subtree. The function can filter out directories to prune
// the tree, e.g., using `IsDir` method.
// If func is nil, all Value nodes are returned.
func (dir *Node) ValuesFunc(fun func(nd *Node) bool) []tensor.Tensor {
	if err := dir.mustDir("ValuesFunc", ""); err != nil {
		return nil
	}
	var nds []tensor.Tensor
	for _, it := range dir.Dir.Values {
		if fun != nil && !fun(it) {
			continue
		}
		if it.IsDir() {
			subs := it.ValuesFunc(fun)
			nds = append(nds, subs...)
		} else {
			nds = append(nds, it.Tensor)
		}
	}
	return nds
}

// NodesFunc returns leaf Nodes under given directory, filtered by
// given function, recursively descending into directories
// to return a flat list of the entire subtree, in directory order
// (e.g., order added).
// The function can filter out directories to prune the tree.
// If func is nil, all leaf Nodes are returned.
func (dir *Node) NodesFunc(fun func(nd *Node) bool) []*Node {
	if err := dir.mustDir("NodesFunc", ""); err != nil {
		return nil
	}
	var nds []*Node
	for _, it := range dir.Dir.Values {
		if fun != nil && !fun(it) {
			continue
		}
		if it.IsDir() {
			subs := it.NodesFunc(fun)
			nds = append(nds, subs...)
		} else {
			nds = append(nds, it)
		}
	}
	return nds
}

// ValuesAlphaFunc returns all Value nodes (tensors) in given directory,
// recursively descending into directories to return a flat list of
// the entire subtree, filtered by given function, with nodes at each
// directory level traversed in alphabetical order.
// The function can filter out directories to prune the tree.
// If func is nil, all Values are returned.
func (dir *Node) ValuesAlphaFunc(fun func(nd *Node) bool) []tensor.Tensor {
	if err := dir.mustDir("ValuesAlphaFunc", ""); err != nil {
		return nil
	}
	names := dir.dirNamesAlpha()
	var nds []tensor.Tensor
	for _, nm := range names {
		it := dir.Dir.At(nm)
		if fun != nil && !fun(it) {
			continue
		}
		if it.IsDir() {
			subs := it.ValuesAlphaFunc(fun)
			nds = append(nds, subs...)
		} else {
			nds = append(nds, it.Tensor)
		}
	}
	return nds
}

// NodesAlphaFunc returns leaf nodes under given directory, filtered
// by given function, with nodes at each directory level
// traversed in alphabetical order, recursively descending into directories
// to return a flat list of the entire subtree, in directory order
// (e.g., order added).
// The function can filter out directories to prune the tree.
// If func is nil, all leaf Nodes are returned.
func (dir *Node) NodesAlphaFunc(fun func(nd *Node) bool) []*Node {
	if err := dir.mustDir("NodesAlphaFunc", ""); err != nil {
		return nil
	}
	names := dir.dirNamesAlpha()
	var nds []*Node
	for _, nm := range names {
		it := dir.Dir.At(nm)
		if fun != nil && !fun(it) {
			continue
		}
		if it.IsDir() {
			subs := it.NodesAlphaFunc(fun)
			nds = append(nds, subs...)
		} else {
			nds = append(nds, it)
		}
	}
	return nds
}

// todo: these must handle going up the tree using ..

// DirAtPath returns directory at given relative path
// from this starting dir.
func (dir *Node) DirAtPath(dirPath string) (*Node, error) {
	var err error
	dirPath = path.Clean(dirPath)
	sdf, err := dir.Sub(dirPath) // this ensures that dir is a dir
	if err != nil {
		return nil, err
	}
	return sdf.(*Node), nil
}

// NodeAtPath returns node at given relative path from this starting dir.
func (dir *Node) NodeAtPath(name string) (*Node, error) {
	if err := dir.mustDir("NodeAtPath", name); err != nil {
		return nil, err
	}
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "NodeAtPath", Path: name, Err: errors.New("invalid path")}
	}
	dirPath, file := path.Split(name)
	sd, err := dir.DirAtPath(dirPath)
	if err != nil {
		return nil, err
	}
	nd, ok := sd.Dir.AtTry(file)
	if !ok {
		if dirPath == "" && (file == dir.name || file == ".") {
			return dir, nil
		}
		return nil, &fs.PathError{Op: "NodeAtPath", Path: name, Err: errors.New("file not found")}
	}
	return nd, nil
}

// Path returns the full path to this data node
func (dir *Node) Path() string {
	pt := dir.name
	cur := dir.Parent
	loops := make(map[*Node]struct{})
	for {
		if cur == nil {
			return pt
		}
		if _, ok := loops[cur]; ok {
			return pt
		}
		pt = path.Join(cur.name, pt)
		loops[cur] = struct{}{}
		cur = cur.Parent
	}
}

// dirNamesAlpha returns the names of nodes in the directory
// sorted alphabetically. Node must be dir by this point.
func (dir *Node) dirNamesAlpha() []string {
	names := slices.Clone(dir.Dir.Keys)
	sort.Strings(names)
	return names
}

// dirNamesByTime returns the names of nodes in the directory
// sorted by modTime. Node must be dir by this point.
func (dir *Node) dirNamesByTime() []string {
	names := slices.Clone(dir.Dir.Keys)
	slices.SortFunc(names, func(a, b string) int {
		return dir.Dir.At(a).ModTime().Compare(dir.Dir.At(b).ModTime())
	})
	return names
}

// mustDir returns an error for given operation and path
// if this data node is not a directory.
func (dir *Node) mustDir(op, path string) error {
	if !dir.IsDir() {
		return &fs.PathError{Op: op, Path: path, Err: errors.New("tensorfs node is not a directory")}
	}
	return nil
}

// Add adds an node to this directory data node.
// The only errors are if this node is not a directory,
// or the name already exists, in which case an [fs.ErrExist] is returned.
// Names must be unique within a directory.
func (dir *Node) Add(it *Node) error {
	if err := dir.mustDir("Add", it.name); err != nil {
		return err
	}
	err := dir.Dir.Add(it.name, it)
	if err != nil {
		return fs.ErrExist
	}
	return nil
}
