// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pi

import (
	"fmt"
	"log"
	"time"

	"goki.dev/fi"
	"goki.dev/pi/v2/langs"
	"goki.dev/pi/v2/lex"
)

// LangFlags are special properties of a given language
type LangFlags int32 //enums:enum

// LangFlags
const (
	// NoFlags = nothing special
	NoFlags LangFlags = iota

	// IndentSpace means that spaces must be used for this language
	IndentSpace

	// IndentTab means that tabs must be used for this language
	IndentTab

	// ReAutoIndent causes current line to be re-indented during AutoIndent for Enter
	// (newline) -- this should only be set for strongly indented languages where
	// the previous + current line can tell you exactly what indent the current line
	// should be at.
	ReAutoIndent
)

// LangProps contains properties of languages supported by the Pi parser
// framework
type LangProps struct {

	// known language -- must be a supported one from Known list
	Known fi.Known

	// character(s) that start a single-line comment -- if empty then multi-line comment syntax will be used
	CommentLn string

	// character(s) that start a multi-line comment or one that requires both start and end
	CommentSt string

	// character(s) that end a multi-line comment or one that requires both start and end
	CommentEd string

	// special properties for this language -- as an explicit list of options to make them easier to see and set in defaults
	Flags []LangFlags

	// Lang interface for this language
	Lang Lang `json:"-" xml:"-"`

	// parser for this language -- initialized in OpenStd
	Parser *Parser `json:"-" xml:"-"`
}

// HasFlag returns true if given flag is set in Flags
func (lp *LangProps) HasFlag(flg LangFlags) bool {
	for _, f := range lp.Flags {
		if f == flg {
			return true
		}
	}
	return false
}

// StdLangProps is the standard compiled-in set of language properties
var StdLangProps = map[fi.Known]*LangProps{
	fi.Ada:        {fi.Ada, "--", "", "", nil, nil, nil},
	fi.Bash:       {fi.Bash, "# ", "", "", nil, nil, nil},
	fi.Csh:        {fi.Csh, "# ", "", "", nil, nil, nil},
	fi.C:          {fi.C, "// ", "/* ", " */", nil, nil, nil},
	fi.CSharp:     {fi.CSharp, "// ", "/* ", " */", nil, nil, nil},
	fi.D:          {fi.D, "// ", "/* ", " */", nil, nil, nil},
	fi.ObjC:       {fi.ObjC, "// ", "/* ", " */", nil, nil, nil},
	fi.Go:         {fi.Go, "// ", "/* ", " */", []LangFlags{IndentTab}, nil, nil},
	fi.Java:       {fi.Java, "// ", "/* ", " */", nil, nil, nil},
	fi.JavaScript: {fi.JavaScript, "// ", "/* ", " */", nil, nil, nil},
	fi.Eiffel:     {fi.Eiffel, "--", "", "", nil, nil, nil},
	fi.Haskell:    {fi.Haskell, "--", "{- ", "-}", nil, nil, nil},
	fi.Lisp:       {fi.Lisp, "; ", "", "", nil, nil, nil},
	fi.Lua:        {fi.Lua, "--", "---[[ ", "--]]", nil, nil, nil},
	fi.Makefile:   {fi.Makefile, "# ", "", "", []LangFlags{IndentTab}, nil, nil},
	fi.Matlab:     {fi.Matlab, "% ", "%{ ", " %}", nil, nil, nil},
	fi.OCaml:      {fi.OCaml, "", "(* ", " *)", nil, nil, nil},
	fi.Pascal:     {fi.Pascal, "// ", " ", " }", nil, nil, nil},
	fi.Perl:       {fi.Perl, "# ", "", "", nil, nil, nil},
	fi.Python:     {fi.Python, "# ", "", "", []LangFlags{IndentSpace}, nil, nil},
	fi.Php:        {fi.Php, "// ", "/* ", " */", nil, nil, nil},
	fi.R:          {fi.R, "# ", "", "", nil, nil, nil},
	fi.Ruby:       {fi.Ruby, "# ", "", "", nil, nil, nil},
	fi.Rust:       {fi.Rust, "// ", "/* ", " */", nil, nil, nil},
	fi.Scala:      {fi.Scala, "// ", "/* ", " */", nil, nil, nil},
	fi.Html:       {fi.Html, "", "<!-- ", " -->", nil, nil, nil},
	fi.TeX:        {fi.TeX, "% ", "", "", nil, nil, nil},
	fi.Markdown:   {fi.Markdown, "", "<!--- ", " -->", []LangFlags{IndentSpace}, nil, nil},
	fi.Yaml:       {fi.Yaml, "#", "", "", []LangFlags{IndentSpace}, nil, nil},
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

	for sl, lp := range StdLangProps {
		pib, err := langs.OpenParser(sl)
		if err != nil {
			continue
		}
		pr := NewParser()
		err = pr.ReadJSON(pib)
		if err != nil {
			log.Println(err)
			return nil
		}
		pr.ModTime = time.Date(2023, 02, 10, 00, 00, 00, 0, time.UTC)
		pr.InitAll()
		lp.Parser = pr
	}
	return nil
}

// Props looks up language properties by fi.Known const int type
func (ll *LangSupporter) Props(sup fi.Known) (*LangProps, error) {
	lp, has := StdLangProps[sup]
	if !has {
		err := fmt.Errorf("pi.LangSupport.Props: no specific support for language: %v", sup)
		//		log.Println(err.Error()) // don't want output
		return nil, err
	}
	return lp, nil
}

// PropsByName looks up language properties by string name of language
// (with case-insensitive fallback). Returns error if not supported.
func (ll *LangSupporter) PropsByName(lang string) (*LangProps, error) {
	sup, err := fi.KnownByName(lang)
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
