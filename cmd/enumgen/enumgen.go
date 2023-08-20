// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package main provides the actual command line
// implementation of the enumgen library.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/goki/enums/enumgen"
)

// the configuration object we use
var config enumgen.Config

var ()

func main() {
	flag.StringVar(&config.Dir, "dir", ".", "the directory to look for enums in")
	flag.StringVar(&config.Output, "output", "enumgen.go", "the file name of the output file")

	log.SetPrefix("enumgen")
	flag.Usage = Usage
	flag.Parse()
	err := enumgen.Generate(config)
	if err != nil {
		fmt.Println(err)
	}
}

// Usage is a replacement usage function for the flags package.
func Usage() {
	_, _ = fmt.Fprintf(os.Stderr, "Enumgen is a tool to generate Go code that adds helpful methods to Go enums.\n")
	_, _ = fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	_, _ = fmt.Fprintf(os.Stderr, "\tenumgen [flags]\n")
	_, _ = fmt.Fprintf(os.Stderr, "For more information, see:\n")
	_, _ = fmt.Fprintf(os.Stderr, "\thttps://github.com/goki/enums\n")
	_, _ = fmt.Fprintf(os.Stderr, "Flags:\n")
	flag.PrintDefaults()
}
