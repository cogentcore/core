
#include "fastexp.wgsl"

// DataStruct has the test data
struct DataStruct {

	// raw value
	Raw: f32,

	// integrated value
	Integ: f32,

	// exp of integ
	Exp: f32,

	// must pad to multiple of 4 floats for arrays
	pad: f32,
}

// ParamStruct has the test params
struct ParamStruct {

	// rate constant in msec
	Tau: f32,

	// 1/Tau
	Dt: f32,

	pad:  f32,
	pad1: f32,
}

// IntegFromRaw computes integrated value from current raw value
fn ParamStruct_IntegFromRaw(ps: ptr<function,ParamStruct>, ds: ptr<function,DataStruct>) {
	(*ds).Integ += (*ps).Dt * ((*ds).Raw - (*ds).Integ);
	(*ds).Exp = FastExp(-(*ds).Integ);
}

@group(0) @binding(0)
var<storage, read_write> Params: array<ParamStruct>;

@group(0) @binding(1)
var<storage, read_write> Data: array<DataStruct>;

@compute
@workgroup_size(64)
fn main(@builtin(global_invocation_id) idx: vec3<u32>) {
	var pars = Params[0];
	var data = Data[idx.x];
	ParamStruct_IntegFromRaw(&pars, &data);
	Data[idx.x] = data;
}
