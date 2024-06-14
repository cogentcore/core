// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tree

import (
	"log/slog"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/jinzhu/copier"

	"cogentcore.org/core/base/elide"
	"cogentcore.org/core/types"
)

// NodeBase implements the [Node] interface and provides the core functionality
// for the Cogent Core tree system. You must use NodeBase as an embedded struct
// in all higher-level tree types.
//
// All nodes must be properly initialized by using one of [New], [NodeBase.NewChild],
// [NodeBase.AddChild], [NodeBase.InsertChild], [NodeBase.InsertNewChild],
// [NodeBase.Clone], [Update], or [cogentcore.org/core/core.Plan]. This ensures
// that the [Node.This] field is set correctly and that the [Node.Init] method is
// called.
//
// All nodes support JSON marshalling and unmarshalling through the standard [encoding/json]
// interfaces, so you can use the standard functions for loading and saving trees. However,
// if you want to load a root node of the correct type from JSON, you need to use the
// [UnmarshalRootJSON] function.
//
// All node types must be added to the Cogent Core type registry via typegen,
// so you must add a go:generate line that runs `core generate` to any packages
// you write that have new node types defined.
type NodeBase struct {

	// Name is the name of this node, which is typically unique relative to other children of
	// the same parent. It can be used for finding and serializing nodes. If not otherwise set,
	// it defaults to the ID (kebab-case) name of the node type combined with the total number
	// of children that have ever been added to the node's parent.
	Name string `copier:"-"`

	// This is the value of this Node as its true underlying type. This allows methods
	// defined on base types to call methods defined on higher-level types, which
	// is necessary for various parts of tree and widget functionality. This is set
	// to nil when the node is deleted.
	This Node `copier:"-" json:"-" xml:"-" display:"-" set:"-"`

	// Parent is the parent of this node, which is set automatically when this node is
	// added as a child of a parent. To change the parent of a node, use [MoveToParent];
	// you should typically not set this field directly. Nodes can only have one parent
	// at a time.
	Parent Node `copier:"-" json:"-" xml:"-" display:"-" set:"-"`

	// Children is the list of children of this node. All of them are set to have this node
	// as their parent. You can directly modify this list, but you should typically use the
	// various NodeBase child helper functions when applicable so that everything is updated
	// properly, such as when deleting children.
	Children []Node `table:"-" copier:"-" set:"-" json:",omitempty"`

	// Properties is a property map for arbitrary key-value properties.
	// When possible, use typed fields on a new type embedding NodeBase instead of this.
	// You should typically use the [NodeBase.SetProperty], [NodeBase.Property], and
	// [NodeBase.DeleteProperty] methods for modifying and accessing properties.
	Properties map[string]any `table:"-" xml:"-" copier:"-" set:"-" json:",omitempty"`

	// numLifetimeChildren is the number of children that have ever been added to this
	// node, which is used for automatic unique naming.
	numLifetimeChildren uint64

	// index is the last value of our index, which is used as a starting point for
	// finding us in our parent next time. It is not guaranteed to be accurate;
	// use the [NodeBase.IndexInParent] method.
	index int

	// depth is the depth of the node while using [NodeBase.WalkDownBreadth].
	depth int
}

// String implements the [fmt.Stringer] interface by returning the path of the node.
func (n *NodeBase) String() string {
	if n == nil || n.This == nil {
		return "nil"
	}
	return elide.Middle(n.Path(), 38)
}

// AsTree returns the [NodeBase] for this Node.
func (n *NodeBase) AsTree() *NodeBase {
	return n
}

// PlanName implements [plan.Namer].
func (n *NodeBase) PlanName() string {
	return n.Name
}

// BaseType returns the base node type for all elements within this tree.
// This is used in the GUI for determining what types of children can be created.
func (n *NodeBase) BaseType() *types.Type {
	return NodeBaseType
}

// Parents:

// IndexInParent returns our index within our parent node. It caches the
// last value and uses that for an optimized search so subsequent calls
// are typically quite fast. Returns -1 if we don't have a parent.
func (n *NodeBase) IndexInParent() int {
	if n.Parent == nil {
		return -1
	}
	idx := IndexOf(n.Parent.AsTree().Children, n.This, n.index) // very fast if index is close
	n.index = idx
	return idx
}

// ParentLevel finds a given potential parent node recursively up the
// hierarchy, returning the level above the current node that the parent was
// found, and -1 if not found.
func (n *NodeBase) ParentLevel(parent Node) int {
	parLev := -1
	level := 0
	n.WalkUpParent(func(k Node) bool {
		if k == parent {
			parLev = level
			return Break
		}
		level++
		return Continue
	})
	return parLev
}

// ParentByName finds first parent recursively up hierarchy that matches
// the given name. It returns nil if not found.
func (n *NodeBase) ParentByName(name string) Node {
	if IsRoot(n) {
		return nil
	}
	if n.Parent.AsTree().Name == name {
		return n.Parent
	}
	return n.Parent.AsTree().ParentByName(name)
}

// ParentByType finds parent recursively up hierarchy, by type, and
// returns nil if not found. If embeds is true, then it looks for any
// type that embeds the given type at any level of anonymous embedding.
func (n *NodeBase) ParentByType(t *types.Type, embeds bool) Node {
	if IsRoot(n) {
		return nil
	}
	if embeds {
		if n.Parent.NodeType().HasEmbed(t) {
			return n.Parent
		}
	} else {
		if n.Parent.NodeType() == t {
			return n.Parent
		}
	}
	return n.Parent.AsTree().ParentByType(t, embeds)
}

// Children:

// HasChildren returns whether this node has any children.
func (n *NodeBase) HasChildren() bool {
	return len(n.Children) > 0
}

// NumChildren returns the number of children this node has.
func (n *NodeBase) NumChildren() int {
	return len(n.Children)
}

// Child returns the child of this node at the given index and returns nil if
// the index is out of range.
func (n *NodeBase) Child(i int) Node {
	if i >= len(n.Children) || i < 0 {
		return nil
	}
	return n.Children[i]
}

// ChildByName returns the first child that has the given name, and nil
// if no such element is found. startIndex arg allows for optimized
// bidirectional find if you have an idea where it might be, which
// can be a key speedup for large lists. If no value is specified for
// startIndex, it starts in the middle, which is a good default.
func (n *NodeBase) ChildByName(name string, startIndex ...int) Node {
	return n.Child(IndexByName(n.Children, name, startIndex...))
}

// ChildByType returns the first child that has the given type, and nil
// if not found. If embeds is true, then it also looks for any type that
// embeds the given type at any level of anonymous embedding.
// startIndex arg allows for optimized bidirectional find if you have an
// idea where it might be, which can be a key speedup for large lists. If
// no value is specified for startIndex, it starts in the middle, which is a
// good default.
func (n *NodeBase) ChildByType(t *types.Type, embeds bool, startIndex ...int) Node {
	return n.Child(IndexByType(n.Children, t, embeds, startIndex...))
}

// Paths:

// TODO: is this the best way to escape paths?

// EscapePathName returns a name that replaces any / with \\
func EscapePathName(name string) string {
	return strings.ReplaceAll(name, "/", `\\`)
}

// UnescapePathName returns a name that replaces any \\ with /
func UnescapePathName(name string) string {
	return strings.ReplaceAll(name, `\\`, "/")
}

// Path returns the path to this node from the tree root,
// using [Node.Name]s separated by / delimeters. Any
// existing / characters in names are escaped to \\
func (n *NodeBase) Path() string {
	if n.Parent != nil {
		return n.Parent.AsTree().Path() + "/" + EscapePathName(n.Name)
	}
	return "/" + EscapePathName(n.Name)
}

// PathFrom returns the path to this node from the given parent node,
// using [Node.Name]s separated by / delimeters. Any
// existing / characters in names are escaped to \\
//
// The paths that it returns exclude the
// name of the parent and the leading slash; for example, in the tree
// a/b/c/d/e, the result of d.PathFrom(b) would be c/d. PathFrom
// automatically gets the [NodeBase.This] version of the given parent,
// so a base type can be passed in without manually accessing [NodeBase.This].
func (n *NodeBase) PathFrom(parent Node) string {
	// critical to get `This`
	parent = parent.AsTree().This
	// we bail a level below the parent so it isn't in the path
	if n.Parent == nil || n.Parent == parent {
		return EscapePathName(n.Name)
	}
	ppath := ""
	if n.Parent == parent {
		ppath = "/" + EscapePathName(parent.AsTree().Name)
	} else {
		ppath = n.Parent.AsTree().PathFrom(parent)
	}
	return ppath + "/" + EscapePathName(n.Name)

}

// FindPath returns the node at the given path from this node.
// FindPath only works correctly when names are unique.
// The given path must be consistent with the format produced
// by [NodeBase.PathFrom]. There is also support for index-based
// access (ie: [0] for the first child) for cases where indexes
// are more useful than names. It returns nil if no node is found
// at the given path.
func (n *NodeBase) FindPath(path string) Node {
	curn := n.This
	pels := strings.Split(strings.Trim(strings.TrimSpace(path), "\""), "/")
	for _, pe := range pels {
		if len(pe) == 0 {
			continue
		}
		idx := findPathChild(curn, UnescapePathName(pe))
		if idx < 0 {
			return nil
		}
		curn = curn.AsTree().Children[idx]
	}
	return curn
}

// findPathChild finds the child with the given string representation in [NodeBase.FindPath].
func findPathChild(n Node, child string) int {
	if child[0] == '[' && child[len(child)-1] == ']' {
		idx, err := strconv.Atoi(child[1 : len(child)-1])
		if err != nil {
			return idx
		}
		if idx < 0 { // from end
			idx = len(n.AsTree().Children) + idx
		}
		return idx
	}
	return IndexByName(n.AsTree().Children, child)
}

// Adding and Inserting Children:

// AddChild adds given child at end of children list.
// The kid node is assumed to not be on another tree (see [MoveToParent])
// and the existing name should be unique among children.
// Any error is automatically logged in addition to being returned.
func (n *NodeBase) AddChild(kid Node) error {
	if err := checkThis(n); err != nil {
		return err
	}
	initNode(kid)
	n.Children = append(n.Children, kid)
	SetParent(kid, n) // key to set new parent before deleting: indicates move instead of delete
	return nil
}

// NewChild creates a new child of the given type and adds it at the end
// of the list of children. The name defaults to the ID (kebab-case) name
// of the type, plus the [Node.NumLifetimeChildren] of the parent.
func (n *NodeBase) NewChild(typ *types.Type) Node {
	if err := checkThis(n); err != nil {
		return nil
	}
	kid := NewOfType(typ)
	initNode(kid)
	n.Children = append(n.Children, kid)
	SetParent(kid, n)
	return kid
}

// InsertChild adds given child at position in children list.
// The kid node is assumed to not be on another tree (see [MoveToParent])
// and the existing name should be unique among children.
// Any error is automatically logged in addition to being returned.
func (n *NodeBase) InsertChild(kid Node, index int) error {
	if err := checkThis(n); err != nil {
		return err
	}
	initNode(kid)
	n.Children = slices.Insert(n.Children, index, kid)
	SetParent(kid, n)
	return nil
}

// InsertNewChild creates a new child of given type and add at position
// in children list. The name defaults to the ID (kebab-case) name
// of the type, plus the [Node.NumLifetimeChildren] of the parent.
func (n *NodeBase) InsertNewChild(typ *types.Type, index int) Node {
	if err := checkThis(n); err != nil {
		return nil
	}
	kid := NewOfType(typ)
	initNode(kid)
	n.Children = slices.Insert(n.Children, index, kid)
	SetParent(kid, n)
	return kid
}

// Deleting Children:

// DeleteChildAt deletes child at the given index. It returns false
// if there is no child at the given index.
func (n *NodeBase) DeleteChildAt(index int) bool {
	child := n.Child(index)
	if child == nil {
		return false
	}
	n.Children = slices.Delete(n.Children, index, index+1)
	child.Destroy()
	return true
}

// DeleteChild deletes the given child node, returning false if
// it can not find it.
func (n *NodeBase) DeleteChild(child Node) bool {
	if child == nil {
		return false
	}
	idx := IndexOf(n.Children, child)
	if idx < 0 {
		return false
	}
	return n.DeleteChildAt(idx)
}

// DeleteChildByName deletes child node by name, returning false
// if it can not find it.
func (n *NodeBase) DeleteChildByName(name string) bool {
	idx := IndexByName(n.Children, name)
	if idx < 0 {
		return false
	}
	return n.DeleteChildAt(idx)
}

// DeleteChildren deletes all children nodes.
func (n *NodeBase) DeleteChildren() {
	kids := n.Children
	n.Children = n.Children[:0] // preserves capacity of list
	for _, kid := range kids {
		if kid == nil {
			continue
		}
		kid.Destroy()
	}
}

// Delete deletes this node from its parent's children list
// and then destroys itself.
func (n *NodeBase) Delete() {
	if n.Parent == nil {
		n.This.Destroy()
	} else {
		n.Parent.AsTree().DeleteChild(n.This)
	}
}

// Destroy recursively deletes and destroys the node, all of its children,
// and all of its children's children, etc.
func (n *NodeBase) Destroy() {
	if n.This == nil { // already destroyed
		return
	}
	n.DeleteChildren()
	n.This = nil
}

// Property Storage:

// SetProperty sets given the given property to the given value.
func (n *NodeBase) SetProperty(key string, value any) {
	if n.Properties == nil {
		n.Properties = map[string]any{}
	}
	n.Properties[key] = value
}

// Property returns the property value for the given key.
// It returns nil if it doesn't exist.
func (n *NodeBase) Property(key string) any {
	return n.Properties[key]
}

// DeleteProperty deletes the property with the given key.
func (n *NodeBase) DeleteProperty(key string) {
	if n.Properties == nil {
		return
	}
	delete(n.Properties, key)
}

// Tree Walking:

// WalkUp calls the given function on the node and all of its parents,
// sequentially in the current goroutine (generally necessary for going up,
// which is typically quite fast anyway). It stops walking if the function
// returns [Break] and keeps walking if it returns [Continue]. It returns
// whether walking was finished (false if it was aborted with [Break]).
func (n *NodeBase) WalkUp(fun func(n Node) bool) bool {
	cur := n.This
	for {
		if !fun(cur) { // false return means stop
			return false
		}
		parent := cur.AsTree().Parent
		if parent == nil || parent == cur { // prevent loops
			return true
		}
		cur = parent
	}
}

// WalkUpParent calls the given function on all of the node's parents (but not
// the node itself), sequentially in the current goroutine (generally necessary
// for going up, which is typically quite fast anyway). It stops walking if the
// function returns [Break] and keeps walking if it returns [Continue]. It returns
// whether walking was finished (false if it was aborted with [Break]).
func (n *NodeBase) WalkUpParent(fun func(n Node) bool) bool {
	if IsRoot(n) {
		return true
	}
	cur := n.Parent
	for {
		if !fun(cur) { // false return means stop
			return false
		}
		parent := cur.AsTree().Parent
		if parent == nil || parent == cur { // prevent loops
			return true
		}
		cur = parent
	}
}

// WalkDown strategy: https://stackoverflow.com/questions/5278580/non-recursive-depth-first-search-algorithm

// WalkDown calls the given function on the node and all of its children
// in a depth-first manner over all of the children, sequentially in the
// current goroutine. It stops walking the current branch of the tree if
// the function returns [Break] and keeps walking if it returns [Continue].
// It is non-recursive and safe for concurrent calling. The [Node.NodeWalkDown]
// method is called for every node after the given function, which enables nodes
// to also traverse additional nodes, like widget parts.
func (n *NodeBase) WalkDown(fun func(n Node) bool) {
	if n.This == nil {
		return
	}
	tm := map[Node]int{} // traversal map
	start := n.This
	cur := start
	tm[cur] = -1
outer:
	for {
		cb := cur.AsTree()
		if cb.This != nil && fun(cur) { // false return means stop
			cb.This.NodeWalkDown(fun)
			if cb.HasChildren() {
				tm[cur] = 0 // 0 for no fields
				nxt := cb.Child(0)
				if nxt != nil && nxt.AsTree().This != nil {
					cur = nxt.AsTree().This
					tm[cur] = -1
					continue
				}
			}
		} else {
			tm[cur] = cb.NumChildren()
		}
		// if we get here, we're in the ascent branch -- move to the right and then up
		for {
			cb := cur.AsTree() // may have changed, so must get again
			curChild := tm[cur]
			if (curChild + 1) < cb.NumChildren() {
				curChild++
				tm[cur] = curChild
				nxt := cb.Child(curChild)
				if nxt != nil && nxt.AsTree().This != nil {
					cur = nxt.AsTree().This
					tm[cur] = -1
					continue outer
				}
				continue
			}
			delete(tm, cur)
			// couldn't go right, move up..
			if cur == start {
				break outer // done!
			}
			parent := cb.Parent
			if parent == nil || parent == cur { // shouldn't happen, but does..
				// fmt.Printf("nil / cur parent %v\n", par)
				break outer
			}
			cur = parent
		}
	}
}

// NodeWalkDown is a placeholder implementation of [Node.NodeWalkDown]
// that does nothing.
func (n *NodeBase) NodeWalkDown(fun func(n Node) bool) {}

// WalkDownPost iterates in a depth-first manner over the children, calling
// shouldContinue on each node to test if processing should proceed (if it returns
// [Break] then that branch of the tree is not further processed),
// and then calls the given function after all of a node's children
// have been iterated over. In effect, this means that the given function
// is called for deeper nodes first. This uses node state information to manage
// the traversal and is very fast, but can only be called by one goroutine at a
// time, so you should use a Mutex if there is a chance of multiple threads
// running at the same time. The nodes are processed in the current goroutine.
func (n *NodeBase) WalkDownPost(shouldContinue func(n Node) bool, fun func(n Node) bool) {
	if n.This == nil {
		return
	}
	tm := map[Node]int{} // traversal map
	start := n.This
	cur := start
	tm[cur] = -1
outer:
	for {
		cb := cur.AsTree()
		if cb.This != nil && shouldContinue(cur) { // false return means stop
			if cb.HasChildren() {
				tm[cur] = 0 // 0 for no fields
				nxt := cb.Child(0)
				if nxt != nil && nxt.AsTree().This != nil {
					cur = nxt.AsTree().This
					tm[cur] = -1
					continue
				}
			}
		} else {
			tm[cur] = cb.NumChildren()
		}
		// if we get here, we're in the ascent branch -- move to the right and then up
		for {
			cb := cur.AsTree() // may have changed, so must get again
			curChild := tm[cur]
			if (curChild + 1) < cb.NumChildren() {
				curChild++
				tm[cur] = curChild
				nxt := cb.Child(curChild)
				if nxt != nil && nxt.AsTree().This != nil {
					cur = nxt.AsTree().This
					tm[cur] = -1
					continue outer
				}
				continue
			}
			fun(cur) // now we call the function, last..
			// couldn't go right, move up..
			delete(tm, cur)
			if cur == start {
				break outer // done!
			}
			parent := cb.Parent
			if parent == nil || parent == cur { // shouldn't happen
				break outer
			}
			cur = parent
		}
	}
}

// Note: it does not appear that there is a good
// recursive breadth-first-search strategy:
// https://herringtondarkholme.github.io/2014/02/17/generator/
// https://stackoverflow.com/questions/2549541/performing-breadth-first-search-recursively/2549825#2549825

// WalkDownBreadth calls the given function on the node and all of its children
// in breadth-first order. It stops walking the current branch of the tree if the
// function returns [Break] and keeps walking if it returns [Continue]. It is
// non-recursive, but not safe for concurrent calling.
func (n *NodeBase) WalkDownBreadth(fun func(n Node) bool) {
	start := n.This

	level := 0
	start.AsTree().depth = level
	queue := make([]Node, 1)
	queue[0] = start

	for {
		if len(queue) == 0 {
			break
		}
		cur := queue[0]
		depth := cur.AsTree().depth
		queue = queue[1:]

		if cur.AsTree().This != nil && fun(cur) { // false return means don't proceed
			for _, cn := range cur.AsTree().Children {
				if cn != nil && cn.AsTree().This != nil {
					cn.AsTree().depth = depth + 1
					queue = append(queue, cn)
				}
			}
		}
	}
}

// Deep Copy:

// note: we use the copy from direction (instead of copy to), as the receiver
// is modified whereas the from is not and assignment is typically in the same
// direction.

// CopyFrom copies the data and children of the given node to this node.
// It is essential that the source node has unique names. It is very efficient
// by using the [Node.ConfigChildren] method which attempts to preserve any
// existing nodes in the destination if they have the same name and type, so a
// copy from a source to a target that only differ minimally will be
// minimally destructive. Only copying to the same type is supported.
// The struct field tag copier:"-" can be added for any fields that
// should not be copied. Also, unexported fields are not copied.
// See [Node.CopyFieldsFrom] for more information on field copying.
func (n *NodeBase) CopyFrom(from Node) {
	if from == nil {
		slog.Error("tree.NodeBase.CopyFrom: nil source", "destinationNode", n)
		return
	}
	copyFrom(n.This, from)
}

// copyFrom is the implementation of [NodeBase.CopyFrom].
func copyFrom(to, from Node) {
	fc := from.AsTree().Children
	if len(fc) == 0 {
		to.AsTree().DeleteChildren()
	} else {
		p := make(TypePlan, len(fc))
		for i, c := range fc {
			p[i].Type = c.NodeType()
			p[i].Name = c.AsTree().Name
		}
		UpdateSlice(&to.AsTree().Children, to, p)
	}

	maps.Copy(to.AsTree().Properties, from.AsTree().Properties)

	to.AsTree().This.CopyFieldsFrom(from)
	for i, kid := range to.AsTree().Children {
		fmk := from.AsTree().Child(i)
		copyFrom(kid, fmk)
	}
}

// Clone creates and returns a deep copy of the tree from this node down.
// Any pointers within the cloned tree will correctly point within the new
// cloned tree (see [Node.CopyFrom] for more information).
func (n *NodeBase) Clone() Node {
	nc := NewOfType(n.This.NodeType())
	initNode(nc)
	nc.AsTree().SetName(n.Name)
	nc.AsTree().CopyFrom(n.This)
	return nc
}

// CopyFieldsFrom copies the fields of the node from the given node.
// By default, it is [NodeBase.CopyFieldsFrom], which automatically does
// a deep copy of all of the fields of the node that do not a have a
// `copier:"-"` struct tag. Node types should only implement a custom
// CopyFieldsFrom method when they have fields that need special copying
// logic that can not be automatically handled. All custom CopyFieldsFrom
// methods should call [NodeBase.CopyFieldsFrom] first and then only do manual
// handling of specific fields that can not be automatically copied. See
// [cogentcore.org/core/core.WidgetBase.CopyFieldsFrom] for an example of a
// custom CopyFieldsFrom method.
func (n *NodeBase) CopyFieldsFrom(from Node) {
	err := copier.CopyWithOption(n.This, from.AsTree().This, copier.Option{CaseSensitive: true, DeepCopy: true})
	if err != nil {
		slog.Error("tree.NodeBase.CopyFieldsFrom", "err", err)
	}
}

// Event methods:

// Init is a placeholder implementation of
// [Node.Init] that does nothing.
func (n *NodeBase) Init() {}

// OnAdd is a placeholder implementation of
// [Node.OnAdd] that does nothing.
func (n *NodeBase) OnAdd() {}

// OnChildAdded is a placeholder implementation of
// [Node.OnChildAdded] that does nothing.
func (n *NodeBase) OnChildAdded(child Node) {}
