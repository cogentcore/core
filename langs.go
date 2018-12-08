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

// LangProps contains properties of languages supported by the Pi parser
// framework
type LangProps struct {
	Lang      filecat.Supported `desc:"language -- must be a supported one from Supported list"`
	CommentLn string            `desc:"character(s) that start a single-line comment -- if empty then multi-line comment syntax will be used"`
	CommentSt string            `desc:"character(s) that start a multi-line comment or one that requires both start and end"`
	CommentEd string            `desc:"character(s) that end a multi-line comment or one that requires both start and end"`
	Parser    *Parser           `json:"-" xml:"-" desc:"parser for this language"`
}

// StdLangProps is the standard compiled-in set of langauge properties
var StdLangProps = map[filecat.Supported]LangProps{
	filecat.Ada:        {filecat.Ada, "--", "", "", nil},
	filecat.Bash:       {filecat.Bash, "# ", "", "", nil},
	filecat.Csh:        {filecat.Csh, "# ", "", "", nil},
	filecat.C:          {filecat.C, "// ", "/* ", " */", nil},
	filecat.CSharp:     {filecat.CSharp, "// ", "/* ", " */", nil},
	filecat.D:          {filecat.D, "// ", "/* ", " */", nil},
	filecat.ObjC:       {filecat.ObjC, "// ", "/* ", " */", nil},
	filecat.Go:         {filecat.Go, "// ", "/* ", " */", nil},
	filecat.Java:       {filecat.Java, "// ", "/* ", " */", nil},
	filecat.JavaScript: {filecat.JavaScript, "// ", "/* ", " */", nil},
	filecat.Eiffel:     {filecat.Eiffel, "--", "", "", nil},
	filecat.Haskell:    {filecat.Haskell, "--", "{- ", "-}", nil},
	filecat.Lisp:       {filecat.Lisp, "; ", "", "", nil},
	filecat.Lua:        {filecat.Lua, "--", "---[[ ", "--]]", nil},
	filecat.Makefile:   {filecat.Makefile, "# ", "", "", nil},
	filecat.Matlab:     {filecat.Matlab, "% ", "%{ ", " %}", nil},
	filecat.OCaml:      {filecat.OCaml, "", "(* ", " *)", nil},
	filecat.Pascal:     {filecat.Pascal, "// ", " ", " }", nil},
	filecat.Perl:       {filecat.Perl, "# ", "", "", nil},
	filecat.Python:     {filecat.Python, "# ", "", "", nil},
	filecat.Php:        {filecat.Php, "// ", "/* ", " */", nil},
	filecat.R:          {filecat.R, "# ", "", "", nil},
	filecat.Ruby:       {filecat.Ruby, "# ", "", "", nil},
	filecat.Rust:       {filecat.Rust, "// ", "/* ", " */", nil},
	filecat.Scala:      {filecat.Scala, "// ", "/* ", " */", nil},
	filecat.Html:       {filecat.Html, "", "<!-- ", " -->", nil},
	filecat.TeX:        {filecat.TeX, "% ", "", "", nil},
	filecat.Markdown:   {filecat.Markdown, "", "<!--- ", " -->", nil},
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
