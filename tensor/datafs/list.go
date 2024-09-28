// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package datafs

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

func (d *Data) String() string {
	if !d.IsDir() {
		return d.Data.Label()
	}
	return d.List(Short, DirOnly)
}

func (d *Data) List(long, recursive bool) string {
	if long {
		return d.ListLong(recursive, 0)
	}
	return d.ListShort(recursive, 0)
}

func (d *Data) ListShort(recursive bool, ident int) string {
	var b strings.Builder
	items := d.ItemsFunc(nil)
	for _, it := range items {
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

func (d *Data) ListLong(recursive bool, ident int) string {
	var b strings.Builder
	items := d.ItemsFunc(nil)
	for _, it := range items {
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
