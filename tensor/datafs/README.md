# datafs: a virtual filesystem for data

`datafs` is a virtual file system that implements the Go `fs` interface, and can be accessed using fs-general tools, including the cogent core `filetree` and the `goal` shell.

Data is represented using the [tensor] package universal data type: the `tensor.Indexed` `Tensor`, which can represent everything from a single scalar value up to n-dimensional collections of patterns, in a range of data types.

A given `Data` node either has a tensor `Value` or is a directory with an ordered map of other nodes under it.  Each value has a name which must be unique within the directory. The nodes are processed in the order of this list, which initially reflects the order added, and can be re-ordered as needed.

The hierarchical structure of a filesystem naturally supports various kinds of functions, such as various time scales of logging, with lower-level data aggregated into upper levels.  Or hierarchical splits for a pivot-table effect.


