#ifndef __RAND_HLSL__
#define __RAND_HLSL__


#include "slrand.hlsl"

struct Rnds {
	uint2  Uints;
	int         pad, pad1;
	float2 Floats;
	int         pad2, pad3;
	float2 Floats11;
	int         pad4, pad5;
	float2 Gauss;
	int         pad6, pad7;
// Note that the counter to the outer-most computation function
// is passed by *value*, so the same counter goes to each element
// as it is computed, but within this scope, counter is passed by
// reference (as a pointer) so subsequent calls get a new counter value.
// The counter should be incremented by the number of random calls
// outside of the overall update function.
	void RndGen(uint2 counter, uint idx) {
		this.Uints = RandUint2(counter, idx);
		this.Floats = RandFloat2(counter, idx);
		this.Floats11 = RandFloat112(counter, idx);
		this.Gauss = RandNormFloat2(counter, idx);
	}

};


// from file: rand.hlsl

// binding is var, set
[[vk::binding(0, 0)]] RWStructuredBuffer<uint2> Counter;
[[vk::binding(0, 1)]] RWStructuredBuffer<Rnds> Data;

[numthreads(64, 1, 1)]

void main(uint3 idx : SV_DispatchThreadID) {
	Data[idx.x].RndGen(Counter[0], idx.x);
}


#endif // __RAND_HLSL__
