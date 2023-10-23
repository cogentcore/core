// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"bytes"
	"fmt"
	"log"
	"log/slog"

	"goki.dev/gi/v2/gi"
	"goki.dev/girl/states"
	"goki.dev/goosi/events"
	"goki.dev/goosi/mimedata"
	"goki.dev/gti"
	"goki.dev/ki/v2"
	"goki.dev/laser"
	"goki.dev/pi/v2/filecat"
)

// note: see this file has all the SyncNode specific
// functions for TreeView.

// SyncRootNode sets the root view to the root
// of the sync source node for this TreeView,
// and syncs the rest of the tree to match.
// Calls ki.UniquifyNamesAll on source tree to
// ensure that node names are unique
// which is essential for proper viewing!
func (tv *TreeView) SyncRootNode(sk ki.Ki) {
	updt := false
	ki.UniquifyNamesAll(sk)
	if tv.SyncNode != sk {
		updt = tv.UpdateStart()
		tv.SyncNode = sk
	}
	tv.RootView = tv
	tvIdx := 0
	tv.SyncToSrc(&tvIdx, true, 0)
	tv.UpdateEndLayout(updt)
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

// SyncToSrc updates the view tree to match the source tree, using
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
	tvPar := tv.TreeViewParent()
	if tvPar != nil {
		tv.RootView = tvPar.RootView
		// if init && depth >= tv.RootView.OpenDepth {
		// 	tv.SetClosed(true)
		// }
	}
	vcprop := "view-closed"
	skids := *sk.Children()
	tnl := make(ki.Config, 0, len(skids))
	typ := tv.This().KiType()
	for _, skid := range skids {
		tnl.Add(typ, "tv_"+skid.Name())
	}
	mods, updt := tv.ConfigChildren(tnl) // false = don't use unique names -- needs to!
	if mods {
		tv.SetNeedsLayout()
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
	return tv.Name()
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
			tv.SetState(true, states.Disabled)
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
		sn = append(sn, v.SyncNode)
	}
	return sn
}

// FindSyncNode finds TreeView node for given source node,
// or nil if not found
func (tv *TreeView) FindSyncNode(kn ki.Ki) *TreeView {
	var ttv *TreeView
	tv.WalkPre(func(k ki.Ki) bool {
		tvki := AsTreeView(k)
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

// SrcInsertAfter inserts a new node in the source tree
// after this node, at the same (sibling) level,
// prompting for the type of node to insert
func (tv *TreeView) SrcInsertAfter() {
	tv.SrcInsertAt(1, "Insert After")
}

// SrcInsertBefore inserts a new node in the source tree
// before this node, at the same (sibling) level,
// prompting for the type of node to insert
func (tv *TreeView) SrcInsertBefore() {
	tv.SrcInsertAt(0, "Insert Before")
}

// SrcInsertAt inserts a new node in the source tree
// at given relative offset from this node,
// at the same (sibling) level,
// prompting for the type of node to insert
func (tv *TreeView) SrcInsertAt(rel int, actNm string) {
	if tv.SyncNode == nil {
		return
	}
	if tv.IsRoot(actNm) {
		return
	}
	sk := tv.SyncNode
	myidx, ok := sk.IndexInParent()
	if !ok {
		return
	}
	myidx += rel
	gi.NewKiDialog(tv, gi.DlgOpts{Title: actNm, Prompt: "Number and Type of Items to Insert:"}, sk.BaseType(), func(dlg *gi.Dialog) {
		if !dlg.Accepted {
			return
		}
		par := tv.SyncNode
		typ := dlg.Data.(*gti.Type)
		n := 1 // todo
		updt := par.UpdateStart()
		var ski ki.Ki
		for i := 0; i < n; i++ {
			nm := fmt.Sprintf("New%v%v", typ.Name, myidx+rel+i)
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
	})
}

// SrcAddChild adds a new child node to this one in the source tree,
// prompting the user for the type of node to add
func (tv *TreeView) SrcAddChild() {
	ttl := "Add Child"
	sk := tv.SyncNode
	if sk == nil {
		log.Printf("TreeView %v nil SyncNode in: %v\n", ttl, tv.Path())
		return
	}
	gi.NewKiDialog(tv, gi.DlgOpts{Title: ttl, Prompt: "Number and Type of Items to Add:"}, sk.BaseType(),
		func(dlg *gi.Dialog) {
			if !dlg.Accepted {
				return
			}
			sk := tv.SyncNode
			typ := dlg.Data.(*gti.Type)
			n := 1 // todo
			updt := sk.UpdateStart()
			sk.SetChildAdded()
			var ski ki.Ki
			for i := 0; i < n; i++ {
				nm := fmt.Sprintf("New%v%v", typ.Name, i)
				nki := sk.NewChild(typ, nm)
				if i == n-1 {
					ski = nki
				}
				// tv.TreeViewSig.Emit(tv.RootView.This(), int64(TreeViewInserted), nki.This())
			}
			tv.SendChangeEventReSync(nil)
			sk.UpdateEnd(updt)
			if ski != nil {
				tv.Open()
				if tvk := tv.ChildByName("tv_"+ski.Name(), 0); tvk != nil {
					stv := AsTreeView(tvk)
					stv.SelectAction(events.SelectOne)
				}
			}
		})
}

// SrcDelete deletes the source node corresponding
// to this view node in the source tree.
func (tv *TreeView) SrcDelete() {
	ttl := "Delete"
	if tv.IsRoot(ttl) {
		return
	}
	if tv.MoveDown(events.SelectOne) == nil {
		tv.MoveUp(events.SelectOne)
	}
	sk := tv.SyncNode
	if sk == nil {
		log.Printf("TreeView %v nil SyncNode in: %v\n", ttl, tv.Path())
		return
	}
	sk.Delete(true)
	// tv.TreeViewSig.Emit(tv.RootView.This(), int64(TreeViewDeleted), sk.This())
	tv.SendChangeEventReSync(nil)
}

// SrcDuplicate duplicates the source node corresponding to this view node in
// the source tree, and inserts the duplicate after this node (as a new
// sibling)
func (tv *TreeView) SrcDuplicate() {
	ttl := "TreeView Duplicate"
	if tv.IsRoot(ttl) {
		return
	}
	sk := tv.SyncNode
	if sk == nil {
		log.Printf("TreeView %v nil SyncNode in: %v\n", ttl, tv.Path())
		return
	}
	if tv.Par == nil {
		return
	}
	tvpar := AsTreeView(tv.Par)
	par := tvpar.SyncNode
	if par == nil {
		log.Printf("TreeView %v nil SyncNode in: %v\n", ttl, tvpar.Path())
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
	// tv.RootView.TreeViewSig.Emit(tv.RootView.This(), int64(TreeViewInserted), nwkid.This())
	par.UpdateEnd(updt)
	tvpar.SendChangeEventReSync(nil)
	if tvk := tvpar.ChildByName("tv_"+nm, 0); tvk != nil {
		stv := AsTreeView(tvk)
		stv.SelectAction(events.SelectOne)
	}
}

// SrcEdit pulls up a StructViewDialog window on the source object
// viewed by this node
func (tv *TreeView) SrcEdit() {
	if tv.SyncNode == nil {
		slog.Error("TreeView SrcEdit nil SyncNode:", tv)
		return
	}
	// tynm := laser.NonPtrType(tv.SyncNode.KiType()).Name()
	StructViewDialog(tv, DlgOpts{Title: "type"}, tv.SyncNode, nil)
}

// SrcGoGiEditor pulls up a new GoGiEditor window on the source
// object viewed by this node
func (tv *TreeView) SrcGoGiEditor() {
	if tv.SyncNode == nil {
		slog.Error("TreeView SrcGoGiEditor nil SyncNode:", tv)
		return
	}
	GoGiEditorDialog(tv.SyncNode)
}

// MimeDataSync adds mimedata for this node: a text/plain of the Path,
// and an application/json of the source node.
// satisfies Clipper.MimeData interface
func (tv *TreeView) MimeDataSync(md *mimedata.Mimes) {
	sroot := tv.RootView.SyncNode
	src := tv.SyncNode
	*md = append(*md, mimedata.NewTextData(src.PathFrom(sroot)))
	var buf bytes.Buffer
	err := ki.WriteNewJSON(src, &buf)
	if err == nil {
		*md = append(*md, &mimedata.Data{Type: filecat.DataJson, Data: buf.Bytes()})
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
		if d.Type == filecat.DataJson {
			nki, err := ki.ReadNewJSON(bytes.NewReader(d.Data))
			if err == nil {
				sl = append(sl, nki)
			} else {
				slog.Error("giv.TreeView SyncNodesFromMimeData: JSON load error:", err)
			}
		} else if d.Type == filecat.TextPlain { // paths
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
