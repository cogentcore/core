// Copyright (c) 2020, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package tensormpi wraps the Message Passing Interface for distributed memory
data sharing across a collection of processors (procs).

It also contains some useful abstractions and error logging support in Go.

The wrapping code was initially copied  from https://github.com/cpmech/gosl/mpi
and significantly modified.

All standard Go types are supported using the apache arrow tmpl generation tool.
Int is assumed to be 64bit and is defined as a []int because that is typically
more convenient.
*/
package tensormpi
