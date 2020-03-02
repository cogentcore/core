// Copyright (c) 2020, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Copied and only lightly modified from:
// https://github.com/nickng/bibtex
// Licenced under an Apache-2.0 licence
// and presumably Copyright (c) 2017 by Nick Ng

%{
package bibtex

import (
	"io"
)

type bibTag struct {
	key string
	val BibString
}

var bib *BibTex // Only for holding current bib
%}

%union {
	bibtex   *BibTex
	strval   string
	bibentry *BibEntry
	bibtag   *bibTag
	bibtags  []*bibTag
	strings  BibString
}

%token COMMENT STRING PREAMBLE
%token ATSIGN COLON EQUAL COMMA POUND LBRACE RBRACE DQUOTE LPAREN RPAREN
%token <strval> BAREIDENT IDENT
%type <bibtex> bibtex
%type <bibentry> bibentry
%type <bibtag> tag stringentry
%type <bibtags> tags
%type <strings> longstring preambleentry

%%

top : bibtex { }
    ;

bibtex : /* empty */          { $$ = NewBibTex(); bib = $$ }
       | bibtex bibentry      { $$ = $1; $$.AddEntry($2) }
       | bibtex commententry  { $$ = $1 }
       | bibtex stringentry   { $$ = $1; $$.AddStringVar($2.key, $2.val) }
       | bibtex preambleentry { $$ = $1; $$.AddPreamble($2) }
       ;

bibentry : ATSIGN BAREIDENT LBRACE BAREIDENT COMMA tags RBRACE { $$ = NewBibEntry($2, $4); for _, t := range $6 { $$.AddField(t.key, t.val) } }
         | ATSIGN BAREIDENT LPAREN BAREIDENT COMMA tags RPAREN { $$ = NewBibEntry($2, $4); for _, t := range $6 { $$.AddField(t.key, t.val) } }
         ;

commententry : ATSIGN COMMENT LBRACE longstring RBRACE {}
             | ATSIGN COMMENT LPAREN longstring RBRACE {}
             ;

stringentry : ATSIGN STRING LBRACE BAREIDENT EQUAL longstring RBRACE { $$ = &bibTag{key: $4, val: $6 } }
            | ATSIGN STRING LPAREN BAREIDENT EQUAL longstring RBRACE { $$ = &bibTag{key: $4, val: $6 } }
            ;

preambleentry : ATSIGN PREAMBLE LBRACE longstring RBRACE { $$ = $4 }
              | ATSIGN PREAMBLE LPAREN longstring RPAREN { $$ = $4 }
              ;

longstring :                  IDENT     { $$ = NewBibConst($1) }
           |                  BAREIDENT { $$ = bib.GetStringVar($1) }
           | longstring POUND IDENT     { $$ = NewBibComposite($1); $$.(*BibComposite).Append(NewBibConst($3))}
           | longstring POUND BAREIDENT { $$ = NewBibComposite($1); $$.(*BibComposite).Append(bib.GetStringVar($3)) }
           ;

tag : /* empty */                { }
    | BAREIDENT EQUAL longstring { $$ = &bibTag{key: $1, val: $3} }
    ;

tags : tag            { $$ = []*bibTag{$1} }
     | tags COMMA tag { if $3 == nil { $$ = $1 } else { $$ = append($1, $3) } }
     ;

%%

// Parse is the entry point to the bibtex parser.
func Parse(r io.Reader) (*BibTex, error) {
	l := NewLexer(r)
	bibtexParse(l)
	select {
	case err := <-l.Errors:
		return nil, err
	default:
		return bib, nil
	}
}
