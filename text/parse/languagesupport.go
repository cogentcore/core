// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parse

import (
	"fmt"
	"time"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/text/parse/languages"
	"cogentcore.org/core/text/parse/lexer"
)

// LanguageFlags are special properties of a given language
type LanguageFlags int32 //enums:enum

// LangFlags
const (
	// NoFlags = nothing special
	NoFlags LanguageFlags = iota

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

// LanguageProperties contains properties of languages supported by the parser
// framework
type LanguageProperties struct {

	// known language -- must be a supported one from Known list
	Known fileinfo.Known

	// character(s) that start a single-line comment -- if empty then multi-line comment syntax will be used
	CommentLn string

	// character(s) that start a multi-line comment or one that requires both start and end
	CommentSt string

	// character(s) that end a multi-line comment or one that requires both start and end
	CommentEd string

	// special properties for this language -- as an explicit list of options to make them easier to see and set in defaults
	Flags []LanguageFlags

	// Lang interface for this language
	Lang Language `json:"-" xml:"-"`

	// parser for this language -- initialized in OpenStandard
	Parser *Parser `json:"-" xml:"-"`
}

// HasFlag returns true if given flag is set in Flags
func (lp *LanguageProperties) HasFlag(flg LanguageFlags) bool {
	for _, f := range lp.Flags {
		if f == flg {
			return true
		}
	}
	return false
}

// StandardLanguageProperties is the standard compiled-in set of language properties
var StandardLanguageProperties = map[fileinfo.Known]*LanguageProperties{
	fileinfo.Ada:        {fileinfo.Ada, "--", "", "", nil, nil, nil},
	fileinfo.Bash:       {fileinfo.Bash, "# ", "", "", nil, nil, nil},
	fileinfo.Csh:        {fileinfo.Csh, "# ", "", "", nil, nil, nil},
	fileinfo.C:          {fileinfo.C, "// ", "/* ", " */", nil, nil, nil},
	fileinfo.CSharp:     {fileinfo.CSharp, "// ", "/* ", " */", nil, nil, nil},
	fileinfo.D:          {fileinfo.D, "// ", "/* ", " */", nil, nil, nil},
	fileinfo.ObjC:       {fileinfo.ObjC, "// ", "/* ", " */", nil, nil, nil},
	fileinfo.Go:         {fileinfo.Go, "// ", "/* ", " */", []LanguageFlags{IndentTab}, nil, nil},
	fileinfo.Java:       {fileinfo.Java, "// ", "/* ", " */", nil, nil, nil},
	fileinfo.JavaScript: {fileinfo.JavaScript, "// ", "/* ", " */", nil, nil, nil},
	fileinfo.Eiffel:     {fileinfo.Eiffel, "--", "", "", nil, nil, nil},
	fileinfo.Haskell:    {fileinfo.Haskell, "--", "{- ", "-}", nil, nil, nil},
	fileinfo.Lisp:       {fileinfo.Lisp, "; ", "", "", nil, nil, nil},
	fileinfo.Lua:        {fileinfo.Lua, "--", "---[[ ", "--]]", nil, nil, nil},
	fileinfo.Makefile:   {fileinfo.Makefile, "# ", "", "", []LanguageFlags{IndentTab}, nil, nil},
	fileinfo.Matlab:     {fileinfo.Matlab, "% ", "%{ ", " %}", nil, nil, nil},
	fileinfo.OCaml:      {fileinfo.OCaml, "", "(* ", " *)", nil, nil, nil},
	fileinfo.Pascal:     {fileinfo.Pascal, "// ", " ", " }", nil, nil, nil},
	fileinfo.Perl:       {fileinfo.Perl, "# ", "", "", nil, nil, nil},
	fileinfo.Python:     {fileinfo.Python, "# ", "", "", []LanguageFlags{IndentSpace}, nil, nil},
	fileinfo.Php:        {fileinfo.Php, "// ", "/* ", " */", nil, nil, nil},
	fileinfo.R:          {fileinfo.R, "# ", "", "", nil, nil, nil},
	fileinfo.Ruby:       {fileinfo.Ruby, "# ", "", "", nil, nil, nil},
	fileinfo.Rust:       {fileinfo.Rust, "// ", "/* ", " */", nil, nil, nil},
	fileinfo.Scala:      {fileinfo.Scala, "// ", "/* ", " */", nil, nil, nil},
	fileinfo.Html:       {fileinfo.Html, "", "<!-- ", " -->", nil, nil, nil},
	fileinfo.TeX:        {fileinfo.TeX, "% ", "", "", nil, nil, nil},
	fileinfo.Markdown:   {fileinfo.Markdown, "", "<!--- ", " -->", []LanguageFlags{IndentSpace}, nil, nil},
	fileinfo.Yaml:       {fileinfo.Yaml, "#", "", "", []LanguageFlags{IndentSpace}, nil, nil},
}

// LanguageSupporter provides general support for supported languages.
// e.g., looking up lexers and parsers by name.
// Also implements the lexer.LangLexer interface to provide access to other
// Guest Lexers
type LanguageSupporter struct{}

// LanguageSupport is the main language support hub for accessing parse
// support interfaces for each supported language
var LanguageSupport = LanguageSupporter{}

// OpenStandard opens all the standard parsers for languages, from the langs/ directory
func (ll *LanguageSupporter) OpenStandard() error {
	lexer.TheLanguageLexer = &LanguageSupport

	for sl, lp := range StandardLanguageProperties {
		pib, err := languages.OpenParser(sl)
		if err != nil {
			continue
		}
		pr := NewParser()
		err = pr.ReadJSON(pib)
		if err != nil {
			return errors.Log(err)
		}
		pr.ModTime = time.Date(2023, 02, 10, 00, 00, 00, 0, time.UTC)
		pr.InitAll()
		lp.Parser = pr
	}
	return nil
}

// Properties looks up language properties by fileinfo.Known const int type
func (ll *LanguageSupporter) Properties(sup fileinfo.Known) (*LanguageProperties, error) {
	lp, has := StandardLanguageProperties[sup]
	if !has {
		err := fmt.Errorf("parse.LangSupport.Properties: no specific support for language: %v", sup)
		return nil, err
	}
	return lp, nil
}

// PropertiesByName looks up language properties by string name of language
// (with case-insensitive fallback). Returns error if not supported.
func (ll *LanguageSupporter) PropertiesByName(lang string) (*LanguageProperties, error) {
	sup, err := fileinfo.KnownByName(lang)
	if err != nil {
		// log.Println(err.Error()) // don't want output during lexing..
		return nil, err
	}
	return ll.Properties(sup)
}

// LexerByName looks up Lexer for given language by name
// (with case-insensitive fallback). Returns nil if not supported.
func (ll *LanguageSupporter) LexerByName(lang string) *lexer.Rule {
	lp, err := ll.PropertiesByName(lang)
	if err != nil {
		return nil
	}
	if lp.Parser == nil {
		// log.Printf("core.LangSupport: no lexer / parser support for language: %v\n", lang)
		return nil
	}
	return lp.Parser.Lexer
}
