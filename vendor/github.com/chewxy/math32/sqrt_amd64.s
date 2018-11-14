#include "textflag.h"

// func Sqrt(x float32) float32
TEXT ·Sqrt(SB),NOSPLIT,$0
	MOVSS x+0(FP), X0
	SQRTSS X0, X0
	MOVSS X0, ret+8(FP)
	RET
