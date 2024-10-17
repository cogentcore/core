# datafs: a virtual filesystem for data

`datafs` is a virtual file system that implements the Go `fs` interface, and can be accessed using fs-general tools, including the cogent core `filetree` and the `goal` shell.

Data is represented using the [tensor] package universal data type: the `tensor.Tensor`, which can represent everything from a single scalar value up to n-dimensional collections of patterns, in a range of data types.

A given `Data` node is either:
* A _Value_, with a tensor encoding its `Data` value. These are terminal "leaves" in the hierarchical data tree, equivalent to "files" in a standard filesystem.
* A _Directory_, with an ordered map of other Data nodes under it.

Each Data node has a name which must be unique within the directory. The nodes in a directory are processed in the order of its ordered map list, which initially reflects the order added, and can be re-ordered as needed.  An alphabetical sort is also available with the `Alpha` versions of methods, and is the default sort for standard FS operations.

The hierarchical structure of a filesystem naturally supports various kinds of functions, such as various time scales of logging, with lower-level data aggregated into upper levels.  Or hierarchical splits for a pivot-table effect.


