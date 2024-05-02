#version 450

// DataStruct has the test data
struct DataStruct  {
    float Raw;
    float Integ;
    float Pad1;
    float Pad2;
};

// ParamStruct has the test params
struct ParamStruct  {
    float Tau;
    float Dt;

// IntegFmRaw computes integrated value from current raw value
//   void IntegFmRaw(inout DataStruct ds) {
//       ds.Integ = ds.Integ + this.Dt * (ds.Raw - ds.Integ);
//   }
};

layout (local_size_x = 256) in;

layout(set = 0, binding = 0) uniform Params {
	ParamStruct params;
};

layout(set = 1, binding = 0) buffer Data {
	DataStruct data[];
};
	
void main() {
	uint idx = gl_GlobalInvocationID.x;
	data[idx].Integ = data[idx].Integ + params.Dt * (data[idx].Raw - data[idx].Integ);
}

