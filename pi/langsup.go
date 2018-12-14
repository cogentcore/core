// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pi

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/goki/gi/filecat"
	"github.com/goki/ki/dirs"
	"github.com/goki/ki/kit"
	"github.com/goki/pi/lex"
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
	Sup       filecat.Supported `desc:"language -- must be a supported one from Supported list"`
	CommentLn string            `desc:"character(s) that start a single-line comment -- if empty then multi-line comment syntax will be used"`
	CommentSt string            `desc:"character(s) that start a multi-line comment or one that requires both start and end"`
	CommentEd string            `desc:"character(s) that end a multi-line comment or one that requires both start and end"`
	Flags     []LangFlags       `desc:"special properties for this language"`
	Lang      Lang              `json:"-" xml:"-" desc:"Lang interface for this language"`
	Parser    *Parser           `json:"-" xml:"-" desc:"parser for this language -- initialized in OpenStd"`
}

// StdLangProps is the standard compiled-in set of langauge properties
var StdLangProps = map[filecat.Supported]LangProps{
	filecat.Ada:        {filecat.Ada, "--", "", "", nil, nil, nil},
	filecat.Bash:       {filecat.Bash, "# ", "", "", nil, nil, nil},
	filecat.Csh:        {filecat.Csh, "# ", "", "", nil, nil, nil},
	filecat.C:          {filecat.C, "// ", "/* ", " */", nil, nil, nil},
	filecat.CSharp:     {filecat.CSharp, "// ", "/* ", " */", nil, nil, nil},
	filecat.D:          {filecat.D, "// ", "/* ", " */", nil, nil, nil},
	filecat.ObjC:       {filecat.ObjC, "// ", "/* ", " */", nil, nil, nil},
	filecat.Go:         {filecat.Go, "// ", "/* ", " */", []LangFlags{IndentTab}, &TheGoLang, nil},
	filecat.Java:       {filecat.Java, "// ", "/* ", " */", nil, nil, nil},
	filecat.JavaScript: {filecat.JavaScript, "// ", "/* ", " */", nil, nil, nil},
	filecat.Eiffel:     {filecat.Eiffel, "--", "", "", nil, nil, nil},
	filecat.Haskell:    {filecat.Haskell, "--", "{- ", "-}", nil, nil, nil},
	filecat.Lisp:       {filecat.Lisp, "; ", "", "", nil, nil, nil},
	filecat.Lua:        {filecat.Lua, "--", "---[[ ", "--]]", nil, nil, nil},
	filecat.Makefile:   {filecat.Makefile, "# ", "", "", []LangFlags{IndentTab}, nil, nil},
	filecat.Matlab:     {filecat.Matlab, "% ", "%{ ", " %}", nil, nil, nil},
	filecat.OCaml:      {filecat.OCaml, "", "(* ", " *)", nil, nil, nil},
	filecat.Pascal:     {filecat.Pascal, "// ", " ", " }", nil, nil, nil},
	filecat.Perl:       {filecat.Perl, "# ", "", "", nil, nil, nil},
	filecat.Python:     {filecat.Python, "# ", "", "", []LangFlags{IndentSpace}, nil, nil},
	filecat.Php:        {filecat.Php, "// ", "/* ", " */", nil, nil, nil},
	filecat.R:          {filecat.R, "# ", "", "", nil, nil, nil},
	filecat.Ruby:       {filecat.Ruby, "# ", "", "", nil, nil, nil},
	filecat.Rust:       {filecat.Rust, "// ", "/* ", " */", nil, nil, nil},
	filecat.Scala:      {filecat.Scala, "// ", "/* ", " */", nil, nil, nil},
	filecat.Html:       {filecat.Html, "", "<!-- ", " -->", nil, nil, nil},
	filecat.TeX:        {filecat.TeX, "% ", "", "", nil, nil, nil},
	filecat.Markdown:   {filecat.Markdown, "", "<!--- ", " -->", []LangFlags{IndentSpace}, &TheMarkdownLang, nil},
}

// LangSupporter provides general support for supported languages.
// e.g., looking up lexers and parsers by name.
// Also implements the lex.LangLexer interface to provide access to other
// Guest Lexers
type LangSupporter struct {
}

// LangSupport is the main language support hub for accessing GoPi
// support interfaces for each supported language
var LangSupport = LangSupporter{}

// OpenStd opens all the standard parsers for languages, from the langs/ directory
func (ll *LangSupporter) OpenStd() error {
	lex.TheLangLexer = &LangSupport

	path, err := dirs.GoSrcDir("github.com/goki/pi/langs")
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

// Props looks up language properties by filecat.Supported const int type
func (ll *LangSupporter) Props(sup filecat.Supported) (*LangProps, error) {
	lp, has := StdLangProps[sup]
	if !has {
		err := fmt.Errorf("pi.LangSupport.Props: no specific support for language: %v", sup)
		//		log.Println(err.Error()) // don't want output
		return nil, err
	}
	return &lp, nil
}

// PropsByName looks up language properties by string name of language
// (with case-insensitive fallback). Returns error if not supported.
func (ll *LangSupporter) PropsByName(lang string) (*LangProps, error) {
	sup, err := filecat.SupportedByName(lang)
	if err != nil {
		// log.Println(err.Error()) // don't want output during lexing..
		return nil, err
	}
	return ll.Props(sup)
}

// LexerByName looks up Lexer for given language by name
// (with case-insensitive fallback). Returns nil if not supported.
func (ll *LangSupporter) LexerByName(lang string) *lex.Rule {
	lp, err := ll.PropsByName(lang)
	if err != nil {
		return nil
	}
	if lp.Parser == nil {
		// log.Printf("gi.LangSupport: no lexer / parser support for language: %v\n", lang)
		return nil
	}
	return &lp.Parser.Lexer
}
