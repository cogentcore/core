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
	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/events"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/types"
)

// note: see this file has all the SyncNode specific
// functions for Tree.

// SyncTree sets the root view to the root of the sync source tree node
// for this Tree, and syncs the rest of the tree to match.
// The source tree must have unique names for each child within a given parent.
func (tr *Tree) SyncTree(n tree.Node) *Tree {
	if tr.SyncNode != n {
		tr.SyncNode = n
	}
	tvIndex := 0
	tr.SyncToSrc(&tvIndex, true, 0)
	tr.Update()
	return tr
}

// SetSyncNode sets the sync source node that we are viewing,
// and syncs the view of its tree.  It is called routinely
// via SyncToSrc during tree updating.
// It uses tree Config mechanism to perform minimal updates to
// remain in sync.
func (tr *Tree) SetSyncNode(sn tree.Node, tvIndex *int, init bool, depth int) {
	if tr.SyncNode != sn {
		tr.SyncNode = sn
	}
	tr.SyncToSrc(tvIndex, init, depth)
}

// ReSync resynchronizes the view relative to the underlying nodes
// and forces a full rerender
func (tr *Tree) ReSync() {
	tvIndex := tr.viewIndex
	tr.SyncToSrc(&tvIndex, false, 0)
	tr.Update()
}

// SyncToSrc updates the view tree to match the sync tree, using
// ConfigChildren to maximally preserve existing tree elements.
// init means we are doing initial build, and depth tracks depth
// (only during init).
func (tr *Tree) SyncToSrc(tvIndex *int, init bool, depth int) {
	sn := tr.SyncNode
	// root must keep the same name for continuity with surrounding context
	if tr != tr.RootView {
		nm := "tv_" + sn.AsTree().Name
		tr.SetName(nm)
	}
	tr.viewIndex = *tvIndex
	*tvIndex++
	if init && depth >= tr.RootView.OpenDepth {
		tr.SetClosed(true)
	}
	skids := sn.AsTree().Children
	p := make(tree.TypePlan, 0, len(skids))
	typ := tr.This.NodeType()
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
		vk.SetSyncNode(skid, tvIndex, init, depth+1)
		idx++
	}
	if !sn.AsTree().HasChildren() {
		tr.SetClosed(true)
	}
}

// Label returns the display label for this node,
// satisfying the Labeler interface
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

// UpdateReadOnly updates the ReadOnly state based on SyncNode.
// Returns true if ReadOnly.
// The inactivity of individual nodes only affects display properties
// typically, and not overall functional behavior, which is controlled by
// inactivity of the root node (i.e, make the root ReadOnly
// to make entire tree read-only and non-modifiable)
func (tr *Tree) UpdateReadOnly() bool {
	if tr.SyncNode == nil {
		return false
	}
	tr.SetState(false, states.Disabled)
	if inact := tr.SyncNode.AsTree().Property("ReadOnly"); inact != nil {
		if bo, err := reflectx.ToBool(inact); bo && err == nil {
			tr.SetReadOnly(true)
		}
	}
	return tr.IsReadOnly()
}

// SelectedSyncNodes returns a slice of the currently selected
// sync source nodes in the entire tree
func (tr *Tree) SelectedSyncNodes() []tree.Node {
	var res []tree.Node
	sl := tr.SelectedViews()
	for _, v := range sl {
		res = append(res, v.AsCoreTree().SyncNode)
	}
	return res
}

// FindSyncNode finds Tree node for given source node,
// or nil if not found
func (tr *Tree) FindSyncNode(kn tree.Node) *Tree {
	var ttv *Tree
	tr.WidgetWalkDown(func(wi Widget, wb *WidgetBase) bool {
		tvn := AsTree(wi)
		if tvn != nil {
			if tvn.SyncNode == kn {
				ttv = tvn
				return tree.Break
			}
		}
		return tree.Continue
	})
	return ttv
}

// InsertAfter inserts a new node in the tree
// after this node, at the same (sibling) level,
// prompting for the type of node to insert.
// If SyncNode is set, operates on Sync Tree.
func (tr *Tree) InsertAfter() { //types:add
	tr.InsertAt(1, "Insert After")
}

// InsertBefore inserts a new node in the tree
// before this node, at the same (sibling) level,
// prompting for the type of node to insert
// If SyncNode is set, operates on Sync Tree.
func (tr *Tree) InsertBefore() { //types:add
	tr.InsertAt(0, "Insert Before")
}

func (tr *Tree) AddTreeNodes(rel, myidx int, typ *types.Type, n int) {
	var stv *Tree
	for i := 0; i < n; i++ {
		nn := tr.InsertNewChild(typ, myidx+i)
		nn.AsTree().SetName(fmt.Sprintf("new-%v-%v", typ.IDName, myidx+rel+i))
		ntv := AsTree(nn)
		ntv.Update()
		if i == n-1 {
			stv = ntv
		}
	}
	tr.Update()
	tr.Open()
	tr.TreeChanged(nil)
	if stv != nil {
		stv.SelectAction(events.SelectOne)
	}
}

func (tr *Tree) AddSyncNodes(rel, myidx int, typ *types.Type, n int) {
	parent := tr.SyncNode
	var sn tree.Node
	for i := 0; i < n; i++ {
		nn := parent.AsTree().InsertNewChild(typ, myidx+i)
		nn.AsTree().SetName(fmt.Sprintf("new-%v-%v", typ.IDName, myidx+rel+i))
		if i == n-1 {
			sn = nn
		}
	}
	tr.SendChangeEventReSync(nil)
	if sn != nil {
		if tvk := tr.ChildByName("tv_"+sn.AsTree().Name, 0); tvk != nil {
			stv := AsTree(tvk)
			stv.SelectAction(events.SelectOne)
		}
	}
}

// InsertAt inserts a new node in the tree
// at given relative offset from this node,
// at the same (sibling) level,
// prompting for the type of node to insert
// If SyncNode is set, operates on Sync Tree.
func (tr *Tree) InsertAt(rel int, actNm string) {
	if tr.IsRoot(actNm) {
		return
	}
	myidx := tr.IndexInParent()
	if myidx < 0 {
		return
	}
	myidx += rel
	typ := tr.This.BaseType()
	if tr.SyncNode != nil {
		typ = tr.SyncNode.BaseType()
	}
	d := NewBody().AddTitle(actNm).AddText("Number and type of items to insert:")
	nd := &NewItemsData{Number: 1, Type: typ}
	sv := NewForm(d).SetStruct(nd) // TODO(config)
	tree.ChildByType[*Chooser](sv, tree.Embeds).SetTypes(types.AllEmbeddersOf(typ)...).SetCurrentIndex(0)
	d.AddBottomBar(func(parent Widget) {
		d.AddCancel(parent)
		d.AddOK(parent).OnClick(func(e events.Event) {
			parent := AsTree(tr.Parent)
			if tr.SyncNode != nil {
				parent.AddSyncNodes(rel, myidx, nd.Type, nd.Number)
			} else {
				parent.AddTreeNodes(rel, myidx, nd.Type, nd.Number)
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
	typ := tr.This.BaseType()
	if tr.SyncNode != nil {
		typ = tr.SyncNode.BaseType()
	}
	d := NewBody().AddTitle(ttl).AddText("Number and type of items to insert:")
	nd := &NewItemsData{Number: 1, Type: typ}
	sv := NewForm(d).SetStruct(nd)
	tree.ChildByType[*TypeChooser](sv, tree.Embeds).SetTypes(types.AllEmbeddersOf(typ)...).SetCurrentIndex(0) // TODO(config)
	d.AddBottomBar(func(parent Widget) {
		d.AddCancel(parent)
		d.AddOK(parent).OnClick(func(e events.Event) {
			if tr.SyncNode != nil {
				tr.AddSyncNodes(0, 0, nd.Type, nd.Number)
			} else {
				tr.AddTreeNodes(0, 0, nd.Type, nd.Number)
			}
		})
	})
	d.RunDialog(tr)
}

// DeleteNode deletes the tree node or sync node corresponding
// to this view node in the sync tree.
// If SyncNode is set, operates on Sync Tree.
func (tr *Tree) DeleteNode() { //types:add
	ttl := "Delete"
	if tr.IsRoot(ttl) {
		return
	}
	tr.Close()
	if tr.MoveDown(events.SelectOne) == nil {
		tr.MoveUp(events.SelectOne)
	}
	if tr.SyncNode != nil {
		tr.SyncNode.AsTree().Delete()
		tr.SendChangeEventReSync(nil)
	} else {
		parent := AsTree(tr.Parent)
		tr.Delete()
		parent.Update()
		parent.TreeChanged(nil)
	}
}

// Duplicate duplicates the sync node corresponding to this view node in
// the tree, and inserts the duplicate after this node (as a new sibling).
// If SyncNode is set, operates on Sync Tree.
func (tr *Tree) Duplicate() { //types:add
	ttl := "Tree Duplicate"
	if tr.IsRoot(ttl) {
		return
	}
	if tr.Parent == nil {
		return
	}
	if tr.SyncNode != nil {
		tr.DuplicateSync()
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
	ntv := AsTree(nwkid)
	parent.InsertChild(nwkid, myidx+1)
	ntv.Update()
	parent.Update()
	parent.TreeChanged(nil)
	// ntv.SelectAction(events.SelectOne)
}

func (tr *Tree) DuplicateSync() {
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
	parent.AsTree().InsertChild(nwkid, myidx+1)
	tvparent.SendChangeEventReSync(nil)
	if tvk := tvparent.ChildByName("tv_"+nm, 0); tvk != nil {
		stv := AsTree(tvk)
		stv.SelectAction(events.SelectOne)
	}
}

// EditNode pulls up a FormDialog window on the node.
// If SyncNode is set, operates on Sync Tree.
func (tr *Tree) EditNode() { //types:add
	if tr.SyncNode != nil {
		tynm := tr.SyncNode.NodeType().Name
		d := NewBody().AddTitle(tynm)
		NewForm(d).SetStruct(tr.SyncNode).SetReadOnly(tr.IsReadOnly())
		d.RunFullDialog(tr)
	} else {
		tynm := tr.NodeType().Name
		d := NewBody().AddTitle(tynm)
		NewForm(d).SetStruct(tr.This).SetReadOnly(tr.IsReadOnly())
		d.RunFullDialog(tr)
	}
}

// InspectNode pulls up a new Inspector window on the node.
// If SyncNode is set, operates on Sync Tree.
func (tr *Tree) InspectNode() { //types:add
	if tr.SyncNode != nil {
		InspectorWindow(tr.SyncNode)
	} else {
		InspectorWindow(tr)
	}
}

// MimeDataSync adds mimedata for this node: a text/plain of the Path,
// and an application/json of the sync node.
func (tr *Tree) MimeDataSync(md *mimedata.Mimes) {
	sroot := tr.RootView.SyncNode
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

// SyncNodesFromMimeData creates a slice of tree node(s)
// from given mime data and also a corresponding slice
// of original paths.
func (tr *Tree) SyncNodesFromMimeData(md mimedata.Mimes) ([]tree.Node, []string) {
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

// PasteAssignSync assigns mime data (only the first one!) to this node
func (tr *Tree) PasteAssignSync(md mimedata.Mimes) {
	sl, _ := tr.SyncNodesFromMimeData(md)
	if len(sl) == 0 {
		return
	}
	tr.SyncNode.AsTree().CopyFrom(sl[0])
	tr.NeedsLayout()
	tr.SendChangeEvent(nil)
}

// PasteAtSync inserts object(s) from mime data at rel position to this node.
// If another item with the same name already exists, it will
// append _Copy on the name of the inserted objects
func (tr *Tree) PasteAtSync(md mimedata.Mimes, mod events.DropMods, rel int, actNm string) {
	sn := tr.SyncNode
	sl, pl := tr.NodesFromMimeData(md)
	tvparent := AsTree(tr.Parent)
	parent := sn.AsTree().Parent
	myidx := sn.AsTree().IndexInParent()
	if myidx < 0 {
		return
	}
	myidx += rel
	sroot := tr.RootView.SyncNode
	sz := len(sl)
	var seln tree.Node
	for i, ns := range sl {
		orgpath := pl[i]
		if mod != events.DropMove {
			if cn := parent.AsTree().ChildByName(ns.AsTree().Name, 0); cn != nil {
				ns.AsTree().SetName(ns.AsTree().Name + "_Copy")
			}
		}
		parent.AsTree().InsertChild(ns, myidx+i)
		npath := ns.AsTree().PathFrom(sroot)
		if mod == events.DropMove && npath == orgpath { // we will be nuked immediately after drag
			ns.AsTree().SetName(ns.AsTree().Name + TreeTempMovedTag) // special keyword :)
		}
		if i == sz-1 {
			seln = ns
		}
	}
	tvparent.SendChangeEventReSync(nil)
	if seln != nil {
		if tvk := tvparent.ChildByName("tv_"+seln.AsTree().Name, myidx); tvk != nil {
			stv := AsTree(tvk)
			stv.SelectAction(events.SelectOne)
		}
	}
}

// PasteChildrenSync inserts object(s) from mime data at
// end of children of this node
func (tr *Tree) PasteChildrenSync(md mimedata.Mimes, mod events.DropMods) {
	sl, _ := tr.NodesFromMimeData(md)
	sk := tr.SyncNode
	for _, ns := range sl {
		sk.AsTree().AddChild(ns)
	}
	tr.SendChangeEventReSync(nil)
}

// CutSync copies to system.Clipboard and deletes selected items.
func (tr *Tree) CutSync() {
	tr.Copy(false)
	sels := tr.SelectedSyncNodes()
	tr.UnselectAll()
	for _, sn := range sels {
		sn.AsTree().Delete()
	}
	tr.SendChangeEventReSync(nil)
}

// DropDeleteSourceSync handles delete source event for DropMove case, for Sync
func (tr *Tree) DropDeleteSourceSync(de *events.DragDrop) {
	md := de.Data.(mimedata.Mimes)
	sroot := tr.RootView.SyncNode
	for _, d := range md {
		if d.Type != fileinfo.TextPlain { // link
			continue
		}
		path := string(d.Data)
		sn := sroot.AsTree().FindPath(path)
		if sn != nil {
			sn.AsTree().Delete()
		}
		sn = sroot.AsTree().FindPath(path + TreeTempMovedTag)
		if sn != nil {
			psplt := strings.Split(path, "/")
			orgnm := psplt[len(psplt)-1]
			sn.AsTree().SetName(orgnm)
			_, swb := AsWidget(sn)
			swb.NeedsRender()
		}
	}
	tr.SendChangeEventReSync(nil)
}
