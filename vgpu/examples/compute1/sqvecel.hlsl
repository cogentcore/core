// HLSL compute example

[[vk::binding(0, 0)]] RWStructuredBuffer<float4> In;
[[vk::binding(1, 0)]] RWStructuredBuffer<float4> Out;

[numthreads(64, 1, 1)]

void main(uint3 idx : SV_DispatchThreadID) {
    Out[idx.x] = In[idx.x] * In[idx.x];
}

