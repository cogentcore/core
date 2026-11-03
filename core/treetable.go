// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"encoding/json"
	"image"
	"log/slog"
	"maps"
	"reflect"
	"slices"
	"strconv"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/base/fileinfo/mimedata"
	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/tree"
)

// TreeTable represents a tree of structs as a table, where the fields are
// the columns and the items are the rows, as in a [Table]. The first column
// shows the tree: the items are indented under their parents, with branches
// that open and close, as in a [Tree]. It is a full-featured editor of the
// tree, with multiple selection, cut-and-paste, and drag-and-drop, which
// insert items before, after, or as the children of other items.
//
// The items are a pointer to a slice of pointers to structs, set with
// [TreeTable.SetItems]. The children of an item are in its field named
// Children, or with a `treetable:"children"` tag, which is a slice of
// pointers to the same struct type, or a []any of them, as needed for
// struct types made with [reflect.StructOf], which cannot refer to
// themselves. If a bool field has a `treetable:"closed"` tag, it records
// whether the item is closed, so that this is saved with the items.
type TreeTable struct {
	Table

	// Items is the pointer to the slice of the top-level items.
	Items any `set:"-"`

	// ShowItem, if set, returns whether to show the given item, a pointer
	// to a struct, with its children, as for filtering the items.
	ShowItem func(item any) bool `json:"-" xml:"-"`

	// HideGroupValues is whether the items with children, which are groups
	// of items, such as categories, show only the tree, and not the values
	// of their other fields.
	HideGroupValues bool

	// itemsSet is the Items for which the item type and rows were set.
	itemsSet any

	// itemType is the struct type of the items.
	itemType reflect.Type

	// childrenIndex is the index of the children field of the items,
	// or nil if they have none.
	childrenIndex []int

	// closedIndex is the index of the closed field of the items,
	// or nil if they have none, in which case closed records it.
	closedIndex []int

	// closed has the closed items, if they have no closed field.
	closed map[any]bool

	// rows are the visible items, in order.
	rows []treeTableRow

	// draggedItems are the items last copied for dragging,
	// which are deleted when they are moved.
	draggedItems []reflect.Value
}

// treeTableRow is a visible item of a [TreeTable].
type treeTableRow struct {
	// item is the pointer to the item.
	item reflect.Value

	// parent is the pointer to the parent item, or invalid for a top-level item.
	parent reflect.Value

	// depth is the number of ancestors of the item.
	depth int
}

func (tt *TreeTable) WidgetValue() any { return &tt.Items }

func (tt *TreeTable) Init() {
	tt.Table.Init()
	tt.closed = map[any]bool{}
	tt.ContextMenus = nil // replaces those of the flat lists
	tt.AddContextMenu(tt.contextMenu)

	// these run before those of the flat lists, which they replace
	tt.OnFinal(events.KeyChord, tt.keyInput)
	tt.On(events.Drop, tt.dragDrop)

	tt.Makers.Normal[0] = func(p *tree.Plan) {
		if tt.Items != tt.itemsSet {
			tt.setItems()
		}
		if tt.itemType == nil {
			return
		}
		tt.sortItems()
		tt.updateRows()

		if tt.InitSelectedIndex >= 0 {
			tt.SelectedIndex = tt.InitSelectedIndex
			tt.InitSelectedIndex = -1
			tt.ScrollToIndex(tt.SelectedIndex)
		}
		tt.UpdateStartIndex()

		tt.makeHeader(p)
		tt.MakeGrid(p, func(p *tree.Plan) {
			for i := range tt.VisibleRows {
				tt.MakeRow(p, i)
			}
		})
	}
}

// SetItems sets the items to the given pointer to a slice of pointers
// to structs, which are the top-level items. If they have the same type as
// the current items, as for new values of them, such as after an undo, the
// view stays the same: the scroll position, the selected rows, and the
// widgets of the rows, with the focus.
func (tt *TreeTable) SetItems(items any) *TreeTable {
	if items == tt.itemsSet && items != nil {
		tt.MakeIter = 0
		return tt
	}
	tt.Items = items
	tt.setItems()
	return tt
}

// setItems sets the item type and rows for the Items.
func (tt *TreeTable) setItems() {
	if tt.sameItemType() {
		tt.itemsSet = tt.Items
		// the selection is by index, as the items are new
		selected, current := maps.Clone(tt.SelectedIndexes), tt.SelectedIndex
		tt.rows = nil
		tt.closed = map[any]bool{}
		tt.updateRows()
		for idx := range selected {
			if tt.validIndex(idx) {
				tt.SelectedIndexes[idx] = struct{}{}
			}
		}
		tt.SelectedIndex = min(current, len(tt.rows)-1)
		return
	}
	sort := tt.SortFieldName() // kept by name, for the new fields
	tt.itemsSet = tt.Items
	tt.itemType = nil
	tt.rows = nil
	tt.closed = map[any]bool{}
	if reflectx.IsNil(reflect.ValueOf(tt.Items)) {
		tt.Slice = nil
		return
	}
	typ := reflect.TypeOf(tt.Items)
	if typ.Kind() != reflect.Pointer || typ.Elem().Kind() != reflect.Slice || typ.Elem().Elem().Kind() != reflect.Pointer || typ.Elem().Elem().Elem().Kind() != reflect.Struct {
		slog.Error("TreeTable requires a pointer to a slice of pointers to structs", "type", typ)
		return
	}
	tt.setItemType(typ.Elem().Elem().Elem())

	// the Table shows the rows, a slice of pointers to the visible items
	rows := reflect.New(typ.Elem())
	tt.Slice = rows.Interface()
	tt.sliceUnderlying = rows.Elem()
	tt.elementValue = reflect.New(tt.itemType).Elem()
	tt.cacheVisibleFields()
	tt.SetSortFieldName(sort) // before any update in SetSliceBase, which can set it
	tt.updateRows()
	tt.SetSliceBase()
}

// sameItemType returns whether the Items have the same type as those
// for which the item type and rows were set.
func (tt *TreeTable) sameItemType() bool {
	if tt.itemType == nil || reflectx.IsNil(reflect.ValueOf(tt.Items)) {
		return false
	}
	typ := reflect.TypeOf(tt.Items)
	return typ.Kind() == reflect.Pointer && typ.Elem() == tt.sliceUnderlying.Type()
}

// setItemType sets the type of the items and their tree fields.
func (tt *TreeTable) setItemType(typ reflect.Type) {
	tt.itemType = typ
	tt.childrenIndex, tt.closedIndex = nil, nil
	children, ok := treeTableField(typ, "children")
	if !ok {
		children, ok = typ.FieldByName("Children")
	}
	if ok {
		if ct := children.Type; ct.Kind() == reflect.Slice && (ct.Elem() == reflect.PointerTo(typ) || ct.Elem() == reflect.TypeFor[any]()) {
			tt.childrenIndex = children.Index
		} else {
			slog.Error("TreeTable: the children field must be a slice of pointers to the item type, or a []any", "type", typ, "field", children.Name)
		}
	}
	if closed, ok := treeTableField(typ, "closed"); ok && closed.Type.Kind() == reflect.Bool {
		tt.closedIndex = closed.Index
	}
}

// treeTableField returns the field of the given struct type
// with the given treetable tag.
func treeTableField(typ reflect.Type, tag string) (reflect.StructField, bool) {
	for i := range typ.NumField() {
		if f := typ.Field(i); f.Tag.Get("treetable") == tag {
			return f, true
		}
	}
	return reflect.StructField{}, false
}

// cacheVisibleFields caches the visible struct fields,
// which do not include the tree fields.
func (tt *TreeTable) cacheVisibleFields() {
	tt.Table.cacheVisibleFields()
	tt.visibleFields = slices.DeleteFunc(tt.visibleFields, func(f reflect.StructField) bool {
		return slices.Equal(f.Index, tt.childrenIndex) || slices.Equal(f.Index, tt.closedIndex)
	})
	tt.numVisibleFields = len(tt.visibleFields)
	tt.headerWidths = make([]int, tt.numVisibleFields)
	tt.colMaxWidths = make([]int, tt.numVisibleFields)
}

//////// Items

// itemsValue returns the slice of the top-level items.
func (tt *TreeTable) itemsValue() reflect.Value {
	return reflect.ValueOf(tt.Items).Elem()
}

// children returns the slice of the children of the given item.
func (tt *TreeTable) children(item reflect.Value) reflect.Value {
	return item.Elem().FieldByIndex(tt.childrenIndex)
}

// numChildren returns the number of children of the given item.
func (tt *TreeTable) numChildren(item reflect.Value) int {
	if tt.childrenIndex == nil {
		return 0
	}
	return tt.children(item).Len()
}

// newItem returns a new item, with the values of its default tags.
func (tt *TreeTable) newItem() reflect.Value {
	item := reflect.New(tt.itemType)
	errors.Log(reflectx.SetFromDefaultTags(item.Interface()))
	return item
}

// walkItems calls the given function on each item in the given slice,
// with its parent and depth, going into its children if it returns true.
func (tt *TreeTable) walkItems(sl, parent reflect.Value, depth int, fun func(item, parent reflect.Value, depth int) bool) {
	for i := range sl.Len() {
		item := itemAt(sl, i)
		if !isItem(item) {
			continue
		}
		if fun(item, parent, depth) && tt.childrenIndex != nil {
			tt.walkItems(tt.children(item), item, depth+1, fun)
		}
	}
}

// itemAt returns the pointer to the item at the given index
// in the given slice of items, which can be a []any. It is a copy,
// which stays the same when the slice changes.
func itemAt(sl reflect.Value, i int) reflect.Value {
	item := sl.Index(i)
	if item.Kind() == reflect.Interface {
		return item.Elem()
	}
	return reflect.ValueOf(item.Interface())
}

// isItem returns whether the given value from [itemAt] is an item.
func isItem(item reflect.Value) bool {
	return item.IsValid() && item.Kind() == reflect.Pointer && !item.IsNil()
}

// indexOfItem returns the index of the given item in the given slice, or -1.
func indexOfItem(sl, item reflect.Value) int {
	for i := range sl.Len() {
		if it := itemAt(sl, i); isItem(it) && it.UnsafePointer() == item.UnsafePointer() {
			return i
		}
	}
	return -1
}

// insertItem inserts the given item into the given slice at the given index.
func insertItem(sl reflect.Value, i int, item reflect.Value) {
	n := sl.Len()
	ns := reflect.Append(sl, reflect.Zero(sl.Type().Elem()))
	reflect.Copy(ns.Slice(i+1, n+1), ns.Slice(i, n))
	ns.Index(i).Set(item)
	sl.Set(ns)
}

// deleteItem deletes the item at the given index from the given slice.
func deleteItem(sl reflect.Value, i int) {
	n := sl.Len()
	reflect.Copy(sl.Slice(i, n-1), sl.Slice(i+1, n))
	sl.Index(n - 1).SetZero()
	sl.Set(sl.Slice(0, n-1))
}

// removeItem removes the given item from the given slice
// or the children of its items, returning whether it was found.
func (tt *TreeTable) removeItem(sl, item reflect.Value) bool {
	if k := indexOfItem(sl, item); k >= 0 {
		deleteItem(sl, k)
		return true
	}
	if tt.childrenIndex == nil {
		return false
	}
	for i := range sl.Len() {
		if c := itemAt(sl, i); isItem(c) && tt.removeItem(tt.children(c), item) {
			return true
		}
	}
	return false
}

// sortItems sorts the items and the children of each item by the
// sort column, if there is one.
func (tt *TreeTable) sortItems() {
	if tt.sortIndex < 0 || tt.sortIndex >= len(tt.visibleFields) {
		return
	}
	tt.sortSiblings(tt.itemsValue(), tt.visibleFields[tt.sortIndex].Index)
}

// sortSiblings sorts the items in the given slice and their children
// by the field with the given index.
func (tt *TreeTable) sortSiblings(sl reflect.Value, field []int) {
	n := sl.Len()
	items := reflect.MakeSlice(reflect.SliceOf(reflect.PointerTo(tt.itemType)), 0, n)
	for i := range n {
		item := itemAt(sl, i)
		if !isItem(item) {
			return // nothing to sort by
		}
		items = reflect.Append(items, item)
	}
	reflectx.StructSliceSort(items.Interface(), field, !tt.sortDescending)
	for i := range n {
		item := items.Index(i)
		sl.Index(i).Set(item)
		if tt.childrenIndex != nil {
			tt.sortSiblings(tt.children(item), field)
		}
	}
}

//////// Rows

// updateRows updates the rows of the visible items,
// keeping the selection of the same items.
func (tt *TreeTable) updateRows() {
	selected := map[any]bool{}
	for i := range tt.SelectedIndexes {
		if i < len(tt.rows) {
			selected[tt.rows[i].item.Interface()] = true
		}
	}
	var current any
	if tt.validIndex(tt.SelectedIndex) {
		current = tt.rows[tt.SelectedIndex].item.Interface()
	}

	tt.rows = tt.rows[:0]
	tt.walkItems(tt.itemsValue(), reflect.Value{}, 0, func(item, parent reflect.Value, depth int) bool {
		if tt.ShowItem != nil && !tt.ShowItem(item.Interface()) {
			return false
		}
		tt.rows = append(tt.rows, treeTableRow{item: item, parent: parent, depth: depth})
		return !tt.itemClosed(item)
	})
	rs := reflect.MakeSlice(tt.sliceUnderlying.Type(), len(tt.rows), len(tt.rows))
	for i, r := range tt.rows {
		rs.Index(i).Set(r.item)
	}
	tt.sliceUnderlying.Set(rs)
	tt.UpdateSliceSize()

	tt.ResetSelectedIndexes()
	for i, r := range tt.rows {
		item := r.item.Interface()
		if selected[item] {
			tt.SelectedIndexes[i] = struct{}{}
		}
		if item == current {
			tt.SelectedIndex = i
		}
	}
	tt.SelectedIndex = min(tt.SelectedIndex, len(tt.rows)-1)
}

// validIndex returns whether the given row index is that of a visible item.
func (tt *TreeTable) validIndex(idx int) bool {
	return idx >= 0 && idx < len(tt.rows)
}

// rowOf returns the row index of the given item, or -1 if it is not visible.
func (tt *TreeTable) rowOf(item reflect.Value) int {
	return slices.IndexFunc(tt.rows, func(r treeTableRow) bool {
		return r.item.UnsafePointer() == item.UnsafePointer()
	})
}

// rowDepth returns the depth of the item at the given widget row index.
func (tt *TreeTable) rowDepth(i int) int {
	si, _, invis := tt.SliceIndex(i)
	if invis || !tt.validIndex(si) {
		return 0
	}
	return tt.rows[si].depth
}

// siblingsOf returns the slice with the item at the given row index,
// and its index there.
func (tt *TreeTable) siblingsOf(idx int) (reflect.Value, int) {
	r := tt.rows[idx]
	sl := tt.itemsValue()
	if r.parent.IsValid() {
		sl = tt.children(r.parent)
	}
	return sl, indexOfItem(sl, r.item)
}

// edited updates the rows after the items are edited, selects
// the given item, and sends a change event.
func (tt *TreeTable) edited(item reflect.Value) {
	tt.sortItems() // as the update does, so that the item is in its row
	tt.updateRows()
	idx := tt.rowOf(item)
	if idx >= 0 {
		tt.SelectIndexEvent(idx, events.SelectOne)
	}
	tt.UpdateChange()
	if idx = tt.rowOf(item); idx >= 0 {
		tt.IndexGrabFocus(idx)
	}
}

//////// Open and close

// IsClosed returns whether the item at the given row index is closed,
// with its children not shown.
func (tt *TreeTable) IsClosed(idx int) bool {
	return tt.validIndex(idx) && tt.itemClosed(tt.rows[idx].item)
}

// itemClosed returns whether the given item is closed.
func (tt *TreeTable) itemClosed(item reflect.Value) bool {
	if tt.closedIndex != nil {
		return item.Elem().FieldByIndex(tt.closedIndex).Bool()
	}
	return tt.closed[item.Interface()]
}

// setItemClosed sets whether the given item is closed.
func (tt *TreeTable) setItemClosed(item reflect.Value, closed bool) {
	if tt.closedIndex != nil {
		item.Elem().FieldByIndex(tt.closedIndex).SetBool(closed)
		return
	}
	if closed {
		tt.closed[item.Interface()] = true
	} else {
		delete(tt.closed, item.Interface())
	}
}

// Open opens the item at the given row index, showing its children.
func (tt *TreeTable) Open(idx int) {
	tt.setClosed(idx, false)
}

// Close closes the item at the given row index, hiding its children.
func (tt *TreeTable) Close(idx int) {
	tt.setClosed(idx, true)
}

// ToggleClose opens the item at the given row index if it is closed,
// and otherwise closes it.
func (tt *TreeTable) ToggleClose(idx int) {
	tt.setClosed(idx, !tt.IsClosed(idx))
}

// setClosed sets whether the item at the given row index is closed,
// if it has children.
func (tt *TreeTable) setClosed(idx int, closed bool) {
	if !tt.validIndex(idx) || tt.numChildren(tt.rows[idx].item) == 0 || tt.IsClosed(idx) == closed {
		return
	}
	tt.setItemClosed(tt.rows[idx].item, closed)
	tt.closedChanged()
}

// OpenAll opens all of the items.
func (tt *TreeTable) OpenAll() { //types:add
	tt.setAllClosed(false)
}

// CloseAll closes all of the items.
func (tt *TreeTable) CloseAll() { //types:add
	tt.setAllClosed(true)
}

// setAllClosed sets whether all of the items with children are closed.
func (tt *TreeTable) setAllClosed(closed bool) {
	tt.walkItems(tt.itemsValue(), reflect.Value{}, 0, func(item, parent reflect.Value, depth int) bool {
		if tt.numChildren(item) > 0 {
			tt.setItemClosed(item, closed)
		}
		return true
	})
	tt.closedChanged()
}

// closedChanged updates after items are opened or closed,
// which is a change of the items if they have a closed field.
func (tt *TreeTable) closedChanged() {
	tt.updateRows()
	if tt.closedIndex != nil {
		tt.UpdateChange()
	} else {
		tt.Update()
	}
}

//////// Editing

// NewAt inserts a new item after the item in the row before the given
// row index, as its sibling, or before the first item for 0, or at the
// end of the top-level items for -1.
func (tt *TreeTable) NewAt(idx int) {
	switch {
	case idx < 0 || len(tt.rows) == 0:
		item := tt.newItem()
		sl := tt.itemsValue()
		insertItem(sl, sl.Len(), item)
		tt.edited(item)
	case idx == 0:
		tt.InsertBefore(0)
	default:
		tt.InsertAfter(min(idx, len(tt.rows)) - 1)
	}
}

// InsertBefore inserts a new item before the item at the given row index,
// as its sibling.
func (tt *TreeTable) InsertBefore(idx int) {
	tt.insertNew(idx, 0)
}

// InsertAfter inserts a new item after the item at the given row index,
// as its sibling.
func (tt *TreeTable) InsertAfter(idx int) {
	tt.insertNew(idx, 1)
}

// insertNew inserts a new item at the given offset from the item at
// the given row index among its siblings.
func (tt *TreeTable) insertNew(idx, offset int) {
	if !tt.validIndex(idx) {
		tt.NewAt(-1)
		return
	}
	item := tt.newItem()
	sl, k := tt.siblingsOf(idx)
	insertItem(sl, k+offset, item)
	tt.edited(item)
}

// AddChild adds a new item at the end of the children of the item at
// the given row index, and opens it.
func (tt *TreeTable) AddChild(idx int) {
	if !tt.validIndex(idx) || tt.childrenIndex == nil {
		return
	}
	parent := tt.rows[idx].item
	item := tt.newItem()
	ch := tt.children(parent)
	insertItem(ch, ch.Len(), item)
	tt.setItemClosed(parent, false)
	tt.edited(item)
}

// DeleteAt deletes the item at the given row index, with its children.
func (tt *TreeTable) DeleteAt(idx int) {
	if !tt.validIndex(idx) {
		return
	}
	sl, k := tt.siblingsOf(idx)
	deleteItem(sl, k)
	tt.updateRows()
	tt.UpdateChange()
}

// deleteSelected deletes the selected items, with their children.
func (tt *TreeTable) deleteSelected() {
	for _, idx := range tt.SelectedIndexesList(true) { // last first, so the others stay
		tt.DeleteAt(idx)
	}
}

// Indent moves the item at the given row index into the item before it
// among its siblings, as its last child.
func (tt *TreeTable) Indent(idx int) {
	if !tt.validIndex(idx) || tt.childrenIndex == nil {
		return
	}
	sl, k := tt.siblingsOf(idx)
	if k <= 0 {
		return
	}
	item := tt.rows[idx].item
	parent := itemAt(sl, k-1)
	deleteItem(sl, k)
	ch := tt.children(parent)
	insertItem(ch, ch.Len(), item)
	tt.setItemClosed(parent, false)
	tt.edited(item)
}

// Outdent moves the item at the given row index out of its parent,
// to after its parent among the siblings of the parent.
func (tt *TreeTable) Outdent(idx int) {
	if !tt.validIndex(idx) || !tt.rows[idx].parent.IsValid() {
		return
	}
	r := tt.rows[idx]
	sl, k := tt.siblingsOf(idx)
	psl, pk := tt.siblingsOf(tt.rowOf(r.parent))
	deleteItem(sl, k)
	insertItem(psl, pk+1, r.item)
	tt.edited(r.item)
}

// MoveItemUp moves the item at the given row index before
// the previous one among its siblings.
func (tt *TreeTable) MoveItemUp(idx int) {
	tt.moveItem(idx, -1)
}

// MoveItemDown moves the item at the given row index after
// the next one among its siblings.
func (tt *TreeTable) MoveItemDown(idx int) {
	tt.moveItem(idx, 1)
}

// moveItem moves the item at the given row index by the given number
// of places among its siblings.
func (tt *TreeTable) moveItem(idx, delta int) {
	if !tt.validIndex(idx) {
		return
	}
	sl, k := tt.siblingsOf(idx)
	if j := k + delta; j >= 0 && j < sl.Len() {
		reflect.Swapper(sl.Interface())(k, j)
		tt.edited(tt.rows[idx].item)
	}
}

// Duplicate inserts copies of the selected items after the last of them.
func (tt *TreeTable) Duplicate() {
	ixs := tt.SelectedIndexesList(true)
	if len(ixs) == 0 {
		return
	}
	tt.pasteAfter(tt.CopySelectToMime(), ixs[0])
}

//////// Copy and paste

// selectedTopItems returns the selected items, in order, without those
// within another selected item, with which they are copied.
func (tt *TreeTable) selectedTopItems() []reflect.Value {
	var items []reflect.Value
	for _, idx := range tt.SelectedIndexesList(false) {
		if tt.validIndex(idx) && !tt.hasSelectedAncestor(idx) {
			items = append(items, tt.rows[idx].item)
		}
	}
	return items
}

// hasSelectedAncestor returns whether an ancestor of the item at
// the given row index is selected.
func (tt *TreeTable) hasSelectedAncestor(idx int) bool {
	for p := tt.rows[idx].parent; p.IsValid(); {
		pi := tt.rowOf(p)
		if pi < 0 {
			return false
		}
		if tt.indexIsSelected(pi) {
			return true
		}
		p = tt.rows[pi].parent
	}
	return false
}

// CopySelectToMime returns the selected items, with their children,
// as mime data. It records them as the dragged items.
func (tt *TreeTable) CopySelectToMime() mimedata.Mimes {
	items := tt.selectedTopItems()
	tt.draggedItems = items
	md := make(mimedata.Mimes, 0, len(items))
	for _, item := range items {
		b, err := json.MarshalIndent(item.Interface(), "", "  ")
		if errors.Log(err) == nil {
			md = append(md, &mimedata.Data{Type: fileinfo.DataJSON, Data: b})
		}
	}
	return md
}

// itemsFromMime returns new items from the given mime data.
func (tt *TreeTable) itemsFromMime(md mimedata.Mimes) []reflect.Value {
	var items []reflect.Value
	for _, d := range md {
		if d.Type != fileinfo.DataJSON {
			continue
		}
		item, err := tt.itemFromJSON(d.Data)
		if errors.Log(err) == nil {
			items = append(items, item)
		}
	}
	return items
}

// itemFromJSON returns a new item decoded from the given JSON, with its
// children, which are decoded as items when they are a []any.
func (tt *TreeTable) itemFromJSON(b []byte) (reflect.Value, error) {
	item := reflect.New(tt.itemType)
	if err := json.Unmarshal(b, item.Interface()); err != nil {
		return reflect.Value{}, err
	}
	if tt.childrenIndex == nil {
		return item, nil
	}
	ch := tt.children(item)
	if ch.Type().Elem().Kind() != reflect.Interface {
		return item, nil
	}
	// the children were decoded as maps
	items := reflect.MakeSlice(ch.Type(), 0, ch.Len())
	for i := range ch.Len() {
		cb, err := json.Marshal(ch.Index(i).Interface())
		if err != nil {
			return reflect.Value{}, err
		}
		c, err := tt.itemFromJSON(cb)
		if err != nil {
			return reflect.Value{}, err
		}
		items = reflect.Append(items, c)
	}
	ch.Set(items)
	return item, nil
}

// pasteItems inserts the items from the given mime data into the given
// slice at the given index, and selects the first of them.
func (tt *TreeTable) pasteItems(md mimedata.Mimes, sl reflect.Value, k int) {
	items := tt.itemsFromMime(md)
	for i, item := range items {
		insertItem(sl, k+i, item)
	}
	if len(items) > 0 {
		tt.edited(items[0])
	}
}

// PasteAtIndex inserts the items from the given mime data before the
// item at the given row index, as its siblings, or at the end of the
// top-level items if it is out of range.
func (tt *TreeTable) PasteAtIndex(md mimedata.Mimes, idx int) {
	if !tt.validIndex(idx) {
		sl := tt.itemsValue()
		tt.pasteItems(md, sl, sl.Len())
		return
	}
	sl, k := tt.siblingsOf(idx)
	tt.pasteItems(md, sl, k)
}

// pasteAfter inserts the items from the given mime data after the
// item at the given row index, as its siblings.
func (tt *TreeTable) pasteAfter(md mimedata.Mimes, idx int) {
	if !tt.validIndex(idx) {
		tt.PasteAtIndex(md, -1)
		return
	}
	sl, k := tt.siblingsOf(idx)
	tt.pasteItems(md, sl, k+1)
}

// pasteChildren inserts the items from the given mime data at the end
// of the children of the item at the given row index, and opens it.
func (tt *TreeTable) pasteChildren(md mimedata.Mimes, idx int) {
	if !tt.validIndex(idx) || tt.childrenIndex == nil {
		return
	}
	parent := tt.rows[idx].item
	tt.setItemClosed(parent, false)
	ch := tt.children(parent)
	tt.pasteItems(md, ch, ch.Len())
}

// PasteAssign assigns the first item from the given mime data,
// with its children, to the item at the given row index.
func (tt *TreeTable) PasteAssign(md mimedata.Mimes, idx int) {
	items := tt.itemsFromMime(md)
	if len(items) == 0 || !tt.validIndex(idx) {
		return
	}
	item := tt.rows[idx].item
	item.Elem().Set(items[0].Elem())
	tt.edited(item)
}

// makeTreePasteMenu makes the menu of the ways to paste the given mime
// data relative to the item at the given row index.
func (tt *TreeTable) makeTreePasteMenu(m *Scene, md mimedata.Mimes, idx int, mod events.DropMods, fun func()) {
	paste := func(text string, f func()) {
		NewButton(m).SetText(text).OnClick(func(e events.Event) {
			f()
			if fun != nil {
				fun()
			}
		})
	}
	if !tt.validIndex(idx) {
		paste("Insert at end", func() { tt.PasteAtIndex(md, -1) })
		NewButton(m).SetText("Cancel")
		return
	}
	if mod == events.DropCopy {
		paste("Assign to", func() { tt.PasteAssign(md, idx) })
	}
	paste("Insert before", func() { tt.PasteAtIndex(md, idx) })
	paste("Insert after", func() { tt.pasteAfter(md, idx) })
	if tt.childrenIndex != nil {
		paste("Insert as child", func() { tt.pasteChildren(md, idx) })
	}
	NewButton(m).SetText("Cancel")
}

//////// Drag and drop

// dragStart starts dragging the selected items from the given tree cell,
// selecting the item under the mouse if it is not selected.
func (tt *TreeTable) dragStart(e events.Event, cell Widget) {
	row, _, ok := tt.ListGrid.indexFromPixel(e.StartPos()) // where the mouse went down
	if !ok || !tt.validIndex(tt.StartIndex+row) {
		return
	}
	if !tt.indexIsSelected(tt.StartIndex + row) {
		tt.updateSelectRow(row, events.SelectOne)
	}
	md := tt.CopySelectToMime()
	if len(md) == 0 {
		return
	}
	tt.Scene.Events.DragStart(cell, md, e)
	e.SetHandled()
}

// dragDrop handles a drop onto the tree table, with a menu of the ways
// to paste the dropped items relative to the item under the mouse.
func (tt *TreeTable) dragDrop(e events.Event) {
	e.SetHandled()
	de := e.(*events.DragDrop)
	md, ok := de.Data.(mimedata.Mimes)
	if !ok || tt.itemType == nil {
		return
	}
	idx := -1
	if row, _, ok := tt.ListGrid.indexFromPixel(de.Pos()); ok && tt.validIndex(tt.StartIndex+row) {
		idx = tt.StartIndex + row
	}
	if idx >= 0 && de.DropMod == events.DropMove && tt.isOwnDrag(de) && tt.withinDragged(tt.rows[idx].item) {
		return // cannot move items into themselves
	}
	mf := func(m *Scene, pos image.Point) {
		tt.Scene.Events.DragMenuAddModText(m, de.DropMod)
		tt.makeTreePasteMenu(m, md, idx, de.DropMod, func() {
			tt.dropFinalize(de)
		})
	}
	NewMenu(mf, tt.This.(Widget), de.WindowPos()).Run()
}

// isOwnDrag returns whether the given drop is of items dragged from
// this tree table.
func (tt *TreeTable) isOwnDrag(de *events.DragDrop) bool {
	src, ok := de.Source.(tree.Node)
	for ok && src != nil {
		if src == tt.This {
			return true
		}
		src = src.AsTree().Parent
	}
	return false
}

// withinDragged returns whether the given item is one of the dragged
// items or one of their descendants.
func (tt *TreeTable) withinDragged(item reflect.Value) bool {
	for item.IsValid() {
		if slices.ContainsFunc(tt.draggedItems, func(d reflect.Value) bool {
			return d.UnsafePointer() == item.UnsafePointer()
		}) {
			return true
		}
		idx := tt.rowOf(item)
		if idx < 0 {
			return false
		}
		item = tt.rows[idx].parent
	}
	return false
}

// dropDeleteItems deletes the dragged items after they are moved.
func (tt *TreeTable) dropDeleteItems() {
	for _, item := range tt.draggedItems {
		tt.removeItem(tt.itemsValue(), item)
	}
	tt.draggedItems = nil
	tt.updateRows()
	tt.UpdateChange()
}

//////// Widgets

// MakeRow adds the widgets for the given widget row index to the given
// plan, with the tree cell for the first field.
func (tt *TreeTable) MakeRow(p *tree.Plan, i int) {
	si, _, invis := tt.SliceIndex(i)
	if tt.ShowIndexes {
		tt.MakeGridIndex(p, i, si, strconv.Itoa(i), invis)
	}
	if tt.numVisibleFields == 0 {
		return
	}
	tt.makeTreeCell(p, i)
	for fli := 1; fli < tt.numVisibleFields; fli++ {
		tt.makeFieldValue(p, i, fli)
	}
	tt.makeRowButton(p, i)
}

// makeTreeCell adds the tree cell for the given widget row index to the
// given plan: the indented branch and the value of the first field.
func (tt *TreeTable) makeTreeCell(p *tree.Plan, i int) {
	tree.AddAt(p, "tree-"+strconv.Itoa(i), func(w *treeTableCell) {
		w.tt = tt
		w.SetProperty(ListRowProperty, i)
		w.SetProperty(ListColProperty, 0)
		w.Styler(func(s *styles.Style) {
			s.SetAbilities(!tt.IsReadOnly(), abilities.Draggable)
			s.Direction = styles.Row
			s.Align.Items = styles.Center
			s.Gap.X.Em(0.1)
			s.Padding.Zero()
			// the indentation is padding, which the spacing setting scales,
			// but it is not for spacing, so it stays the same, as in a Tree
			s.Padding.Left.Em(ConstantSpacing(float32(tt.rowDepth(i))))
			s.Grow.Set(1, 0)
			if tt.columnWidth(0) > 0 { // see ListGrid.fixColumnWidths
				s.Overflow.X = styles.OverflowHidden
			}
		})
		w.On(events.DragStart, func(e events.Event) {
			tt.dragStart(e, w)
		})
		w.On(events.DropDeleteSource, func(e events.Event) {
			tt.dropDeleteItems()
		})
		w.On(events.ContextMenu, tt.rowContextMenu(i))
		w.Updater(func() {
			_, _, invis := tt.SliceIndex(i)
			w.SetState(invis, states.Invisible)
		})
		w.Maker(func(p *tree.Plan) {
			tt.makeBranch(p, i)
			tt.makeFieldValue(p, i, 0)
		})
	})
}

// StyleValue styles the value widgets, where the value of the tree cell
// fills it, if the width of the first column is set.
func (tt *TreeTable) StyleValue(w Widget, s *styles.Style, row, col int) {
	if col != 0 || tt.columnWidth(0) == 0 {
		tt.Table.StyleValue(w, s, row, col)
		return
	}
	s.Min.X.Em(2)
	s.Max.X.Zero() // such as the 40ch of a TextField
	s.Grow.X = 1
	s.SetTextWrap(false)
}

// treeTableCell is the tree cell of a [TreeTable], with the indented branch
// and the value of the first field, which fits in the width of the cell when
// the width of the column is set, so that a text field scrolls its text,
// instead of having the width of all of it.
type treeTableCell struct {
	Frame

	// tt is the tree table of the cell.
	tt *TreeTable
}

func (tc *treeTableCell) SizeDown(iter int) bool {
	if tc.fitValue() {
		tc.sizeFromChildrenFit(iter, SizeDownPass) // the layout of the new size
	}
	return tc.Frame.SizeDown(iter)
}

func (tc *treeTableCell) SizeFinal() {
	tc.fitValue() // for the final width of the cell
	tc.Frame.SizeFinal()
}

// fitValue fits the value in the width of the cell after the branch,
// if the width of the column is set, returning whether it is.
func (tc *treeTableCell) fitValue() bool {
	if tc.tt == nil || tc.tt.columnWidth(0) == 0 || tc.NumChildren() < 2 {
		return false
	}
	other := float32(0)
	for _, c := range tc.Children[:tc.NumChildren()-1] {
		other += AsWidget(c).Geom.Size.Actual.Total.X + tc.Styles.Gap.X.Dots
	}
	vs := &AsWidget(tc.Child(tc.NumChildren() - 1)).Geom.Size
	width := max(tc.Geom.Size.Alloc.Content.X-other, 0)
	space := vs.Actual.Total.X - vs.Actual.Content.X
	vs.Actual.Total.X, vs.Actual.Content.X = width, max(width-space, 0)
	vs.Max.X = vs.Actual.Content.X // as a TextField sizes itself to its text, up to its max
	return true
}

// makeBranch adds the branch for the given widget row index to the given
// plan, which opens and closes the item.
func (tt *TreeTable) makeBranch(p *tree.Plan, i int) {
	tree.AddAt(p, "branch", func(w *Switch) {
		w.SetType(SwitchCheckbox)
		w.SetIconOn(icons.KeyboardArrowDown).SetIconOff(icons.KeyboardArrowRight).SetIconIndeterminate(icons.Blank)
		w.Styler(func(s *styles.Style) {
			s.SetAbilities(!tt.IsReadOnly(), abilities.Focusable)
			s.Color = colors.Scheme.Primary.Base
			s.Padding.Zero()
			s.Align.Self = styles.Center
			if w.StateIs(states.Indeterminate) {
				// a leaf, with nothing to open
				s.Abilities = 0
				s.StateLayer = 0
			}
		})
		w.OnClick(func(e events.Event) {
			e.SetHandled() // the item, not the switch, has the state
			si, _, _ := tt.SliceIndex(i)
			tt.ToggleClose(si)
		})
		w.On(events.ContextMenu, tt.rowContextMenu(i))
		w.On(events.KeyChord, func(e events.Event) {
			si, _, _ := tt.SliceIndex(i)
			switch keymap.Of(e.KeyChord()) {
			case keymap.MoveRight:
				tt.Open(si)
				e.SetHandled()
			case keymap.MoveLeft:
				tt.Close(si)
				e.SetHandled()
			}
		})
		w.Updater(func() {
			si, _, invis := tt.SliceIndex(i)
			leaf := invis || !tt.validIndex(si) || tt.numChildren(tt.rows[si].item) == 0
			w.SetState(leaf, states.Indeterminate)
			w.SetState(!leaf && !tt.IsClosed(si), states.Checked)
		})
	})
}

// rowContextMenu returns the handler of context menu events on the tree
// cell for the given widget row index, and its branch, which selects the
// row and shows the context menu of the tree table, as for the values.
func (tt *TreeTable) rowContextMenu(i int) func(e events.Event) {
	return func(e events.Event) {
		if si, _, invis := tt.SliceIndex(i); !invis && !tt.indexIsSelected(si) {
			tt.updateSelectRow(i, events.SelectOne)
		}
		tt.HandleEvent(e)
	}
}

// hideValue returns whether to hide the value of the given visible
// field at the given slice index, for [TreeTable.HideGroupValues].
func (tt *TreeTable) hideValue(si, fli int) bool {
	return tt.HideGroupValues && fli > 0 && tt.validIndex(si) && tt.numChildren(tt.rows[si].item) > 0
}

// RowGrabFocus grabs the focus for the first focusable value widget in the
// given row, returning it, or nil if there is none.
func (tt *TreeTable) RowGrabFocus(row int) *WidgetBase {
	if !tt.IsRowInBounds(row) || tt.InFocusGrab || tt.ListGrid == nil {
		return nil
	}
	nWidgPerRow, idxOff := tt.RowWidgetNs()
	ridx := nWidgPerRow * row
	lg := tt.ListGrid
	if ridx+nWidgPerRow > lg.NumChildren() {
		return nil
	}
	value := func(fli int) *WidgetBase {
		w := lg.Child(ridx + idxOff + fli).(Widget).AsWidget()
		if fli == 0 { // the tree cell, with the value last
			if !w.HasChildren() {
				return nil
			}
			w = w.Child(w.NumChildren() - 1).(Widget).AsWidget()
		}
		return w
	}
	for fli := range tt.numVisibleFields {
		if w := value(fli); w != nil && (w.StateIs(states.Focused) || w.ContainsFocus()) {
			return w
		}
	}
	tt.InFocusGrab = true
	defer func() { tt.InFocusGrab = false }()
	for fli := range tt.numVisibleFields {
		if w := value(fli); w != nil && w.CanFocus() {
			w.SetFocus()
			return w
		}
	}
	return nil
}

// keyInput handles the keys for the tree: opening and closing items
// when read-only, and inserting, duplicating, and deleting items. The
// keys of a text field, such as [keymap.Delete], come here after its
// edits are accepted with [keymap.Accept].
func (tt *TreeTable) keyInput(e events.Event) {
	if !tt.IsReadOnly() {
		tt.selectFocusedRow()
	}
	idx := tt.SelectedIndex
	kf := keymap.Of(e.KeyChord())
	if len(tt.rows) == 0 && !tt.IsReadOnly() {
		switch kf {
		case keymap.Insert, keymap.InsertAfter, keymap.InsertChild:
			tt.NewAt(-1) // the first item
			e.SetHandled()
		}
		return
	}
	if !tt.validIndex(idx) {
		return
	}
	if tt.IsReadOnly() {
		switch kf {
		case keymap.MoveRight:
			tt.Open(idx)
			e.SetHandled()
		case keymap.MoveLeft:
			if !tt.IsClosed(idx) && tt.numChildren(tt.rows[idx].item) > 0 {
				tt.Close(idx)
			} else if pi := tt.rowOf(tt.rows[idx].parent); tt.rows[idx].parent.IsValid() && pi >= 0 {
				tt.ScrollToIndex(pi)
				tt.updateSelectIndex(pi, true, events.SelectOne)
			}
			e.SetHandled()
		}
		return
	}
	switch kf {
	case keymap.Delete: // after the edits of a text field are accepted
		tt.deleteSelected()
		tt.IndexGrabFocus(tt.SelectedIndex) // if the focus was in the last row
	case keymap.Insert:
		tt.InsertBefore(idx)
	case keymap.InsertAfter:
		tt.InsertAfter(idx)
	case keymap.InsertChild:
		tt.AddChild(idx)
	case keymap.Duplicate:
		tt.Duplicate()
	default:
		return
	}
	tt.SelectMode = false
	e.SetHandled()
}

func (tt *TreeTable) contextMenu(m *Scene, pos image.Point) {
	idx := tt.SelectedIndex
	view := NewButton(m)
	view.OnClick(func(e events.Event) {
		tt.editIndex(idx)
	})
	if tt.IsReadOnly() {
		view.SetText("View").SetIcon(icons.Visibility)
		NewButton(m).SetText("Copy").SetIcon(icons.Copy).SetKey(keymap.Copy).OnClick(func(e events.Event) {
			tt.copyIndexes(true)
		})
	} else {
		view.SetText("Edit").SetIcon(icons.Edit)
		NewSeparator(m)
		NewButton(m).SetText("Insert before").SetIcon(icons.Add).SetKey(keymap.Insert).OnClick(func(e events.Event) {
			tt.InsertBefore(idx)
		})
		NewButton(m).SetText("Insert after").SetIcon(icons.Add).SetKey(keymap.InsertAfter).OnClick(func(e events.Event) {
			tt.InsertAfter(idx)
		})
		NewButton(m).SetText("Add child").SetIcon(icons.SubdirectoryArrowRight).SetKey(keymap.InsertChild).OnClick(func(e events.Event) {
			tt.AddChild(idx)
		})
		NewButton(m).SetText("Delete").SetIcon(icons.Delete).SetKey(keymap.Delete).OnClick(func(e events.Event) {
			tt.deleteSelected()
		})
		NewSeparator(m)
		NewButton(m).SetText("Indent").SetIcon(icons.FormatIndentIncrease).OnClick(func(e events.Event) {
			tt.Indent(idx)
		})
		NewButton(m).SetText("Outdent").SetIcon(icons.FormatIndentDecrease).OnClick(func(e events.Event) {
			tt.Outdent(idx)
		})
		NewButton(m).SetText("Move up").SetIcon(icons.ArrowUpward).OnClick(func(e events.Event) {
			tt.MoveItemUp(idx)
		})
		NewButton(m).SetText("Move down").SetIcon(icons.ArrowDownward).OnClick(func(e events.Event) {
			tt.MoveItemDown(idx)
		})
		NewSeparator(m)
		NewButton(m).SetText("Copy").SetIcon(icons.Copy).SetKey(keymap.Copy).OnClick(func(e events.Event) {
			tt.copyIndexes(true)
		})
		NewButton(m).SetText("Cut").SetIcon(icons.Cut).SetKey(keymap.Cut).OnClick(func(e events.Event) {
			tt.cutIndexes()
		})
		NewButton(m).SetText("Paste").SetIcon(icons.Paste).SetKey(keymap.Paste).OnClick(func(e events.Event) {
			tt.pasteIndex(idx)
		})
		NewButton(m).SetText("Duplicate").SetIcon(icons.Copy).SetKey(keymap.Duplicate).OnClick(func(e events.Event) {
			tt.Duplicate()
		})
	}
	NewSeparator(m)
	NewButton(m).SetText("Open all").SetIcon(icons.UnfoldMore).OnClick(func(e events.Event) {
		tt.OpenAll()
	})
	NewButton(m).SetText("Close all").SetIcon(icons.UnfoldLess).OnClick(func(e events.Event) {
		tt.CloseAll()
	})
	NewButton(m).SetText("Toggle indexes").SetIcon(icons.Numbers).OnClick(func(e events.Event) {
		tt.ShowIndexes = !tt.ShowIndexes
		tt.Update()
	})
}
