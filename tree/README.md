# Trees in Go

The **Tree** is one of the most flexible, widely-used data structures in programming, including the DOM structure at the core of a web browser, scene graphs for 3D and 2D graphics systems, JSON, XML, SVG, filesystems, programs themselves, etc.  This is because trees can capture most relevant forms of *structure* (hierarchical groupings, categories, relationships, etc) and are most powerful when they are fully *generative* -- arbitrary new types can be inserted flexibly.

Cogent Core provides a general-purpose tree container type, that can support all of these applications, by embedding and extending the `NodeBase` struct type that implements the `Node` `interface`.  Unlike many cases in Go, the need to be able to arbitrarily extend the type space of nodes in the tree within a consistent API, means that the more traditional object-oriented model works best here, with a single common base type, and derived types that handle diverse cases (e.g., different types of widgets in a GUI).  Cogent Core stores a `Node` interface of each node, enabling correct virtual function calling on these derived types.

A virtue of using an appropriate data representation is that some important operations can be performed particularly concisely and efficiently when they are naturally supported by the data structure.  For example, matrices and vectors as supported by numpy or MATLAB provide a concise high-level language for expressing many algorithms.

In addition, Cogent Core provides functions that traverse the tree in the usual relevant ways and take a `func` function argument, so you can easily apply a common operation across the whole tree in a transparent and self-contained manner, like this:

```go
func (mn *MyNode) DoSomethingOnMyTree() {
	mn.WalkPre(func(n tree.Node) bool {
		DoSomething(n)
		// ...
		return tree.Continue // return value determines whether tree traversal continues or not
	})
}
```

Many operations are naturally expressed in terms of these traversal algorithms.

Three core Cogent Core features include:

* `ConfigChildren` uses a list of types and names and performs a minimal, efficient update of the children of a node to configure them to match (including no changes if already configured accordingly).  This is used during loading from JSON, and extensively in the `Cogent Core` GUI system to efficiently re-use existing tree elements.  There is often complex logic to determine what elements need to be present in a Widget, so separating that out from then configuring the elements that actually are present is efficient and simplifies the code.

## Cogent Core Graphical Interface

The first and most important application of ki is the Cogent Core graphical interface system, in the [core](https://pkg.go.dev/cogentcore.org/core/core) package.  The scene graph of tree elements automatically drives minimal refresh updates, allowing Cogent Core to provide a complete interactive 2D and 3D GUI environment in native Go, in a compact codebase.  Part of this is the natural elegance of Go, but Cogent Core enhances that by providing the robust natural primitives needed to express all the GUI functionality.

The [parse](https://pkg.go.dev/cogentcore.org/core/parse) interactive parsing framework also leverages Cogent Core trees to represent the AST (abstract syntax tree) of programs in different languages.  Further, the parser grammar itself is written (in a GUI interactive way) as a tree of parsing elements using tree nodes.

# Simple Example 

See `node_test.go` for lots of simple usage examples.

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
