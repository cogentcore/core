// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package token

import (
	"fmt"
	"testing"
)

func CatItem(t *testing.T, itm, cat, subcat Tokens) {
	if itm.Cat() != cat {
		t.Error(fmt.Errorf("%v cat wrong: %v != %v", itm, itm.Cat(), cat))
	}
	if itm.SubCat() != subcat {
		t.Error(fmt.Errorf("%v subcat wrong: %v != %v", itm, itm.SubCat(), subcat))
	}
}

func TestCats(t *testing.T) {
	CatItem(t, Error, None, None)
	CatItem(t, EOS, None, None)
	CatItem(t, Keyword, Keyword, Keyword)
	CatItem(t, KeywordPseudo, Keyword, Keyword)
	CatItem(t, KeywordType, Keyword, Keyword)
	CatItem(t, Name, Name, Name)
	CatItem(t, NameTag, Name, Name)
	CatItem(t, NameVar, Name, NameVar)
	CatItem(t, NameVarMagic, Name, NameVar)
	CatItem(t, Literal, Literal, Literal)
	CatItem(t, LitNum, Literal, LitNum)
	CatItem(t, LitNumComplex, Literal, LitNum)
	CatItem(t, Operator, Operator, Operator)
	CatItem(t, OpMathRem, Operator, OpMath)
	CatItem(t, OpBitOr, Operator, OpBit)
	CatItem(t, OpAsgnArrow, Operator, OpAsgn)
	CatItem(t, OpMathAsgn, Operator, OpMathAsgn)
	CatItem(t, OpBitAsgnAnd, Operator, OpBitAsgn)
	CatItem(t, OpLogNot, Operator, OpLog)
	CatItem(t, OpRelGtEq, Operator, OpRel)
	CatItem(t, OpList, Operator, OpList)
	CatItem(t, Punctuation, Punctuation, Punctuation)
	CatItem(t, PunctGpRBrace, Punctuation, PunctGp)
	CatItem(t, PunctStr, Punctuation, PunctStr)
	CatItem(t, CommentSingle, Comment, Comment)
	CatItem(t, TextSpellErr, Text, Text)
	CatItem(t, TextStylePrompt, Text, TextStyle)
}
