#ifndef __FASTEXP_HLSL__
#define __FASTEXP_HLSL__


// FastExp is a quartic spline approximation to the Exp function, by N.N. Schraudolph
// It does not have any of the sanity checking of a standard method -- returns
// nonsense when arg is out of range.  Runs in 2.23ns vs. 6.3ns for 64bit which is faster
// than exp actually.
float FastExp(float x) {
	if (x <= -88.02969) { // this doesn't add anything and -exp is main use-case anyway
		return 0;
	}
	int i = int(12102203*x) + 127*(1<<23);
	int m = i >> 7 & 0xFFFF; // copy mantissa
	i += (((((((((((3537 * m) >> 16) + 13668) * m) >> 18) + 15817) * m) >> 14) - 80470) * m) >> 11);
	return asfloat(uint(i));
}
#endif // __FASTEXP_HLSL__
