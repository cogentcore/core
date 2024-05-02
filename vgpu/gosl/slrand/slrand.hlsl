// Copyright (c) 2022, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Original file is in Go package: github.com/emer/gosl/v2/slrand
// See README.md there for documentation.

// These random number generation (RNG) functions are optimized for
// use on the GPU, with equivalent Go versions available in slrand.go.
// This is using the Philox2x32 counter-based RNG.

// vulkan glslang does not support 64 bit integers:
// https://github.com/KhronosGroup/glslang/issues/2965
// so we have to use the slower version of the MulHiLo algorithm:

// MulHiLo32 does 32 bit hi-lo multiply using only 32 bit uints
void MulHiLo32(uint a, uint b, out uint lo, out uint hi) {
    const uint LOMASK = ((((uint)1)<<16)-1);
    lo = a * b;               /* full low multiply */
    uint ahi = a >> 16;
    uint alo = a & LOMASK;
    uint bhi = b >> 16;
    uint blo = b & LOMASK;

    uint ahbl = ahi * blo;
    uint albh = alo * bhi;

    uint ahbl_albh = ((ahbl&LOMASK) + (albh&LOMASK));
    uint hit = ahi*bhi + (ahbl>>16) +  (albh>>16);
    hit += ahbl_albh >> 16; /* carry from the sum of lo(ahbl) + lo(albh) ) */
    /* carry from the sum with alo*blo */
    hit += ((lo >> 16) < (ahbl_albh&LOMASK));
    hi = hit; 
}

/*
// MulHiLo64 is the fast, simpler version when 64 bit uints become available
void MulHiLo64(uint a, uint b, out uint lo, out uint hi) {
	uint64_t prod = uint64_t(a) * uint64_t(b);
	hi = uint(prod >> 32);
	lo = uint(prod);
}
*/

// Philox2x32round does one round of updating of the counter
void Philox2x32round(inout uint2 counter, uint key) {
	uint hi;
	uint lo;
	MulHiLo32(0xD256D193, counter.x, lo, hi);
	counter.x = hi ^ key ^ counter.y;
	counter.y = lo;
}

// Philox2x32bumpkey does one round of updating of the key
void Philox2x32bumpkey(inout uint key) {
	key += uint(0x9E3779B9);
}

// Philox2x32 implements the stateless counter-based RNG algorithm
// returning a random number as 2 uint32 32 bit values, given a
// counter and key input that determine the result.
uint2 Philox2x32(uint2 counter, uint key) {
	Philox2x32round(counter, key); // 1
	Philox2x32bumpkey(key);
	Philox2x32round(counter, key); // 2
	Philox2x32bumpkey(key);
	Philox2x32round(counter, key); // 3
	Philox2x32bumpkey(key);
	Philox2x32round(counter, key); // 4
	Philox2x32bumpkey(key);
	Philox2x32round(counter, key); // 5
	Philox2x32bumpkey(key);
	Philox2x32round(counter, key); // 6
	Philox2x32bumpkey(key);
	Philox2x32round(counter, key); // 7
	Philox2x32bumpkey(key);
	Philox2x32round(counter, key); // 8
	Philox2x32bumpkey(key);
	Philox2x32round(counter, key); // 9
	Philox2x32bumpkey(key);
	
	Philox2x32round(counter, key); // 10
	return counter;
}

// UintToFloat converts a uint 32 bit integer into a 32 bit float
// in the (0,1) interval (i.e., exclusive of 1).
// This differs from the Go standard by excluding 0, which is handy for passing
// directly to Log function, and from the reference Philox code by excluding 1
// which is in the Go standard and most other standard RNGs.
float UintToFloat(uint val) {
	const float factor = float(1.) / (float(uint(0xffffffff)) + float(1.));
	const float halffactor = float(0.5) * factor;
	float f = float(val) * factor + halffactor;
	if(f == 1.0) { // exclude 1
		return asfloat(0x3F7FFFFF);
	}
	return f;
}

// UintToFloat11 converts a uint 32 bit integer into a 32 bit float
// in the [1..1] interval (inclusive of -1 and 1, never identically == 0)
float UintToFloat11(uint val) {
	const float factor = float(1.) / (float(int(0x7fffffff)) + float(1.));
	const float halffactor = float(0.5) * factor;
	return (float(int(val)) * factor + halffactor);
}

// Uint2ToFloat converts two uint 32 bit integers (uint2)
// into two corresponding 32 bit float values (float2)
// in the (0,1) interval (i.e., exclusive of 1).
float2 Uint2ToFloat(uint2 val) {
	float2 r;
	r.x = UintToFloat(val.x);
	r.y = UintToFloat(val.y);
	return r;
}

// Uint2ToFloat11 converts two uint 32 bit integers into 32 bit floats
// in the [1,1] interval (inclusive of -1 and 1, never identically == 0)
float2 Uint2ToFloat11(uint2 val) {
	float2 r;
	r.x = UintToFloat11(val.x);
	r.y = UintToFloat11(val.y);
	return r;
}

// CounterIncr increments the given counter as if it was 
// a uint64 integer.
void CounterIncr(inout uint2 counter) {
	if(counter.x == 0xffffffff) {
		counter.y++;
		counter.x = 0;
	} else {
		counter.x++;
	}
}

// CounterAdd adds the given increment to the counter
void CounterAdd(inout uint2 counter, uint inc) {
	if(inc == 0) {
		return;
	}
	if(counter.x > uint(0xffffffff) - inc) {
		counter.y++;
		counter.x = (inc - 1) - (uint(0xffffffff) - counter.x);
	} else {
		counter.x+=inc;
	}
}

////////////////////////////////////////////////////////////
//   Methods below provide a standard interface
//   with more readable names, mapping onto the Go rand methods.
//   These are what should be called by end-user code.

// RandUint2 returns two uniformly distributed 32 unsigned integers,
// based on given counter and key.
// The counter is incremented by 1 (in a 64-bit equivalent manner)
// as a result of this call, ensuring that the next call will produce
// the next random number in the sequence.  The key should be the 
// unique index of the element being updated.
uint2 RandUint2(inout uint2 counter, uint key) {
	uint2 res = Philox2x32(counter, key);
	CounterIncr(counter);
	return res;
}

// RandUint returns a uniformly distributed 32 unsigned integer,
// based on given counter and key.
// The counter is incremented by 1 (in a 64-bit equivalent manner)
// as a result of this call, ensuring that the next call will produce
// the next random number in the sequence.  The key should be the 
// unique index of the element being updated.
uint RandUint(inout uint2 counter, uint key) {
	uint2 res = Philox2x32(counter, key);
	CounterIncr(counter);
	return res.x;
}

// RandFloat2 returns two uniformly distributed 32 floats
// in range (0,1) based on given counter and key.
// The counter is incremented by 1 (in a 64-bit equivalent manner)
// as a result of this call, ensuring that the next call will produce
// the next random number in the sequence.  The key should be the 
// unique index of the element being updated.
float2 RandFloat2(inout uint2 counter, uint key) {
	return Uint2ToFloat(RandUint2(counter, key));
}

// RandFloat returns a uniformly distributed 32 float
// in range (0,1) based on given counter and key.
// The counter is incremented by 1 (in a 64-bit equivalent manner)
// as a result of this call, ensuring that the next call will produce
// the next random number in the sequence.  The key should be the 
// unique index of the element being updated.
float RandFloat(inout uint2 counter, uint key) {
	return UintToFloat(RandUint(counter, key));
}

// RandFloat112 returns two uniformly distributed 32 floats
// in range [-1,1] based on given counter and key.
// The counter is incremented by 1 (in a 64-bit equivalent manner)
// as a result of this call, ensuring that the next call will produce
// the next random number in the sequence.  The key should be the 
// unique index of the element being updated.
float2 RandFloat112(inout uint2 counter, uint key) {
	return Uint2ToFloat11(RandUint2(counter, key));
}

// RandFloat11 returns a uniformly distributed 32 float
// in range [-1..1] based on given counter and key.
// The counter is incremented by 1 (in a 64-bit equivalent manner)
// as a result of this call, ensuring that the next call will produce
// the next random number in the sequence.  The key should be the 
// unique index of the element being updated.
float RandFloat11(inout uint2 counter, uint key) {
	return UintToFloat11(RandUint(counter, key));
}

// RandBoolP returns a bool true value with probability p
bool RandBoolP(inout uint2 counter, uint key, float p) {
	return (RandFloat(counter, key) < p);
}

void sincospi(float x, out float s, out float c) {
	const float PIf = 3.1415926535897932;
	sincos(PIf*x, s, c);
}

// RandNormFloat2 returns two random 32 bit floating numbers 
// distributed according to the normal, Gaussian distribution
// with zero mean and unit variance.
// This is done very efficiently using the Box-Muller algorithm
// that consumes two random 32 bit uint values.
float2 RandNormFloat2(inout uint2 counter, uint key) {
	uint2 ur = RandUint2(counter, key);
	float r;
	float2 f;
	sincospi(UintToFloat11(ur.x), f.x, f.y);
	r = sqrt(-2. * log(UintToFloat(ur.y))); // guaranteed to avoid 0.
	f.x *= r;
	f.y *= r;
	return f;
}

// RandNormFloat returns a random 32 bit floating number 
// distributed according to the normal, Gaussian distribution
// with zero mean and unit variance.
float RandNormFloat(inout uint2 counter, uint key) {
	float2 f = RandNormFloat2(counter, key);
	return f.x;
}

// RandUintn returns a uint 32 bit in the range [0,n)
uint RandUintn(inout uint2 counter, uint key, uint n) {
	float v = RandFloat(counter, key);
	return uint(v * float(n));
}

// Counter is used for storing the random counter using aligned 16 byte storage,
// with convenience methods for typical use cases.
// It retains a copy of the last Seed value, which is applied to the Hi uint32 value.
struct RandCounter {
	uint Lo;
	uint Hi;
	uint HiSeed;
	
	uint pad1;
	
	// Reset resets counter to last set Seed state
	void Reset() {
		this.Lo = 0;
		this.Hi = this.HiSeed;
	}
	
	// Uint2 returns counter as a Uint2
	uint2 Uint2() {
		uint2 r;
		r.x = this.Lo;
		r.y = this.Hi;
		return r;
	}
	
	// Set sets the counter from a Uint2
	void Set(uint2 c) {
		this.Lo = c.x;
		this.Hi = c.y;
	}

	// Seed sets the Hi uint32 value from given seed, saving it in Seed field.
	// Each increment in seed generates a unique sequence of over 4 billion numbers,
	// so it is reasonable to just use incremental values there, but more widely
	// spaced numbers will result in longer unique sequences.
	// Resets Lo to 0.
	// This same seed will be restored during Reset
	void Seed(uint seed) {
		this.Lo = 0;
		this.Hi = seed;
		this.HiSeed = seed;
	}
	
	// Add increments the counter by given amount.
	// Call this after thread completion with number of random numbers
	// generated per thread.
	uint2 Add(int inc) {
		uint2 c = this.Uint2();
		CounterAdd(c, inc);
		this.Set(c);
		return c;
	}
};

