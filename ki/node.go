// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ki

//go:generate core generate
//go:generate core generate ./testdata

import (
	"errors"
	"fmt"
	"log/slog"
	"strconv"

	"log"
	"strings"

	"cogentcore.org/core/enums"
	"cogentcore.org/core/glop/elide"
	"cogentcore.org/core/gti"
	"github.com/jinzhu/copier"
)

// The Node struct implements the [Ki] interface and provides the core functionality
// for the Cogent Core tree system. You can use the Node as an embedded struct or as a struct
// field; the embedded version supports full JSON saving and loading. All types that
// implement the [Ki] interface will automatically be added to gti in `core generate`, which
// is required for various pieces of core functionality.
type Node struct {

	// Nm is the user-supplied name of this node, which can be empty and/or non-unique.
	Nm string `copier:"-" set:"-" label:"Name"`

	// Flags are bit flags for internal node state, which can be extended using the enums package.
	Flags Flags `tableview:"-" copier:"-" json:"-" xml:"-" set:"-" max-width:"80" height:"3"`

	// Props is a property map for arbitrary extensible properties.
	Props Props `tableview:"-" xml:"-" copier:"-" set:"-" label:"Properties"`

	// Par is the parent of this node, which is set automatically when this node is added as a child of a parent.
	Par Ki `tableview:"-" copier:"-" json:"-" xml:"-" view:"-" set:"-" label:"Parent"`

	// Kids is the list of children of this node. All of them are set to have this node
	// as their parent. They can be reordered, but you should generally use Ki Node methods
	// to Add / Delete to ensure proper usage.
	Kids Slice `tableview:"-" copier:"-" set:"-" label:"Children"`

	// Ths is a pointer to ourselves as a Ki. It can always be used to extract the true underlying type
	// of an object when [Node] is embedded in other structs; function receivers do not have this ability
	// so this is necessary. This is set to nil when deleted. Typically use [Ki.This] convenience accessor
	// which protects against concurrent access.
	Ths Ki `copier:"-" json:"-" xml:"-" view:"-" set:"-"`

	// NumLifetimeKids is the number of children that have ever been added to this node, which is used for automatic unique naming.
	NumLifetimeKids uint64 `copier:"-" json:"-" xml:"-" view:"-" set:"-"`

	// index is the last value of our index, which is used as a starting point for finding us in our parent next time.
	// It is not guaranteed to be accurate; use the [Ki.IndexInParent] method.
	index int `copier:"-" json:"-" xml:"-" view:"-" set:"-"`

	// depth is an optional depth parameter of this node, which is only valid during specific contexts, not generally.
	// For example, it is used in the WalkBreadth function
	depth int `copier:"-" json:"-" xml:"-" view:"-" set:"-"`
}

// check implementation of [Ki] interface
var _ = Ki(&Node{})

// StringElideMax is the Max width for [Node.String] path printout of Ki nodes.
var StringElideMax = 38

//////////////////////////////////////////////////////////////////////////
//  fmt.Stringer

// String implements the fmt.stringer interface -- returns the Path of the node
func (n *Node) String() string {
	return elide.Middle(n.This().Path(), StringElideMax)
}

//////////////////////////////////////////////////////////////////////////
//  Basic Ki fields

// This returns the Ki interface that guarantees access to the Ki
// interface in a way that always reveals the underlying type
// (e.g., in reflect calls).  Returns nil if node is nil,
// has been destroyed, or is improperly constructed.
func (n *Node) This() Ki {
	if n == nil {
		return nil
	}
	return n.Ths
}

// AsNode returns the *ki.Node base type for this node.
func (n *Node) AsKi() *Node {
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
func (n *Node) InitName(k Ki, name ...string) {
	InitNode(k)
	if len(name) > 0 {
		n.SetName(name[0])
	}
}

// BaseType returns the base node type for all elements within this tree.
// Used e.g., for determining what types of children can be created.
func (n *Node) BaseType() *gti.Type {
	return NodeType
}

// Name returns the user-defined name of the object (Node.Nm),
// for finding elements, generating paths, IO, etc.
func (n *Node) Name() string {
	return n.Nm
}

// SetName sets the name of this node.
// Names should generally be unique across children of each node.
// See Unique* functions to check / fix.
// If node requires non-unique names, add a separate Label field.
func (n *Node) SetName(name string) {
	n.Nm = name
}

// OnInit is a placeholder implementation of
// [Ki.OnInit] that does nothing.
func (n *Node) OnInit() {}

// OnAdd is a placeholder implementation of
// [Ki.OnAdd] that does nothing.
func (n *Node) OnAdd() {}

// OnChildAdded is a placeholder implementation of
// [Ki.OnChildAdded] that does nothing.
func (n *Node) OnChildAdded(child Ki) {}

//////////////////////////////////////////////////////////////////////////
//  Parents

// Parent returns the parent of this Ki (Node.Par) -- Ki has strict
// one-parent, no-cycles structure -- see SetParent.
func (n *Node) Parent() Ki {
	return n.Par
}

// IndexInParent returns our index within our parent object. It caches the
// last value and uses that for an optimized search so subsequent calls
// are typically quite fast. Returns -1 if we don't have a parent.
func (n *Node) IndexInParent() int {
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
func (n *Node) ParentLevel(par Ki) int {
	parLev := -1
	level := 0
	n.WalkUpParent(func(k Ki) bool {
		if k == par {
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
func (n *Node) ParentByName(name string) Ki {
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
func (n *Node) ParentByType(t *gti.Type, embeds bool) Ki {
	if IsRoot(n) {
		return nil
	}
	if embeds {
		if n.Par.KiType().HasEmbed(t) {
			return n.Par
		}
	} else {
		if n.Par.KiType() == t {
			return n.Par
		}
	}
	return n.Par.ParentByType(t, embeds)
}

//////////////////////////////////////////////////////////////////////////
//  Children

// HasChildren tests whether this node has children (i.e., non-terminal).
func (n *Node) HasChildren() bool {
	return len(n.Kids) > 0
}

// NumChildren returns the number of children of this node.
func (n *Node) NumChildren() int {
	return len(n.Kids)
}

func (n *Node) NumLifetimeChildren() uint64 {
	return n.NumLifetimeKids
}

// Children returns a pointer to the slice of children (Node.Kids) -- use
// methods on ki.Slice for further ways to access (ByName, ByType, etc).
// Slice can be modified directly (e.g., sort, reorder) but Add* / Delete*
// methods on parent node should be used to ensure proper tracking.
func (n *Node) Children() *Slice {
	return &n.Kids
}

// Child returns the child at given index and returns nil if
// the index is out of range.
func (n *Node) Child(idx int) Ki {
	if idx >= len(n.Kids) || idx < 0 {
		return nil
	}
	return n.Kids[idx]
}

// ChildByName returns the first element that has given name, and nil
// if no such element is found. startIdx arg allows for optimized
// bidirectional find if you have an idea where it might be, which
// can be a key speedup for large lists. If no value is specified for
// startIdx, it starts in the middle, which is a good default.
func (n *Node) ChildByName(name string, startIdx ...int) Ki {
	return n.Kids.ElemByName(name, startIdx...)
}

// ChildByType returns the first element that has the given type, and nil
// if not found. If embeds is true, then it also looks for any type that
// embeds the given type at any level of anonymous embedding.
// startIdx arg allows for optimized bidirectional find if you have an
// idea where it might be, which can be a key speedup for large lists. If
// no value is specified for startIdx, it starts in the middle, which is a
// good default.
func (n *Node) ChildByType(t *gti.Type, embeds bool, startIdx ...int) Ki {
	return n.Kids.ElemByType(t, embeds, startIdx...)
}

//////////////////////////////////////////////////////////////////////////
//  Paths

// TODO: is this the best way to escape paths?

// EscapePathName returns a name that replaces any path delimiter symbols
// . or / with \, and \\ escaped versions.
func EscapePathName(name string) string {
	return strings.Replace(strings.Replace(name, ".", `\,`, -1), "/", `\\`, -1)
}

// UnescapePathName returns a name that replaces any escaped path delimiter symbols
// \, or \\ with . and / unescaped versions.
func UnescapePathName(name string) string {
	return strings.Replace(strings.Replace(name, `\,`, ".", -1), `\\`, "/", -1)
}

// Path returns path to this node from the tree root, using node Names
// separated by / and fields by .
// Node names escape any existing / and . characters to \\ and \,
// Path is only valid when child names are unique (see Unique* functions)
func (n *Node) Path() string {
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
// automatically gets the [Ki.This] version of the given parent,
// so a base type can be passed in without manually calling [Ki.This].
func (n *Node) PathFrom(par Ki) string {
	// critical to get "This"
	par = par.This()
	// we bail a level below the parent so it isn't in the path
	if n.Par == nil || n.Par == par {
		return n.Nm
	}
	ppath := ""
	if n.Par == par {
		ppath = "/" + EscapePathName(par.Name())
	} else {
		ppath = n.Par.PathFrom(par)
	}
	if n.Is(Field) {
		return ppath + "." + EscapePathName(n.Nm)
	}
	return ppath + "/" + EscapePathName(n.Nm)

}

// find the child on the path
func findPathChild(k Ki, child string) (int, bool) {
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
func (n *Node) FindPath(path string) Ki {
	if n.Par != nil { // we are not root..
		myp := n.Path()
		path = strings.TrimPrefix(path, myp)
	}
	curn := Ki(n)
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
					slog.Debug("ki.FindPath: %v", err)
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

func (n *Node) FieldByName(field string) (Ki, error) {
	return nil, errors.New("ki.FieldByName: no Ki fields defined for this node")
}

//////////////////////////////////////////////////////////////////////////
//  Adding, Inserting Children

// AddChild adds given child at end of children list.
// The kid node is assumed to not be on another tree (see MoveToParent)
// and the existing name should be unique among children.
func (n *Node) AddChild(kid Ki) error {
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
func (n *Node) NewChild(typ *gti.Type, name ...string) Ki {
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
func (n *Node) SetChild(kid Ki, idx int, name ...string) error {
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
func (n *Node) InsertChild(kid Ki, at int) error {
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
func (n *Node) InsertNewChild(typ *gti.Type, at int, name ...string) Ki {
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
func (n *Node) SetNChildren(trgn int, typ *gti.Type, nameStub ...string) bool {
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
func (n *Node) ConfigChildren(config Config) bool {
	return n.Kids.Config(n.This(), config)
}

//////////////////////////////////////////////////////////////////////////
//  Deleting Children

// DeleteChildAtIndex deletes child at given index. It returns false
// if there is no child at the given index.
func (n *Node) DeleteChildAtIndex(idx int) bool {
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
func (n *Node) DeleteChild(child Ki) bool {
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
func (n *Node) DeleteChildByName(name string) bool {
	idx, ok := n.Kids.IndexByName(name)
	if !ok {
		return false
	}
	return n.DeleteChildAtIndex(idx)
}

// DeleteChildren deletes all children nodes.
func (n *Node) DeleteChildren() {
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
func (n *Node) Delete() {
	if n.Par == nil {
		n.This().Destroy()
	} else {
		n.Par.DeleteChild(n.This())
	}
}

// Destroy recursively deletes and destroys all children and
// their children's children, etc.
func (n *Node) Destroy() {
	if n.This() == nil { // already dead!
		return
	}
	n.DeleteChildren() // delete and destroy all my children
	n.Ths = nil        // last gasp: lose our own sense of self..
}

//////////////////////////////////////////////////////////////////////////
//  Flags

// Is checks if flag is set, using atomic, safe for concurrent access
func (n *Node) Is(f enums.BitFlag) bool {
	return n.Flags.HasFlag(f)
}

// SetFlag sets the given flag(s) to given state
// using atomic, safe for concurrent access
func (n *Node) SetFlag(on bool, f ...enums.BitFlag) {
	n.Flags.SetFlag(on, f...)
}

// FlagType is the base implementation of [Ki.FlagType] that returns a
// value of type [Flags].
func (n *Node) FlagType() enums.BitFlagSetter {
	return &n.Flags
}

//////////////////////////////////////////////////////////////////////////
//  Property interface with inheritance -- nodes can inherit props from parents

// Properties (Node.Props) tell the Cogent Core GUI or other frameworks operating
// on Trees about special features of each node -- functions below support
// inheritance up Tree.
func (n *Node) Properties() *Props {
	return &n.Props
}

// SetProp sets given property key to value val.
// initializes property map if nil.
func (n *Node) SetProp(key string, val any) {
	if n.Props == nil {
		n.Props = make(Props)
	}
	n.Props[key] = val
}

// SetProps sets a whole set of properties
func (n *Node) SetProps(props Props) {
	if n.Props == nil {
		n.Props = make(Props, len(props))
	}
	for key, val := range props {
		n.Props[key] = val
	}
}

// Prop returns the property value for the given key.
// It returns nil if it doesn't exist.
func (n *Node) Prop(key string) any {
	return n.Props[key]
}

// PropInherit gets property value from key with options for inheriting
// property from parents.  If inherit, then checks all parents.
// Returns false if not set anywhere.
func (n *Node) PropInherit(key string, inherit bool) (any, bool) {
	// pr := prof.Start("PropInherit")
	// defer pr.End()
	v, ok := n.Props[key]
	if ok {
		return v, ok
	}
	if inherit && n.Par != nil {
		v, ok = n.Par.PropInherit(key, inherit)
		if ok {
			return v, ok
		}
	}
	return nil, false
}

// DeleteProp deletes property key on this node.
func (n *Node) DeleteProp(key string) {
	if n.Props == nil {
		return
	}
	delete(n.Props, key)
}

// PropTag returns the name to look for in type properties, for types
// that are valid options for values that can be set in Props.  For example
// in Cogent Core, it is "style-props" which is then set for all types that can
// be used in a style (colors, enum options, etc)
func (n *Node) PropTag() string {
	return ""
}

//////////////////////////////////////////////////////////////////////////
//  Tree walking and state updating

// WalkUp calls function on given node and all the way up to its parents,
// and so on -- sequentially all in current go routine (generally
// necessary for going up, which is typically quite fast anyway) -- level
// is incremented after each step (starts at 0, goes up), and passed to
// function -- returns false if fun aborts with false, else true.
func (n *Node) WalkUp(fun func(k Ki) bool) bool {
	cur := n.This()
	for {
		if !fun(cur) { // false return means stop
			return false
		}
		par := cur.Parent()
		if par == nil || par == cur { // prevent loops
			return true
		}
		cur = par
	}
}

// WalkUpParent calls function on parent of node and all the way up to its
// parents, and so on -- sequentially all in current go routine (generally
// necessary for going up, which is typically quite fast anyway) -- level
// is incremented after each step (starts at 0, goes up), and passed to
// function -- returns false if fun aborts with false, else true.
func (n *Node) WalkUpParent(fun func(k Ki) bool) bool {
	if IsRoot(n) {
		return true
	}
	cur := n.Parent()
	for {
		if !fun(cur) { // false return means stop
			return false
		}
		par := cur.Parent()
		if par == nil || par == cur { // prevent loops
			return true
		}
		cur = par
	}
}

////////////////////////////////////////////////////////////////////////
// FuncDown -- Traversal records

// TravMap is a map for recording the traversal of nodes
type TravMap map[Ki]int

// Start is called at start of traversal
func (tm TravMap) Start(k Ki) {
	tm[k] = -1
}

// End deletes node once done at end of traversal
func (tm TravMap) End(k Ki) {
	delete(tm, k)
}

// Set updates traversal state
func (tm TravMap) Set(k Ki, curChild int) {
	tm[k] = curChild
}

// Get retrieves current traversal state
func (tm TravMap) Get(k Ki) int {
	return tm[k]
}

// strategy -- same as used in TreeView:
// https://stackoverflow.com/questions/5278580/non-recursive-depth-first-search-algorithm

// WalkPre calls function on this node (MeFirst) and then iterates
// in a depth-first manner over all the children.
// The [WalkPreNode] method is called for every node, after the given function,
// which e.g., enables nodes to also traverse additional Ki Trees (e.g., Fields).
// The node traversal is non-recursive and uses locally-allocated state -- safe
// for concurrent calling (modulo conflict management in function call itself).
// Function calls are sequential all in current go routine.
// If fun returns false then any further traversal of that branch of the tree is
// aborted, but other branches continue -- i.e., if fun on current node
// returns false, children are not processed further.
func (n *Node) WalkPre(fun func(Ki) bool) {
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
			n.This().WalkPreNode(fun)
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
			par := cur.Parent()
			if par == nil || par == cur { // shouldn't happen, but does..
				// fmt.Printf("nil / cur parent %v\n", par)
				break outer
			}
			cur = par
		}
	}
}

// WalkPreNode is called for every node during WalkPre with the function
// passed to WalkPre.  This e.g., enables nodes to also traverse additional
// Ki Trees (e.g., Fields).
func (n *Node) WalkPreNode(fun func(Ki) bool) {}

// WalkPreLevel calls function on this node (MeFirst) and then iterates
// in a depth-first manner over all the children.
// This version has a level var that tracks overall depth in the tree.
// If fun returns false then any further traversal of that branch of the tree is
// aborted, but other branches continue -- i.e., if fun on current node
// returns false, children are not processed further.
// Because WalkPreLevel is not used within Ki itself, it does not have its
// own version of WalkPreNode -- that can be handled within the closure.
func (n *Node) WalkPreLevel(fun func(k Ki, level int) bool) {
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
			par := cur.Parent()
			if par == nil || par == cur { // shouldn't happen, but does..
				// fmt.Printf("nil / cur parent %v\n", par)
				break outer
			}
			cur = par
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
func (n *Node) WalkPost(doChildTestFunc func(Ki) bool, fun func(Ki) bool) {
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
			par := cur.Parent()
			if par == nil || par == cur { // shouldn't happen
				break outer
			}
			cur = par
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
func (n *Node) WalkBreadth(fun func(k Ki) bool) {
	start := n.This()

	level := 0
	SetDepth(start, level)
	queue := make([]Ki, 1)
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
// children etc, copying Props too.  It is very efficient by
// using the ConfigChildren method which attempts to preserve any existing
// nodes in the destination if they have the same name and type -- so a
// copy from a source to a target that only differ minimally will be
// minimally destructive.  Only copies to same types are supported.
// Signal connections are NOT copied.  No other Ki pointers are copied,
// and the field tag copier:"-" can be added for any other fields that
// should not be copied (unexported, lower-case fields are not copyable).
func (n *Node) CopyFrom(frm Ki) error {
	if frm == nil {
		err := fmt.Errorf("ki.Node CopyFrom into %v: nil 'from' source", n)
		log.Println(err)
		return err
	}
	CopyFromRaw(n.This(), frm)
	return nil
}

// Clone creates and returns a deep copy of the tree from this node down.
// Any pointers within the cloned tree will correctly point within the new
// cloned tree (see Copy info).
func (n *Node) Clone() Ki {
	nki := NewOfType(n.This().KiType())
	nki.InitName(nki, n.Nm)
	nki.CopyFrom(n.This())
	return nki
}

// CopyFromRaw performs a raw copy that just does the deep copy of the
// bits and doesn't do anything with pointers.
func CopyFromRaw(kn, frm Ki) {
	kn.Children().ConfigCopy(kn.This(), *frm.Children())
	n := kn.AsKi()
	fmp := *frm.Properties()
	n.Props = make(Props, len(fmp))
	n.Props.CopyFrom(fmp, DeepCopy)

	kn.This().CopyFieldsFrom(frm)
	for i, kid := range *kn.Children() {
		fmk := frm.Child(i)
		CopyFromRaw(kid, fmk)
	}
}

// CopyFieldsFrom is the base implementation of [Ki.CopyFieldsFrom] that copies the fields
// of the [Node.This] from the fields of the given [Ki.This], recursively following anonymous
// embedded structs. It uses [copier.Copy] for this. It ignores any fields with a `copier:"-"`
// struct tag. Other implementations of [Ki.CopyFieldsFrom] should call this method first and
// then only do manual handling of specific fields that can not be automatically copied.
func (n *Node) CopyFieldsFrom(from Ki) {
	err := copier.CopyWithOption(n.This(), from.This(), copier.Option{CaseSensitive: true, DeepCopy: true})
	if err != nil {
		slog.Error("ki.Node.CopyFieldsFrom", "err", err)
	}
}
