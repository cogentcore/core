// Copyright (c) 2022, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// copied and heavily edited from go src/cmd/gofmt/gofmt.go:

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/emer/gosl/v2/slprint"
)

// flags
var (
	outDir        = flag.String("out", "shaders", "output directory for shader code, relative to where gosl is invoked -- must not be an empty string")
	excludeFuns   = flag.String("exclude", "Update,Defaults", "comma-separated list of names of functions to exclude from exporting to HLSL")
	keepTmp       = flag.Bool("keep", false, "keep temporary converted versions of the source files, for debugging")
	debug         = flag.Bool("debug", false, "enable debugging messages while running")
	excludeFunMap = map[string]bool{}
)

// Keep these in sync with go/format/format.go.
const (
	tabWidth    = 8
	printerMode = slprint.UseSpaces | slprint.TabIndent | printerNormalizeNumbers

	// printerNormalizeNumbers means to canonicalize number literal prefixes
	// and exponents while printing. See https://golang.org/doc/go1.13#gosl.
	//
	// This value is defined in go/printer specifically for go/format and cmd/gosl.
	printerNormalizeNumbers = 1 << 30
)

func usage() {
	fmt.Fprintf(os.Stderr, "usage: gosl [flags] [path ...]\n")
	flag.PrintDefaults()
}

func main() {
	flag.Usage = usage
	flag.Parse()
	goslMain()
}

func GoslArgs() {
	exs := *excludeFuns
	ex := strings.Split(exs, ",")
	for _, fn := range ex {
		excludeFunMap[fn] = true
	}
}

func goslMain() {
	if *outDir == "" {
		fmt.Printf("Must have an output directory (default shaders), specified in -out arg\n")
		return
	}
	os.MkdirAll(*outDir, 0755)
	RemoveGenFiles(*outDir)

	args := flag.Args()
	if len(args) == 0 {
		fmt.Printf("at least one file name must be passed\n")
		return
	}

	GoslArgs()
	ProcessFiles(args)
}
