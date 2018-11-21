// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package filecat

// LangProps contains properties of various languages -- this will be replaced
// by the full parsing information from the GoPi project. But it serves for now.
type LangProps struct {
	Lang      Supported `desc:"language -- must be a supported one from Supported list"`
	CommentLn string    `desc:"character(s) that start a single-line comment -- if empty then multi-line comment syntax will be used"`
	CommentSt string    `desc:"character(s) that start a multi-line comment or one that requires both start and end"`
	CommentEd string    `desc:"character(s) that end a multi-line comment or one that requires both start and end"`
}

// StdLangProps is the standard compiled-in set of langauge properties
var StdLangProps = map[Supported]LangProps{
	Ada:        {Ada, "--", "", ""},
	Bash:       {Bash, "# ", "", ""},
	Csh:        {Csh, "# ", "", ""},
	C:          {C, "// ", "/* ", " */"},
	CSharp:     {CSharp, "// ", "/* ", " */"},
	D:          {D, "// ", "/* ", " */"},
	ObjC:       {ObjC, "// ", "/* ", " */"},
	Go:         {Go, "// ", "/* ", " */"},
	Java:       {Java, "// ", "/* ", " */"},
	JavaScript: {JavaScript, "// ", "/* ", " */"},
	Eiffel:     {Eiffel, "--", "", ""},
	Haskell:    {Haskell, "--", "{- ", "-}"},
	Lisp:       {Lisp, "; ", "", ""},
	Lua:        {Lua, "--", "---[[ ", "--]]"},
	Makefile:   {Makefile, "# ", "", ""},
	Matlab:     {Matlab, "% ", "%{ ", " %}"},
	OCaml:      {OCaml, "", "(* ", " *)"},
	Pascal:     {Pascal, "// ", "{ ", " }"},
	Perl:       {Perl, "# ", "", ""},
	Python:     {Python, "# ", "", ""},
	Php:        {Php, "// ", "/* ", " */"},
	R:          {R, "# ", "", ""},
	Ruby:       {Ruby, "# ", "", ""},
	Rust:       {Rust, "// ", "/* ", " */"},
	Scala:      {Scala, "// ", "/* ", " */"},
	Html:       {Html, "", "<!-- ", " -->"},
	TeX:        {TeX, "% ", "", ""},
	Markdown:   {Markdown, "", "<!--- ", " -->"},
}
