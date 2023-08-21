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
	flag.StringVar(&config.Dir, "dir", ".", "the source directory to look for enums in")
	flag.StringVar(&config.Output, "output", "enumgen.go", "the file name of the output file")
	flag.BoolVar(&config.SQL, "sql", false, "whether to generate methods that implement the SQL Scanner and Valuer interfaces")
	flag.BoolVar(&config.Text, "text", true, "whether to generate text marshaling methods")
	flag.BoolVar(&config.JSON, "json", false, "whether to generate JSON marshaling methods (note that text marshaling methods will also work for JSON, so this should be unnecessary in almost all cases; see the text option)")
	flag.BoolVar(&config.YAML, "yaml", false, "whether to generate YAML marshaling methods")
	flag.BoolVar(&config.GQLGEN, "gqlgen", false, "whether to generate GraphQL marshaling methods for gqlgen")
	flag.StringVar(&config.Transform, "transform", "noop", "if specified, the enum item transformation method (eg: snake_case)")
	flag.StringVar(&config.TrimPrefix, "trimprefix", "", "if specified, the prefix to trim from each item")
	flag.StringVar(&config.AddPrefix, "addprefix", "", "if specified, the prefix to add to each item")
	flag.BoolVar(&config.LineComment, "linecomment", false, "whether to use line comment text as printed text when present")
	flag.StringVar(&config.Comment, "comment", "", "a comment to include at the top of the generated code")

	log.SetPrefix("enumgen")
	flag.Usage = Usage
	flag.Parse()
	config.Defaults()
	err := enumgen.Generate(config)
	if err != nil {
		fmt.Println("error: " + err.Error())
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
