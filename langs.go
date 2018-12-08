// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pi

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/goki/gi/filecat"
	"github.com/goki/ki/kit"
)

// LangFlags are special properties of a given language
type LangFlags int

//go:generate stringer -type=LangFlags

var KiT_LangFlags = kit.Enums.AddEnum(LangFlagsN, false, nil)

func (ev LangFlags) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *LangFlags) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

// LangFlags
const (
	// NoFlags = nothing special
	NoFlags LangFlags = iota

	// IndentSpace means that spaces must be used for this language
	IndentSpace

	// IndentTab means that tabs must be used for this language
	IndentTab

	LangFlagsN
)

// LangProps contains properties of languages supported by the Pi parser
// framework
type LangProps struct {
	Lang      filecat.Supported `desc:"language -- must be a supported one from Supported list"`
	CommentLn string            `desc:"character(s) that start a single-line comment -- if empty then multi-line comment syntax will be used"`
	CommentSt string            `desc:"character(s) that start a multi-line comment or one that requires both start and end"`
	CommentEd string            `desc:"character(s) that end a multi-line comment or one that requires both start and end"`
	Flags     []LangFlags       `desc:"special properties for this language"`
	Parser    *Parser           `json:"-" xml:"-" desc:"parser for this language"`
}

// StdLangProps is the standard compiled-in set of langauge properties
var StdLangProps = map[filecat.Supported]LangProps{
	filecat.Ada:        {filecat.Ada, "--", "", "", nil, nil},
	filecat.Bash:       {filecat.Bash, "# ", "", "", nil, nil},
	filecat.Csh:        {filecat.Csh, "# ", "", "", nil, nil},
	filecat.C:          {filecat.C, "// ", "/* ", " */", nil, nil},
	filecat.CSharp:     {filecat.CSharp, "// ", "/* ", " */", nil, nil},
	filecat.D:          {filecat.D, "// ", "/* ", " */", nil, nil},
	filecat.ObjC:       {filecat.ObjC, "// ", "/* ", " */", nil, nil},
	filecat.Go:         {filecat.Go, "// ", "/* ", " */", []LangFlags{IndentTab}, nil},
	filecat.Java:       {filecat.Java, "// ", "/* ", " */", nil, nil},
	filecat.JavaScript: {filecat.JavaScript, "// ", "/* ", " */", nil, nil},
	filecat.Eiffel:     {filecat.Eiffel, "--", "", "", nil, nil},
	filecat.Haskell:    {filecat.Haskell, "--", "{- ", "-}", nil, nil},
	filecat.Lisp:       {filecat.Lisp, "; ", "", "", nil, nil},
	filecat.Lua:        {filecat.Lua, "--", "---[[ ", "--]]", nil, nil},
	filecat.Makefile:   {filecat.Makefile, "# ", "", "", []LangFlags{IndentTab}, nil},
	filecat.Matlab:     {filecat.Matlab, "% ", "%{ ", " %}", nil, nil},
	filecat.OCaml:      {filecat.OCaml, "", "(* ", " *)", nil, nil},
	filecat.Pascal:     {filecat.Pascal, "// ", " ", " }", nil, nil},
	filecat.Perl:       {filecat.Perl, "# ", "", "", nil, nil},
	filecat.Python:     {filecat.Python, "# ", "", "", []LangFlags{IndentSpace}, nil},
	filecat.Php:        {filecat.Php, "// ", "/* ", " */", nil, nil},
	filecat.R:          {filecat.R, "# ", "", "", nil, nil},
	filecat.Ruby:       {filecat.Ruby, "# ", "", "", nil, nil},
	filecat.Rust:       {filecat.Rust, "// ", "/* ", " */", nil, nil},
	filecat.Scala:      {filecat.Scala, "// ", "/* ", " */", nil, nil},
	filecat.Html:       {filecat.Html, "", "<!-- ", " -->", nil, nil},
	filecat.TeX:        {filecat.TeX, "% ", "", "", nil, nil},
	filecat.Markdown:   {filecat.Markdown, "", "<!--- ", " -->", []LangFlags{IndentSpace}, nil},
}

// OpenStdParsers opens all the standard parsers for languages, from the langs/ directory
func OpenStdParsers() error {
	path, err := kit.GoSrcDir("github.com/goki/pi/langs")
	if err != nil {
		log.Println(err)
		return err
	}
	for sl, lp := range StdLangProps {
		ln := strings.ToLower(sl.String())
		fd := filepath.Join(path, ln)
		fn := filepath.Join(fd, ln+".pi")
		if _, err := os.Stat(fn); os.IsNotExist(err) {
			continue
		}
		pr := NewParser()
		pr.OpenJSON(fn)
		lp.Parser = pr
		StdLangProps[sl] = lp
	}
	return nil
}
