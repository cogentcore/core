// HLSL compute example

[[vk::binding(0, 0)]] RWStructuredBuffer<float> In;
[[vk::binding(1, 0)]] RWStructuredBuffer<float> Out;

// FastExp is a quartic spline approximation to the Exp function, by N.N. Schraudolph
// It does not have any of the sanity checking of a standard method -- returns
// nonsense when arg is out of range.  Runs in 2.23ns vs. 6.3ns for 64bit which is faster
// than math32.Exp actually.
float FastExp(float x) {
	if (x <= -88.02969) { // this doesn't add anything and -exp is main use-case anyway
		return 0;
	}
	int i = int(12102203*x) + 127*(1<<23);
	int m = i >> 7 & 0xFFFF; // copy mantissa
	i += (((((((((((3537 * m) >> 16) + 13668) * m) >> 18) + 15817) * m) >> 14) - 80470) * m) >> 11);
	return asfloat(uint(i));
}


[numthreads(64, 1, 1)]
void main(uint3 idx : SV_DispatchThreadID) {
    // Out[idx.x] = FastExp(In[idx.x]); // 0 diffs
	float vbio = In[idx.x];
	float eval = 0.1 * ((vbio + 90.0) + 10.0);
	// Out[idx.x] = (vbio + 90.0) / (1.0 + FastExp(eval)); // lots of diffs
	// Out[idx.x] = (vbio + 90.0) / (1.0 + exp(eval)); // worse
	// Out[idx.x] = eval; // 0 diff
	Out[idx.x] = float(1.0) / eval; // a few 2.98e-8 diffs already!  no diff from casting
	// Out[idx.x] = 1.0 / FastExp(eval); // lots more diffs, e-08, -09
}

