// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"bytes"
	"fmt"
	"log"
	"strings"

	"cogentcore.org/core/events"
	"cogentcore.org/core/fi"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/gti"
	"cogentcore.org/core/ki"
	"cogentcore.org/core/laser"
	"cogentcore.org/core/mimedata"
	"cogentcore.org/core/states"
)

// note: see this file has all the SyncNode specific
// functions for TreeView.

// SyncTree sets the root view to the root of the sync source tree node
// for this TreeView, and syncs the rest of the tree to match.
// It calls [ki.UniquifyNamesAll] on the source tree to ensure
// that node names are unique, which is essential for proper viewing.
func (tv *TreeView) SyncTree(tree ki.Ki) *TreeView {
	ki.UniquifyNamesAll(tree)
	if tv.SyncNode != tree {
		tv.SyncNode = tree
	}
	tvIndex := 0
	tv.SyncToSrc(&tvIndex, true, 0)
	tv.NeedsLayout()
	return tv
}

// SetSyncNode sets the sync source node that we are viewing,
// and syncs the view of its tree.  It is called routinely
// via SyncToSrc during tree updating.
// It uses ki Config mechanism to perform minimal updates to
// remain in sync.
func (tv *TreeView) SetSyncNode(sk ki.Ki, tvIndex *int, init bool, depth int) {
	if tv.SyncNode != sk {
		tv.SyncNode = sk
	}
	tv.SyncToSrc(tvIndex, init, depth)
}

// ReSync resynchronizes the view relative to the underlying nodes
// and forces a full rerender
func (tv *TreeView) ReSync() {
	tvIndex := tv.ViewIndex
	tv.SyncToSrc(&tvIndex, false, 0)
}

// SyncToSrc updates the view tree to match the sync tree, using
// ConfigChildren to maximally preserve existing tree elements.
// init means we are doing initial build, and depth tracks depth
// (only during init).
func (tv *TreeView) SyncToSrc(tvIndex *int, init bool, depth int) {
	// pr := prof.Start("TreeView.SyncToSrc")
	// defer pr.End()
	sk := tv.SyncNode
	nm := "tv_" + sk.Name()
	tv.SetName(nm)
	tv.ViewIndex = *tvIndex
	*tvIndex++
	// tvPar := tv.TreeViewParent()
	// if tvPar != nil {
	// 	if init && depth >= tv.RootView.OpenDepth {
	// 		tv.SetClosed(true)
	// 	}
	// }
	vcprop := "view-closed"
	skids := *sk.Children()
	tnl := make(ki.Config, 0, len(skids))
	typ := tv.This().KiType()
	for _, skid := range skids {
		tnl.Add(typ, "tv_"+skid.Name())
	}
	mods := tv.ConfigChildren(tnl)
	idx := 0
	for _, skid := range *sk.Children() {
		if len(tv.Kids) <= idx {
			break
		}
		vk := AsTreeView(tv.Kids[idx])
		vk.SetSyncNode(skid, tvIndex, init, depth+1)
		if mods {
			if vcp, ok := skid.PropInherit(vcprop, ki.NoInherit); ok {
				if vc, err := laser.ToBool(vcp); vc && err != nil {
					vk.SetClosed(true)
				}
			}
		}
		idx++
	}
	if !sk.HasChildren() {
		tv.SetClosed(true)
	}
	if mods {
		tv.Update()
		tv.TreeViewChanged(nil)
	}
	tv.NeedsLayout()
}

// Label returns the display label for this node,
// satisfying the Labeler interface
func (tv *TreeView) Label() string {
	if tv.SyncNode != nil {
		if lbl, has := gi.ToLabeler(tv.SyncNode); has {
			return lbl
		}
		return tv.SyncNode.Name()
	}
	if tv.Text != "" {
		return tv.Text
	}
	return tv.Nm
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
	if inact := tv.SyncNode.Prop("ReadOnly"); inact != nil {
		if bo, err := laser.ToBool(inact); bo && err == nil {
			tv.SetReadOnly(true)
		}
	}
	return tv.IsReadOnly()
}

// SelectedSyncNodes returns a slice of the currently-selected
// sync source nodes in the entire tree view
func (tv *TreeView) SelectedSyncNodes() ki.Slice {
	var sn ki.Slice
	sl := tv.SelectedViews()
	for _, v := range sl {
		sn = append(sn, v.AsTreeView().SyncNode)
	}
	return sn
}

// FindSyncNode finds TreeView node for given source node,
// or nil if not found
func (tv *TreeView) FindSyncNode(kn ki.Ki) *TreeView {
	var ttv *TreeView
	tv.WidgetWalkPre(func(wi gi.Widget, wb *gi.WidgetBase) bool {
		tvki := AsTreeView(wi)
		if tvki != nil {
			if tvki.SyncNode == kn {
				ttv = tvki
				return ki.Break
			}
		}
		return ki.Continue
	})
	return ttv
}

// InsertAfter inserts a new node in the tree
// after this node, at the same (sibling) level,
// prompting for the type of node to insert.
// If SyncNode is set, operates on Sync Tree.
func (tv *TreeView) InsertAfter() { //gti:add
	tv.InsertAt(1, "Insert After")
}

// InsertBefore inserts a new node in the tree
// before this node, at the same (sibling) level,
// prompting for the type of node to insert
// If SyncNode is set, operates on Sync Tree.
func (tv *TreeView) InsertBefore() { //gti:add
	tv.InsertAt(0, "Insert Before")
}

func (tv *TreeView) AddTreeNodes(rel, myidx int, typ *gti.Type, n int) {
	var stv *TreeView
	for i := 0; i < n; i++ {
		nm := fmt.Sprintf("new-%v-%v", typ.IDName, myidx+rel+i)
		nki := tv.InsertNewChild(typ, myidx+i, nm)
		ntv := AsTreeView(nki)
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

func (tv *TreeView) AddSyncNodes(rel, myidx int, typ *gti.Type, n int) {
	parent := tv.SyncNode
	var ski ki.Ki
	for i := 0; i < n; i++ {
		nm := fmt.Sprintf("new-%v-%v", typ.IDName, myidx+rel+i)
		nki := parent.InsertNewChild(typ, myidx+i, nm)
		if i == n-1 {
			ski = nki
		}
	}
	tv.SendChangeEventReSync(nil)
	if ski != nil {
		if tvk := tv.ChildByName("tv_"+ski.Name(), 0); tvk != nil {
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
		typ = tv.SyncNode.This().BaseType()
	}
	d := gi.NewBody().AddTitle(actNm).AddText("Number and type of items to insert:")
	nd := &gi.NewItemsData{Number: 1, Type: typ}
	sg := NewStructView(d).SetStruct(nd).StructGrid()
	ki.ChildByType[*gi.Chooser](sg, ki.Embeds).SetTypes(gti.AllEmbeddersOf(typ)...).SetCurrentIndex(0)
	d.AddBottomBar(func(parent gi.Widget) {
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
	d.NewDialog(tv).Run()
}

// AddChildNode adds a new child node to this one in the tree,
// prompting the user for the type of node to add
// If SyncNode is set, operates on Sync Tree.
func (tv *TreeView) AddChildNode() { //gti:add
	ttl := "Add child"
	typ := tv.This().BaseType()
	if tv.SyncNode != nil {
		typ = tv.SyncNode.This().BaseType()
	}
	d := gi.NewBody().AddTitle(ttl).AddText("Number and type of items to insert:")
	nd := &gi.NewItemsData{Number: 1, Type: typ}
	sg := NewStructView(d).SetStruct(nd).StructGrid()
	ki.ChildByType[*gi.Chooser](sg, ki.Embeds).SetTypes(gti.AllEmbeddersOf(typ)...).SetCurrentIndex(0)
	d.AddBottomBar(func(parent gi.Widget) {
		d.AddCancel(parent)
		d.AddOK(parent).OnClick(func(e events.Event) {
			if tv.SyncNode != nil {
				tv.AddSyncNodes(0, 0, nd.Type, nd.Number)
			} else {
				tv.AddTreeNodes(0, 0, nd.Type, nd.Number)
			}
		})
	})
	d.NewDialog(tv).Run()
}

// DeleteNode deletes the tree node or sync node corresponding
// to this view node in the sync tree.
// If SyncNode is set, operates on Sync Tree.
func (tv *TreeView) DeleteNode() { //gti:add
	ttl := "Delete"
	if tv.IsRoot(ttl) {
		return
	}
	tv.Close()
	if tv.MoveDown(events.SelectOne) == nil {
		tv.MoveUp(events.SelectOne)
	}
	if tv.SyncNode != nil {
		tv.SyncNode.Delete()
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
func (tv *TreeView) Duplicate() { //gti:add
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
	nm := fmt.Sprintf("%v_Copy", tv.Name())
	tv.Unselect()
	nwkid := tv.Clone()
	nwkid.SetName(nm)
	ntv := AsTreeView(nwkid)
	parent.InsertChild(nwkid, myidx+1)
	ntv.Update()
	parent.Update()
	parent.TreeViewChanged(nil)
	// ntv.SelectAction(events.SelectOne)
}

func (tv *TreeView) DuplicateSync() {
	sk := tv.SyncNode
	tvparent := AsTreeView(tv.Par)
	parent := tvparent.SyncNode
	if parent == nil {
		log.Printf("TreeView %v nil SyncNode in: %v\n", tv, tvparent.Path())
		return
	}
	myidx := sk.IndexInParent()
	if myidx < 0 {
		return
	}
	nm := fmt.Sprintf("%v_Copy", sk.Name())
	nwkid := sk.Clone()
	nwkid.SetName(nm)
	parent.InsertChild(nwkid, myidx+1)
	tvparent.SendChangeEventReSync(nil)
	if tvk := tvparent.ChildByName("tv_"+nm, 0); tvk != nil {
		stv := AsTreeView(tvk)
		stv.SelectAction(events.SelectOne)
	}
}

// EditNode pulls up a StructViewDialog window on the node.
// If SyncNode is set, operates on Sync Tree.
func (tv *TreeView) EditNode() { //gti:add
	if tv.SyncNode != nil {
		tynm := tv.SyncNode.KiType().Name
		d := gi.NewBody().AddTitle(tynm)
		NewStructView(d).SetStruct(tv.SyncNode).SetReadOnly(tv.IsReadOnly())
		d.NewFullDialog(tv).Run()
	} else {
		tynm := tv.KiType().Name
		d := gi.NewBody().AddTitle(tynm)
		NewStructView(d).SetStruct(tv.This()).SetReadOnly(tv.IsReadOnly())
		d.NewFullDialog(tv).Run()
	}
}

// InspectNode pulls up a new Inspector window on the node.
// If SyncNode is set, operates on Sync Tree.
func (tv *TreeView) InspectNode() { //gti:add
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
	*md = append(*md, mimedata.NewTextData(src.PathFrom(sroot)))
	var buf bytes.Buffer
	err := ki.WriteNewJSON(src, &buf)
	if err == nil {
		*md = append(*md, &mimedata.Data{Type: fi.DataJson, Data: buf.Bytes()})
	} else {
		gi.ErrorSnackbar(tv, err, "Error encoding node")
	}
}

// SyncNodesFromMimeData creates a slice of Ki node(s)
// from given mime data and also a corresponding slice
// of original paths.
func (tv *TreeView) SyncNodesFromMimeData(md mimedata.Mimes) (ki.Slice, []string) {
	ni := len(md) / 2
	sl := make(ki.Slice, 0, ni)
	pl := make([]string, 0, ni)
	for _, d := range md {
		if d.Type == fi.DataJson {
			nki, err := ki.ReadNewJSON(bytes.NewReader(d.Data))
			if err == nil {
				sl = append(sl, nki)
			} else {
				gi.ErrorSnackbar(tv, err, "Error loading node")
			}
		} else if d.Type == fi.TextPlain { // paths
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
	tv.SyncNode.CopyFrom(sl[0])
	tv.NeedsLayout()
	tv.SendChangeEvent(nil)
}

// PasteAtSync inserts object(s) from mime data at rel position to this node.
// If another item with the same name already exists, it will
// append _Copy on the name of the inserted objects
func (tv *TreeView) PasteAtSync(md mimedata.Mimes, mod events.DropMods, rel int, actNm string) {
	sk := tv.SyncNode
	sl, pl := tv.NodesFromMimeData(md)
	tvparent := AsTreeView(tv.Par)
	parent := sk.Parent()
	myidx := sk.IndexInParent()
	if myidx < 0 {
		return
	}
	myidx += rel
	sroot := tv.RootView.SyncNode
	sz := len(sl)
	var selKi ki.Ki
	for i, ns := range sl {
		orgpath := pl[i]
		if mod != events.DropMove {
			if cn := parent.ChildByName(ns.Name(), 0); cn != nil {
				ns.SetName(ns.Name() + "_Copy")
			}
		}
		parent.InsertChild(ns, myidx+i)
		npath := ns.PathFrom(sroot)
		if mod == events.DropMove && npath == orgpath { // we will be nuked immediately after drag
			ns.SetName(ns.Name() + TreeViewTempMovedTag) // special keyword :)
		}
		if i == sz-1 {
			selKi = ns
		}
	}
	tvparent.SendChangeEventReSync(nil)
	if selKi != nil {
		if tvk := tvparent.ChildByName("tv_"+selKi.Name(), myidx); tvk != nil {
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
		sk.AddChild(ns)
	}
	tv.SendChangeEventReSync(nil)
}

// CutSync copies to goosi.Clipboard and deletes selected items.
func (tv *TreeView) CutSync() {
	tv.Copy(false)
	sels := tv.SelectedSyncNodes()
	tv.UnselectAll()
	for _, sn := range sels {
		sn.Delete()
	}
	tv.SendChangeEventReSync(nil)
}

// DropDeleteSourceSync handles delete source event for DropMove case, for Sync
func (tv *TreeView) DropDeleteSourceSync(de *events.DragDrop) {
	md := de.Data.(mimedata.Mimes)
	sroot := tv.RootView.SyncNode
	for _, d := range md {
		if d.Type != fi.TextPlain { // link
			continue
		}
		path := string(d.Data)
		sn := sroot.FindPath(path)
		if sn != nil {
			sn.Delete()
		}
		sn = sroot.FindPath(path + TreeViewTempMovedTag)
		if sn != nil {
			psplt := strings.Split(path, "/")
			orgnm := psplt[len(psplt)-1]
			sn.SetName(orgnm)
			_, swb := gi.AsWidget(sn)
			swb.NeedsRender()
		}
	}
	tv.SendChangeEventReSync(nil)
}
