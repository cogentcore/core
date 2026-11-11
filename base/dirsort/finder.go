// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dirsort

import (
	"os"
	"path/filepath"
)

// finderPrefs is the path of the preferences of the Finder, relative to
// the home directory, which has its standard view settings.
var finderPrefs = "Library/Preferences/com.apple.finder.plist"

// finderCache caches the sort orders of directories in the Finder.
var finderCache cache[nativeSort]

// finderSort returns the sort order of the given directory in the Finder,
// and whether it has one: that of the view settings in its .DS_Store file
// (see [dsStoreRecords]), for the view that it is shown in (vstl), or else
// that of the standard view settings of the Finder, for its preferred view.
// The list views (lsvC, lsvP, lsvp) have a sort column, with whether each
// column is ascending, and the icon and gallery views (icvp, glvp) arrange
// by a field, or none.
func finderSort(dir string) (Sort, bool) {
	ns := finderCache.get(dir, func() nativeSort {
		if s, ok := finderDirSort(dir); ok {
			return nativeSort{s, true}
		}
		// (keyed by "", which is not a clean directory)
		return finderCache.get("", finderStandardSort)
	})
	return ns.s, ns.ok
}

// finderDirSort returns the sort order of the given directory in the view
// settings of its .DS_Store file, and whether it has one.
func finderDirSort(dir string) (Sort, bool) {
	b, err := os.ReadFile(filepath.Join(dir, ".DS_Store"))
	if err != nil {
		return Sort{}, false
	}
	recs, err := dsStoreRecords(b, ".")
	if err != nil {
		return Sort{}, false
	}
	style, _ := recs["vstl"].(string)
	views := map[string]map[string]any{}
	for _, code := range []string{"lsvC", "lsvP", "lsvp", "icvp", "glvp"} {
		if blob, ok := recs[code].([]byte); ok {
			if pl, err := decodeBplist(blob); err == nil {
				if m, ok := pl.(map[string]any); ok {
					views[code] = m
				}
			}
		}
	}
	list := views["lsvC"]
	if list == nil {
		list = views["lsvP"]
	}
	if list == nil {
		list = views["lsvp"]
	}
	return viewSort(style, list, views["icvp"], views["glvp"])
}

// finderStandardSort returns the sort order of the standard view settings
// of the Finder, for its preferred view, and whether it has one.
func finderStandardSort() nativeSort {
	b, err := os.ReadFile(filepath.Join(homeDir(), finderPrefs))
	if err != nil {
		return nativeSort{}
	}
	pl, err := decodeBplist(b)
	if err != nil {
		return nativeSort{}
	}
	prefs, _ := pl.(map[string]any)
	style, _ := prefs["FXPreferredViewStyle"].(string)
	std, _ := prefs["StandardViewSettings"].(map[string]any)
	view := func(name string) map[string]any {
		m, _ := std[name].(map[string]any)
		return m
	}
	list := view("ExtendedListViewSettingsV2")
	if list == nil {
		list = view("ListViewSettings")
	}
	s, ok := viewSort(style, list, view("IconViewSettings"), view("GalleryViewSettings"))
	return nativeSort{s, ok}
}

// viewSort returns the sort order of the given view style of the Finder
// (as Nlsv for the list view), from the given view settings of its list,
// icon and gallery views, and whether it has one. The column view (clmv)
// is sorted as the list view, as is a view that is not sorted by itself.
func viewSort(style string, list, icon, gallery map[string]any) (Sort, bool) {
	switch style {
	case "icnv":
		if s, ok := arrangeSort(icon); ok {
			return s, true
		}
	case "glyv":
		if s, ok := arrangeSort(gallery); ok {
			return s, true
		}
	}
	return listSort(list)
}

// listSort returns the sort order of the given view settings of a list
// view, and whether it has one: its sort column, which is ascending as
// given for it in its columns (an array of them, by identifier, or a
// dictionary of them).
func listSort(list map[string]any) (Sort, bool) {
	col, _ := list["sortColumn"].(string)
	by, ok := byField(col)
	if !ok {
		return Sort{}, false
	}
	s := Sort{By: by, Down: downByDefault(by)}
	var colInfo map[string]any
	switch cols := list["columns"].(type) {
	case []any:
		for _, c := range cols {
			if cm, ok := c.(map[string]any); ok && cm["identifier"] == col {
				colInfo = cm
			}
		}
	case map[string]any:
		colInfo, _ = cols[col].(map[string]any)
	}
	if asc, ok := colInfo["ascending"].(bool); ok {
		s.Down = !asc
	}
	return s, true
}

// arrangeSort returns the sort order of the given view settings of an icon
// or gallery view, and whether it has one: the field that it is arranged
// by, which is not none, in its default direction.
func arrangeSort(view map[string]any) (Sort, bool) {
	arr, _ := view["arrangeBy"].(string)
	by, ok := byField(arr)
	if !ok {
		return Sort{}, false
	}
	return Sort{By: by, Down: downByDefault(by)}, true
}
