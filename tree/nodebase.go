// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tree

import (
	"errors"
	"log/slog"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/jinzhu/copier"

	"cogentcore.org/core/base/elide"
	"cogentcore.org/core/enums"
	"cogentcore.org/core/types"
)

// NodeBase implements the [Node] interface and provides the core functionality
// for the Cogent Core tree system. You should use NodeBase as an embedded struct
// in higher-level tree types.
type NodeBase struct {

	// Nm is the user-supplied name of this node, which can be empty and/or non-unique.
	// It is typically accessed through [Node.Name].
	Nm string `copier:"-" set:"-" label:"Name"`

	// Flags are bit flags for internal node state, which can be extended using
	// the enums package.
	Flags Flags `tableview:"-" copier:"-" json:"-" xml:"-" set:"-" max-width:"80" height:"3"`

	// Properties is a property map for arbitrary key-value properties.
	Properties map[string]any `tableview:"-" xml:"-" copier:"-" set:"-"`

	// Par is the parent of this node, which is set automatically when this node is
	// added as a child of a parent. It is typically accessed through [Node.Parent].
	Par Node `copier:"-" json:"-" xml:"-" view:"-" set:"-"`

	// Children is the list of children of this node. All of them are set to have this node
	// as their parent. They can be reordered, but you should generally use [Node]
	// methods when adding and deleting children to ensure everything gets updated.
	Children Slice `tableview:"-" copier:"-" set:"-"`

	// Ths is a pointer to ourselves as a [Node]. It can always be used to extract the
	// true underlying type of an object when [NodeBase] is embedded in other structs;
	// function receivers do not have this ability, so this is necessary. This is set
	// to nil when the node is deleted. It is typically accessed through [Node.This].
	// It needs to be exported so that it can be interacted with through reflection
	// during field copying.
	Ths Node `copier:"-" json:"-" xml:"-" view:"-" set:"-"`

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
	if n == nil || n.This() == nil {
		return "nil"
	}
	return elide.Middle(n.This().Path(), 38)
}

// This returns the Node as its true underlying type.
// It returns nil if the node is nil, has been destroyed,
// or is improperly constructed.
func (n *NodeBase) This() Node {
	if n == nil {
		return nil
	}
	return n.Ths
}

// AsTree returns the [NodeBase] for this Node.
func (n *NodeBase) AsTree() *NodeBase {
	return n
}

// Name returns the user-defined name of the Node, which can be
// used for finding elements, generating paths, I/O, etc.
func (n *NodeBase) Name() string {
	return n.Nm
}

// SetName sets the name of this node. Names should generally be unique
// across children of each node. If the node requires some non-unique name,
// add a separate Label field.
func (n *NodeBase) SetName(name string) {
	n.Nm = name
}

// BaseType returns the base node type for all elements within this tree.
// This is used in the GUI for determining what types of children can be created.
func (n *NodeBase) BaseType() *types.Type {
	return NodeBaseType
}

// Parents:

// Parent returns the parent of this Node.
// Each Node can only have one parent.
func (n *NodeBase) Parent() Node {
	return n.Par
}

// IndexInParent returns our index within our parent node. It caches the
// last value and uses that for an optimized search so subsequent calls
// are typically quite fast. Returns -1 if we don't have a parent.
func (n *NodeBase) IndexInParent() int {
	if n.Par == nil {
		return -1
	}
	idx := IndexOf(n.Par.AsTree().Children, n.This(), n.index) // very fast if index is close
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
	if n.Par.Name() == name {
		return n.Par
	}
	return n.Par.AsTree().ParentByName(name)
}

// ParentByType finds parent recursively up hierarchy, by type, and
// returns nil if not found. If embeds is true, then it looks for any
// type that embeds the given type at any level of anonymous embedding.
func (n *NodeBase) ParentByType(t *types.Type, embeds bool) Node {
	if IsRoot(n) {
		return nil
	}
	if embeds {
		if n.Par.NodeType().HasEmbed(t) {
			return n.Par
		}
	} else {
		if n.Par.NodeType() == t {
			return n.Par
		}
	}
	return n.Par.AsTree().ParentByType(t, embeds)
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
	return n.Child(n.Children.IndexByName(name, startIndex...))
}

// ChildByType returns the first child that has the given type, and nil
// if not found. If embeds is true, then it also looks for any type that
// embeds the given type at any level of anonymous embedding.
// startIndex arg allows for optimized bidirectional find if you have an
// idea where it might be, which can be a key speedup for large lists. If
// no value is specified for startIndex, it starts in the middle, which is a
// good default.
func (n *NodeBase) ChildByType(t *types.Type, embeds bool, startIndex ...int) Node {
	return n.Child(n.Children.IndexByType(t, embeds, startIndex...))
}

// Paths:

// TODO: is this the best way to escape paths?

// EscapePathName returns a name that replaces any path delimiter symbols
// . or / with \, and \\ escaped versions.
func EscapePathName(name string) string {
	return strings.ReplaceAll(strings.ReplaceAll(name, ".", `\,`), "/", `\\`)
}

// UnescapePathName returns a name that replaces any escaped path delimiter symbols
// \, or \\ with . and / unescaped versions.
func UnescapePathName(name string) string {
	return strings.ReplaceAll(strings.ReplaceAll(name, `\,`, "."), `\\`, "/")
}

// Path returns the path to this node from the tree root,
// using [Node.Name]s separated by / and fields by .
// Path is only valid for finding items when child names
// are unique. Any existing / and . characters in names
// are escaped to \\ and \,
func (n *NodeBase) Path() string {
	if n.Par != nil {
		if n.Is(Field) {
			return n.Par.Path() + "." + EscapePathName(n.Nm)
		}
		return n.Par.Path() + "/" + EscapePathName(n.Nm)
	}
	return "/" + EscapePathName(n.Nm)
}

// PathFrom returns path to this node from the given parent node, using
// [Node.Name]s separated by / and fields by .
// Path is only valid for finding items when child names
// are unique. Any existing / and . characters in names
// are escaped to \\ and \,
//
// The paths that it returns exclude the
// name of the parent and the leading slash; for example, in the tree
// a/b/c/d/e, the result of d.PathFrom(b) would be c/d. PathFrom
// automatically gets the [Node.This] version of the given parent,
// so a base type can be passed in without manually calling [Node.This].
func (n *NodeBase) PathFrom(parent Node) string {
	// critical to get "This"
	parent = parent.This()
	// we bail a level below the parent so it isn't in the path
	if n.Par == nil || n.Par == parent {
		return EscapePathName(n.Nm)
	}
	ppath := ""
	if n.Par == parent {
		ppath = "/" + EscapePathName(parent.Name())
	} else {
		ppath = n.Par.PathFrom(parent)
	}
	if n.Is(Field) {
		return ppath + "." + EscapePathName(n.Nm)
	}
	return ppath + "/" + EscapePathName(n.Nm)

}

// findPathChild finds the child on the path.
func findPathChild(n Node, child string) int {
	if len(child) == 0 {
		return -1
	}
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
	return n.AsTree().Children.IndexByName(child, 0)
}

// FindPath returns the node at the given path from this node.
// FindPath only works correctly when names are unique.
// Path has [Node.Name]s separated by / and fields by .
// Node names escape any existing / and . characters to \\ and \,
// There is also support for [idx] index-based access for any given path
// element, for cases when indexes are more useful than names.
// Returns nil if not found.
func (n *NodeBase) FindPath(path string) Node {
	curn := n.This()
	pels := strings.Split(strings.Trim(strings.TrimSpace(path), "\""), "/")
	for _, pe := range pels {
		if len(pe) == 0 {
			continue
		}
		if strings.Contains(pe, ".") { // has fields
			fels := strings.Split(pe, ".")
			// find the child first, then the fields
			idx := findPathChild(curn, UnescapePathName(fels[0]))
			if idx < 0 {
				return nil
			}
			curn = curn.AsTree().Children[idx]
			for i := 1; i < len(fels); i++ {
				fe := UnescapePathName(fels[i])
				fk, err := curn.FieldByName(fe)
				if err != nil {
					slog.Debug("tree.FindPath: %v", err)
					return nil
				}
				curn = fk
			}
		} else {
			idx := findPathChild(curn, UnescapePathName(pe))
			if idx < 0 {
				return nil
			}
			curn = curn.AsTree().Children[idx]
		}
	}
	return curn
}

// FieldByName is a placeholder implementation of [Node.FieldByName]
// that returns an error.
func (n *NodeBase) FieldByName(field string) (Node, error) {
	return nil, errors.New("tree.NodeBase.FieldByName: no tree fields defined for this node")
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
	idx := n.Children.IndexByName(name)
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
		kid.SetFlag(true)
		kid.Destroy()
	}
}

// Delete deletes this node from its parent's children list.
func (n *NodeBase) Delete() {
	if n.Par == nil {
		n.This().Destroy()
	} else {
		n.Par.AsTree().DeleteChild(n.This())
	}
}

// Destroy recursively deletes and destroys the node, all of its children,
// and all of its children's children, etc.
func (n *NodeBase) Destroy() {
	if n.This() == nil { // already destroyed
		return
	}
	n.DeleteChildren()
	n.Ths = nil
}

// Flags:

// Is checks if the given flag is set, using atomic,
// which is safe for concurrent access.
func (n *NodeBase) Is(f enums.BitFlag) bool {
	return n.Flags.HasFlag(f)
}

// SetFlag sets the given flag(s) to the given state
// using atomic, which is safe for concurrent access.
func (n *NodeBase) SetFlag(on bool, f ...enums.BitFlag) {
	n.Flags.SetFlag(on, f...)
}

// FlagType returns the flags of the node as the true flag type of the node,
// which may be a type that extends the standard [Flags]. Each node type
// that extends the flag type should define this method; for example:
//
//	func (wb *WidgetBase) FlagType() enums.BitFlagSetter {
//		return (*WidgetFlags)(&wb.Flags)
//	}
func (n *NodeBase) FlagType() enums.BitFlagSetter {
	return &n.Flags
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
	cur := n.This()
	for {
		if !fun(cur) { // false return means stop
			return false
		}
		parent := cur.Parent()
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
	cur := n.Parent()
	for {
		if !fun(cur) { // false return means stop
			return false
		}
		parent := cur.Parent()
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
	if n.This() == nil {
		return
	}
	tm := map[Node]int{} // traversal map
	start := n.This()
	cur := start
	tm[cur] = -1
outer:
	for {
		if cur.This() != nil && fun(cur) { // false return means stop
			cur.This().NodeWalkDown(fun)
			if cur.HasChildren() {
				tm[cur] = 0 // 0 for no fields
				nxt := cur.Child(0)
				if nxt != nil && nxt.This() != nil {
					cur = nxt.This()
					tm[cur] = -1
					continue
				}
			}
		} else {
			tm[cur] = cur.NumChildren()
		}
		// if we get here, we're in the ascent branch -- move to the right and then up
		for {
			curChild := tm[cur]
			if (curChild + 1) < cur.NumChildren() {
				curChild++
				tm[cur] = curChild
				nxt := cur.Child(curChild)
				if nxt != nil && nxt.This() != nil {
					cur = nxt.This()
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
			parent := cur.Parent()
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
// doChildTest on each node to test if processing should proceed (if it returns
// [Break] then that branch of the tree is not further processed),
// and then calls the given function after all of a node's children
// have been iterated over. In effect, this means that the given function
// is called for deeper nodes first. This uses node state information to manage
// the traversal and is very fast, but can only be called by one goroutine at a
// time, so you should use a Mutex if there is a chance of multiple threads
// running at the same time. The nodes are processed in the current goroutine.
func (n *NodeBase) WalkDownPost(doChildTestFunc func(n Node) bool, fun func(n Node) bool) {
	if n.This() == nil {
		return
	}
	tm := map[Node]int{} // traversal map
	start := n.This()
	cur := start
	tm[cur] = -1
outer:
	for {
		if cur.This() != nil && doChildTestFunc(cur) { // false return means stop
			if cur.HasChildren() {
				tm[cur] = 0 // 0 for no fields
				nxt := cur.Child(0)
				if nxt != nil && nxt.This() != nil {
					cur = nxt.This()
					tm[cur] = -1
					continue
				}
			}
		} else {
			tm[cur] = cur.NumChildren()
		}
		// if we get here, we're in the ascent branch -- move to the right and then up
		for {
			curChild := tm[cur]
			if (curChild + 1) < cur.NumChildren() {
				curChild++
				tm[cur] = curChild
				nxt := cur.Child(curChild)
				if nxt != nil && nxt.This() != nil {
					cur = nxt.This()
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
			parent := cur.Parent()
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
	start := n.This()

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

		if cur.This() != nil && fun(cur) { // false return means don't proceed
			for _, cn := range cur.AsTree().Children {
				if cn != nil && cn.This() != nil {
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
	copyFrom(n.This(), from)
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
			p[i].Name = c.Name()
		}
		UpdateSlice(&to.AsTree().Children, to, p)
	}

	maps.Copy(to.AsTree().Properties, from.AsTree().Properties)

	to.This().CopyFieldsFrom(from)
	for i, kid := range to.AsTree().Children {
		fmk := from.Child(i)
		copyFrom(kid, fmk)
	}
}

// Clone creates and returns a deep copy of the tree from this node down.
// Any pointers within the cloned tree will correctly point within the new
// cloned tree (see [Node.CopyFrom] for more information).
func (n *NodeBase) Clone() Node {
	nc := NewOfType(n.This().NodeType())
	initNode(nc)
	nc.SetName(n.Nm)
	nc.CopyFrom(n.This())
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
	err := copier.CopyWithOption(n.This(), from.This(), copier.Option{CaseSensitive: true, DeepCopy: true})
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
