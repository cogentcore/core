// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package pi provides the overall integration and coordination of the several
// sub-packages that comprise the interactive parser (GoPi) system.  It has
// the end-user high-level functions in the pi.Parser class for processing
// files through the Lexer and Parser stages, and specific custom functions
// for processing supported languages, via a common interface.
//
// The basic machinery of lexing and parsing is implemented in the lex and parse
// sub-packages, which each work in a completely general-purpose manner across
// all supported languages, and can be re-used independently for any other such
// specific purpose outside of the full pi system.  Each of them depend on the
// token.Tokens constants defined in the token package, which provides a
// "universal language" of lexical tokens used across all supported languages
// and syntax highlighting cases, based originally on pygments via chroma and
// since expanded and systematized from there.
//
// The parse package produces an abstract syntax tree (AST) representation
// of the source, and lists (maps) of symbols that can be used for name lookup
// and completion (types, variables, functions, etc).  Those symbol structures
// are defined in the syms sub-package.
//
// To more effectively manage and organize the symbols from parsing,
// language-specific logic is required, and this is supported by the
// Lang interface, which is implemented for each of the supported
// languages (see lang.go and e.g., go.go).
//
// The LangSupport variable provides the hub for accessing interfaces
// for supported languages, using the StdLangProps map which
// provides a lookup from the filecat.Supported language name to its
// associated Lang interface and pi.Parser parser.
// Thus you can go from the GoGi giv.FileInfo.Sup field to its
// associated GoPi methods using this map (and associated LangSupport
// methods).  This map is extensible and other supported languages
// can be added in other packages.  This requires a dependency on
// gi/filecat sub-module in GoGi, which defines a broad set of supported
// file categories and associated mime types, etc, which are generally
// supported within the GoGi gui framework -- a subset of these are the
// languages and file formats supported by GoPi parsing / lexing.
//
// The piv sub-package provides the GUI for constructing and testing a
// lexer and parser interactively.  It is the only sub-package with
// significant dependencies, especially on GoGi and Gide.
//
package pi
