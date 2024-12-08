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
@workgroup_size(64,1,1)
fn main(@builtin(workgroup_id) wgid: vec3<u32>, @builtin(num_workgroups) nwg: vec3<u32>, @builtin(local_invocation_index) loci: u32) {
	// note: wgid.x is the inner loop, then y, then z
	let idx = loci + (wgid.x + wgid.y * nwg.x + wgid.z * nwg.x * nwg.y) * 64;
	var d = In[idx];
	compute(&d);
	In[idx] = d;
	// the following is for testing indexing: uncomment to see.
	// In[idx].A = f32(loci); 
	// In[idx].B = f32(wgid.x); 
	// In[idx].C = f32(wgid.y); 
	// In[idx].D = f32(idx); 
}

