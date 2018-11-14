// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSS-style
// license that can be found in the LICENSE file.

#include "textflag.h"

#define HSqrt2 7.07106781186547524401e-01 // sqrt(2)/2
#define Ln2Hi  6.9313812256e-01   // 0x3f317180
#define Ln2Lo  9.0580006145e-06   // 0x3717f7d1
#define L1     6.6666668653e-01   // 0x3f2aaaab
#define L2     4.0000000596e-01   // 0x3ecccccd
#define L3     2.8571429849e-01   // 0x3e924925
#define L4     2.2222198546e-01   // 0x3e638e29
#define L5     1.8183572590e-01   // 0x3e3a3325
#define L6     1.5313838422e-01   // 0x3e1cd04f
#define L7     1.4798198640e-01   // 0x3e178897
#define NaN    0x7FE00000
#define PosInf 0x7F800000
#define NegInf 0xFF800000

// func Log(x float64) float64
TEXT ·Log(SB),NOSPLIT,$0
	// test bits for special cases
	MOVL    x+0(FP), BX
	MOVQ    $~(1<<31), AX // sign bit mask
	ANDQ    BX, AX
	JEQ     isZero
	MOVL    $0, AX
	CMPL    AX, BX
	JGT     isNegative
	MOVL    $PosInf, AX
	CMPQ    AX, BX
	JLE     isInfOrNaN
	// f1, ki := math.Frexp(x); k := float64(ki)
	MOVL    BX, X0
	MOVL    $0x007FFFFF, AX
	MOVL    AX, X2
	ANDPS   X0, X2
	MOVSS   $0.5, X0 // 0x3FE0000000000000
	ORPS    X0, X2 // X2= f1
	SHRQ    $23, BX
	ANDL    $0xFF, BX
	SUBL    $0x7E, BX
	CVTSL2SS BX, X1 // x1= k, x2= f1
	// if f1 < math.Sqrt2/2 { k -= 1; f1 *= 2 }
	MOVSS   $HSqrt2, X0 // x0= 0.7071, x1= k, x2= f1
	CMPSS   X2, X0, 5 // cmpnlt; x0= 0 or ^0, x1= k, x2 = f1
	MOVSS   $1.0, X3 // x0= 0 or ^0, x1= k, x2 = f1, x3= 1
	ANDPS   X0, X3 // x0= 0 or ^0, x1= k, x2 = f1, x3= 0 or 1
	SUBSS   X3, X1 // x0= 0 or ^0, x1= k, x2 = f1, x3= 0 or 1
	MOVSS   $1.0, X0 // x0= 1, x1= k, x2= f1, x3= 0 or 1
	ADDSS   X0, X3 // x0= 1, x1= k, x2= f1, x3= 1 or 2
	MULSS   X3, X2 // x0= 1, x1= k, x2= f1
	// f := f1 - 1
	SUBSS   X0, X2 // x1= k, x2= f
	// s := f / (2 + f)
	MOVSS   $2.0, X0
	ADDSS   X2, X0
	MOVUPS  X2, X3
	DIVSS   X0, X3 // x1=k, x2= f, x3= s
	// s2 := s * s
	MOVUPS  X3, X4 // x1= k, x2= f, x3= s
	MULSS   X4, X4 // x1= k, x2= f, x3= s, x4= s2
	// s4 := s2 * s2
	MOVUPS  X4, X5 // x1= k, x2= f, x3= s, x4= s2
	MULSS   X5, X5 // x1= k, x2= f, x3= s, x4= s2, x5= s4
	// t1 := s2 * (L1 + s4*(L3+s4*(L5+s4*L7)))
	MOVSS   $L7, X6
	MULSS   X5, X6
	ADDSS   $L5, X6
	MULSS   X5, X6
	ADDSS   $L3, X6
	MULSS   X5, X6
	ADDSS   $L1, X6
	MULSS   X6, X4 // x1= k, x2= f, x3= s, x4= t1, x5= s4
	// t2 := s4 * (L2 + s4*(L4+s4*L6))
	MOVSS   $L6, X6
	MULSS   X5, X6
	ADDSS   $L4, X6
	MULSS   X5, X6
	ADDSS   $L2, X6
	MULSS   X6, X5 // x1= k, x2= f, x3= s, x4= t1, x5= t2
	// R := t1 + t2
	ADDSS   X5, X4 // x1= k, x2= f, x3= s, x4= R
	// hfsq := 0.5 * f * f
	MOVSS   $0.5, X0
	MULSS   X2, X0
	MULSS   X2, X0 // x0= hfsq, x1= k, x2= f, x3= s, x4= R
	// return k*Ln2Hi - ((hfsq - (s*(hfsq+R) + k*Ln2Lo)) - f)
	ADDSS   X0, X4 // x0= hfsq, x1= k, x2= f, x3= s, x4= hfsq+R
	MULSS   X4, X3 // x0= hfsq, x1= k, x2= f, x3= s*(hfsq+R)
	MOVSS   $Ln2Lo, X4
	MULSS   X1, X4 // x4= k*Ln2Lo
	ADDSS   X4, X3 // x0= hfsq, x1= k, x2= f, x3= s*(hfsq+R)+k*Ln2Lo
	SUBSS   X3, X0 // x0= hfsq-(s*(hfsq+R)+k*Ln2Lo), x1= k, x2= f
	SUBSS   X2, X0 // x0= (hfsq-(s*(hfsq+R)+k*Ln2Lo))-f, x1= k
	MULSS   $Ln2Hi, X1 // x0= (hfsq-(s*(hfsq+R)+k*Ln2Lo))-f, x1= k*Ln2Hi
	SUBSS   X0, X1 // x1= k*Ln2Hi-((hfsq-(s*(hfsq+R)+k*Ln2Lo))-f)
  	MOVSS   X1, ret+8(FP)
	RET
isInfOrNaN:
	MOVL    BX, ret+8(FP) // +Inf or NaN, return x
	RET
isNegative:
	MOVL    $NaN, AX
	MOVL    AX, ret+8(FP) // return NaN
	RET
isZero:
	MOVL    $NegInf, AX
	MOVL    AX, ret+8(FP) // return -Inf
	RET
