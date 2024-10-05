// sltensor indexing functions

fn F32Index2D(s0: f32, s1: f32, i0: u32, i1: u32) -> u32 {
	return u32(2) + bitcast<u32>(s0) * i0 + bitcast<u32>(s1) * i1;
}

fn F32Index3D(s0: f32, s1: f32, s2: f32, i0: u32, i1: u32, i2: u32) -> u32 {
	return u32(3) + bitcast<u32>(s0) * i0 + bitcast<u32>(s1) * i1 + bitcast<u32>(s2) * i2;
}

fn U32Index2D(s0: u32, s1: u32, i0: u32, i1: u32) -> u32 {
	return u32(2) + s0 * i0 + s1 * i1;
}

fn U32Index3D(s0: u32, s1: u32, s2: u32, i0: u32, i1: u32, i2: u32) -> u32 {
	return u32(3) + s0 * i0 + s1 * i1 + s2 * i2;
}

