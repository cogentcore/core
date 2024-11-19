# tensorfs: a virtual filesystem for tensor data

`tensorfs` is a virtual file system that implements the Go `fs` interface, and can be accessed using fs-general tools, including the cogent core `filetree` and the `goal` shell.

Values are represented using the [tensor] package universal data type: the `tensor.Tensor`, which can represent everything from a single scalar value up to n-dimensional collections of patterns, in a range of data types.

A given `Node` in the file system is either:
* A _Value_, with a tensor encoding its value. These are terminal "leaves" in the hierarchical data tree, equivalent to "files" in a standard filesystem.
* A _Directory_, with an ordered map of other Node nodes under it.

Each Node has a name which must be unique within the directory. The nodes in a directory are processed in the order of its ordered map list, which initially reflects the order added, and can be re-ordered as needed. An alphabetical sort is also available with the `Alpha` versions of methods, and is the default sort for standard FS operations.

The hierarchical structure of a filesystem naturally supports various kinds of functions, such as various time scales of logging, with lower-level data aggregated into upper levels.  Or hierarchical splits for a pivot-table effect.

# Usage

There are two main APIs, one for direct usage within Go, and another that is used by the `goal` framework for interactive shell-based access, which always operates relative to a current working directory.

## Go API

The primary Go access function is the generic `Value`:

```Go
tsr := tensorfs.Value[float64](dir, "filename", 5, 5)
```

This returns a `tensor.Values` for the node `"filename"` in the directory Node `dir` with the tensor shape size of 5x5, and `float64` values.

If the tensor was previously created, then it is returned, and otherwise it is created.  This provides a robust single-function API for access and creation, and it doesn't return any errors, so the return value can used directly, in inline expressions etc.

For efficiency, _there are no checks_ on the existing value relative to the arguments passed, so if you end up using the same name for two different things, that will cause problems that will hopefully become evident. If you want to ensure that the size is correct, you should use an explicit `tensor.SetShapeSizes` call, which is still quite efficient if the size is the same. You can also have an initial call to `Value` that has no size args, and then set the size later -- that works fine.

All tensor value access is via standalone functions, not methods, with methods reserved for directory node functionality.

There are also a few other variants of the `Value` functionality:
* `Scalar` calls `Value` with a size of 1.
* `Values` makes multiple tensor values of the same shape, with a final variadic list of names.
* `ValueType` takes a `reflect.Kind` arg for the data type, which can then be a variable.
* `NewForTensor` creates a node for an existing tensor.

`DirTable` returns a `table.Table` with all the tensors under a given directory node, which can then be used for making plots or doing other forms of data analysis. This works best when each tensor has the same outer-most row dimension. The table is persistent and very efficient, using direct pointers to the underlying tensor values.

### Directories

Directories are `Node` elements that have a `Dir` value (ordered map of named nodes) instead of a tensor value.

```Go
dir := tensorfs.NewDir("subdir", dir) // make a new directory
```

There are parallel `Node` and `Value` access methods for directory nodes, with the Value ones being:

* `tsr := dir.Value("name")` returns tensor directly, will panic if not valid
* `tsrs, err := dir.Values("name1", "name2")` returns a slice of tensors and error if any issues
* `tsrs := dir.ValuesFunc(<filter func>)` walks down directories (unless filtered) and returns a flat list of all tensors found. Goes in "directory order" = order nodes were added.
* `tsrs := dir.ValuesAlphaFunc(<filter func>)` is like `ValuesFunc` but traverses in alpha order at each node.

### Existing items and unique names

As in a real filesystem, names must be unique within each directory, which creates issues for how to manage conflicts between existing and new items. We adopt the same behavior as the Go `os` package in general:

* If an existing item with the same name is present, return that existing item and an `fs.ErrExist` error, so that the caller can decide how to proceed, using `errors.Is(fs.ErrExist)`.

* There are also `Recycle` versions of functions that do not return an error and are preferred when specifically expecting an existing item.

## `goal` Command API

The following shell command style functions always operate relative to the global `CurDir` current directory and `CurRoot` root, and `goal` in math mode exposes these methods directly. Goal operates on tensor valued variables always.

* `Chdir("subdir")` change current directory to subdir.
* `Mkdir("subdir")` make a new directory.
* `List()` print a list of nodes.
* `tsr := Get("mydata")` get tensor value at "mydata" node.
* `Set("mydata", tsr)` set tensor to "mydata" node.


