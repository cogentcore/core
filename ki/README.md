Go language (golang) tree structure (ki = 木 = tree in Japanese)

[![Go Reference](https://pkg.go.dev/badge/cogentcore.org/core/ki.svg)](https://pkg.go.dev/cogentcore.org/core/ki)

**NOTE:** A new version of Ki is being developed on this branch, so there may be breaking changes and unstable code. For the last stable version of Ki, import v1.1.16 and see the [v1 branch](https://cogentcore.org/core/ki/tree/v1).

# Overview

The **Tree** is one of the most flexible, widely-used data structures in programming, including the DOM structure at the core of a web browser, scene graphs for 3D and 2D graphics systems, JSON, XML, SVG, filesystems, programs themselves, etc.  This is because trees can capture most relevant forms of *structure* (hierarchical groupings, categories, relationships, etc) and are most powerful when they are fully *generative* -- arbitrary new types can be inserted flexibly.

Cogent Core provides a general-purpose tree container type, that can support all of these applications, by embedding and extending the `Node` struct type that implements the `Ki` (Ki = Tree in Japanese) `interface`.  Unlike many cases in Go, the need to be able to arbitrarily extend the type space of nodes in the tree within a consistent API, means that the more traditional object-oriented model works best here, with a single common base type, and derived types that handle diverse cases (e.g., different types of widgets in a GUI).  Cogent Core stores a Ki interface of each node, enabling correct virtual function calling on these derived types.

A virtue of using an appropriate data representation is that some important operations can be performed particularly concisely and efficiently when they are naturally supported by the data structure.  For example, matrices and vectors as supported by numpy or MATLAB provide a concise high-level language for expressing many algorithms.


In addition, Cogent Core provides functions that traverse the tree in the usual relevant ways ("natural" me-first depth-first, me-last depth-first, and breadth-first) and take a `func` function argument, so you can easily apply a common operation across the whole tree in a transparent and self-contained manner, like this:

```go
func (n *MyNode) DoSomethingOnMyTree() {
	n.FuncDownMeFirst(0, nil, func(k Ki, level int, d interface{}) bool {
		mn := KiToMyNode(k) // function converts a Ki to relevant node type -- you must write
		mn.DoSomething()
		...
		return ki.Continue // return value determines whether tree traversal continues or not
	})
}
```

Many operations are naturally expressed in terms of these traversal algorithms.

Three core Cogent Core features include:

* `ConfigChildren` uses a list of types and names and performs a minimal, efficient update of the children of a node to configure them to match (including no changes if already configured accordingly).  This is used during loading from JSON, and extensively in the `Cogent Core` GUI system to efficiently re-use existing tree elements.  There is often complex logic to determine what elements need to be present in a Widget, so separating that out from then configuring the elements that actually are present is efficient and simplifies the code.

In addition, Ki nodes support a general-purpose `Props` property `map`, and the `kit` (Ki Types) package provides a `TypeRegistry` and an `EnumRegistry`, along with various `reflect` utilities, to enable fully-automatic saving / loading of Ki trees from JSON or XML, including converting const int (enum) values to / from strings so those numeric values can change in the code without invalidating existing files.

Ki Nodes can be used as fields in a struct -- they function much like pre-defined Children elements, and all the standard FuncDown* iterators traverse the fields automatically.  The Ki Init function automatically names these structs with their field names, and sets the parent to the parent struct.  This was essential in the Cogent Core framework to support separate Widget Parts independent of the larger scenegraph.

## Cogent Core Graphical Interface

The first and most important application of ki is the Cogent Core graphical interface system, in the [gi](https://pkg.go.dev/cogentcore.org/core/gi) package.  The scene graph of Ki elements automatically drives minimal refresh updates, allowing Cogent Core to provide a complete interactive 2D and 3D GUI environment in native Go, in a compact codebase.  Part of this is the natural elegance of Go, but Cogent Core enhances that by providing the robust natural primitives needed to express all the GUI functionality.

The [pi](https://pkg.go.dev/cogentcore.org/core/pi) interactive parsing framework also leverages Cogent Core trees to represent the AST (abstract syntax tree) of programs in different languages.  Further, the parser grammar itself is written (in a GUI interactive way) as a tree of parsing elements using Ki nodes.

# Code Map

* `kit` package: `kit.Types` `TypeRegistry` provides name-to-type map for looking up types by name, and types can have default properties. `kit.Enums` `EnumRegistry` provides enum (const int) <-> string conversion, including `bitflag` enums.  Also has robust generic `kit.ToInt` `kit.ToFloat` etc converters from `interface{}` to specific type, for processing properties, and several utilities in `embeds.go` for managing embedded structure types (e.g., `TypeEmbeds` checks if one type embeds another, and `EmbeddedStruct` returns the embedded struct from a given struct, providing flexible access to elements of an embedded type hierarchy -- there are also methods for navigating the flattened list of all embedded fields within a struct).  Also has a `kit.Type` struct that supports saving / loading of type information using type names.

* `walki` package provides tree-walking methods for more ad-hoc, special-case tree traversal, as compared to the standard Func* methods on Ki itself.

* `bitflag` package: simple bit flag setting, checking, and clearing methods that take bit position args as ints (from const int eunum iota's) and do the bit shifting from there

* `ki.go` = `Ki` interface for all major tree node functionality.

* `slice.go` = `ki.Slice []Ki` supports saving / loading of Ki objects in a slice, by recording the size and types of elements in the slice -- requires `kit.Types` type registry to lookup types by name.

* `props.go` = `ki.Props map[string]interface{}` supports saving / loading of property values using actual `struct` types and named const int enums, using the `kit` type registries.  Used for CSS styling in `Cogent Core`.

* `signal.go` = `Signal` that calls function on a receiver Ki objects that have been previously `Connect`ed to the signal -- also supports signal type so the same signal sender can send different types of signals over the same connection -- used for signaling changes in tree structure, and more general tree updating signals.

# Status

* Feb, 2021: version 1.1.0 reflects major simplification pass to reduce API footprint and remove separate Unique names (names should in general be unique -- add a separate non-unique name where needed).  Now that Cogent Core etc is complete, could get rid if quite a few things.

* April, 2020: version 1.0.0 release -- all stable and well tested.

# Simple Example 

See `ki/node_test.go` for lots of simple usage examples.  Here's some code from there that makes a tree with a parent and 2 children.

```go
parent := NodeEmbed{}
parent.InitName(&parent, "par1") // root must be initialized -- this also names it.
typ := reflect.TypeOf(parent)
parent.NewChild(typ, "child1") // Add etc methods auto-initialize children
parent.NewChild(typ, "child2")

// traverse the tree calling the parent node first and then its children, recursively
// "natural" order
parent.FuncDownMeFirst(0, nil, func(k Ki, level int, d interface{}) bool {
	fmt.Printf("level: %d  node: %s\n", level, k.Path())
	return ki.Continue
})
```

# Trick for fast finding in a slice

Cogent Core takes an extra starting index arg for all methods that lookup a value in a slice, such as ChildByName.  The search for the item starts at that index, and goes up and down from there.  Thus, if you have any idea where the item might be in the list, it can save (considerable for large lists) time finding it.

Furthermore, it enables a robust optimized lookup map that remembers these indexes for each item, but then always searches from the index, so it is always correct under list modifications, but if the list is unchanged, then it is very efficient, and does not require saving pointers, which minimizes any impact on the GC, prevents stale pointers, etc.

The `IndexInParent()` method uses this trick, using the cached `Node.index` value.

Here's example code for a separate Find method where the indexes are stored in a map:

```Go
// FindByName finds item by name, using cached indexes for speed
func (ob *Obj) FindByName(nm string) *Obj {
	if sv.FindIndexes == nil {
		ob.FindIndexes = make(map[string]int) // field on object
	}
	idx, has := ob.FindIndexes[nm]
	if !has {
		idx = len(ob.Kids) / 2 // start in middle first time
	}
	idx, has = ob.Kids.IndexByName(nm, idx)
	if has {
		ob.FindIndexes[nm] = idx
		return ob.Kids[idx].(*Obj)
  	}
	delete(ob.FindIndexes, nm) // must have been deleted
	return nil
}
```
