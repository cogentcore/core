// Copyright (c) 2022, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// copied from go src/cmd/gofmt/doc.go:

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
gosl translates Go source code into HLSL compatible shader code.
use //gosl: start <filename> and //gosl: end <filename> to
bracket code that should be copied into shaders/<filename>.hlsl
use //gosl: main <filename> instead of start for shader code that is
commented out in the .go file, which will be copied into the filename
and uncommented.

pass filenames or directory names for files to process.

Usage:

	gosl [flags] [path ...]

The flags are:

	-out string
	  	output directory for shader code, relative to where gosl is invoked (default "shaders")
*/
package main
