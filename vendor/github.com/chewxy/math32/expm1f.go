// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package math32

// The original C code, the long comment, and the constants
// below are from FreeBSD's /usr/src/lib/msun/src/s_expm1.c
// and came with this notice. The go code is a simplified
// version of the original C.
//
// ====================================================
// Copyright (C) 1993 by Sun Microsystems, Inc. All rights reserved.
//
// Developed at SunPro, a Sun Microsystems, Inc. business.
// Permission to use, copy, modify, and distribute this
// software is freely granted, provided that this notice
// is preserved.
// ====================================================
//
// expm1(x)
// Returns exp(x)-1, the exponential of x minus 1.
//
// Method
//   1. Argument reduction:
//      Given x, find r and integer k such that
//
//               x = k*ln2 + r,  |r| <= 0.5*ln2 ~ 0.34658
//
//      Here a correction term c will be computed to compensate
//      the error in r when rounded to a floating-point number.
//
//   2. Approximating expm1(r) by a special rational function on
//      the interval [0,0.34658]:
//      Since
//          r*(exp(r)+1)/(exp(r)-1) = 2+ r**2/6 - r**4/360 + ...
//      we define R1(r*r) by
//          r*(exp(r)+1)/(exp(r)-1) = 2+ r**2/6 * R1(r*r)
//      That is,
//          R1(r**2) = 6/r *((exp(r)+1)/(exp(r)-1) - 2/r)
//                   = 6/r * ( 1 + 2.0*(1/(exp(r)-1) - 1/r))
//                   = 1 - r**2/60 + r**4/2520 - r**6/100800 + ...
//      We use a special Reme algorithm on [0,0.347] to generate
//      a polynomial of degree 5 in r*r to approximate R1. The
//      maximum error of this polynomial approximation is bounded
//      by 2**-61. In other words,
//          R1(z) ~ 1.0 + Q1*z + Q2*z**2 + Q3*z**3 + Q4*z**4 + Q5*z**5
//      where   Q1  =  -1.6666666666666567384E-2,
//              Q2  =   3.9682539681370365873E-4,
//              Q3  =  -9.9206344733435987357E-6,
//              Q4  =   2.5051361420808517002E-7,
//              Q5  =  -6.2843505682382617102E-9;
//      (where z=r*r, and the values of Q1 to Q5 are listed below)
//      with error bounded by
//          |                  5           |     -61
//          | 1.0+Q1*z+...+Q5*z   -  R1(z) | <= 2
//          |                              |
//
//      expm1(r) = exp(r)-1 is then computed by the following
//      specific way which minimize the accumulation rounding error:
//                             2     3
//                            r     r    [ 3 - (R1 + R1*r/2)  ]
//            expm1(r) = r + --- + --- * [--------------------]
//                            2     2    [ 6 - r*(3 - R1*r/2) ]
//
//      To compensate the error in the argument reduction, we use
//              expm1(r+c) = expm1(r) + c + expm1(r)*c
//                         ~ expm1(r) + c + r*c
//      Thus c+r*c will be added in as the correction terms for
//      expm1(r+c). Now rearrange the term to avoid optimization
//      screw up:
//                      (      2                                    2 )
//                      ({  ( r    [ R1 -  (3 - R1*r/2) ]  )  }    r  )
//       expm1(r+c)~r - ({r*(--- * [--------------------]-c)-c} - --- )
//                      ({  ( 2    [ 6 - r*(3 - R1*r/2) ]  )  }    2  )
//                      (                                             )
//
//                 = r - E
//   3. Scale back to obtain expm1(x):
//      From step 1, we have
//         expm1(x) = either 2**k*[expm1(r)+1] - 1
//                  = or     2**k*[expm1(r) + (1-2**-k)]
//   4. Implementation notes:
//      (A). To save one multiplication, we scale the coefficient Qi
//           to Qi*2**i, and replace z by (x**2)/2.
//      (B). To achieve maximum accuracy, we compute expm1(x) by
//        (i)   if x < -56*ln2, return -1.0, (raise inexact if x!=inf)
//        (ii)  if k=0, return r-E
//        (iii) if k=-1, return 0.5*(r-E)-0.5
//        (iv)  if k=1 if r < -0.25, return 2*((r+0.5)- E)
//                     else          return  1.0+2.0*(r-E);
//        (v)   if (k<-2||k>56) return 2**k(1-(E-r)) - 1 (or exp(x)-1)
//        (vi)  if k <= 20, return 2**k((1-2**-k)-(E-r)), else
//        (vii) return 2**k(1-((E+2**-k)-r))
//
// Special cases:
//      expm1(INF) is INF, expm1(NaN) is NaN;
//      expm1(-INF) is -1, and
//      for finite argument, only expm1(0)=0 is exact.
//
// Accuracy:
//      according to an error analysis, the error is always less than
//      1 ulp (unit in the last place).
//
// Misc. info.
//      For IEEE double
//          if x >  7.09782712893383973096e+02 then expm1(x) overflow
//
// Constants:
// The hexadecimal values are the intended ones for the following
// constants. The decimal values may be used, provided that the
// compiler will convert from decimal to binary accurately enough
// to produce the hexadecimal values shown.
//

// Expm1 returns e**x - 1, the base-e exponential of x minus 1.
// It is more accurate than Exp(x) - 1 when x is near zero.
//
// Special cases are:
//	Expm1(+Inf) = +Inf
//	Expm1(-Inf) = -1
//	Expm1(NaN) = NaN
// Very large values overflow to -1 or +Inf.
func Expm1(x float32) float32 {
	return expm1(x)
}

func expm1(x float32) float32 {
	const (
		Othreshold = 89.415985                   // 0x42b2d4fc
		Ln2X27     = 1.871497344970703125e+01    // 0x4195b844
		Ln2HalfX3  = 1.0397207736968994140625    // 0x3F851592
		Ln2Half    = 3.465735912322998046875e-01 // 0x3eb17218
		Ln2Hi      = 6.9313812256e-01            // 0x3f317180
		Ln2Lo      = 9.0580006145e-06            // 0x3717f7d1
		InvLn2     = 1.4426950216e+00            // 0x3fb8aa3b
		Tiny       = 1.0 / (1 << 54)             // 2**-54 = 0x3c90000000000000

		/* scaled coefficients related to expm1 */
		Q1 = -3.3333335072e-02 /* 0xbd088889 */
		Q2 = 1.5873016091e-03  /* 0x3ad00d01 */
		Q3 = -7.9365076090e-05 /* 0xb8a670cd */
		Q4 = 4.0082177293e-06  /* 0x36867e54 */
		Q5 = -2.0109921195e-07 /* 0xb457edbb */
	)

	// special cases
	switch {
	case IsInf(x, 1) || IsNaN(x):
		return x
	case IsInf(x, -1):
		return -1
	}

	absx := x
	sign := false
	if x < 0 {
		absx = -absx
		sign = true
	}

	// filter out huge argument
	if absx >= Ln2X27 { // if |x| >= 27 * ln2
		if sign {
			return -1 // x < -56*ln2, return -1
		}
		if absx >= Othreshold { // if |x| >= 89.415985...
			return Inf(1)
		}
	}

	// argument reduction
	var c float32
	var k int
	if absx > Ln2Half { // if  |x| > 0.5 * ln2
		var hi, lo float32
		if absx < Ln2HalfX3 { // and |x| < 1.5 * ln2
			if !sign {
				hi = x - Ln2Hi
				lo = Ln2Lo
				k = 1
			} else {
				hi = x + Ln2Hi
				lo = -Ln2Lo
				k = -1
			}
		} else {
			if !sign {
				k = int(InvLn2*x + 0.5)
			} else {
				k = int(InvLn2*x - 0.5)
			}
			t := float32(k)
			hi = x - t*Ln2Hi // t * Ln2Hi is exact here
			lo = t * Ln2Lo
		}
		x = hi - lo
		c = (hi - x) - lo
	} else if absx < Tiny { // when |x| < 2**-54, return x
		return x
	} else {
		k = 0
	}

	// x is now in primary range
	hfx := 0.5 * x
	hxs := x * hfx
	r1 := 1 + hxs*(Q1+hxs*(Q2+hxs*(Q3+hxs*(Q4+hxs*Q5))))
	t := 3 - r1*hfx
	e := hxs * ((r1 - t) / (6.0 - x*t))
	if k != 0 {
		e = (x*(e-c) - c)
		e -= hxs
		switch {
		case k == -1:
			return 0.5*(x-e) - 0.5
		case k == 1:
			if x < -0.25 {
				return -2 * (e - (x + 0.5))
			}
			return 1 + 2*(x-e)
		case k <= -2 || k > 56: // suffice to return exp(x)-1
			y := 1 - (e - x)
			y = Float32frombits(Float32bits(y) + uint32(k)<<23) // add k to y's exponent
			return y - 1
		}
		if k < 20 {
			t := Float32frombits(0x3f800000 - (0x1000000 >> uint(k))) // t=1-2**-k
			y := t - (e - x)
			y = Float32frombits(Float32bits(y) + uint32(k)<<23) // add k to y's exponent
			return y
		}
		t := Float32frombits(uint32(0x7f-k) << 23) // 2**-k
		y := x - (e + t)
		y += 1
		y = Float32frombits(Float32bits(y) + uint32(k)<<23) // add k to y's exponent
		return y
	}
	return x - (x*e - hxs) // c is 0
}
