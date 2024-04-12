// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pi

import (
	"fmt"
	"log"
	"time"

	"cogentcore.org/core/fileinfo"
	"cogentcore.org/core/pi/langs"
	"cogentcore.org/core/pi/lexer"
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

// LangProperties contains properties of languages supported by the Pi parser
// framework
type LangProperties struct {

	// known language -- must be a supported one from Known list
	Known fileinfo.Known

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
func (lp *LangProperties) HasFlag(flg LangFlags) bool {
	for _, f := range lp.Flags {
		if f == flg {
			return true
		}
	}
	return false
}

// StandardLangProperties is the standard compiled-in set of language properties
var StandardLangProperties = map[fileinfo.Known]*LangProperties{
	fileinfo.Ada:        {fileinfo.Ada, "--", "", "", nil, nil, nil},
	fileinfo.Bash:       {fileinfo.Bash, "# ", "", "", nil, nil, nil},
	fileinfo.Csh:        {fileinfo.Csh, "# ", "", "", nil, nil, nil},
	fileinfo.C:          {fileinfo.C, "// ", "/* ", " */", nil, nil, nil},
	fileinfo.CSharp:     {fileinfo.CSharp, "// ", "/* ", " */", nil, nil, nil},
	fileinfo.D:          {fileinfo.D, "// ", "/* ", " */", nil, nil, nil},
	fileinfo.ObjC:       {fileinfo.ObjC, "// ", "/* ", " */", nil, nil, nil},
	fileinfo.Go:         {fileinfo.Go, "// ", "/* ", " */", []LangFlags{IndentTab}, nil, nil},
	fileinfo.Java:       {fileinfo.Java, "// ", "/* ", " */", nil, nil, nil},
	fileinfo.JavaScript: {fileinfo.JavaScript, "// ", "/* ", " */", nil, nil, nil},
	fileinfo.Eiffel:     {fileinfo.Eiffel, "--", "", "", nil, nil, nil},
	fileinfo.Haskell:    {fileinfo.Haskell, "--", "{- ", "-}", nil, nil, nil},
	fileinfo.Lisp:       {fileinfo.Lisp, "; ", "", "", nil, nil, nil},
	fileinfo.Lua:        {fileinfo.Lua, "--", "---[[ ", "--]]", nil, nil, nil},
	fileinfo.Makefile:   {fileinfo.Makefile, "# ", "", "", []LangFlags{IndentTab}, nil, nil},
	fileinfo.Matlab:     {fileinfo.Matlab, "% ", "%{ ", " %}", nil, nil, nil},
	fileinfo.OCaml:      {fileinfo.OCaml, "", "(* ", " *)", nil, nil, nil},
	fileinfo.Pascal:     {fileinfo.Pascal, "// ", " ", " }", nil, nil, nil},
	fileinfo.Perl:       {fileinfo.Perl, "# ", "", "", nil, nil, nil},
	fileinfo.Python:     {fileinfo.Python, "# ", "", "", []LangFlags{IndentSpace}, nil, nil},
	fileinfo.Php:        {fileinfo.Php, "// ", "/* ", " */", nil, nil, nil},
	fileinfo.R:          {fileinfo.R, "# ", "", "", nil, nil, nil},
	fileinfo.Ruby:       {fileinfo.Ruby, "# ", "", "", nil, nil, nil},
	fileinfo.Rust:       {fileinfo.Rust, "// ", "/* ", " */", nil, nil, nil},
	fileinfo.Scala:      {fileinfo.Scala, "// ", "/* ", " */", nil, nil, nil},
	fileinfo.Html:       {fileinfo.Html, "", "<!-- ", " -->", nil, nil, nil},
	fileinfo.TeX:        {fileinfo.TeX, "% ", "", "", nil, nil, nil},
	fileinfo.Markdown:   {fileinfo.Markdown, "", "<!--- ", " -->", []LangFlags{IndentSpace}, nil, nil},
	fileinfo.Yaml:       {fileinfo.Yaml, "#", "", "", []LangFlags{IndentSpace}, nil, nil},
}

// LangSupporter provides general support for supported languages.
// e.g., looking up lexers and parsers by name.
// Also implements the lexer.LangLexer interface to provide access to other
// Guest Lexers
type LangSupporter struct {
}

// LangSupport is the main language support hub for accessing GoPi
// support interfaces for each supported language
var LangSupport = LangSupporter{}

// OpenStandard opens all the standard parsers for languages, from the langs/ directory
func (ll *LangSupporter) OpenStandard() error {
	lexer.TheLangLexer = &LangSupport

	for sl, lp := range StandardLangProperties {
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

// Properties looks up language properties by fileinfo.Known const int type
func (ll *LangSupporter) Properties(sup fileinfo.Known) (*LangProperties, error) {
	lp, has := StandardLangProperties[sup]
	if !has {
		err := fmt.Errorf("pi.LangSupport.Properties: no specific support for language: %v", sup)
		//		log.Println(err.Error()) // don't want output
		return nil, err
	}
	return lp, nil
}

// PropertiesByName looks up language properties by string name of language
// (with case-insensitive fallback). Returns error if not supported.
func (ll *LangSupporter) PropertiesByName(lang string) (*LangProperties, error) {
	sup, err := fileinfo.KnownByName(lang)
	if err != nil {
		// log.Println(err.Error()) // don't want output during lexing..
		return nil, err
	}
	return ll.Properties(sup)
}

// LexerByName looks up Lexer for given language by name
// (with case-insensitive fallback). Returns nil if not supported.
func (ll *LangSupporter) LexerByName(lang string) *lexer.Rule {
	lp, err := ll.PropertiesByName(lang)
	if err != nil {
		return nil
	}
	if lp.Parser == nil {
		// log.Printf("core.LangSupport: no lexer / parser support for language: %v\n", lang)
		return nil
	}
	return &lp.Parser.Lexer
}
