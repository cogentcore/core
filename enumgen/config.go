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
	Dir         string // the source directory
	Output      string // the output file
	SQL         bool   // whether to generate methods that implement the SQL Scanner and Valuer interfaces
	JSON        bool   // whether to generate JSON marshaling methods
	YAML        bool   // whether to generate YAML marshaling methods
	Text        bool   // whether to generate text marshaling methods
	GQLGEN      bool   // whether to generate GraphQL marshaling methods for gqlgen
	AltValues   bool   // whether to generate alternative string values methods
	Transform   string // if specified, the enum item transformation method (eg: snake_case)
	TrimPrefix  string // if specified, a comma-separated list of prefixes to trim from each item
	AddPrefix   string // if specified, the prefix to add to each item
	LineComment bool   // whether to use line comment text as printed text when present
	Comment     string // a comment to include at the top of the generated code
}
