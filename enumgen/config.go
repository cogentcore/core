// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on http://github.com/dmarkham/enumer and
// golang.org/x/tools/cmd/stringer:

// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package enumgen

// Config contains the configuration information
// used by enumgen
//
//gti:add
type Config struct {

	// the source directory to run enumgen on (can be set to multiple through paths like ./...)
	Dir string `def:"." posarg:"0" required:"-"`

	// the output file location relative to the package on which enumgen is being called
	Output string `def:"enumgen.go"`

	// if specified, the enum item transformation method (eg: snake, kebab, lower)
	Transform string

	// if specified, a comma-separated list of prefixes to trim from each item
	TrimPrefix string

	// if specified, the prefix to add to each item
	AddPrefix string

	// whether to use line comment text as printed text when present
	LineComment bool

	// whether to accept lowercase versions of enum names in SetString
	AcceptLower bool `def:"true"`

	// whether to generate text marshaling methods
	Text bool `def:"true"`

	// whether to generate JSON marshaling methods (note that text marshaling methods will also work for JSON, so this should be unnecessary in almost all cases; see the text option)
	JSON bool

	// whether to generate YAML marshaling methods
	YAML bool

	// whether to generate methods that implement the SQL Scanner and Valuer interfaces
	SQL bool

	// whether to generate GraphQL marshaling methods for gqlgen
	GQL bool
}
