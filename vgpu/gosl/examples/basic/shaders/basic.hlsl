#ifndef __BASIC_HLSL__
#define __BASIC_HLSL__


#include "fastexp.hlsl"

// DataStruct has the test data
struct DataStruct {

	// raw value
	float Raw;

	// integrated value
	float Integ;

	// exp of integ
	float Exp;

	// must pad to multiple of 4 floats for arrays
	float Pad2;
};

// ParamStruct has the test params
struct ParamStruct {

	// rate constant in msec
	float Tau;

	// 1/Tau
	float Dt;

	float pad, pad1;
	void IntegFromRaw(inout DataStruct ds) {
		ds.Integ += this.Dt * (ds.Raw - ds.Integ);
		ds.Exp = FastExp(-ds.Integ);
	}

};

// note: double-commented lines required here -- binding is var, set
[[vk::binding(0, 0)]] RWStructuredBuffer<ParamStruct> Params;
[[vk::binding(0, 1)]] RWStructuredBuffer<DataStruct> Data;

[numthreads(64, 1, 1)]

void main(uint3 idx : SV_DispatchThreadID) {
    Params[0].IntegFromRaw(Data[idx.x]);
}
#endif // __BASIC_HLSL__
