// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tensorfs

import (
	"strings"

	"cogentcore.org/core/base/indent"
)

const (
	Short = false
	Long  = true

	DirOnly   = false
	Recursive = true
)

// todo: list options string

func (nd *Node) String() string {
	if !nd.IsDir() {
		return nd.Tensor.Label()
	}
	return nd.List(Short, DirOnly)
}

// List returns a listing of nodes in the given directory.
//   - long = include detailed information about each node, vs just the name.
//   - recursive = descend into subdirectories.
func (dir *Node) List(long, recursive bool) string {
	if long {
		return dir.ListLong(recursive, 0)
	}
	return dir.ListShort(recursive, 0)
}

// ListShort returns a name-only listing of given directory.
func (dir *Node) ListShort(recursive bool, ident int) string {
	var b strings.Builder
	nodes, _ := dir.Nodes()
	for _, it := range nodes {
		b.WriteString(indent.Tabs(ident))
		if it.IsDir() {
			if recursive {
				b.WriteString("\n" + it.ListShort(recursive, ident+1))
			} else {
				b.WriteString(it.name + "/ ")
			}
		} else {
			b.WriteString(it.name + " ")
		}
	}
	return b.String()
}

// ListLong returns a detailed listing of given directory.
func (dir *Node) ListLong(recursive bool, ident int) string {
	var b strings.Builder
	nodes, _ := dir.Nodes()
	for _, it := range nodes {
		b.WriteString(indent.Tabs(ident))
		if it.IsDir() {
			b.WriteString(it.name + "/\n")
			if recursive {
				b.WriteString(it.ListLong(recursive, ident+1))
			}
		} else {
			b.WriteString(it.String() + "\n")
		}
	}
	return b.String()
}
