// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"bytes"
	"fmt"
	"log"
	"log/slog"

	"goki.dev/fi"
	"goki.dev/gi/v2/gi"
	"goki.dev/girl/states"
	"goki.dev/goosi/events"
	"goki.dev/goosi/mimedata"
	"goki.dev/gti"
	"goki.dev/ki/v2"
	"goki.dev/laser"
)

// note: see this file has all the SyncNode specific
// functions for TreeView.

// SyncRootNode sets the root view to the root
// of the sync source node for this TreeView,
// and syncs the rest of the tree to match.
// Calls ki.UniquifyNamesAll on source tree to
// ensure that node names are unique
// which is essential for proper viewing!
func (tv *TreeView) SyncRootNode(sk ki.Ki) *TreeView {
	updt := false
	ki.UniquifyNamesAll(sk)
	if tv.SyncNode != sk {
		updt = tv.UpdateStart()
		tv.SyncNode = sk
	}
	tvIdx := 0
	tv.SyncToSrc(&tvIdx, true, 0)
	tv.UpdateEndLayout(updt)
	return tv
}

// SetSyncNode sets the sync source node that we are viewing,
// and syncs the view of its tree.  It is called routinely
// via SyncToSrc during tree updating.
// It uses ki Config mechanism to perform minimal updates to
// remain in sync.
func (tv *TreeView) SetSyncNode(sk ki.Ki, tvIdx *int, init bool, depth int) {
	updt := false
	if tv.SyncNode != sk {
		updt = tv.UpdateStart()
		tv.SyncNode = sk
	}
	tv.SyncToSrc(tvIdx, init, depth)
	tv.UpdateEnd(updt)
}

// ReSync resynchronizes the view relative to the underlying nodes
// and forces a full rerender
func (tv *TreeView) ReSync() {
	tvIdx := tv.ViewIdx
	tv.SyncToSrc(&tvIdx, false, 0)
}

// SyncToSrc updates the view tree to match the sync tree, using
// ConfigChildren to maximally preserve existing tree elements.
// init means we are doing initial build, and depth tracks depth
// (only during init).
func (tv *TreeView) SyncToSrc(tvIdx *int, init bool, depth int) {
	// pr := prof.Start("TreeView.SyncToSrc")
	// defer pr.End()
	sk := tv.SyncNode
	nm := "tv_" + sk.Name()
	tv.SetName(nm)
	tv.ViewIdx = *tvIdx
	(*tvIdx)++
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
	mods, updt := tv.ConfigChildren(tnl) // false = don't use unique names -- needs to!
	if mods {
		tv.SetNeedsLayout(true)
		// fmt.Printf("got mod on %v\n", tv.Path())
	}
	idx := 0
	for _, skid := range *sk.Children() {
		if len(tv.Kids) <= idx {
			break
		}
		vk := AsTreeView(tv.Kids[idx])
		vk.SetSyncNode(skid, tvIdx, init, depth+1)
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
	tv.UpdateEndLayout(updt)
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
	return tv.Text
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
	if inact, err := tv.SyncNode.PropTry("ReadOnly"); err == nil {
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
	updt := tv.UpdateStart()
	var stv *TreeView
	for i := 0; i < n; i++ {
		nm := fmt.Sprintf("new-%v-%v", typ.IDName, myidx+rel+i)
		tv.SetChildAdded()
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
	tv.UpdateEndLayout(updt)
	if stv != nil {
		stv.SelectAction(events.SelectOne)
	}
}

func (tv *TreeView) AddSyncNodes(rel, myidx int, typ *gti.Type, n int) {
	par := tv.SyncNode
	updt := par.UpdateStart()
	var ski ki.Ki
	for i := 0; i < n; i++ {
		nm := fmt.Sprintf("new-%v-%v", typ.IDName, myidx+rel+i)
		par.SetChildAdded()
		nki := par.InsertNewChild(typ, myidx+i, nm)
		if i == n-1 {
			ski = nki
		}
	}
	tv.SendChangeEventReSync(nil)
	par.UpdateEnd(updt)
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
	myidx, ok := tv.IndexInParent()
	if !ok {
		return
	}
	myidx += rel
	typ := tv.This().BaseType()
	if tv.SyncNode != nil {
		typ = tv.SyncNode.This().BaseType()
	}
	d := gi.NewBody().AddTitle(actNm).AddText("Number and Type of Items to Insert:")
	nd := &gi.NewItemsData{Number: 1, Type: typ}
	sg := NewStructView(d).SetStruct(nd).StructGrid()
	ki.ChildByType[*gi.Chooser](sg, true).SetTypes(gti.AllEmbeddersOf(typ), true, true, 50)
	d.AddBottomBar(func(pw gi.Widget) {
		d.AddCancel(pw)
		d.AddOk(pw).OnClick(func(e events.Event) {
			par := AsTreeView(tv.Par)
			if tv.SyncNode != nil {
				par.AddSyncNodes(rel, myidx, nd.Type, nd.Number)
			} else {
				par.AddTreeNodes(rel, myidx, nd.Type, nd.Number)
			}
		})
	})
	d.NewDialog(tv).Run()
}

// AddChildNode adds a new child node to this one in the tree,
// prompting the user for the type of node to add
// If SyncNode is set, operates on Sync Tree.
func (tv *TreeView) AddChildNode() { //gti:add
	ttl := "Add Child"
	typ := tv.This().BaseType()
	if tv.SyncNode != nil {
		typ = tv.SyncNode.This().BaseType()
	}
	d := gi.NewBody().AddTitle(ttl).AddText("Number and Type of Items to Add:")
	nd := &gi.NewItemsData{Number: 1, Type: typ}
	sg := NewStructView(d).SetStruct(nd).StructGrid()
	ki.ChildByType[*gi.Chooser](sg, true).SetTypes(gti.AllEmbeddersOf(typ), true, true, 50)
	d.AddBottomBar(func(pw gi.Widget) {
		d.AddCancel(pw)
		d.AddOk(pw).OnClick(func(e events.Event) {
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
		tv.SyncNode.Delete(true)
		tv.SendChangeEventReSync(nil)
	} else {
		par := AsTreeView(tv.Par)
		updt := par.UpdateStart()
		tv.Delete(true)
		par.Update()
		par.TreeViewChanged(nil)
		par.UpdateEndLayout(updt)
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
	par := AsTreeView(tv.Par)
	myidx, ok := tv.IndexInParent()
	if !ok {
		return
	}
	updt := par.UpdateStart()
	nm := fmt.Sprintf("%v_Copy", tv.Name())
	tv.Unselect()
	nwkid := tv.Clone()
	nwkid.SetName(nm)
	par.SetChildAdded()
	par.InsertChild(nwkid, myidx+1)
	ntv := AsTreeView(nwkid)
	ntv.Update()
	par.Update()
	par.TreeViewChanged(nil)
	par.UpdateEndLayout(updt)
	ntv.SelectAction(events.SelectOne)
}

func (tv *TreeView) DuplicateSync() {
	sk := tv.SyncNode
	tvpar := AsTreeView(tv.Par)
	par := tvpar.SyncNode
	if par == nil {
		log.Printf("TreeView %v nil SyncNode in: %v\n", tv, tvpar.Path())
		return
	}
	myidx, ok := sk.IndexInParent()
	if !ok {
		return
	}
	updt := par.UpdateStart()
	nm := fmt.Sprintf("%v_Copy", sk.Name())
	nwkid := sk.Clone()
	nwkid.SetName(nm)
	par.SetChildAdded()
	par.InsertChild(nwkid, myidx+1)
	par.UpdateEnd(updt)
	tvpar.SendChangeEventReSync(nil)
	if tvk := tvpar.ChildByName("tv_"+nm, 0); tvk != nil {
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
		NewStructView(d).SetStruct(tv.SyncNode)
		d.NewFullDialog(tv).Run()
	} else {
		tynm := tv.KiType().Name
		d := gi.NewBody().AddTitle(tynm)
		NewStructView(d).SetStruct(tv.This())
		d.NewFullDialog(tv).Run()
	}
}

// InspectNode pulls up a new Inspector window on the node.
// If SyncNode is set, operates on Sync Tree.
func (tv *TreeView) InspectNode() { //gti:add
	if tv.SyncNode != nil {
		InspectorDialog(tv.SyncNode)
	} else {
		InspectorDialog(tv)
	}
}

// MimeDataSync adds mimedata for this node: a text/plain of the Path,
// and an application/json of the sync node.
// satisfies Clipper.MimeData interface
func (tv *TreeView) MimeDataSync(md *mimedata.Mimes) {
	sroot := tv.RootView.SyncNode
	src := tv.SyncNode
	*md = append(*md, mimedata.NewTextData(src.PathFrom(sroot)))
	var buf bytes.Buffer
	err := ki.WriteNewJSON(src, &buf)
	if err == nil {
		*md = append(*md, &mimedata.Data{Type: fi.DataJson, Data: buf.Bytes()})
	} else {
		slog.Error("giv.TreeView MimeData Write JSON error:", err)
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
				slog.Error("giv.TreeView SyncNodesFromMimeData: JSON load error:", err)
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
	updt := tv.UpdateStart()
	tv.SyncNode.CopyFrom(sl[0])
	tv.UpdateEndLayout(updt)
	tv.SendChangeEvent(nil)
}

// PasteAt inserts object(s) from mime data at rel position to this node.
// If another item with the same name already exists, it will
// append _Copy on the name of the inserted objects
func (tv *TreeView) PasteAtSync(md mimedata.Mimes, mod events.DropMods, rel int, actNm string) {
	sk := tv.SyncNode
	sl, pl := tv.NodesFromMimeData(md)
	tvpar := AsTreeView(tv.Par)
	par := sk.Parent()
	myidx, ok := sk.IndexInParent()
	if !ok {
		return
	}
	myidx += rel
	sroot := tv.RootView.SyncNode
	updt := par.UpdateStart()
	sz := len(sl)
	var selKi ki.Ki
	for i, ns := range sl {
		orgpath := pl[i]
		if mod != events.DropMove {
			if cn := par.ChildByName(ns.Name(), 0); cn != nil {
				ns.SetName(ns.Name() + "_Copy")
			}
		}
		par.SetChildAdded()
		par.InsertChild(ns, myidx+i)
		npath := ns.PathFrom(sroot)
		if mod == events.DropMove && npath == orgpath { // we will be nuked immediately after drag
			ns.SetName(ns.Name() + TreeViewTempMovedTag) // special keyword :)
		}
		if i == sz-1 {
			selKi = ns
		}
	}
	par.UpdateEnd(updt)
	tvpar.SendChangeEventReSync(nil)
	if selKi != nil {
		if tvk := tvpar.ChildByName("tv_"+selKi.Name(), myidx); tvk != nil {
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
	updt := sk.UpdateStart()
	sk.SetChildAdded()
	for _, ns := range sl {
		sk.AddChild(ns)
	}
	sk.UpdateEnd(updt)
	tv.SendChangeEventReSync(nil)
}

// CutSync copies to clip.Board and deletes selected items.
// satisfies gi.Clipper interface and can be overridden by subtypes
func (tv *TreeView) CutSync() {
	tv.Copy(false)
	sels := tv.SelectedSyncNodes()
	tv.UnselectAll()
	for _, sn := range sels {
		sn.Delete(true)
	}
	tv.SendChangeEventReSync(nil)
}
