
@group(0) @binding(0)
var<storage, read_write> Counter: array<su64>;

@group(0) @binding(1)
var<storage, read_write> Data: array<Rnds>;

@compute
@workgroup_size(64)
fn main(@builtin(global_invocation_id) idx: vec3<u32>) {
	var ctr = Counter[0];
	var data = Data[idx.x];
	Rnds_RndGen(&data, ctr, idx.x);
	Data[idx.x] = data;
}

