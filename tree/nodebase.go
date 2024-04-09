// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tree

import (
	"errors"
	"fmt"
	"log"
	"log/slog"
	"maps"
	"strconv"
	"strings"

	"github.com/jinzhu/copier"

	"cogentcore.org/core/enums"
	"cogentcore.org/core/glop/elide"
	"cogentcore.org/core/gti"
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

	// Props is a property map for arbitrary key-value properties.
	// They are typically accessed through the property methods on [Node].
	Props map[string]any `tableview:"-" xml:"-" copier:"-" set:"-" label:"Properties"`

	// Par is the parent of this node, which is set automatically when this node is
	// added as a child of a parent. It is typically accessed through [Node.Parent].
	Par Node `copier:"-" json:"-" xml:"-" view:"-" set:"-"`

	// Kids is the list of children of this node. All of them are set to have this node
	// as their parent. They can be reordered, but you should generally use [Node]
	// methods when adding and deleting children to ensure everything gets updated.
	// They are typically accessed through [Node.Children].
	Kids Slice `tableview:"-" copier:"-" set:"-" label:"Children"`

	// this is a pointer to ourselves as a [Node]. It can always be used to extract the
	// true underlying type of an object when [NodeBase] is embedded in other structs;
	// function receivers do not have this ability, so this is necessary. This is set
	// to nil when the node is deleted. It is typically accessed through [Node.This].
	this Node

	// numLifetimeChildren is the number of children that have ever been added to this
	// node, which is used for automatic unique naming. It is typically accessed
	// through [Node.NumLifetimeChildren].
	numLifetimeChildren uint64

	// index is the last value of our index, which is used as a starting point for
	// finding us in our parent next time. It is not guaranteed to be accurate;
	// use the [Node.IndexInParent] method.
	index int

	// depth is the depth of the node while using [Node.WalkDownBreadth].
	depth int
}

// String implements the fmt.Stringer interface by returning the path of the node.
func (n *NodeBase) String() string {
	return elide.Middle(n.This().Path(), 38)
}

// This returns the Node as its true underlying type.
// It returns nil if the node is nil, has been destroyed,
// or is improperly constructed.
func (n *NodeBase) This() Node {
	if n == nil {
		return nil
	}
	return n.this
}

// AsTreeNode returns the [NodeBase] for this Node.
func (n *NodeBase) AsTreeNode() *NodeBase {
	return n
}

// InitName initializes this node to the given actual object as a Node interface
// and sets its name. The names should be unique among children of a node.
// This is called automatically when adding child nodes and using [NewRoot].
// If the name is unspecified, it defaults to the ID (kebab-case) name of the type.
// Even though this is a method and gets the method receiver, it needs
// an "external" version of itself passed as the first arg, from which
// the proper Node interface pointer will be obtained. This is the only
// way to get virtual functional calling to work within the Go language.
func (n *NodeBase) InitName(k Node, name ...string) {
	InitNode(k)
	if len(name) > 0 {
		n.SetName(name[0])
	}
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
func (n *NodeBase) BaseType() *gti.Type {
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
	idx, ok := n.Par.Children().IndexOf(n.This(), n.index) // very fast if index is close..
	if !ok {
		return -1
	}
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
// given name. Returns nil if not found.
func (n *NodeBase) ParentByName(name string) Node {
	if IsRoot(n) {
		return nil
	}
	if n.Par.Name() == name {
		return n.Par
	}
	return n.Par.ParentByName(name)
}

// ParentByType finds parent recursively up hierarchy, by type, and
// returns nil if not found. If embeds is true, then it looks for any
// type that embeds the given type at any level of anonymous embedding.
func (n *NodeBase) ParentByType(t *gti.Type, embeds bool) Node {
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
	return n.Par.ParentByType(t, embeds)
}

// Children:

// HasChildren returns whether this node has any children.
func (n *NodeBase) HasChildren() bool {
	return len(n.Kids) > 0
}

// NumChildren returns the number of children this node has.
func (n *NodeBase) NumChildren() int {
	return len(n.Kids)
}

// NumLifetimeChildren returns the number of children that this node
// has ever had added to it (it is not decremented when a child is removed).
// It is used for unique naming of children.
func (n *NodeBase) NumLifetimeChildren() uint64 {
	return n.numLifetimeChildren
}

// Children returns a pointer to the slice of children of this node.
// The resultant slice can be modified directly (e.g., sort, reorder),
// but new children should be added via New/Add/Insert Child methods on
// Node to ensure proper initialization.
func (n *NodeBase) Children() *Slice {
	return &n.Kids
}

// Child returns the child of this node at the given index and returns nil if
// the index is out of range.
func (n *NodeBase) Child(i int) Node {
	if i >= len(n.Kids) || i < 0 {
		return nil
	}
	return n.Kids[i]
}

// ChildByName returns the first child that has the given name, and nil
// if no such element is found. startIndex arg allows for optimized
// bidirectional find if you have an idea where it might be, which
// can be a key speedup for large lists. If no value is specified for
// startIndex, it starts in the middle, which is a good default.
func (n *NodeBase) ChildByName(name string, startIndex ...int) Node {
	return n.Kids.ElemByName(name, startIndex...)
}

// ChildByType returns the first child that has the given type, and nil
// if not found. If embeds is true, then it also looks for any type that
// embeds the given type at any level of anonymous embedding.
// startIndex arg allows for optimized bidirectional find if you have an
// idea where it might be, which can be a key speedup for large lists. If
// no value is specified for startIndex, it starts in the middle, which is a
// good default.
func (n *NodeBase) ChildByType(t *gti.Type, embeds bool, startIndex ...int) Node {
	return n.Kids.ElemByType(t, embeds, startIndex...)
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
		return n.Nm
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

// find the child on the path
func findPathChild(k Node, child string) (int, bool) {
	if len(child) == 0 {
		return 0, false
	}
	if child[0] == '[' && child[len(child)-1] == ']' {
		idx, err := strconv.Atoi(child[1 : len(child)-1])
		if err != nil {
			return idx, false
		}
		if idx < 0 { // from end
			idx = len(*k.Children()) + idx
		}
		if k.Children().IsValidIndex(idx) != nil {
			return idx, false
		}
		return idx, true
	}
	return k.Children().IndexByName(child, 0)
}

// FindPath returns the node at the given path, starting from this node.
// If this node is not the root, then the path to this node is subtracted
// from the start of the path if present there.
// FindPath only works correctly when names are unique.
// Path has [Node.Name]s separated by / and fields by .
// Node names escape any existing / and . characters to \\ and \,
// There is also support for [idx] index-based access for any given path
// element, for cases when indexes are more useful than names.
// Returns nil if not found.
func (n *NodeBase) FindPath(path string) Node {
	if n.Par != nil { // we are not root..
		myp := n.Path()
		path = strings.TrimPrefix(path, myp)
	}
	curn := n.This()
	pels := strings.Split(strings.Trim(strings.TrimSpace(path), "\""), "/")
	for i, pe := range pels {
		if len(pe) == 0 {
			continue
		}
		if i <= 1 && curn.Name() == UnescapePathName(pe) {
			continue
		}
		if strings.Contains(pe, ".") { // has fields
			fels := strings.Split(pe, ".")
			// find the child first, then the fields
			idx, ok := findPathChild(curn, UnescapePathName(fels[0]))
			if !ok {
				return nil
			}
			curn = (*(curn.Children()))[idx]
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
			idx, ok := findPathChild(curn, UnescapePathName(pe))
			if !ok {
				return nil
			}
			curn = (*(curn.Children()))[idx]
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
func (n *NodeBase) AddChild(kid Node) error {
	if err := ThisCheck(n); err != nil {
		return err
	}
	InitNode(kid)
	n.Kids = append(n.Kids, kid)
	SetParent(kid, n.This()) // key to set new parent before deleting: indicates move instead of delete
	return nil
}

// NewChild creates a new child of the given type and adds it at end
// of children list. The name should be unique among children. If the
// name is unspecified, it defaults to the ID (kebab-case) name of the
// type, plus the [Ki.NumLifetimeChildren] of its parent.
func (n *NodeBase) NewChild(typ *gti.Type, name ...string) Node {
	if err := ThisCheck(n); err != nil {
		return nil
	}
	kid := NewOfType(typ)
	InitNode(kid)
	n.Kids = append(n.Kids, kid)
	if len(name) > 0 {
		kid.SetName(name[0])
	}
	SetParent(kid, n.This())
	return kid
}

// SetChild sets child at given index to be the given item; if it is passed
// a name, then it sets the name of the child as well; just calls Init
// (or InitName) on the child, and SetParent. Names should be unique
// among children.
func (n *NodeBase) SetChild(kid Node, idx int, name ...string) error {
	if err := n.Kids.IsValidIndex(idx); err != nil {
		return err
	}
	if len(name) > 0 {
		kid.InitName(kid, name[0])
	} else {
		InitNode(kid)
	}
	n.Kids[idx] = kid
	SetParent(kid, n.This())
	return nil
}

// InsertChild adds given child at position in children list.
// The kid node is assumed to not be on another tree (see [MoveToParent])
// and the existing name should be unique among children.
func (n *NodeBase) InsertChild(kid Node, at int) error {
	if err := ThisCheck(n); err != nil {
		return err
	}
	InitNode(kid)
	n.Kids.Insert(kid, at)
	SetParent(kid, n.This())
	return nil
}

// InsertNewChild creates a new child of given type and add at position
// in children list. The name should be unique among children. If the
// name is unspecified, it defaults to the ID (kebab-case) name of the
// type, plus the [Ki.NumLifetimeChildren] of its parent.
func (n *NodeBase) InsertNewChild(typ *gti.Type, at int, name ...string) Node {
	if err := ThisCheck(n); err != nil {
		return nil
	}
	kid := NewOfType(typ)
	InitNode(kid)
	n.Kids.Insert(kid, at)
	if len(name) > 0 {
		kid.SetName(name[0])
	}
	SetParent(kid, n.This())
	return kid
}

// SetNChildren ensures that there are exactly n children, deleting any
// extra, and creating any new ones, using NewChild with given type and
// naming according to nameStubX where X is the index of the child.
// If nameStub is not specified, it defaults to the ID (kebab-case)
// name of the type. It returns whether any changes were made to the
// children.
//
// Note that this does not ensure existing children are of given type, or
// change their names, or call UniquifyNames; use ConfigChildren for
// those cases; this function is for simpler cases where a parent uses
// this function consistently to manage children all of the same type.
func (n *NodeBase) SetNChildren(trgn int, typ *gti.Type, nameStub ...string) bool {
	sz := len(n.Kids)
	if trgn == sz {
		return false
	}
	mods := false
	for sz > trgn {
		mods = true
		sz--
		n.DeleteChildAtIndex(sz)
	}
	ns := typ.IDName
	if len(nameStub) > 0 {
		ns = nameStub[0]
	}
	for sz < trgn {
		mods = true
		nm := fmt.Sprintf("%s%d", ns, sz)
		n.InsertNewChild(typ, sz, nm)
		sz++
	}
	return mods
}

// ConfigChildren configures children according to the given list of
// [TypeAndName]s; it attempts to have minimal impact relative to existing
// items that fit the type and name constraints (they are moved into the
// corresponding positions), and any extra children are removed, and new
// ones added, to match the specified config. It is important that names
// are unique. It returns whether any changes were made to the children.
func (n *NodeBase) ConfigChildren(config Config) bool {
	return n.Kids.Config(n.This(), config)
}

// Deleting Children:

// DeleteChildAtIndex deletes child at given index. It returns false
// if there is no child at the given index.
func (n *NodeBase) DeleteChildAtIndex(idx int) bool {
	child := n.Child(idx)
	if child == nil {
		return false
	}
	n.Kids.DeleteAtIndex(idx)
	child.Destroy()
	return true
}

// DeleteChild deletes the given child node, returning false if
// it can not find it.
func (n *NodeBase) DeleteChild(child Node) bool {
	if child == nil {
		return false
	}
	idx, ok := n.Kids.IndexOf(child)
	if !ok {
		return false
	}
	return n.DeleteChildAtIndex(idx)
}

// DeleteChildByName deletes child node by name, returning false
// if it can not find it.
func (n *NodeBase) DeleteChildByName(name string) bool {
	idx, ok := n.Kids.IndexByName(name)
	if !ok {
		return false
	}
	return n.DeleteChildAtIndex(idx)
}

// DeleteChildren deletes all children nodes.
func (n *NodeBase) DeleteChildren() {
	kids := n.Kids
	n.Kids = n.Kids[:0] // preserves capacity of list
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
		n.Par.DeleteChild(n.This())
	}
}

// Destroy recursively deletes and destroys all children and
// their children's children, etc.
func (n *NodeBase) Destroy() {
	if n.This() == nil { // already destroyed
		return
	}
	n.DeleteChildren()
	n.this = nil
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

// Properties returns the key-value properties set for this node.
func (n *NodeBase) Properties() map[string]any {
	return n.Props
}

// SetProperty sets given the given property to the given value.
func (n *NodeBase) SetProperty(key string, value any) {
	if n.Props == nil {
		n.Props = map[string]any{}
	}
	n.Props[key] = value
}

// Property returns the property value for the given key.
// It returns nil if it doesn't exist.
func (n *NodeBase) Property(key string) any {
	return n.Props[key]
}

// DeleteProperty deletes the property with the given key.
func (n *NodeBase) DeleteProperty(key string) {
	if n.Props == nil {
		return
	}
	delete(n.Props, key)
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

// walking strategy: https://stackoverflow.com/questions/5278580/non-recursive-depth-first-search-algorithm

// WalkDown calls function on this node (MeFirst) and then iterates
// in a depth-first manner over all the children.
// The [WalkPreNode] method is called for every node, after the given function,
// which e.g., enables nodes to also traverse additional Ki Trees (e.g., Fields).
// The node traversal is non-recursive and uses locally-allocated state -- safe
// for concurrent calling (modulo conflict management in function call itself).
// Function calls are sequential all in current go routine.
// If fun returns false then any further traversal of that branch of the tree is
// aborted, but other branches continue -- i.e., if fun on current node
// returns false, children are not processed further.
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
			n.This().NodeWalkDown(fun)
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

// NodeWalkDown is called for every node during WalkPre with the function
// passed to WalkPre.  This e.g., enables nodes to also traverse additional
// Ki Trees (e.g., Fields).
func (n *NodeBase) NodeWalkDown(fun func(n Node) bool) {}

// WalkDownPost iterates in a depth-first manner over the children, calling
// doChildTestFunc on each node to test if processing should proceed (if it returns
// false then that branch of the tree is not further processed), and then
// calls given fun function after all of a node's children.
// have been iterated over ("Me Last").
// The node traversal is non-recursive and uses locally-allocated state -- safe
// for concurrent calling (modulo conflict management in function call itself).
// Function calls are sequential all in current go routine.
// The level var tracks overall depth in the tree.
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

// Note: it does not appear that there is a good recursive BFS search strategy
// https://herringtondarkholme.github.io/2014/02/17/generator/
// https://stackoverflow.com/questions/2549541/performing-breadth-first-search-recursively/2549825#2549825

// WalkDownBreadth calls function on all children in breadth-first order
// using the standard queue strategy.  This depends on and updates the
// Depth parameter of the node.  If fun returns false then any further
// traversal of that branch of the tree is aborted, but other branches continue.
func (n *NodeBase) WalkDownBreadth(fun func(n Node) bool) {
	start := n.This()

	level := 0
	start.AsTreeNode().depth = level
	queue := make([]Node, 1)
	queue[0] = start

	for {
		if len(queue) == 0 {
			break
		}
		cur := queue[0]
		depth := cur.AsTreeNode().depth
		queue = queue[1:]

		if cur.This() != nil && fun(cur) { // false return means don't proceed
			for _, cn := range *cur.Children() {
				if cn != nil && cn.This() != nil {
					cn.AsTreeNode().depth = depth + 1
					queue = append(queue, cn)
				}
			}
		}
	}
}

//////////////////////////////////////////////////////////////////////////
//  Deep Copy / Clone

// note: we use the copy from direction as the receiver is modified whereas the
// from is not and assignment is typically in same direction

// CopyFrom another Ki node.  It is essential that source has Unique names!
// The Ki copy function recreates the entire tree in the copy, duplicating
// children etc, copying Properties too.  It is very efficient by
// using the ConfigChildren method which attempts to preserve any existing
// nodes in the destination if they have the same name and type -- so a
// copy from a source to a target that only differ minimally will be
// minimally destructive.  Only copies to same types are supported.
// Signal connections are NOT copied.  No other Ki pointers are copied,
// and the field tag copier:"-" can be added for any other fields that
// should not be copied (unexported, lower-case fields are not copyable).
func (n *NodeBase) CopyFrom(frm Node) error {
	if frm == nil {
		err := fmt.Errorf("tree.NodeBase CopyFrom into %v: nil 'from' source", n)
		log.Println(err)
		return err
	}
	CopyFromRaw(n.This(), frm)
	return nil
}

// Clone creates and returns a deep copy of the tree from this node down.
// Any pointers within the cloned tree will correctly point within the new
// cloned tree (see Copy info).
func (n *NodeBase) Clone() Node {
	nc := NewOfType(n.This().NodeType())
	nc.InitName(nc, n.Nm)
	nc.CopyFrom(n.This())
	return nc
}

// CopyFromRaw performs a raw copy that just does the deep copy of the
// bits and doesn't do anything with pointers.
func CopyFromRaw(n, from Node) {
	n.Children().ConfigCopy(n.This(), *from.Children())
	maps.Copy(n.Properties(), from.Properties())

	n.This().CopyFieldsFrom(from)
	for i, kid := range *n.Children() {
		fmk := from.Child(i)
		CopyFromRaw(kid, fmk)
	}
}

// CopyFieldsFrom is the base implementation of [Node.CopyFieldsFrom] that copies the fields
// of the [NodeBase.This] from the fields of the given [Node.This], recursively following anonymous
// embedded structs. It uses [copier.Copy] for this. It ignores any fields with a `copier:"-"`
// struct tag. Other implementations of [Node.CopyFieldsFrom] should call this method first and
// then only do manual handling of specific fields that can not be automatically copied.
func (n *NodeBase) CopyFieldsFrom(from Node) {
	err := copier.CopyWithOption(n.This(), from.This(), copier.Option{CaseSensitive: true, DeepCopy: true})
	if err != nil {
		slog.Error("tree.NodeBase.CopyFieldsFrom", "err", err)
	}
}

// OnInit is a placeholder implementation of
// [Node.OnInit] that does nothing.
func (n *NodeBase) OnInit() {}

// OnAdd is a placeholder implementation of
// [Node.OnAdd] that does nothing.
func (n *NodeBase) OnAdd() {}

// OnChildAdded is a placeholder implementation of
// [Node.OnChildAdded] that does nothing.
func (n *NodeBase) OnChildAdded(child Node) {}
