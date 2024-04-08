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
	Nm string `copier:"-" set:"-" label:"Name"`

	// Flags are bit flags for internal node state, which can be extended using the enums package.
	Flags Flags `tableview:"-" copier:"-" json:"-" xml:"-" set:"-" max-width:"80" height:"3"`

	// Props is a property map for arbitrary key-value properties.
	Props map[string]any `tableview:"-" xml:"-" copier:"-" set:"-" label:"Properties"`

	// Par is the parent of this node, which is set automatically when this node is added as a child of a parent.
	Par Node `tableview:"-" copier:"-" json:"-" xml:"-" view:"-" set:"-" label:"Parent"`

	// Kids is the list of children of this node. All of them are set to have this node
	// as their parent. They can be reordered, but you should generally use Ki Node methods
	// to Add / Delete to ensure proper usage.
	Kids Slice `tableview:"-" copier:"-" set:"-" label:"Children"`

	// Ths is a pointer to ourselves as a Ki. It can always be used to extract the true underlying type
	// of an object when [Node] is embedded in other structs; function receivers do not have this ability
	// so this is necessary. This is set to nil when deleted. Typically use [Ki.This] convenience accessor
	// which protects against concurrent access.
	Ths Node `copier:"-" json:"-" xml:"-" view:"-" set:"-"`

	// NumLifetimeKids is the number of children that have ever been added to this node, which is used for automatic unique naming.
	NumLifetimeKids uint64 `copier:"-" json:"-" xml:"-" view:"-" set:"-"`

	// index is the last value of our index, which is used as a starting point for finding us in our parent next time.
	// It is not guaranteed to be accurate; use the [Ki.IndexInParent] method.
	index int `copier:"-" json:"-" xml:"-" view:"-" set:"-"`

	// depth is an optional depth parameter of this node, which is only valid during specific contexts, not generally.
	// For example, it is used in the WalkBreadth function
	depth int `copier:"-" json:"-" xml:"-" view:"-" set:"-"`
}

// check implementation of [Node] interface
var _ = Node(&NodeBase{})

// StringElideMax is the Max width for [NodeBase.String] path printout of Ki nodes.
var StringElideMax = 38

//////////////////////////////////////////////////////////////////////////
//  fmt.Stringer

// String implements the fmt.stringer interface -- returns the Path of the node
func (n *NodeBase) String() string {
	return elide.Middle(n.This().Path(), StringElideMax)
}

//////////////////////////////////////////////////////////////////////////
//  Basic Ki fields

// This returns the Ki interface that guarantees access to the Ki
// interface in a way that always reveals the underlying type
// (e.g., in reflect calls).  Returns nil if node is nil,
// has been destroyed, or is improperly constructed.
func (n *NodeBase) This() Node {
	if n == nil {
		return nil
	}
	return n.Ths
}

// AsTreeNode returns the *tree.NodeBase base type for this node.
func (n *NodeBase) AsTreeNode() *NodeBase {
	return n
}

// InitName initializes this node to given actual object as a Ki interface
// and sets its name. The names should be unique among children of a node.
// This is needed for root nodes -- automatically done for other nodes
// when they are added to the Ki tree. If the name is unspecified, it
// defaults to the ID (kebab-case) name of the type.
// Even though this is a method and gets the method receiver, it needs
// an "external" version of itself passed as the first arg, from which
// the proper Ki interface pointer will be obtained.  This is the only
// way to get virtual functional calling to work within the Go language.
func (n *NodeBase) InitName(k Node, name ...string) {
	InitNode(k)
	if len(name) > 0 {
		n.SetName(name[0])
	}
}

// BaseType returns the base node type for all elements within this tree.
// Used e.g., for determining what types of children can be created.
func (n *NodeBase) BaseType() *gti.Type {
	return NodeBaseType
}

// Name returns the user-defined name of the object (Node.Nm),
// for finding elements, generating paths, IO, etc.
func (n *NodeBase) Name() string {
	return n.Nm
}

// SetName sets the name of this node.
// Names should generally be unique across children of each node.
// See Unique* functions to check / fix.
// If node requires non-unique names, add a separate Label field.
func (n *NodeBase) SetName(name string) {
	n.Nm = name
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

//////////////////////////////////////////////////////////////////////////
//  Parents

// Parent returns the parent of this Ki (Node.Par) -- Ki has strict
// one-parent, no-cycles structure -- see SetParent.
func (n *NodeBase) Parent() Node {
	return n.Par
}

// IndexInParent returns our index within our parent object. It caches the
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
// hierarchy, returning level above current node that the parent was
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
// given name -- returns nil if not found.
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

//////////////////////////////////////////////////////////////////////////
//  Children

// HasChildren tests whether this node has children (i.e., non-terminal).
func (n *NodeBase) HasChildren() bool {
	return len(n.Kids) > 0
}

// NumChildren returns the number of children of this node.
func (n *NodeBase) NumChildren() int {
	return len(n.Kids)
}

func (n *NodeBase) NumLifetimeChildren() uint64 {
	return n.NumLifetimeKids
}

// Children returns a pointer to the slice of children (Node.Kids) -- use
// methods on [tree.Slice] for further ways to access (ByName, ByType, etc).
// Slice can be modified directly (e.g., sort, reorder) but Add* / Delete*
// methods on parent node should be used to ensure proper tracking.
func (n *NodeBase) Children() *Slice {
	return &n.Kids
}

// Child returns the child at given index and returns nil if
// the index is out of range.
func (n *NodeBase) Child(idx int) Node {
	if idx >= len(n.Kids) || idx < 0 {
		return nil
	}
	return n.Kids[idx]
}

// ChildByName returns the first element that has given name, and nil
// if no such element is found. startIndex arg allows for optimized
// bidirectional find if you have an idea where it might be, which
// can be a key speedup for large lists. If no value is specified for
// startIndex, it starts in the middle, which is a good default.
func (n *NodeBase) ChildByName(name string, startIndex ...int) Node {
	return n.Kids.ElemByName(name, startIndex...)
}

// ChildByType returns the first element that has the given type, and nil
// if not found. If embeds is true, then it also looks for any type that
// embeds the given type at any level of anonymous embedding.
// startIndex arg allows for optimized bidirectional find if you have an
// idea where it might be, which can be a key speedup for large lists. If
// no value is specified for startIndex, it starts in the middle, which is a
// good default.
func (n *NodeBase) ChildByType(t *gti.Type, embeds bool, startIndex ...int) Node {
	return n.Kids.ElemByType(t, embeds, startIndex...)
}

//////////////////////////////////////////////////////////////////////////
//  Paths

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

// Path returns path to this node from the tree root, using node Names
// separated by / and fields by .
// Node names escape any existing / and . characters to \\ and \,
// Path is only valid when child names are unique (see Unique* functions)
func (n *NodeBase) Path() string {
	if n.Par != nil {
		if n.Is(Field) {
			return n.Par.Path() + "." + EscapePathName(n.Nm)
		}
		return n.Par.Path() + "/" + EscapePathName(n.Nm)
	}
	return "/" + EscapePathName(n.Nm)
}

// PathFrom returns path to this node from given parent node, using
// node Names separated by / and fields by .
// Node names escape any existing / and . characters to \\ and \,
// Path is only valid for finding items when child names are unique
// (see Unique* functions). The paths that it returns exclude the
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

// FindPath returns Ki object at given path, starting from this node
// (e.g., the root).  If this node is not the root, then the path
// to this node is subtracted from the start of the path if present there.
// FindPath only works correctly when names are unique.
// Path has node Names separated by / and fields by .
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

func (n *NodeBase) FieldByName(field string) (Node, error) {
	return nil, errors.New("tree.FieldByName: no tree fields defined for this node")
}

//////////////////////////////////////////////////////////////////////////
//  Adding, Inserting Children

// AddChild adds given child at end of children list.
// The kid node is assumed to not be on another tree (see MoveToParent)
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
// type, plus the [Node.NumLifetimeChildren] of its parent.
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
// The kid node is assumed to not be on another tree (see MoveToParent)
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
// type, plus the [Node.NumLifetimeChildren] of its parent.
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
// name of the type. It returns whether any changes were made to the children.
//
// Note that this does not ensure existing children are of given type, or
// change their names, or call UniquifyNames -- use ConfigChildren for
// those cases -- this function is for simpler cases where a parent uses
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

// ConfigChildren configures children according to given list of
// type-and-name's -- attempts to have minimal impact relative to existing
// items that fit the type and name constraints (they are moved into the
// corresponding positions), and any extra children are removed, and new
// ones added, to match the specified config. It is important that names
// are unique! It returns whether any changes were made to the children.
func (n *NodeBase) ConfigChildren(config Config) bool {
	return n.Kids.Config(n.This(), config)
}

//////////////////////////////////////////////////////////////////////////
//  Deleting Children

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
	if n.This() == nil { // already dead!
		return
	}
	n.DeleteChildren() // delete and destroy all my children
	n.Ths = nil        // last gasp: lose our own sense of self..
}

//////////////////////////////////////////////////////////////////////////
//  Flags

// Is checks if flag is set, using atomic, safe for concurrent access
func (n *NodeBase) Is(f enums.BitFlag) bool {
	return n.Flags.HasFlag(f)
}

// SetFlag sets the given flag(s) to given state
// using atomic, safe for concurrent access
func (n *NodeBase) SetFlag(on bool, f ...enums.BitFlag) {
	n.Flags.SetFlag(on, f...)
}

// FlagType is the base implementation of [Node.FlagType] that returns a
// value of type [Flags].
func (n *NodeBase) FlagType() enums.BitFlagSetter {
	return &n.Flags
}

//////////////////////////////////////////////////////////////////////////
//  Property interface with inheritance -- nodes can inherit properties from parents

// Properties (Node.Properties) tell the Cogent Core GUI or other frameworks operating
// on Trees about special features of each node -- functions below support
// inheritance up Tree.
func (n *NodeBase) Properties() map[string]any {
	return n.Props
}

// SetProperty sets given property key to value val.
// initializes property map if nil.
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

// DeleteProperty deletes property key on this node.
func (n *NodeBase) DeleteProperty(key string) {
	if n.Props == nil {
		return
	}
	delete(n.Props, key)
}

//////////////////////////////////////////////////////////////////////////
//  Tree walking and state updating

// WalkUp calls function on given node and all the way up to its parents,
// and so on -- sequentially all in current go routine (generally
// necessary for going up, which is typically quite fast anyway) -- level
// is incremented after each step (starts at 0, goes up), and passed to
// function -- returns false if fun aborts with false, else true.
func (n *NodeBase) WalkUp(fun func(k Node) bool) bool {
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

// WalkUpParent calls function on parent of node and all the way up to its
// parents, and so on -- sequentially all in current go routine (generally
// necessary for going up, which is typically quite fast anyway) -- level
// is incremented after each step (starts at 0, goes up), and passed to
// function -- returns false if fun aborts with false, else true.
func (n *NodeBase) WalkUpParent(fun func(k Node) bool) bool {
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

////////////////////////////////////////////////////////////////////////
// FuncDown -- Traversal records

// TravMap is a map for recording the traversal of nodes
type TravMap map[Node]int

// Start is called at start of traversal
func (tm TravMap) Start(k Node) {
	tm[k] = -1
}

// End deletes node once done at end of traversal
func (tm TravMap) End(k Node) {
	delete(tm, k)
}

// Set updates traversal state
func (tm TravMap) Set(k Node, curChild int) {
	tm[k] = curChild
}

// Get retrieves current traversal state
func (tm TravMap) Get(k Node) int {
	return tm[k]
}

// strategy -- same as used in TreeView:
// https://stackoverflow.com/questions/5278580/non-recursive-depth-first-search-algorithm

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
func (n *NodeBase) WalkDown(fun func(Node) bool) {
	if n.This() == nil {
		return
	}
	tm := TravMap{} // not significantly faster to pre-allocate larger size
	start := n.This()
	cur := start
	tm.Start(cur)
outer:
	for {
		if cur.This() != nil && fun(cur) { // false return means stop
			n.This().NodeWalkDown(fun)
			if cur.HasChildren() {
				tm.Set(cur, 0) // 0 for no fields
				nxt := cur.Child(0)
				if nxt != nil && nxt.This() != nil {
					cur = nxt.This()
					tm.Start(cur)
					continue
				}
			}
		} else {
			tm.Set(cur, cur.NumChildren())
		}
		// if we get here, we're in the ascent branch -- move to the right and then up
		for {
			curChild := tm.Get(cur)
			if (curChild + 1) < cur.NumChildren() {
				curChild++
				tm.Set(cur, curChild)
				nxt := cur.Child(curChild)
				if nxt != nil && nxt.This() != nil {
					cur = nxt.This()
					tm.Start(cur)
					continue outer
				}
				continue
			}
			tm.End(cur)
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
func (n *NodeBase) NodeWalkDown(fun func(Node) bool) {}

// WalkPreLevel calls function on this node (MeFirst) and then iterates
// in a depth-first manner over all the children.
// This version has a level var that tracks overall depth in the tree.
// If fun returns false then any further traversal of that branch of the tree is
// aborted, but other branches continue -- i.e., if fun on current node
// returns false, children are not processed further.
// Because WalkPreLevel is not used within Ki itself, it does not have its
// own version of WalkPreNode -- that can be handled within the closure.
func (n *NodeBase) WalkPreLevel(fun func(k Node, level int) bool) {
	if n.This() == nil {
		return
	}
	level := 0
	tm := TravMap{} // not significantly faster to pre-allocate larger size
	start := n.This()
	cur := start
	tm.Start(cur)
outer:
	for {
		if cur.This() != nil && fun(cur, level) { // false return means stop
			level++ // this is the descent branch
			if cur.HasChildren() {
				tm.Set(cur, 0) // 0 for no fields
				nxt := cur.Child(0)
				if nxt != nil && nxt.This() != nil {
					cur = nxt.This()
					tm.Start(cur)
					continue
				}
			}
		} else {
			tm.Set(cur, cur.NumChildren())
			level++ // we will pop back up out of this next
		}
		// if we get here, we're in the ascent branch -- move to the right and then up
		for {
			curChild := tm.Get(cur)
			if (curChild + 1) < cur.NumChildren() {
				curChild++
				tm.Set(cur, curChild)
				nxt := cur.Child(curChild)
				if nxt != nil && nxt.This() != nil {
					cur = nxt.This()
					tm.Start(cur)
					continue outer
				}
				continue
			}
			tm.End(cur)
			// couldn't go right, move up..
			if cur == start {
				break outer // done!
			}
			level--
			parent := cur.Parent()
			if parent == nil || parent == cur { // shouldn't happen, but does..
				// fmt.Printf("nil / cur parent %v\n", par)
				break outer
			}
			cur = parent
		}
	}
}

// WalkPost iterates in a depth-first manner over the children, calling
// doChildTestFunc on each node to test if processing should proceed (if it returns
// false then that branch of the tree is not further processed), and then
// calls given fun function after all of a node's children.
// have been iterated over ("Me Last").
// The node traversal is non-recursive and uses locally-allocated state -- safe
// for concurrent calling (modulo conflict management in function call itself).
// Function calls are sequential all in current go routine.
// The level var tracks overall depth in the tree.
func (n *NodeBase) WalkPost(doChildTestFunc func(Node) bool, fun func(Node) bool) {
	if n.This() == nil {
		return
	}
	tm := TravMap{} // not significantly faster to pre-allocate larger size
	start := n.This()
	cur := start
	tm.Start(cur)
outer:
	for {
		if cur.This() != nil && doChildTestFunc(cur) { // false return means stop
			if cur.HasChildren() {
				tm.Set(cur, 0) // 0 for no fields
				nxt := cur.Child(0)
				if nxt != nil && nxt.This() != nil {
					cur = nxt.This()
					tm.Set(cur, -1)
					continue
				}
			}
		} else {
			tm.Set(cur, cur.NumChildren())
		}
		// if we get here, we're in the ascent branch -- move to the right and then up
		for {
			curChild := tm.Get(cur)
			if (curChild + 1) < cur.NumChildren() {
				curChild++
				tm.Set(cur, curChild)
				nxt := cur.Child(curChild)
				if nxt != nil && nxt.This() != nil {
					cur = nxt.This()
					tm.Start(cur)
					continue outer
				}
				continue
			}
			fun(cur) // now we call the function, last..
			// couldn't go right, move up..
			tm.End(cur)
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

// WalkBreadth calls function on all children in breadth-first order
// using the standard queue strategy.  This depends on and updates the
// Depth parameter of the node.  If fun returns false then any further
// traversal of that branch of the tree is aborted, but other branches continue.
func (n *NodeBase) WalkBreadth(fun func(k Node) bool) {
	start := n.This()

	level := 0
	SetDepth(start, level)
	queue := make([]Node, 1)
	queue[0] = start

	for {
		if len(queue) == 0 {
			break
		}
		cur := queue[0]
		depth := Depth(cur)
		queue = queue[1:]

		if cur.This() != nil && fun(cur) { // false return means don't proceed
			for _, k := range *cur.Children() {
				if k != nil && k.This() != nil {
					SetDepth(k, depth+1)
					queue = append(queue, k)
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
