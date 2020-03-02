// Copyright (c) 2020, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Copied and only lightly modified from:
// https://github.com/nickng/bibtex
// Licenced under an Apache-2.0 licence
// and presumably Copyright (c) 2017 by Nick Ng

//line bibtex.y:2
package bibtex

import (
	__yyfmt__ "fmt"
	"io"
)

//line bibtex.y:2

type bibTag struct {
	key string
	val BibString
}

var bib *BibTex // Only for holding current bib

//line bibtex.y:16
type bibtexSymType struct {
	yys      int
	bibtex   *BibTex
	strval   string
	bibentry *BibEntry
	bibtag   *bibTag
	bibtags  []*bibTag
	strings  BibString
}

const COMMENT = 57346
const STRING = 57347
const PREAMBLE = 57348
const ATSIGN = 57349
const COLON = 57350
const EQUAL = 57351
const COMMA = 57352
const POUND = 57353
const LBRACE = 57354
const RBRACE = 57355
const DQUOTE = 57356
const LPAREN = 57357
const RPAREN = 57358
const BAREIDENT = 57359
const IDENT = 57360

var bibtexToknames = [...]string{
	"$end",
	"error",
	"$unk",
	"COMMENT",
	"STRING",
	"PREAMBLE",
	"ATSIGN",
	"COLON",
	"EQUAL",
	"COMMA",
	"POUND",
	"LBRACE",
	"RBRACE",
	"DQUOTE",
	"LPAREN",
	"RPAREN",
	"BAREIDENT",
	"IDENT",
}
var bibtexStatenames = [...]string{}

const bibtexEofCode = 1
const bibtexErrCode = 2
const bibtexInitialStackSize = 16

//line bibtex.y:76

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

//line yacctab:1
var bibtexExca = [...]int{
	-1, 1,
	1, -1,
	-2, 0,
}

const bibtexPrivate = 57344

const bibtexLast = 61

var bibtexAct = [...]int{

	22, 39, 40, 41, 9, 10, 11, 24, 23, 44,
	43, 27, 48, 26, 21, 20, 25, 8, 50, 28,
	29, 33, 33, 49, 18, 16, 38, 19, 17, 14,
	31, 12, 15, 42, 13, 30, 45, 46, 33, 33,
	52, 51, 48, 36, 33, 47, 37, 33, 35, 34,
	54, 53, 33, 7, 32, 4, 1, 6, 5, 3,
	2,
}
var bibtexPact = [...]int{

	-1000, -1000, 46, -1000, -1000, -1000, -1000, 0, 19, 17,
	13, 12, -2, -3, -10, -10, -4, -6, -10, -10,
	25, 20, 41, -1000, -1000, 36, 39, 34, 33, 10,
	-14, -14, -1000, -8, -1000, -10, -10, -1000, -1000, 32,
	-1000, 14, 2, -1000, -1000, 28, 27, -1000, -14, -10,
	-1000, -1000, -1000, -1000, 11,
}
var bibtexPgo = [...]int{

	0, 60, 59, 2, 58, 1, 0, 57, 56, 55,
}
var bibtexR1 = [...]int{

	0, 8, 1, 1, 1, 1, 1, 2, 2, 9,
	9, 4, 4, 7, 7, 6, 6, 6, 6, 3,
	3, 5, 5,
}
var bibtexR2 = [...]int{

	0, 1, 0, 2, 2, 2, 2, 7, 7, 5,
	5, 7, 7, 5, 5, 1, 1, 3, 3, 0,
	3, 1, 3,
}
var bibtexChk = [...]int{

	-1000, -8, -1, -2, -9, -4, -7, 7, 17, 4,
	5, 6, 12, 15, 12, 15, 12, 15, 12, 15,
	17, 17, -6, 18, 17, -6, 17, 17, -6, -6,
	10, 10, 13, 11, 13, 9, 9, 13, 16, -5,
	-3, 17, -5, 18, 17, -6, -6, 13, 10, 9,
	16, 13, 13, -3, -6,
}
var bibtexDef = [...]int{

	2, -2, 1, 3, 4, 5, 6, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 15, 16, 0, 0, 0, 0, 0,
	19, 19, 9, 0, 10, 0, 0, 13, 14, 0,
	21, 0, 0, 17, 18, 0, 0, 7, 19, 0,
	8, 11, 12, 22, 20,
}
var bibtexTok1 = [...]int{

	1,
}
var bibtexTok2 = [...]int{

	2, 3, 4, 5, 6, 7, 8, 9, 10, 11,
	12, 13, 14, 15, 16, 17, 18,
}
var bibtexTok3 = [...]int{
	0,
}

var bibtexErrorMessages = [...]struct {
	state int
	token int
	msg   string
}{}

//line yaccpar:1

/*	parser for yacc output	*/

var (
	bibtexDebug        = 0
	bibtexErrorVerbose = false
)

type bibtexLexer interface {
	Lex(lval *bibtexSymType) int
	Error(s string)
}

type bibtexParser interface {
	Parse(bibtexLexer) int
	Lookahead() int
}

type bibtexParserImpl struct {
	lval  bibtexSymType
	stack [bibtexInitialStackSize]bibtexSymType
	char  int
}

func (p *bibtexParserImpl) Lookahead() int {
	return p.char
}

func bibtexNewParser() bibtexParser {
	return &bibtexParserImpl{}
}

const bibtexFlag = -1000

func bibtexTokname(c int) string {
	if c >= 1 && c-1 < len(bibtexToknames) {
		if bibtexToknames[c-1] != "" {
			return bibtexToknames[c-1]
		}
	}
	return __yyfmt__.Sprintf("tok-%v", c)
}

func bibtexStatname(s int) string {
	if s >= 0 && s < len(bibtexStatenames) {
		if bibtexStatenames[s] != "" {
			return bibtexStatenames[s]
		}
	}
	return __yyfmt__.Sprintf("state-%v", s)
}

func bibtexErrorMessage(state, lookAhead int) string {
	const TOKSTART = 4

	if !bibtexErrorVerbose {
		return "syntax error"
	}

	for _, e := range bibtexErrorMessages {
		if e.state == state && e.token == lookAhead {
			return "syntax error: " + e.msg
		}
	}

	res := "syntax error: unexpected " + bibtexTokname(lookAhead)

	// To match Bison, suggest at most four expected tokens.
	expected := make([]int, 0, 4)

	// Look for shiftable tokens.
	base := bibtexPact[state]
	for tok := TOKSTART; tok-1 < len(bibtexToknames); tok++ {
		if n := base + tok; n >= 0 && n < bibtexLast && bibtexChk[bibtexAct[n]] == tok {
			if len(expected) == cap(expected) {
				return res
			}
			expected = append(expected, tok)
		}
	}

	if bibtexDef[state] == -2 {
		i := 0
		for bibtexExca[i] != -1 || bibtexExca[i+1] != state {
			i += 2
		}

		// Look for tokens that we accept or reduce.
		for i += 2; bibtexExca[i] >= 0; i += 2 {
			tok := bibtexExca[i]
			if tok < TOKSTART || bibtexExca[i+1] == 0 {
				continue
			}
			if len(expected) == cap(expected) {
				return res
			}
			expected = append(expected, tok)
		}

		// If the default action is to accept or reduce, give up.
		if bibtexExca[i+1] != 0 {
			return res
		}
	}

	for i, tok := range expected {
		if i == 0 {
			res += ", expecting "
		} else {
			res += " or "
		}
		res += bibtexTokname(tok)
	}
	return res
}

func bibtexlex1(lex bibtexLexer, lval *bibtexSymType) (char, token int) {
	token = 0
	char = lex.Lex(lval)
	if char <= 0 {
		token = bibtexTok1[0]
		goto out
	}
	if char < len(bibtexTok1) {
		token = bibtexTok1[char]
		goto out
	}
	if char >= bibtexPrivate {
		if char < bibtexPrivate+len(bibtexTok2) {
			token = bibtexTok2[char-bibtexPrivate]
			goto out
		}
	}
	for i := 0; i < len(bibtexTok3); i += 2 {
		token = bibtexTok3[i+0]
		if token == char {
			token = bibtexTok3[i+1]
			goto out
		}
	}

out:
	if token == 0 {
		token = bibtexTok2[1] /* unknown char */
	}
	if bibtexDebug >= 3 {
		__yyfmt__.Printf("lex %s(%d)\n", bibtexTokname(token), uint(char))
	}
	return char, token
}

func bibtexParse(bibtexlex bibtexLexer) int {
	return bibtexNewParser().Parse(bibtexlex)
}

func (bibtexrcvr *bibtexParserImpl) Parse(bibtexlex bibtexLexer) int {
	var bibtexn int
	var bibtexVAL bibtexSymType
	var bibtexDollar []bibtexSymType
	_ = bibtexDollar // silence set and not used
	bibtexS := bibtexrcvr.stack[:]

	Nerrs := 0   /* number of errors */
	Errflag := 0 /* error recovery flag */
	bibtexstate := 0
	bibtexrcvr.char = -1
	bibtextoken := -1 // bibtexrcvr.char translated into internal numbering
	defer func() {
		// Make sure we report no lookahead when not parsing.
		bibtexstate = -1
		bibtexrcvr.char = -1
		bibtextoken = -1
	}()
	bibtexp := -1
	goto bibtexstack

ret0:
	return 0

ret1:
	return 1

bibtexstack:
	/* put a state and value onto the stack */
	if bibtexDebug >= 4 {
		__yyfmt__.Printf("char %v in %v\n", bibtexTokname(bibtextoken), bibtexStatname(bibtexstate))
	}

	bibtexp++
	if bibtexp >= len(bibtexS) {
		nyys := make([]bibtexSymType, len(bibtexS)*2)
		copy(nyys, bibtexS)
		bibtexS = nyys
	}
	bibtexS[bibtexp] = bibtexVAL
	bibtexS[bibtexp].yys = bibtexstate

bibtexnewstate:
	bibtexn = bibtexPact[bibtexstate]
	if bibtexn <= bibtexFlag {
		goto bibtexdefault /* simple state */
	}
	if bibtexrcvr.char < 0 {
		bibtexrcvr.char, bibtextoken = bibtexlex1(bibtexlex, &bibtexrcvr.lval)
	}
	bibtexn += bibtextoken
	if bibtexn < 0 || bibtexn >= bibtexLast {
		goto bibtexdefault
	}
	bibtexn = bibtexAct[bibtexn]
	if bibtexChk[bibtexn] == bibtextoken { /* valid shift */
		bibtexrcvr.char = -1
		bibtextoken = -1
		bibtexVAL = bibtexrcvr.lval
		bibtexstate = bibtexn
		if Errflag > 0 {
			Errflag--
		}
		goto bibtexstack
	}

bibtexdefault:
	/* default state action */
	bibtexn = bibtexDef[bibtexstate]
	if bibtexn == -2 {
		if bibtexrcvr.char < 0 {
			bibtexrcvr.char, bibtextoken = bibtexlex1(bibtexlex, &bibtexrcvr.lval)
		}

		/* look through exception table */
		xi := 0
		for {
			if bibtexExca[xi+0] == -1 && bibtexExca[xi+1] == bibtexstate {
				break
			}
			xi += 2
		}
		for xi += 2; ; xi += 2 {
			bibtexn = bibtexExca[xi+0]
			if bibtexn < 0 || bibtexn == bibtextoken {
				break
			}
		}
		bibtexn = bibtexExca[xi+1]
		if bibtexn < 0 {
			goto ret0
		}
	}
	if bibtexn == 0 {
		/* error ... attempt to resume parsing */
		switch Errflag {
		case 0: /* brand new error */
			bibtexlex.Error(bibtexErrorMessage(bibtexstate, bibtextoken))
			Nerrs++
			if bibtexDebug >= 1 {
				__yyfmt__.Printf("%s", bibtexStatname(bibtexstate))
				__yyfmt__.Printf(" saw %s\n", bibtexTokname(bibtextoken))
			}
			fallthrough

		case 1, 2: /* incompletely recovered error ... try again */
			Errflag = 3

			/* find a state where "error" is a legal shift action */
			for bibtexp >= 0 {
				bibtexn = bibtexPact[bibtexS[bibtexp].yys] + bibtexErrCode
				if bibtexn >= 0 && bibtexn < bibtexLast {
					bibtexstate = bibtexAct[bibtexn] /* simulate a shift of "error" */
					if bibtexChk[bibtexstate] == bibtexErrCode {
						goto bibtexstack
					}
				}

				/* the current p has no shift on "error", pop stack */
				if bibtexDebug >= 2 {
					__yyfmt__.Printf("error recovery pops state %d\n", bibtexS[bibtexp].yys)
				}
				bibtexp--
			}
			/* there is no state on the stack with an error shift ... abort */
			goto ret1

		case 3: /* no shift yet; clobber input char */
			if bibtexDebug >= 2 {
				__yyfmt__.Printf("error recovery discards %s\n", bibtexTokname(bibtextoken))
			}
			if bibtextoken == bibtexEofCode {
				goto ret1
			}
			bibtexrcvr.char = -1
			bibtextoken = -1
			goto bibtexnewstate /* try again in the same state */
		}
	}

	/* reduction by production bibtexn */
	if bibtexDebug >= 2 {
		__yyfmt__.Printf("reduce %v in:\n\t%v\n", bibtexn, bibtexStatname(bibtexstate))
	}

	bibtexnt := bibtexn
	bibtexpt := bibtexp
	_ = bibtexpt // guard against "declared and not used"

	bibtexp -= bibtexR2[bibtexn]
	// bibtexp is now the index of $0. Perform the default action. Iff the
	// reduced production is ε, $1 is possibly out of range.
	if bibtexp+1 >= len(bibtexS) {
		nyys := make([]bibtexSymType, len(bibtexS)*2)
		copy(nyys, bibtexS)
		bibtexS = nyys
	}
	bibtexVAL = bibtexS[bibtexp+1]

	/* consult goto table to find next state */
	bibtexn = bibtexR1[bibtexn]
	bibtexg := bibtexPgo[bibtexn]
	bibtexj := bibtexg + bibtexS[bibtexp].yys + 1

	if bibtexj >= bibtexLast {
		bibtexstate = bibtexAct[bibtexg]
	} else {
		bibtexstate = bibtexAct[bibtexj]
		if bibtexChk[bibtexstate] != -bibtexn {
			bibtexstate = bibtexAct[bibtexg]
		}
	}
	// dummy call; replaced with literal code
	switch bibtexnt {

	case 1:
		bibtexDollar = bibtexS[bibtexpt-1 : bibtexpt+1]
		//line bibtex.y:36
		{
		}
	case 2:
		bibtexDollar = bibtexS[bibtexpt-0 : bibtexpt+1]
		//line bibtex.y:39
		{
			bibtexVAL.bibtex = NewBibTex()
			bib = bibtexVAL.bibtex
		}
	case 3:
		bibtexDollar = bibtexS[bibtexpt-2 : bibtexpt+1]
		//line bibtex.y:40
		{
			bibtexVAL.bibtex = bibtexDollar[1].bibtex
			bibtexVAL.bibtex.AddEntry(bibtexDollar[2].bibentry)
		}
	case 4:
		bibtexDollar = bibtexS[bibtexpt-2 : bibtexpt+1]
		//line bibtex.y:41
		{
			bibtexVAL.bibtex = bibtexDollar[1].bibtex
		}
	case 5:
		bibtexDollar = bibtexS[bibtexpt-2 : bibtexpt+1]
		//line bibtex.y:42
		{
			bibtexVAL.bibtex = bibtexDollar[1].bibtex
			bibtexVAL.bibtex.AddStringVar(bibtexDollar[2].bibtag.key, bibtexDollar[2].bibtag.val)
		}
	case 6:
		bibtexDollar = bibtexS[bibtexpt-2 : bibtexpt+1]
		//line bibtex.y:43
		{
			bibtexVAL.bibtex = bibtexDollar[1].bibtex
			bibtexVAL.bibtex.AddPreamble(bibtexDollar[2].strings)
		}
	case 7:
		bibtexDollar = bibtexS[bibtexpt-7 : bibtexpt+1]
		//line bibtex.y:46
		{
			bibtexVAL.bibentry = NewBibEntry(bibtexDollar[2].strval, bibtexDollar[4].strval)
			for _, t := range bibtexDollar[6].bibtags {
				bibtexVAL.bibentry.AddField(t.key, t.val)
			}
		}
	case 8:
		bibtexDollar = bibtexS[bibtexpt-7 : bibtexpt+1]
		//line bibtex.y:47
		{
			bibtexVAL.bibentry = NewBibEntry(bibtexDollar[2].strval, bibtexDollar[4].strval)
			for _, t := range bibtexDollar[6].bibtags {
				bibtexVAL.bibentry.AddField(t.key, t.val)
			}
		}
	case 9:
		bibtexDollar = bibtexS[bibtexpt-5 : bibtexpt+1]
		//line bibtex.y:50
		{
		}
	case 10:
		bibtexDollar = bibtexS[bibtexpt-5 : bibtexpt+1]
		//line bibtex.y:51
		{
		}
	case 11:
		bibtexDollar = bibtexS[bibtexpt-7 : bibtexpt+1]
		//line bibtex.y:54
		{
			bibtexVAL.bibtag = &bibTag{key: bibtexDollar[4].strval, val: bibtexDollar[6].strings}
		}
	case 12:
		bibtexDollar = bibtexS[bibtexpt-7 : bibtexpt+1]
		//line bibtex.y:55
		{
			bibtexVAL.bibtag = &bibTag{key: bibtexDollar[4].strval, val: bibtexDollar[6].strings}
		}
	case 13:
		bibtexDollar = bibtexS[bibtexpt-5 : bibtexpt+1]
		//line bibtex.y:58
		{
			bibtexVAL.strings = bibtexDollar[4].strings
		}
	case 14:
		bibtexDollar = bibtexS[bibtexpt-5 : bibtexpt+1]
		//line bibtex.y:59
		{
			bibtexVAL.strings = bibtexDollar[4].strings
		}
	case 15:
		bibtexDollar = bibtexS[bibtexpt-1 : bibtexpt+1]
		//line bibtex.y:62
		{
			bibtexVAL.strings = NewBibConst(bibtexDollar[1].strval)
		}
	case 16:
		bibtexDollar = bibtexS[bibtexpt-1 : bibtexpt+1]
		//line bibtex.y:63
		{
			bibtexVAL.strings = bib.GetStringVar(bibtexDollar[1].strval)
		}
	case 17:
		bibtexDollar = bibtexS[bibtexpt-3 : bibtexpt+1]
		//line bibtex.y:64
		{
			bibtexVAL.strings = NewBibComposite(bibtexDollar[1].strings)
			bibtexVAL.strings.(*BibComposite).Append(NewBibConst(bibtexDollar[3].strval))
		}
	case 18:
		bibtexDollar = bibtexS[bibtexpt-3 : bibtexpt+1]
		//line bibtex.y:65
		{
			bibtexVAL.strings = NewBibComposite(bibtexDollar[1].strings)
			bibtexVAL.strings.(*BibComposite).Append(bib.GetStringVar(bibtexDollar[3].strval))
		}
	case 19:
		bibtexDollar = bibtexS[bibtexpt-0 : bibtexpt+1]
		//line bibtex.y:68
		{
		}
	case 20:
		bibtexDollar = bibtexS[bibtexpt-3 : bibtexpt+1]
		//line bibtex.y:69
		{
			bibtexVAL.bibtag = &bibTag{key: bibtexDollar[1].strval, val: bibtexDollar[3].strings}
		}
	case 21:
		bibtexDollar = bibtexS[bibtexpt-1 : bibtexpt+1]
		//line bibtex.y:72
		{
			bibtexVAL.bibtags = []*bibTag{bibtexDollar[1].bibtag}
		}
	case 22:
		bibtexDollar = bibtexS[bibtexpt-3 : bibtexpt+1]
		//line bibtex.y:73
		{
			if bibtexDollar[3].bibtag == nil {
				bibtexVAL.bibtags = bibtexDollar[1].bibtags
			} else {
				bibtexVAL.bibtags = append(bibtexDollar[1].bibtags, bibtexDollar[3].bibtag)
			}
		}
	}
	goto bibtexstack /* stack new state and value */
}
