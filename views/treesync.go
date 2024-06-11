// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package views

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
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/types"
)

// note: see this file has all the SyncNode specific
// functions for TreeView.

// SyncTree sets the root view to the root of the sync source tree node
// for this TreeView, and syncs the rest of the tree to match.
// The source tree must have unique names for each child within a given parent.
func (tv *TreeView) SyncTree(n tree.Node) *TreeView {
	if tv.SyncNode != n {
		tv.SyncNode = n
	}
	tvIndex := 0
	tv.SyncToSrc(&tvIndex, true, 0)
	tv.Update()
	return tv
}

// SetSyncNode sets the sync source node that we are viewing,
// and syncs the view of its tree.  It is called routinely
// via SyncToSrc during tree updating.
// It uses tree Config mechanism to perform minimal updates to
// remain in sync.
func (tv *TreeView) SetSyncNode(sn tree.Node, tvIndex *int, init bool, depth int) {
	if tv.SyncNode != sn {
		tv.SyncNode = sn
	}
	tv.SyncToSrc(tvIndex, init, depth)
}

// ReSync resynchronizes the view relative to the underlying nodes
// and forces a full rerender
func (tv *TreeView) ReSync() {
	tvIndex := tv.ViewIndex
	tv.SyncToSrc(&tvIndex, false, 0)
	tv.Update()
}

// SyncToSrc updates the view tree to match the sync tree, using
// ConfigChildren to maximally preserve existing tree elements.
// init means we are doing initial build, and depth tracks depth
// (only during init).
func (tv *TreeView) SyncToSrc(tvIndex *int, init bool, depth int) {
	sn := tv.SyncNode
	// root must keep the same name for continuity with surrounding context
	if tv != tv.RootView {
		nm := "tv_" + sn.AsTree().Name
		tv.SetName(nm)
	}
	tv.ViewIndex = *tvIndex
	*tvIndex++
	if init && depth >= tv.RootView.OpenDepth {
		tv.SetClosed(true)
	}
	skids := sn.AsTree().Children
	p := make(tree.TypePlan, 0, len(skids))
	typ := tv.This().NodeType()
	for _, skid := range skids {
		p.Add(typ, "tv_"+skid.AsTree().Name)
	}
	tree.Update(tv, p)
	idx := 0
	for _, skid := range sn.AsTree().Children {
		if len(tv.Children) <= idx {
			break
		}
		vk := AsTreeView(tv.Children[idx])
		vk.SetSyncNode(skid, tvIndex, init, depth+1)
		idx++
	}
	if !sn.AsTree().HasChildren() {
		tv.SetClosed(true)
	}
}

// Label returns the display label for this node,
// satisfying the Labeler interface
func (tv *TreeView) Label() string {
	if tv.SyncNode != nil {
		// TODO: make this an option?
		if lbl, has := labels.ToLabeler(tv.SyncNode); has {
			return lbl
		}
		return tv.SyncNode.AsTree().Name
	}
	if tv.Text != "" {
		return tv.Text
	}
	return tv.Name
}

// UpdateReadOnly updates the ReadOnly state based on SyncNode.
// Returns true if ReadOnly.
// The inactivity of individual nodes only affects display properties
// typically, and not overall functional behavior, which is controlled by
// inactivity of the root node (i.e, make the root ReadOnly
// to make entire tree read-only and non-modifiable)
func (tv *TreeView) UpdateReadOnly() bool {
	if tv.SyncNode == nil {
		return false
	}
	tv.SetState(false, states.Disabled)
	if inact := tv.SyncNode.AsTree().Property("ReadOnly"); inact != nil {
		if bo, err := reflectx.ToBool(inact); bo && err == nil {
			tv.SetReadOnly(true)
		}
	}
	return tv.IsReadOnly()
}

// SelectedSyncNodes returns a slice of the currently selected
// sync source nodes in the entire tree view
func (tv *TreeView) SelectedSyncNodes() []tree.Node {
	var res []tree.Node
	sl := tv.SelectedViews()
	for _, v := range sl {
		res = append(res, v.AsTreeView().SyncNode)
	}
	return res
}

// FindSyncNode finds TreeView node for given source node,
// or nil if not found
func (tv *TreeView) FindSyncNode(kn tree.Node) *TreeView {
	var ttv *TreeView
	tv.WidgetWalkDown(func(wi core.Widget, wb *core.WidgetBase) bool {
		tvn := AsTreeView(wi)
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
func (tv *TreeView) InsertAfter() { //types:add
	tv.InsertAt(1, "Insert After")
}

// InsertBefore inserts a new node in the tree
// before this node, at the same (sibling) level,
// prompting for the type of node to insert
// If SyncNode is set, operates on Sync Tree.
func (tv *TreeView) InsertBefore() { //types:add
	tv.InsertAt(0, "Insert Before")
}

func (tv *TreeView) AddTreeNodes(rel, myidx int, typ *types.Type, n int) {
	var stv *TreeView
	for i := 0; i < n; i++ {
		nn := tv.InsertNewChild(typ, myidx+i)
		nn.AsTree().SetName(fmt.Sprintf("new-%v-%v", typ.IDName, myidx+rel+i))
		ntv := AsTreeView(nn)
		ntv.Update()
		if i == n-1 {
			stv = ntv
		}
	}
	tv.Update()
	tv.Open()
	tv.TreeViewChanged(nil)
	if stv != nil {
		stv.SelectAction(events.SelectOne)
	}
}

func (tv *TreeView) AddSyncNodes(rel, myidx int, typ *types.Type, n int) {
	parent := tv.SyncNode
	var sn tree.Node
	for i := 0; i < n; i++ {
		nn := parent.AsTree().InsertNewChild(typ, myidx+i)
		nn.AsTree().SetName(fmt.Sprintf("new-%v-%v", typ.IDName, myidx+rel+i))
		if i == n-1 {
			sn = nn
		}
	}
	tv.SendChangeEventReSync(nil)
	if sn != nil {
		if tvk := tv.ChildByName("tv_"+sn.AsTree().Name, 0); tvk != nil {
			stv := AsTreeView(tvk)
			stv.SelectAction(events.SelectOne)
		}
	}
}

// InsertAt inserts a new node in the tree
// at given relative offset from this node,
// at the same (sibling) level,
// prompting for the type of node to insert
// If SyncNode is set, operates on Sync Tree.
func (tv *TreeView) InsertAt(rel int, actNm string) {
	if tv.IsRoot(actNm) {
		return
	}
	myidx := tv.IndexInParent()
	if myidx < 0 {
		return
	}
	myidx += rel
	typ := tv.This().BaseType()
	if tv.SyncNode != nil {
		typ = tv.SyncNode.BaseType()
	}
	d := core.NewBody().AddTitle(actNm).AddText("Number and type of items to insert:")
	nd := &core.NewItemsData{Number: 1, Type: typ}
	sv := NewStructView(d).SetStruct(nd) // TODO(config)
	tree.ChildByType[*core.Chooser](sv, tree.Embeds).SetTypes(types.AllEmbeddersOf(typ)...).SetCurrentIndex(0)
	d.AddBottomBar(func(parent core.Widget) {
		d.AddCancel(parent)
		d.AddOK(parent).OnClick(func(e events.Event) {
			parent := AsTreeView(tv.Par)
			if tv.SyncNode != nil {
				parent.AddSyncNodes(rel, myidx, nd.Type, nd.Number)
			} else {
				parent.AddTreeNodes(rel, myidx, nd.Type, nd.Number)
			}
		})
	})
	d.RunDialog(tv)
}

// AddChildNode adds a new child node to this one in the tree,
// prompting the user for the type of node to add
// If SyncNode is set, operates on Sync Tree.
func (tv *TreeView) AddChildNode() { //types:add
	ttl := "Add child"
	typ := tv.This().BaseType()
	if tv.SyncNode != nil {
		typ = tv.SyncNode.BaseType()
	}
	d := core.NewBody().AddTitle(ttl).AddText("Number and type of items to insert:")
	nd := &core.NewItemsData{Number: 1, Type: typ}
	sv := NewStructView(d).SetStruct(nd)
	tree.ChildByType[*TypeChooser](sv, tree.Embeds).SetTypes(types.AllEmbeddersOf(typ)...).SetCurrentIndex(0) // TODO(config)
	d.AddBottomBar(func(parent core.Widget) {
		d.AddCancel(parent)
		d.AddOK(parent).OnClick(func(e events.Event) {
			if tv.SyncNode != nil {
				tv.AddSyncNodes(0, 0, nd.Type, nd.Number)
			} else {
				tv.AddTreeNodes(0, 0, nd.Type, nd.Number)
			}
		})
	})
	d.RunDialog(tv)
}

// DeleteNode deletes the tree node or sync node corresponding
// to this view node in the sync tree.
// If SyncNode is set, operates on Sync Tree.
func (tv *TreeView) DeleteNode() { //types:add
	ttl := "Delete"
	if tv.IsRoot(ttl) {
		return
	}
	tv.Close()
	if tv.MoveDown(events.SelectOne) == nil {
		tv.MoveUp(events.SelectOne)
	}
	if tv.SyncNode != nil {
		tv.SyncNode.AsTree().Delete()
		tv.SendChangeEventReSync(nil)
	} else {
		parent := AsTreeView(tv.Par)
		tv.Delete()
		parent.Update()
		parent.TreeViewChanged(nil)
	}
}

// Duplicate duplicates the sync node corresponding to this view node in
// the tree, and inserts the duplicate after this node (as a new sibling).
// If SyncNode is set, operates on Sync Tree.
func (tv *TreeView) Duplicate() { //types:add
	ttl := "TreeView Duplicate"
	if tv.IsRoot(ttl) {
		return
	}
	if tv.Par == nil {
		return
	}
	if tv.SyncNode != nil {
		tv.DuplicateSync()
		return
	}
	parent := AsTreeView(tv.Par)
	myidx := tv.IndexInParent()
	if myidx < 0 {
		return
	}
	nm := fmt.Sprintf("%v_Copy", tv.Name)
	tv.Unselect()
	nwkid := tv.Clone()
	nwkid.AsTree().SetName(nm)
	ntv := AsTreeView(nwkid)
	parent.InsertChild(nwkid, myidx+1)
	ntv.Update()
	parent.Update()
	parent.TreeViewChanged(nil)
	// ntv.SelectAction(events.SelectOne)
}

func (tv *TreeView) DuplicateSync() {
	sn := tv.SyncNode
	tvparent := AsTreeView(tv.Par)
	parent := tvparent.SyncNode
	if parent == nil {
		log.Printf("TreeView %v nil SyncNode in: %v\n", tv, tvparent.Path())
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
		stv := AsTreeView(tvk)
		stv.SelectAction(events.SelectOne)
	}
}

// EditNode pulls up a StructViewDialog window on the node.
// If SyncNode is set, operates on Sync Tree.
func (tv *TreeView) EditNode() { //types:add
	if tv.SyncNode != nil {
		tynm := tv.SyncNode.NodeType().Name
		d := core.NewBody().AddTitle(tynm)
		NewStructView(d).SetStruct(tv.SyncNode).SetReadOnly(tv.IsReadOnly())
		d.RunFullDialog(tv)
	} else {
		tynm := tv.NodeType().Name
		d := core.NewBody().AddTitle(tynm)
		NewStructView(d).SetStruct(tv.This()).SetReadOnly(tv.IsReadOnly())
		d.RunFullDialog(tv)
	}
}

// InspectNode pulls up a new Inspector window on the node.
// If SyncNode is set, operates on Sync Tree.
func (tv *TreeView) InspectNode() { //types:add
	if tv.SyncNode != nil {
		InspectorWindow(tv.SyncNode)
	} else {
		InspectorWindow(tv)
	}
}

// MimeDataSync adds mimedata for this node: a text/plain of the Path,
// and an application/json of the sync node.
func (tv *TreeView) MimeDataSync(md *mimedata.Mimes) {
	sroot := tv.RootView.SyncNode
	src := tv.SyncNode
	*md = append(*md, mimedata.NewTextData(src.AsTree().PathFrom(sroot)))
	var buf bytes.Buffer
	err := jsonx.Write(src, &buf)
	if err == nil {
		*md = append(*md, &mimedata.Data{Type: fileinfo.DataJson, Data: buf.Bytes()})
	} else {
		core.ErrorSnackbar(tv, err, "Error encoding node")
	}
}

// SyncNodesFromMimeData creates a slice of tree node(s)
// from given mime data and also a corresponding slice
// of original paths.
func (tv *TreeView) SyncNodesFromMimeData(md mimedata.Mimes) ([]tree.Node, []string) {
	ni := len(md) / 2
	sl := make([]tree.Node, 0, ni)
	pl := make([]string, 0, ni)
	for _, d := range md {
		if d.Type == fileinfo.DataJson {
			nn, err := tree.UnmarshalRootJSON(d.Data)
			if err == nil {
				sl = append(sl, nn)
			} else {
				core.ErrorSnackbar(tv, err, "Error loading node")
			}
		} else if d.Type == fileinfo.TextPlain { // paths
			pl = append(pl, string(d.Data))
		}
	}
	return sl, pl
}

// PasteAssignSync assigns mime data (only the first one!) to this node
func (tv *TreeView) PasteAssignSync(md mimedata.Mimes) {
	sl, _ := tv.SyncNodesFromMimeData(md)
	if len(sl) == 0 {
		return
	}
	tv.SyncNode.AsTree().CopyFrom(sl[0])
	tv.NeedsLayout()
	tv.SendChangeEvent(nil)
}

// PasteAtSync inserts object(s) from mime data at rel position to this node.
// If another item with the same name already exists, it will
// append _Copy on the name of the inserted objects
func (tv *TreeView) PasteAtSync(md mimedata.Mimes, mod events.DropMods, rel int, actNm string) {
	sn := tv.SyncNode
	sl, pl := tv.NodesFromMimeData(md)
	tvparent := AsTreeView(tv.Par)
	parent := sn.AsTree().Parent()
	myidx := sn.AsTree().IndexInParent()
	if myidx < 0 {
		return
	}
	myidx += rel
	sroot := tv.RootView.SyncNode
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
			ns.AsTree().SetName(ns.AsTree().Name + TreeViewTempMovedTag) // special keyword :)
		}
		if i == sz-1 {
			seln = ns
		}
	}
	tvparent.SendChangeEventReSync(nil)
	if seln != nil {
		if tvk := tvparent.ChildByName("tv_"+seln.AsTree().Name, myidx); tvk != nil {
			stv := AsTreeView(tvk)
			stv.SelectAction(events.SelectOne)
		}
	}
}

// PasteChildrenSync inserts object(s) from mime data at
// end of children of this node
func (tv *TreeView) PasteChildrenSync(md mimedata.Mimes, mod events.DropMods) {
	sl, _ := tv.NodesFromMimeData(md)
	sk := tv.SyncNode
	for _, ns := range sl {
		sk.AsTree().AddChild(ns)
	}
	tv.SendChangeEventReSync(nil)
}

// CutSync copies to system.Clipboard and deletes selected items.
func (tv *TreeView) CutSync() {
	tv.Copy(false)
	sels := tv.SelectedSyncNodes()
	tv.UnselectAll()
	for _, sn := range sels {
		sn.AsTree().Delete()
	}
	tv.SendChangeEventReSync(nil)
}

// DropDeleteSourceSync handles delete source event for DropMove case, for Sync
func (tv *TreeView) DropDeleteSourceSync(de *events.DragDrop) {
	md := de.Data.(mimedata.Mimes)
	sroot := tv.RootView.SyncNode
	for _, d := range md {
		if d.Type != fileinfo.TextPlain { // link
			continue
		}
		path := string(d.Data)
		sn := sroot.AsTree().FindPath(path)
		if sn != nil {
			sn.AsTree().Delete()
		}
		sn = sroot.AsTree().FindPath(path + TreeViewTempMovedTag)
		if sn != nil {
			psplt := strings.Split(path, "/")
			orgnm := psplt[len(psplt)-1]
			sn.AsTree().SetName(orgnm)
			_, swb := core.AsWidget(sn)
			swb.NeedsRender()
		}
	}
	tv.SendChangeEventReSync(nil)
}
