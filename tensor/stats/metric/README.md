# metric

`metric` provides various similarity / distance metrics for comparing tensors, operating on the `tensor.Indexed` standard data representation.

The signatures of all such metric functions are identical, captured as types: `metric.Func32` and `metric.Func64` so that other functions that use a metric can take a pointer to any such function.


