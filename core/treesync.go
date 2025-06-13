// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"bytes"
	"fmt"
	"log"
	"strings"

	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/base/fileinfo/mimedata"
	"cogentcore.org/core/base/iox/jsonx"
	"cogentcore.org/core/base/labels"
	"cogentcore.org/core/events"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/types"
)

// note: see this file has all the SyncNode specific
// functions for Tree.

// SyncTree sets the root [Tree.SyncNode] to the root of the given [tree.Node]
// and synchronizes the rest of the tree to match. The source tree must have
// unique names for each child within a given parent.
func (tr *Tree) SyncTree(n tree.Node) *Tree {
	if tr.SyncNode != n {
		tr.SyncNode = n
	}
	tvIndex := 0
	tr.syncToSrc(&tvIndex, true, 0)
	tr.Update()
	return tr
}

// setSyncNode sets the sync source node that we are viewing,
// and syncs the view of its tree.  It is called routinely
// via SyncToSrc during tree updating.
// It uses tree Config mechanism to perform minimal updates to
// remain in sync.
func (tr *Tree) setSyncNode(sn tree.Node, tvIndex *int, init bool, depth int) {
	if tr.SyncNode != sn {
		tr.SyncNode = sn
	}
	tr.syncToSrc(tvIndex, init, depth)
}

// Resync resynchronizes the [Tree] relative to the [Tree.SyncNode]
// underlying nodes and triggers an update.
func (tr *Tree) Resync() {
	tvIndex := tr.viewIndex
	tr.syncToSrc(&tvIndex, false, 0)
	tr.Update()
}

// syncToSrc updates the view tree to match the sync tree, using
// ConfigChildren to maximally preserve existing tree elements.
// init means we are doing initial build, and depth tracks depth
// (only during init).
func (tr *Tree) syncToSrc(tvIndex *int, init bool, depth int) {
	sn := tr.SyncNode
	// root must keep the same name for continuity with surrounding context
	if tr != tr.Root {
		nm := "tv_" + sn.AsTree().Name
		tr.SetName(nm)
	}
	tr.viewIndex = *tvIndex
	*tvIndex++
	if init && depth >= tr.Root.AsCoreTree().OpenDepth {
		tr.SetClosed(true)
	}
	skids := sn.AsTree().Children
	p := make(tree.TypePlan, 0, len(skids))
	typ := tr.NodeType()
	for _, skid := range skids {
		p.Add(typ, "tv_"+skid.AsTree().Name)
	}
	tree.Update(tr, p)
	idx := 0
	for _, skid := range sn.AsTree().Children {
		if len(tr.Children) <= idx {
			break
		}
		vk := AsTree(tr.Children[idx])
		vk.setSyncNode(skid, tvIndex, init, depth+1)
		idx++
	}
	if !sn.AsTree().HasChildren() {
		tr.SetClosed(true)
	}
}

// Label returns the display label for this node,
// satisfying the [labels.Labeler] interface.
func (tr *Tree) Label() string {
	if tr.SyncNode != nil {
		// TODO: make this an option?
		if lbl, has := labels.ToLabeler(tr.SyncNode); has {
			return lbl
		}
		return tr.SyncNode.AsTree().Name
	}
	if tr.Text != "" {
		return tr.Text
	}
	return tr.Name
}

// selectedSyncNodes returns a slice of the currently selected
// sync source nodes in the entire tree
func (tr *Tree) selectedSyncNodes() []tree.Node {
	var res []tree.Node
	sl := tr.GetSelectedNodes()
	for _, v := range sl {
		res = append(res, v.AsCoreTree().SyncNode)
	}
	return res
}

// FindSyncNode returns the [Tree] node for the corresponding given
// source [tree.Node] in [Tree.SyncNode] or nil if not found.
func (tr *Tree) FindSyncNode(n tree.Node) *Tree {
	var res *Tree
	tr.WidgetWalkDown(func(cw Widget, cwb *WidgetBase) bool {
		tvn := AsTree(cw)
		if tvn != nil {
			if tvn.SyncNode == n {
				res = tvn
				return tree.Break
			}
		}
		return tree.Continue
	})
	return res
}

// InsertAfter inserts a new node in the tree
// after this node, at the same (sibling) level,
// prompting for the type of node to insert.
// If SyncNode is set, operates on Sync Tree.
func (tr *Tree) InsertAfter() { //types:add
	tr.insertAt(1, "Insert after")
}

// InsertBefore inserts a new node in the tree
// before this node, at the same (sibling) level,
// prompting for the type of node to insert
// If SyncNode is set, operates on Sync Tree.
func (tr *Tree) InsertBefore() { //types:add
	tr.insertAt(0, "Insert before")
}

func (tr *Tree) addTreeNodes(rel, myidx int, typ *types.Type, n int) {
	var stv *Tree
	for i := 0; i < n; i++ {
		nn := tree.NewOfType(typ)
		tr.InsertChild(nn, myidx+i)
		nn.AsTree().SetName(fmt.Sprintf("new-%v-%v", typ.IDName, myidx+rel+i))
		ntv := AsTree(nn)
		ntv.Update()
		if i == n-1 {
			stv = ntv
		}
	}
	tr.Update()
	tr.Open()
	tr.sendChangeEvent()
	if stv != nil {
		stv.SelectEvent(events.SelectOne)
	}
}

func (tr *Tree) addSyncNodes(rel, myidx int, typ *types.Type, n int) {
	parent := tr.SyncNode
	var sn tree.Node
	for i := 0; i < n; i++ {
		nn := tree.NewOfType(typ)
		parent.AsTree().InsertChild(nn, myidx+i)
		nn.AsTree().SetName(fmt.Sprintf("new-%v-%v", typ.IDName, myidx+rel+i))
		if i == n-1 {
			sn = nn
		}
	}
	tr.sendChangeEventReSync(nil)
	if sn != nil {
		if tvk := tr.ChildByName("tv_"+sn.AsTree().Name, 0); tvk != nil {
			stv := AsTree(tvk)
			stv.SelectEvent(events.SelectOne)
		}
	}
}

// newItemsData contains the data necessary to make a certain
// number of items of a certain type, which can be used with a
// [Form] in new item dialogs.
type newItemsData struct {

	// Number is the number of elements to create
	Number int

	// Type is the type of elements to create
	Type *types.Type
}

// insertAt inserts a new node in the tree
// at given relative offset from this node,
// at the same (sibling) level,
// prompting for the type of node to insert
// If SyncNode is set, operates on Sync Tree.
func (tr *Tree) insertAt(rel int, actNm string) {
	if tr.IsRoot(actNm) {
		return
	}
	myidx := tr.IndexInParent()
	if myidx < 0 {
		return
	}
	myidx += rel
	var typ *types.Type
	if tr.SyncNode == nil {
		typ = types.TypeByValue(tr.This)
	} else {
		typ = types.TypeByValue(tr.SyncNode)
	}
	d := NewBody(actNm)
	NewText(d).SetType(TextSupporting).SetText("Number and type of items to insert:")
	nd := &newItemsData{Number: 1, Type: typ}
	NewForm(d).SetStruct(nd)
	d.AddBottomBar(func(bar *Frame) {
		d.AddCancel(bar)
		d.AddOK(bar).OnClick(func(e events.Event) {
			parent := AsTree(tr.Parent)
			if tr.SyncNode != nil {
				parent.addSyncNodes(rel, myidx, nd.Type, nd.Number)
			} else {
				parent.addTreeNodes(rel, myidx, nd.Type, nd.Number)
			}
		})
	})
	d.RunDialog(tr)
}

// AddChildNode adds a new child node to this one in the tree,
// prompting the user for the type of node to add
// If SyncNode is set, operates on Sync Tree.
func (tr *Tree) AddChildNode() { //types:add
	ttl := "Add child"
	var typ *types.Type
	if tr.SyncNode == nil {
		typ = types.TypeByValue(tr.This)
	} else {
		typ = types.TypeByValue(tr.SyncNode)
	}
	d := NewBody(ttl)
	NewText(d).SetType(TextSupporting).SetText("Number and type of items to insert:")
	nd := &newItemsData{Number: 1, Type: typ}
	NewForm(d).SetStruct(nd)
	d.AddBottomBar(func(bar *Frame) {
		d.AddCancel(bar)
		d.AddOK(bar).OnClick(func(e events.Event) {
			if tr.SyncNode != nil {
				tr.addSyncNodes(0, 0, nd.Type, nd.Number)
			} else {
				tr.addTreeNodes(0, 0, nd.Type, nd.Number)
			}
		})
	})
	d.RunDialog(tr)
}

// DeleteNode deletes the tree node or sync node corresponding
// to this view node in the sync tree.
// If SyncNode is set, operates on Sync Tree.
func (tr *Tree) DeleteNode() { //types:add
	if tr.IsRoot("Delete") {
		return
	}
	tr.Close()
	if tr.moveDown(events.SelectOne) == nil {
		tr.moveUp(events.SelectOne)
	}
	if tr.SyncNode != nil {
		tr.SyncNode.AsTree().Delete()
		tr.sendChangeEventReSync(nil)
	} else {
		parent := AsTree(tr.Parent)
		tr.Delete()
		parent.Update()
		parent.sendChangeEvent()
	}
}

// deleteSync deletes selected items.
func (tr *Tree) deleteSync() {
	sels := tr.selectedSyncNodes()
	tr.UnselectAll()
	for _, sn := range sels {
		sn.AsTree().Delete()
	}
	tr.sendChangeEventReSync(nil)
}

// Duplicate duplicates this node, and inserts the duplicate after this node
// (as a new sibling). If SyncNode is set, operates on Sync Tree.
func (tr *Tree) Duplicate() { //types:add
	ttl := "Duplicate"
	if tr.IsRoot(ttl) {
		return
	}
	if tr.Parent == nil {
		return
	}
	if tr.SyncNode != nil {
		tr.duplicateSync()
		return
	}
	parent := AsTree(tr.Parent)
	myidx := tr.IndexInParent()
	if myidx < 0 {
		return
	}
	nm := fmt.Sprintf("%v_Copy", tr.Name)
	tr.Unselect()
	nwkid := tr.Clone()
	nwkid.AsTree().SetName(nm)
	tree.SetUniqueNameIfDuplicate(parent, nwkid)
	ntv := AsTree(nwkid)
	parent.InsertChild(nwkid, myidx+1)
	ntv.Update()
	parent.Update()
	parent.sendChangeEvent()
	// ntv.SelectEvent(events.SelectOne)
}

func (tr *Tree) duplicateSync() {
	sn := tr.SyncNode
	tvparent := AsTree(tr.Parent)
	parent := tvparent.SyncNode
	if parent == nil {
		log.Printf("Tree %v nil SyncNode in: %v\n", tr, tvparent.Path())
		return
	}
	myidx := sn.AsTree().IndexInParent()
	if myidx < 0 {
		return
	}
	nm := fmt.Sprintf("%v_Copy", sn.AsTree().Name)
	nwkid := sn.AsTree().Clone()
	nwkid.AsTree().SetName(nm)
	tree.SetUniqueNameIfDuplicate(parent, nwkid)
	parent.AsTree().InsertChild(nwkid, myidx+1)
	tvparent.sendChangeEventReSync(nil)
	if tvk := tvparent.ChildByName("tv_"+nm, 0); tvk != nil {
		stv := AsTree(tvk)
		stv.SelectEvent(events.SelectOne)
	}
}

// EditNode pulls up a [Form] dialog for the node.
// If SyncNode is set, operates on Sync Tree.
func (tr *Tree) EditNode() { //types:add
	if tr.SyncNode != nil {
		tynm := tr.SyncNode.AsTree().NodeType().Name
		d := NewBody(tynm)
		NewForm(d).SetStruct(tr.SyncNode).SetReadOnly(tr.IsReadOnly())
		d.RunWindowDialog(tr)
	} else {
		tynm := tr.NodeType().Name
		d := NewBody(tynm)
		NewForm(d).SetStruct(tr.This).SetReadOnly(tr.IsReadOnly())
		d.RunWindowDialog(tr)
	}
}

// inspectNode pulls up a new Inspector window on the node.
// If SyncNode is set, operates on Sync Tree.
func (tr *Tree) inspectNode() { //types:add
	if tr.SyncNode != nil {
		InspectorWindow(tr.SyncNode)
	} else {
		InspectorWindow(tr)
	}
}

// mimeDataSync adds mimedata for this node: a text/plain of the Path,
// and an application/json of the sync node.
func (tr *Tree) mimeDataSync(md *mimedata.Mimes) {
	sroot := tr.Root.AsCoreTree().SyncNode
	src := tr.SyncNode
	*md = append(*md, mimedata.NewTextData(src.AsTree().PathFrom(sroot)))
	var buf bytes.Buffer
	err := jsonx.Write(src, &buf)
	if err == nil {
		*md = append(*md, &mimedata.Data{Type: fileinfo.DataJson, Data: buf.Bytes()})
	} else {
		ErrorSnackbar(tr, err, "Error encoding node")
	}
}

// syncNodesFromMimeData creates a slice of tree node(s)
// from given mime data and also a corresponding slice
// of original paths.
func (tr *Tree) syncNodesFromMimeData(md mimedata.Mimes) ([]tree.Node, []string) {
	ni := len(md) / 2
	sl := make([]tree.Node, 0, ni)
	pl := make([]string, 0, ni)
	for _, d := range md {
		if d.Type == fileinfo.DataJson {
			nn, err := tree.UnmarshalRootJSON(d.Data)
			if err == nil {
				sl = append(sl, nn)
			} else {
				ErrorSnackbar(tr, err, "Error loading node")
			}
		} else if d.Type == fileinfo.TextPlain { // paths
			pl = append(pl, string(d.Data))
		}
	}
	return sl, pl
}

// pasteAssignSync assigns mime data (only the first one!) to this node
func (tr *Tree) pasteAssignSync(md mimedata.Mimes) {
	sl, _ := tr.syncNodesFromMimeData(md)
	if len(sl) == 0 {
		return
	}
	tr.SyncNode.AsTree().CopyFrom(sl[0])
	tr.NeedsLayout()
	tr.sendChangeEvent()
}

// pasteAtSync inserts object(s) from mime data at rel position to this node.
// If another item with the same name already exists, it will
// append _Copy on the name of the inserted objects
func (tr *Tree) pasteAtSync(md mimedata.Mimes, mod events.DropMods, rel int, actNm string) {
	sn := tr.SyncNode
	sl, pl := tr.nodesFromMimeData(md)
	tvparent := AsTree(tr.Parent)
	parent := sn.AsTree().Parent
	myidx := sn.AsTree().IndexInParent()
	if myidx < 0 {
		return
	}
	myidx += rel
	sroot := tr.Root.AsCoreTree().SyncNode
	pt := parent.AsTree()
	sz := len(sl)
	var seln tree.Node
	for i, ns := range sl {
		nst := ns.AsTree()
		orgpath := pl[i]
		tree.SetUniqueNameIfDuplicate(parent, ns)
		pt.InsertChild(ns, myidx+i)
		npath := nst.PathFrom(sroot)
		if mod == events.DropMove && npath == orgpath { // we will be nuked immediately after drag
			nst.SetName(nst.Name + treeTempMovedTag) // special keyword :)
		}
		if i == sz-1 {
			seln = ns
		}
	}
	tvparent.sendChangeEventReSync(nil)
	if seln != nil {
		if tvk := tvparent.ChildByName("tv_"+seln.AsTree().Name, myidx); tvk != nil {
			stv := AsTree(tvk)
			stv.SelectEvent(events.SelectOne)
		}
	}
}

// pasteChildrenSync inserts object(s) from mime data at
// end of children of this node
func (tr *Tree) pasteChildrenSync(md mimedata.Mimes, mod events.DropMods) {
	sl, _ := tr.nodesFromMimeData(md)
	spar := tr.SyncNode
	for _, ns := range sl {
		tree.SetUniqueNameIfDuplicate(spar, ns)
		spar.AsTree().AddChild(ns)
	}
	tr.sendChangeEventReSync(nil)
}

// cutSync copies to system.Clipboard and deletes selected items.
func (tr *Tree) cutSync() {
	tr.Copy()
	sels := tr.selectedSyncNodes()
	tr.UnselectAll()
	for _, sn := range sels {
		sn.AsTree().Delete()
	}
	tr.sendChangeEventReSync(nil)
}

// dropDeleteSourceSync handles delete source event for DropMove case, for Sync
func (tr *Tree) dropDeleteSourceSync(de *events.DragDrop) {
	md := de.Data.(mimedata.Mimes)
	sroot := tr.Root.AsCoreTree().SyncNode
	for _, d := range md {
		if d.Type != fileinfo.TextPlain { // link
			continue
		}
		path := string(d.Data)
		sn := sroot.AsTree().FindPath(path)
		if sn != nil {
			sn.AsTree().Delete()
		}
		sn = sroot.AsTree().FindPath(path + treeTempMovedTag)
		if sn != nil {
			psplt := strings.Split(path, "/")
			orgnm := psplt[len(psplt)-1]
			sn.AsTree().SetName(orgnm)
			AsWidget(sn).NeedsRender()
		}
	}
	tr.sendChangeEventReSync(nil)
}
