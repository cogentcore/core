# Tree

The **Tree** is one of the most flexible, widely used data structures in programming, including the DOM structure at the core of a web browser, scene graphs for 3D and 2D graphics systems, JSON, XML, SVG, file systems, programs themselves, etc.  This is because trees can capture most relevant forms of *structure* (hierarchical groupings, categories, relationships, etc) and are most powerful when they are fully *generative* -- arbitrary new types can be inserted flexibly.

Cogent Core provides a general-purpose tree container type, that can support all of these applications, by embedding and extending the `NodeBase` struct type that implements the `Node` `interface`.  Unlike many cases in Go, the need to be able to arbitrarily extend the type space of nodes in the tree within a consistent API, means that the more traditional object-oriented model works best here, with a single common base type, and derived types that handle diverse cases (e.g., different types of widgets in a GUI).  Cogent Core stores a `Node` interface of each node, enabling correct virtual function calling on these derived types.

In addition, Cogent Core provides functions that traverse the tree in the usual relevant ways and take a `func` function argument, so you can easily apply a common operation across the whole tree in a transparent and self-contained manner, like this:

```go
func (mn *MyNode) DoSomethingOnMyTree() {
	mn.WalkDown(func(n tree.Node) bool {
		DoSomething(n)
		// ...
		return tree.Continue // return value determines whether tree traversal continues or not
	})
}
```

Many operations are naturally expressed in terms of these traversal algorithms.

## Making a Plan for the Tree

The `tree` package implements an efficient mechanism for dynamically updating the tree contents, based on creating a `tree.Plan` that specifies the desired elements of the tree, and functions to make and initialize new nodes when they are needed. The [base/plan](../base/plan) package implements the `Update` function that performs the minimal additions and deletions to update the current state of the tree to match the specified Plan.

The `NodeBase` contains a set of `Makers` functions that are called to create the Plan and define its functions, as well as a set of `Updaters` that are called for all elements when `Update` is called.  Together, these allow significant flexibility in creating efficient, dynamically updatable trees.

In the Cogent Core GUI context, this infrastructure provides a more efficient and effective alternative to the _declarative_ paradigm used in many other GUI frameworks, where the _entire_ new desired state of the GUI is constructed at _every update_, and then a generic "diff"-like algorithm compares that state with the current and drives the necessary updates.  This is typically extremely wasteful in terms of unnecessary memory and processing churn, and also creates problems tracking the current active state of elements.

By contrast, the Plan-based mechanism does the diff only on unique names of each element in the tree, and calls closure functions to configure and update any new elements as needed, instead of running such functions for every single element even when unnecessary.

## Fast finding in a slice

Several tree methods take an optional `startIndex` argument that is used by the generic [base/slicesx](../base/slicesx) `Search` algorithm to search for the item starting at that index, looking up and down from that starting point.  Thus, if you have any idea where the item might be in the list, it can save (considerable for large lists) time finding it.

The `IndexInParent()` method uses this trick, using the cached `NodeBase.index` value, so it is very fast in general, while also robust to dynamically changing trees.


