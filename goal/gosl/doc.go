// Copyright (c) 2022, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// copied from go src/cmd/gofmt/doc.go:

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
gosl translates Go source code into WGSL compatible shader code.
Use //gosl:start and //gosl:end to bracket code to generate.

pass filenames or directory names for files to process.

Usage:

	gosl [flags] [path ...]

The flags are:

	-debug
	  	enable debugging messages while running
	-exclude string
	  	comma-separated list of names of functions to exclude from exporting to WGSL (default "Update,Defaults")
	-keep
	  	keep temporary converted versions of the source files, for debugging
	-out string
	  	output directory for shader code, relative to where gosl is invoked -- must not be an empty string (default "shaders")
*/
package main
