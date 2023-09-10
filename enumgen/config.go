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
type Config struct {

	// [def: .] the source directory to run enumgen on (can be set to multiple through paths like ./...)
	Dir string `def:"." desc:"the source directory to run enumgen on (can be set to multiple through paths like ./...)"`

	// [def: enumgen.go] the output file location relative to the package on which enumgen is being called
	Output string `def:"enumgen.go" desc:"the output file location relative to the package on which enumgen is being called"`

	// if specified, the enum item transformation method (eg: snake, kebab, lower)
	Transform string `desc:"if specified, the enum item transformation method (eg: snake, kebab, lower)"`

	// if specified, a comma-separated list of prefixes to trim from each item
	TrimPrefix string `desc:"if specified, a comma-separated list of prefixes to trim from each item"`

	// if specified, the prefix to add to each item
	AddPrefix string `desc:"if specified, the prefix to add to each item"`

	// whether to use line comment text as printed text when present
	LineComment bool `desc:"whether to use line comment text as printed text when present"`

	// [def: true] whether to accept lowercase versions of enum names in SetString
	AcceptLower bool `def:"true" desc:"whether to accept lowercase versions of enum names in SetString"`

	// [def: true] whether to generate text marshaling methods
	Text bool `def:"true" desc:"whether to generate text marshaling methods"`

	// whether to generate JSON marshaling methods (note that text marshaling methods will also work for JSON, so this should be unnecessary in almost all cases; see the text option)
	JSON bool `desc:"whether to generate JSON marshaling methods (note that text marshaling methods will also work for JSON, so this should be unnecessary in almost all cases; see the text option)"`

	// whether to generate YAML marshaling methods
	YAML bool `desc:"whether to generate YAML marshaling methods"`

	// whether to generate methods that implement the SQL Scanner and Valuer interfaces
	SQL bool `desc:"whether to generate methods that implement the SQL Scanner and Valuer interfaces"`

	// whether to generate GraphQL marshaling methods for gqlgen
	GQL bool `desc:"whether to generate GraphQL marshaling methods for gqlgen"`
}
