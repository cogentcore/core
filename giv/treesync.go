// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"goki.dev/gi/v2/gi"
	"goki.dev/girl/states"
	"goki.dev/ki/v2"
	"goki.dev/laser"
)

// TreeSyncView is a TreeView that synchronizes with another
// Ki tree structure.  Must establish signaling communication
// between the two nodes to keep them synchronized.
type TreeSyncView struct {
	TreeView

	// Ki Node that this widget is viewing in the tree -- the source
	SrcNode ki.Ki `copy:"-" json:"-" xml:"-"`

	// SyncConnectFun is function called to connect given
	// TreeSyncView with its SrcNode, which has just been set.
	SyncConnectFun func(tv *TreeSyncView)

	// if the object we're viewing has its own CtxtMenu property defined,
	// should we also still show the view's own context menu?
	ShowViewCtxtMenu bool
}

// SetRootNode sets the root view to the root of the source node that we are
// viewing, and builds-out the view of its tree.
// Calls ki.UniquifyNamesAll on source tree to ensure that node names are unique
// which is essential for proper viewing!
func (tv *TreeSyncView) SetRootNode(sk ki.Ki) {
	updt := false
	ki.UniquifyNamesAll(sk)
	if tv.SrcNode != sk {
		updt = tv.UpdateStart()
		tv.SrcNode = sk
		if tv.SyncConnectFun != nil {
			tv.SyncConnectFun(tv)
		}
	}
	tv.RootView = &tv.TreeView // todo: should be the interface
	tvIdx := 0
	tv.SyncToSrc(&tvIdx, true, 0)
	tv.UpdateEnd(updt)
}

// SetSrcNode sets the source node that we are viewing,
// and builds-out the view of its tree.  It is called routinely
// via SyncToSrc during tree updating.
func (tv *TreeSyncView) SetSrcNode(sk ki.Ki, tvIdx *int, init bool, depth int) {
	updt := false
	if tv.SrcNode != sk {
		updt = tv.UpdateStart()
		tv.SrcNode = sk
		// sk.NodeSignal().Connect(tv.This(), SrcNodeSignalFunc) // we recv signals from source
	}
	tv.SyncToSrc(tvIdx, init, depth)
	tv.UpdateEnd(updt)
}

// ReSync resynchronizes the view relative to the underlying nodes
// and forces a full rerender
func (tv *TreeSyncView) ReSync() {
	tvIdx := tv.ViewIdx
	tv.SyncToSrc(&tvIdx, false, 0)
}

// SyncToSrc updates the view tree to match the source tree, using
// ConfigChildren to maximally preserve existing tree elements.
// init means we are doing initial build, and depth tracks depth
// (only during init).
func (tv *TreeSyncView) SyncToSrc(tvIdx *int, init bool, depth int) {
	// pr := prof.Start("TreeSyncView.SyncToSrc")
	// defer pr.End()
	sk := tv.SrcNode
	nm := "tv_" + sk.Name()
	tv.SetName(nm)
	tv.ViewIdx = *tvIdx
	(*tvIdx)++
	// tvPar := tv.TreeViewParent()
	// if tvPar != nil {
	// 	tv.RootView = tvPar.RootView
	// 	if init && depth >= tv.RootView.OpenDepth {
	// 		tv.SetClosed()
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
		tv.SetNeedsLayout()
		// fmt.Printf("got mod on %v\n", tv.Path())
	}
	idx := 0
	for _, skid := range *sk.Children() {
		if len(tv.Kids) <= idx {
			break
		}
		vk := AsTreeView(tv.Kids[idx])
		// vk.SetSrcNode(skid, tvIdx, init, depth+1)
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

// Label returns the display label for this node, satisfying the Labeler interface
func (tv *TreeSyncView) Label() string {
	if lbl, has := gi.ToLabeler(tv.SrcNode); has {
		return lbl
	}
	return tv.SrcNode.Name()
}

// UpdateInactive updates the Inactive state based on SrcNode.
// Returns true if inactive.
// The inactivity of individual nodes only affects display properties
// typically, and not overall functional behavior, which is controlled by
// inactivity of the root node (i.e, make the root inactive
// to make entire tree read-only and non-modifiable)
func (tv *TreeSyncView) UpdateInactive() bool {
	tv.SetState(false, states.Disabled)
	if tv.SrcNode == nil {
		tv.SetState(true, states.Disabled)
	} else {
		if inact, err := tv.SrcNode.PropTry("inactive"); err == nil {
			if bo, err := laser.ToBool(inact); bo && err == nil {
				tv.SetState(true, states.Disabled)
			}
		}
	}
	return tv.IsDisabled()
}

/*
// SelectedSrcNodes returns a slice of the currently-selected source nodes
// in the entire tree view
func (tv *TreeSyncView) SelectedSrcNodes() ki.Slice {
	sn := make(ki.Slice, 0)
	sl := tv.SelectedViews()
	for _, v := range sl {
		sn = append(sn, v.SrcNode)
	}
	return sn
}

// FindSrcNode finds TreeView node for given source node,
// or nil if not found
func (tv *TreeSyncView) FindSrcNode(kn ki.Ki) *TreeSyncView {
	var ttv *TreeSyncView
	tv.WalkPre(func(k ki.Ki) bool {
		tvki := AsTreeView(k)
		if tvki != nil {
			if tvki.SrcNode == kn {
				ttv = tvki
				return ki.Break
			}
		}
		return ki.Continue
	})
	return ttv
}

// SrcInsertAfter inserts a new node in the source tree after this node, at
// the same (sibling) level, prompting for the type of node to insert
func (tv *TreeSyncView) SrcInsertAfter() {
	tv.SrcInsertAt(1, "Insert After")
}

// SrcInsertBefore inserts a new node in the source tree before this node, at
// the same (sibling) level, prompting for the type of node to insert
func (tv *TreeSyncView) SrcInsertBefore() {
	tv.SrcInsertAt(0, "Insert Before")
}

// SrcInsertAt inserts a new node in the source tree at given relative offset
// from this node, at the same (sibling) level, prompting for the type of node to insert
func (tv *TreeSyncView) SrcInsertAt(rel int, actNm string) {
	if tv.IsRoot(actNm) {
		return
	}
	sk := tv.SrcNode
	if sk == nil {
		log.Printf("TreeView %v nil SrcNode in: %v\n", actNm, tv.Path())
		return
	}
	myidx, ok := sk.IndexInParent()
	if !ok {
		return
	}
	myidx += rel
	gi.NewKiDialog(tv, gi.DlgOpts{Title: actNm, Prompt: "Number and Type of Items to Insert:"}, sk.BaseType(),
		func(dlg *gi.Dialog) {
			if dlg.Accepted {
				par := tv.SrcNode
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
					// tv.TreeViewSig.Emit(tv.RootView.This(), int64(TreeViewInserted), nki.This())
				}
				tv.SetChanged()
				par.UpdateEnd(updt)
				if ski != nil {
					if tvk := tv.ChildByName("tv_"+ski.Name(), 0); tvk != nil {
						stv := AsTreeView(tvk)
						stv.SelectAction(events.SelectOne)
					}
				}
			}
		})
}

// SrcAddChild adds a new child node to this one in the source tree,
// prompting the user for the type of node to add
func (tv *TreeSyncView) SrcAddChild() {
	ttl := "Add Child"
	sk := tv.SrcNode
	if sk == nil {
		log.Printf("TreeView %v nil SrcNode in: %v\n", ttl, tv.Path())
		return
	}
	gi.NewKiDialog(tv, gi.DlgOpts{Title: ttl, Prompt: "Number and Type of Items to Add:"}, sk.BaseType(),
		func(dlg *gi.Dialog) {
			if dlg.Accepted {
				sk := tv.SrcNode
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
				tv.SetChanged()
				sk.UpdateEnd(updt)
				if ski != nil {
					tv.Open()
					if tvk := tv.ChildByName("tv_"+ski.Name(), 0); tvk != nil {
						stv := AsTreeView(tvk)
						stv.SelectAction(events.SelectOne)
					}
				}
			}
		})
}

// SrcDelete deletes the source node corresponding to this view node in the source tree
func (tv *TreeSyncView) SrcDelete() {
	ttl := "Delete"
	if tv.IsRoot(ttl) {
		return
	}
	if tv.MoveDown(events.SelectOne) == nil {
		tv.MoveUp(events.SelectOne)
	}
	sk := tv.SrcNode
	if sk == nil {
		log.Printf("TreeView %v nil SrcNode in: %v\n", ttl, tv.Path())
		return
	}
	sk.Delete(true)
	// tv.TreeViewSig.Emit(tv.RootView.This(), int64(TreeViewDeleted), sk.This())
	tv.SetChanged()
}

// SrcDuplicate duplicates the source node corresponding to this view node in
// the source tree, and inserts the duplicate after this node (as a new
// sibling)
func (tv *TreeSyncView) SrcDuplicate() {
	ttl := "TreeView Duplicate"
	if tv.IsRoot(ttl) {
		return
	}
	sk := tv.SrcNode
	if sk == nil {
		log.Printf("TreeView %v nil SrcNode in: %v\n", ttl, tv.Path())
		return
	}
	if tv.Par == nil {
		return
	}
	tvpar := AsTreeView(tv.Par)
	par := tvpar.SrcNode
	if par == nil {
		log.Printf("TreeView %v nil SrcNode in: %v\n", ttl, tvpar.Path())
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
	tvpar.SetChanged()
	if tvk := tvpar.ChildByName("tv_"+nm, 0); tvk != nil {
		stv := AsTreeView(tvk)
		stv.SelectAction(events.SelectOne)
	}
}

// SrcEdit pulls up a StructViewDialog window on the source object
// viewed by this node
func (tv *TreeSyncView) SrcEdit() {
	if tv.SrcNode == nil {
		slog.Error("TreeView SrcEdit nil SrcNode:", tv)
		return
	}
	// tynm := laser.NonPtrType(tv.SrcNode.KiType()).Name()
	StructViewDialog(tv, DlgOpts{Title: "type"}, tv.SrcNode, nil)
}

// SrcGoGiEditor pulls up a new GoGiEditor window on the source
// object viewed by this node
func (tv *TreeSyncView) SrcGoGiEditor() {
	if tv.SrcNode == nil {
		slog.Error("TreeView SrcGoGiEditor nil SrcNode:", tv)
		return
	}
	GoGiEditorDialog(tv.SrcNode)
}

// MimeData adds mimedata for this node: a text/plain of the Path,
// and an application/json of the source node.
// satisfies Clipper.MimeData interface
func (tv *TreeSyncView) MimeData(md *mimedata.Mimes) {
	sroot := tv.RootView.SrcNode
	src := tv.SrcNode
	*md = append(*md, mimedata.NewTextData(src.PathFrom(sroot)))
	var buf bytes.Buffer
	err := ki.WriteNewJSON(&buf, src)
	if err == nil {
		*md = append(*md, &mimedata.Data{Type: filecat.DataJson, Data: buf.Bytes()})
	} else {
		log.Printf("gi.TreeView MimeData Write JSON error: %v\n", err)
	}
}

// SrcNodesFromMimeData creates a slice of Ki node(s) from given mime data
// and also a corresponding slice of original paths
func (tv *TreeSyncView) SrcNodesFromMimeData(md mimedata.Mimes) (ki.Slice, []string) {
	ni := len(md) / 2
	sl := make(ki.Slice, 0, ni)
	pl := make([]string, 0, ni)
	for _, d := range md {
		if d.Type == filecat.DataJson {
			nki, err := ki.ReadNewJSON(bytes.NewReader(d.Data))
			if err == nil {
				sl = append(sl, nki)
			} else {
				log.Printf("TreeView NodesFromMimeData: JSON load error: %v\n", err)
			}
		} else if d.Type == filecat.TextPlain { // paths
			pl = append(pl, string(d.Data))
		}
	}
	return sl, pl
}

// PasteAssign assigns mime data (only the first one!) to this node
func (tv *TreeSyncView) PasteAssign(md mimedata.Mimes) {
	sl, _ := tv.SrcNodesFromMimeData(md)
	if len(sl) == 0 {
		return
	}
	sk := tv.SrcNode
	if sk == nil {
		log.Printf("TreeView PasteAssign nil SrcNode in: %v\n", tv.Path())
		return
	}
	sk.CopyFrom(sl[0])
	tv.SetChanged()
}

// todo: these methods require an interface to work, based on base code

// PasteAt inserts object(s) from mime data at rel position to this node.
// If another item with the same name already exists, it will
// append _Copy on the name of the inserted objects
func (tv *TreeSyncView) PasteAt(md mimedata.Mimes, mod events.DropMods, rel int, actNm string) {
	sl, pl := tv.NodesFromMimeData(md)

	if tv.Par == nil {
		return
	}
	tvpar := AsTreeView(tv.Par)
	sk := tv.SrcNode
	if sk == nil {
		log.Printf("TreeView %v nil SrcNode in: %v\n", actNm, tv.Path())
		return
	}
	par := sk.Parent()
	if par == nil {
		gi.PromptDialog(tv, gi.DlgOpts{Title: actNm, Prompt: "Cannot insert after the root of the tree", Ok: true, Cancel: false}, nil)
		return
	}
	myidx, ok := sk.IndexInParent()
	if !ok {
		return
	}
	myidx += rel
	sroot := tv.RootView.SrcNode
	updt := par.UpdateStart()
	sz := len(sl)
	var ski ki.Ki
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
			ski = ns
		}
		// tv.RootView.TreeViewSig.Emit(tv.RootView.This(), int64(TreeViewInserted), ns.This())
	}
	par.UpdateEnd(updt)
	tvpar.SetChanged()
	if ski != nil {
		if tvk := tvpar.ChildByName("tv_"+ski.Name(), 0); tvk != nil {
			stv := AsTreeView(tvk)
			stv.SelectAction(events.SelectOne)
		}
	}
}

// PasteChildren inserts object(s) from mime data at end of children of this
// node
func (tv *TreeSyncView) PasteChildren(md mimedata.Mimes, mod events.DropMods) {
	sl, _ := tv.NodesFromMimeData(md)

	sk := tv.SrcNode
	if sk == nil {
		log.Printf("TreeView PasteChildren nil SrcNode in: %v\n", tv.Path())
		return
	}
	updt := sk.UpdateStart()
	sk.SetChildAdded()
	for _, ns := range sl {
		sk.AddChild(ns)
		// tv.RootView.TreeViewSig.Emit(tv.RootView.This(), int64(TreeViewInserted), ns.This())
	}
	sk.UpdateEnd(updt)
	tv.SetChanged()
}

// Cut copies to clip.Board and deletes selected items.
// satisfies gi.Clipper interface and can be overridden by subtypes
func (tv *TreeView) Cut() {
	if tv.IsRoot("Cut") {
		return
	}
	tv.Copy(false)
	sels := tv.SelectedSrcNodes()
	tv.UnselectAll()
	for _, sn := range sels {
		sn.Delete(true)
	}
	tv.SetChanged()
}

*/
