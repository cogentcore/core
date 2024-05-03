// Copyright (c) 2020, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package mpi wraps the Message Passing Interface for distributed memory
data sharing across a collection of processors (procs).

The wrapping code was initially copied  from https://github.com/cpmech/gosl/mpi
and significantly modified.

All standard Go types are supported using the apache arrow tmpl generation tool.
Int is assumed to be 64bit and is defined as a []int because that is typically
more convenient.
*/
package mpi
