// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package namer holds an interface, Namer, for types with a
// Name() string method, and SetNamer for types that also have SetName()
package namer

// Namer is an interface for anything that returns a Name() string.
type Namer interface {
	Name() string
}
