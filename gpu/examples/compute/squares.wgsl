// WGSL compute example

struct Data {
	A: f32,
	B: f32,
	C: f32,
	D: f32,
}

@group(0) @binding(0)
var<storage, read_write> In: array<Data>;

fn compute(d: ptr<function,Data>) {
	(*d).C = (*d).A + (*d).B;
	(*d).D = (*d).C * (*d).C;
}

@compute
@workgroup_size(64)
fn main(@builtin(global_invocation_id) idx: vec3<u32>) {
	// compute(&In[idx.x]);
	var d = In[idx.x];
	compute(&d);
	In[idx.x] = d;
}

