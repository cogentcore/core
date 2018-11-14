// Copyright 2014 Xuanyi Chew. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//
// The original code is lifted from the Go standard library which is governed by
// a BSD-style licence which can be found here: https://golang.org/LICENSE

#include "textflag.h"

// The method is based on a paper by Naoki Shibata: "Efficient evaluation
// methods of elementary functions suitable for SIMD computation", Proc.
// of International Supercomputing Conference 2010 (ISC'10), pp. 25 -- 32
// (May 2010). The paper is available at
// http://www.springerlink.com/content/340228x165742104/
//
// The original code and the constants below are from the author's
// implementation available at http://freshmeat.net/projects/sleef.
// The README file says, "The software is in public domain.
// You can use the software without any obligation."
//
// This code is a simplified version of the original.
// The magic numbers for the float32 are lifted from the same project

	
#define LN2 0.693147182464599609375 // log_e(2)
#define LOG2E 1.44269502162933349609375 // 1/LN2
#define LN2U 0.693145751953125 // upper half LN2
#define LN2L 1.428606765330187045e-06 // lower half LN2
#define T0 1.0
#define T1 0.5
#define T2 0.166665524244308471679688
#define T3 0.0416710823774337768554688
#define T4 0.00836596917361021041870117
#define PosInf 0x7F800000
#define NegInf 0xFF800000

// func Exp(x float32) float32
TEXT ·Exp(SB),NOSPLIT,$0
// test bits for not-finite
	MOVL    x+0(FP), BX
	MOVQ    $~(1<<31), AX // sign bit mask
	MOVL    BX, DX
	ANDL    AX, DX
	MOVL    $PosInf, AX
	CMPL    AX, DX
	JLE     notFinite
	MOVL    BX, X0
	MOVSS   $LOG2E, X1
	MULSS   X0, X1
	CVTSS2SL X1, BX // BX = exponent
	CVTSL2SS BX, X1
	MOVSS   $LN2U, X2
	MULSS   X1, X2
	SUBSS   X2, X0
	MOVSS   $LN2L, X2
	MULSS   X1, X2
	SUBSS   X2, X0
	// reduce argument
	MULSS   $0.0625, X0
	// Taylor series evaluation
	ADDSS   $T4, X1
	MULSS   X0, X1
	ADDSS   $T3, X1
	MULSS   X0, X1
	ADDSS   $T2, X1
	MULSS   X0, X1
	ADDSS   $T1, X1
	MULSS   X0, X1
	ADDSS   $T0, X1
	MULSS   X1, X0
	MOVSS   $2.0, X1
	ADDSS   X0, X1
	MULSS   X1, X0
	MOVSS   $2.0, X1
	ADDSS   X0, X1
	MULSS   X1, X0
	MOVSS   $2.0, X1
	ADDSS   X0, X1
	MULSS   X1, X0
	MOVSS   $2.0, X1
	ADDSS   X0, X1
	MULSS   X1, X0
	ADDSS   $1.0, X0
	// return fr * 2**exponent
	MOVL    $0x7F, AX // bias
	ADDL    AX, BX
	JLE     underflow
	CMPL    BX, $0xFF
	JGE     overflow
	MOVL    $23, CX
	SHLQ    CX, BX
	MOVL    BX, X1
	MULSS   X1, X0
	MOVSS   X0, ret+8(FP)
	RET
notFinite:
	// test bits for -Inf
	MOVL    $NegInf, AX
	CMPQ    AX, BX
	JNE     notNegInf
	// -Inf, return 0
underflow: // return 0
	MOVL    $0, AX
	MOVL    AX, ret+8(FP)
	RET
overflow: // return +Inf
	MOVL    $PosInf, BX
notNegInf: // NaN or +Inf, return x
	MOVL    BX, ret+8(FP)
	RET
