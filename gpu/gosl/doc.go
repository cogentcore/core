// Copyright (c) 2022, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// copied from go src/cmd/gofmt/doc.go:

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
gosl translates Go source code into WGSL compatible shader code.
use //gosl:start <filename> and //gosl:end <filename> to
bracket code that should be copied into shaders/<filename>.wgsl
Use //gosl:main <filename> instead of start for shader code that is
commented out in the .go file, which will be copied into the filename
and uncommented.

pass filenames or directory names for files to process.

Usage:

	gosl [flags] [path ...]

The flags are:

	-debug
	  	enable debugging messages while running
	-exclude string
	  	comma-separated list of names of functions to exclude from exporting to HLSL (default "Update,Defaults")
	-keep
	  	keep temporary converted versions of the source files, for debugging
	-out string
	  	output directory for shader code, relative to where gosl is invoked -- must not be an empty string (default "shaders")
*/
package main
