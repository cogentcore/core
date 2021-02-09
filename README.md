![alt tag](logo/goki_logo.png)

Go language (golang) tree structure (ki = 木 = tree in Japanese)

[![Go Report Card](https://goreportcard.com/badge/github.com/goki/ki)](https://goreportcard.com/report/github.com/goki/ki)
[![GoDoc](https://godoc.org/github.com/goki/ki?status.svg)](https://godoc.org/github.com/goki/ki)
[![Travis](https://travis-ci.com/goki/ki.svg?branch=master)](https://travis-ci.com/goki/ki)

# Overview

See the [Wiki](https://github.com/goki/ki/wiki) for more docs, and [Google Groups goki-gi](https://groups.google.com/forum/#!forum/goki-gi) emailing list.

The **Tree** is one of the most flexible, widely-used data structures in programming, including the DOM structure at the core of a web browser, scene graphs for 3D and 2D graphics systems, JSON, XML, SVG, filesystems, programs themselves, etc.  This is because trees can capture most relevant forms of *structure* (hierarchical groupings, categories, relationships, etc) and are most powerful when they are fully *generative* -- arbitrary new types can be inserted flexibly.

GoKi provides a general-purpose tree container type, that can support all of these applications, by embedding and extending the `Node` struct type that implements the `Ki` (Ki = Tree in Japanese) `interface`.  Unlike many cases in Go, the need to be able to arbitrarily extend the type space of nodes in the tree within a consistent API, means that the more traditional object-oriented model works best here, with a single common base type, and derived types that handle diverse cases (e.g., different types of widgets in a GUI).  GoKi stores a Ki interface of each node, enabling correct virtual function calling on these derived types.

A virtue of using an appropriate data representation is that some important operations can be performed particularly concisely and efficiently when they are naturally supported by the data structure.  For example, matrices and vectors as supported by numpy or MATLAB provide a concise high-level language for expressing many algorithms.

For trees, GoKi leverages the tree structure for automatically computing the appropriate extent of a scenegraph that needs to be updated, with an arbitrary sequence of individual operations, by propagating updating flags through the tree, and tracking the "high water mark" (see UpdateStart / End).  This makes the GoGi GUI efficient in terms of what needs to be redrawn, while keeping the code local and simple.

In addition, GoKi provides functions that traverse the tree in the usual relevant ways ("natural" me-first depth-first, me-last depth-first, and breadth-first) and take a `func` function argument, so you can easily apply a common operation across the whole tree in a transparent and self-contained manner, like this:

```go
func (n *MyNode) DoSomethingOnMyTree() {
	n.FuncDownMeFirst(0, nil, func(k Ki, level int, d interface{}) bool {
		mn := KiToMyNode(k)
		mn.DoSomething()
		...
		return ki.Continue // return value determines whether tree traversal continues or not
	})
}
```

Many operations are naturally expressed in terms of these traversal algorithms.

Three core GoKi features include:

* A `Signal` mechanism that allows nodes to communicate changes and other events to arbitrary lists of other nodes (similar to the signals and slots from Qt).

* `UpdateStart()` and `UpdateEnd()` functions that wrap around code that changes the tree structure or contents -- these automatically and efficiently determine the highest level node that was affected by changes, and only that highest node sends an `Updated` signal.  This allows arbitrarily nested modifications to proceed independently, each wrapped in their own Start / End blocks, with the optimal minimal update signaling automatically computed.

* `ConfigChildren` uses a list of types and names and performs a minimal, efficient update of the children of a node to configure them to match (including no changes if already configured accordingly).  This is used during loading from JSON, and extensively in the `GoGi` GUI system to efficiently re-use existing tree elements.  There is often complex logic to determine what elements need to be present in a Widget, so separating that out from then configuring the elements that actually are present is efficient and simplifies the code.

In addition, Ki nodes support a general-purpose `Props` property `map`, and the `kit` (Ki Types) package provides a `TypeRegistry` and an `EnumRegistry`, along with various `reflect` utilities, to enable fully-automatic saving / loading of Ki trees from JSON or XML, including converting const int (enum) values to / from strings so those numeric values can change in the code without invalidating existing files.

Ki Nodes can be used as fields in a struct -- they function much like pre-defined Children elements, and all the standard FuncDown* iterators traverse the fields automatically.  The Ki Init function automatically names these structs with their field names, and sets the parent to the parent struct.  This was essential in the GoKi framework to support separate Widget Parts independent of the larger scenegraph.

## GoGi Graphical Interface and Gide IDE App

The first and most important application of GoKi is the [GoGi](https://github.com/goki/gi) graphical interface system, in the `gi` package, and the [Gide](https://github.com/goki/gide) IDE built on top of GoGi.  The scene graph of Ki elements automatically drives minimal refresh updates, and the signaling framework supports gui event delivery and e.g., the "onclick" event signaling from the `Button` widget, etc.  In short, GoGi provides a complete interactive 2D and 3D GUI environment in native Go, in a compact codebase.  Part of this is the natural elegance of Go, but GoKi enhances that by providing the robust natural primitives needed to express all the GUI functionality.  Because GoGi is based around standard CSS styles, SVG rendering, and supports all the major HTML elements, it could even provide a lightweight web browser: [Glide](https://github.com/goki/glide).

The [GoPi](https://github.com/goki/pi) interactive parsing framework also leverages GoKi trees to represent the AST (abstract syntax tree) of programs in different langauges.  Further, the parser grammar itself is written (in a GUI interactive way) as a tree of parsing elements using Ki nodes.

# Code Map

* `kit` package: `kit.Types` `TypeRegistry` provides name-to-type map for looking up types by name, and types can have default properties. `kit.Enums` `EnumRegistry` provides enum (const int) <-> string conversion, including `bitflag` enums.  Also has robust generic `kit.ToInt` `kit.ToFloat` etc converters from `interface{}` to specific type, for processing properties, and several utilities in `embeds.go` for managing embedded structure types (e.g., `TypeEmbeds` checks if one type embeds another, and `EmbeddedStruct` returns the embedded struct from a given struct, providing flexible access to elements of an embedded type hierarchy -- there are also methods for navigating the flattened list of all embedded fields within a struct).  Also has a `kit.Type` struct that supports saving / loading of type information using type names.

* `walki` package provides tree-walking methods for more ad-hoc, special-case tree traversal, as compared to the standard Func* methods on Ki itself.

* `bitflag` package: simple bit flag setting, checking, and clearing methods that take bit position args as ints (from const int eunum iota's) and do the bit shifting from there

* `ki.go` = `Ki` interface for all major tree node functionality.

* `slice.go` = `ki.Slice []Ki` supports saving / loading of Ki objects in a slice, by recording the size and types of elements in the slice -- requires `kit.Types` type registry to lookup types by name.

* `props.go` = `ki.Props map[string]interface{}` supports saving / loading of property values using actual `struct` types and named const int enums, using the `kit` type registries.  Used for CSS styling in `GoGi`.

* `signal.go` = `Signal` that calls function on a receiver Ki objects that have been previously `Connect`ed to the signal -- also supports signal type so the same signal sender can send different types of signals over the same connection -- used for signaling changes in tree structure, and more general tree updating signals.

# Status

* Feb, 2021: version 1.1.0 reflects major simplification pass to reduce API footprint and remove separate Unique names (names should in general be unique -- add a separate non-unique name where needed).  Now that GoGi etc is complete, could get rid if quite a few things.

* April, 2020: version 1.0.0 release -- all stable and well tested.

# Trick for fast finding in a slice

GoKi takes an extra starting index arg for all methods that lookup a value in a slice, such as ChildByName.  The search for the item starts at that index, and goes up and down from there.  Thus, if you have any idea where the item might be in the list, it can save (considerable for large lists) time finding it.

Furthermore, it enables a robust optimized lookup map that remembers these indexes for each item, but then always searches from the index, so it is always correct under list modifications, but if the list is unchanged, then it is very efficient, and does not require saving pointers, which minimizes any impact on the GC, prevents stale pointers, etc.

The `IndexInParent()` method uses this trick, using the cached `Node.index` value.

Here's example code for a separate Find method where the indexes are stored in a map:

```Go
// FindByName finds item by name, using cached indexes for speed
func (ob *Obj) FindByName(nm string) *Obj {
	if sv.FindIdxs == nil {
		ob.FindIdxs = make(map[string]int) // field on object
	}
	idx, has := ob.FindIdxs[nm]
	if !has {
		idx = len(ob.Kids) / 2 // start in middle first time
	}
	idx, has = ob.Kids.IndexByName(nm, idx)
	if has {
		ob.FindIdxs[nm] = idx
		return ob.Kids[idx].(*Obj)
  	}
	delete(ob.FindIdxs, nm) // must have been deleted
	return nil
}
```
