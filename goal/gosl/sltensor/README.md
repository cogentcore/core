# sltensor: tensor access in WGSL

sltensor provides tensor indexing functions used by gosl to translate tensor access functions into direct code, for `array<f32>` and `array<u32>` global variables, which encode the strides in their first NumDims values. The 1D case just uses direct indexing with no strides.

Strides are always encoded for all dimensions to allow complete flexibility in memory organization.


