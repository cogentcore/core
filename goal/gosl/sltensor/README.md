# sltensor: tensor access in WGSL

sltensor has functions to set the shape of a [tensor](../../tensor) to encode the strides in their first NumDims values, which are used to index into the tensor values.

For example:
```Go
sltensor.SetShapeSizes(Data, n, 4)
```

