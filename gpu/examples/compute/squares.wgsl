// WGSL compute example

@group(0) @binding(0)
var<storage, read_write> In: array<f32>;

// note: read_write and both @group and @binding are required
@group(0) @binding(1)
var<storage, read_write> Out: array<f32>;

@compute
@workgroup_size(64)
fn main(@builtin(global_invocation_id) idx: vec3<u32>) {
	Out[idx.x] = In[idx.x] * In[idx.x];
}

