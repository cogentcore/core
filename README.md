# ki
Part of the GoKi Go language (golang) full strength tree structure system (ki = tree in Japanese)

`package ki` -- core `Ki` interface (`ki.go`) and `Node` struct (`node.go`), plus other supporting players

GoDoc documentation: https://godoc.org/github.com/rcoreilly/goki/ki

A Ki tree is recursively composed of Ki Node structs, in a one-Parent / multiple-Child structure.  The typical use is to embed Node in other structs that then implement specific tree-based functionality.  See other packages in GoKi for examples.

# Code Map

* `ki.go` = `Ki` interface for all major tree node functionality
* `kiptr.go` = `KiPtr` struct that supports saving / loading of pointers using paths
* `kislice.go` = `KiSlice []Ki` supports saving / loading of Ki objects in a slice, by recording the size and types of elements in the slice -- requires `KiTypes` type registry to lookup types by name
* `kitype.go` = `KiType struct of reflect.Type` that supports saving / loading of type information using `KiTypes` type registry
* `kitypes.go` = `TypeRegistry` and `KiTypes TypeRegistry` provides name to type map for looking up types by name
* `signal.go` = `Signal struct` that can call any number of functions with receiver Ki objects that have been previously `Connect`ed to the signal -- also supports signal type so the same signal sender can send different types of signals over the same connection -- used for signaling changes in tree structure, and more general tree updating signals.

# Notes on Naming Conventions

https://golang.org/doc/effective_go.html#names

* Don't use Get* -- just use the name itself, when a "getter" is necessary
* Do use Set* -- setter + field name -- this is convention in Qt too
* Uppercase all Fields in general -- Strongly prefer to just deal with the fields directly without having to go through a getter method, and we want embedded objects and anyone else to be able to access those fields, and they also need to be saved / loaded through JSON, etc
* Above means that there can be conflicts for any interfaces that need to provide getter access to those fields.  bummer.  Using Ki prefix in those cases.
* Delete instead of Remove

# TODO

* Find*By* -- FindParentByType, FindChildByType / Name -- easy -- children need recursive
* various new items added to ki.go
* add SetField, FieldValue generic methods -- thin wrappers around reflect
* SetFieldRecursive -- apply to all children, no problem if not found

