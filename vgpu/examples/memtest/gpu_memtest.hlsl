// HLSL memory test example

[[vk::binding(0, 0)]] RWStructuredBuffer<uint> Ba;
[[vk::binding(1, 0)]] RWStructuredBuffer<uint> Bb;

[numthreads(64, 1, 1)]
void main(uint3 idx : SV_DispatchThreadID) {
	Ba[idx.x] = idx.x;
	Bb[idx.x] = idx.x;
}

