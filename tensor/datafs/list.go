// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package datafs

import "strings"

const (
	Short = false
	Long  = true

	DirOnly   = false
	Recursive = true
)

// todo: list options string

func (d *Data) String() string {
	if !d.IsDir() {
		return d.Data.Tensor.Label()
	}
	return d.List(Short, DirOnly)
}

func (d *Data) List(long, recursive bool) string {
	if long {
		return d.ListLong(recursive, 0)
	}
	return d.ListShort(recursive, 0)
}

func (d *Data) ListShort(recursive bool, indent int) string {
	var b strings.Builder
	items := d.ItemsFunc(nil)
	// todo: dirs first, then items
	for _, it := range items {
		if it.IsDir() {
			if recursive {
				b.WriteString(d.ListShort(recursive, indent+1))
			} else {
				b.WriteString(d.name + " ")
			}
		} else {
			b.WriteString(d.name + " ")
		}
	}
	return b.String()
}

func (d *Data) ListLong(recursive bool, indent int) string {
	var b strings.Builder
	items := d.ItemsFunc(nil)
	for _, it := range items {
		if it.IsDir() {
		} else {
			b.WriteString(d.String() + "\n")
		}
	}
	return b.String()
}
