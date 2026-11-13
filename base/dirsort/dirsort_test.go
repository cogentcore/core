// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dirsort

import (
	"encoding/binary"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
	"unicode/utf16"

	"cogentcore.org/core/base/fileinfo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	for _, s := range []Sort{{}, {By: ModTime, Down: true}, {By: Kind}, {By: Size, Down: true}} {
		p, ok := Parse(s.String())
		assert.True(t, ok, s.String())
		assert.Equal(t, s, p)
	}
	assert.Equal(t, "ModTime:down", Sort{By: ModTime, Down: true}.String(), "as core.Table.SortFieldName")
	s, ok := Parse("Size")
	assert.True(t, ok)
	assert.Equal(t, Sort{By: Size}, s)
	for _, bad := range []string{"", "Color:up", "Name:sideways"} {
		_, ok := Parse(bad)
		assert.False(t, ok, bad)
	}
}

func TestSortFiles(t *testing.T) {
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	files := func() []*fileinfo.FileInfo {
		return []*fileinfo.FileInfo{
			{Name: "b.go", Size: 30, Kind: "Code: Go", ModTime: t0.Add(2 * time.Hour)},
			{Name: "A.txt", Size: 10, Kind: "Text: Plain", ModTime: t0},
			{Name: "c.go", Size: 30, Kind: "Code: Go", ModTime: t0.Add(time.Hour)},
			{Name: "a.md", Size: 20, Kind: "Text: Markdown", ModTime: t0.Add(3 * time.Hour)},
		}
	}
	names := func(fs []*fileinfo.FileInfo) []string {
		var ns []string
		for _, f := range fs {
			ns = append(ns, f.Name)
		}
		return ns
	}
	for _, tt := range []struct {
		s    Sort
		want []string
	}{
		{Sort{}, []string{"a.md", "A.txt", "b.go", "c.go"}},
		{Sort{Down: true}, []string{"c.go", "b.go", "A.txt", "a.md"}},
		{Sort{By: Size}, []string{"A.txt", "a.md", "b.go", "c.go"}}, // (by name if equal)
		{Sort{By: Size, Down: true}, []string{"c.go", "b.go", "a.md", "A.txt"}},
		{Sort{By: Kind}, []string{"b.go", "c.go", "a.md", "A.txt"}},
		{Sort{By: ModTime, Down: true}, []string{"a.md", "b.go", "c.go", "A.txt"}},
	} {
		fs := files()
		tt.s.SortFiles(fs)
		assert.Equal(t, tt.want, names(fs), tt.s.String())
	}
}

// testInfo is a [fs.FileInfo] for tests.
type testInfo struct {
	name    string
	size    int64
	modTime time.Time
	dir     bool
}

func (ti testInfo) Name() string       { return ti.name }
func (ti testInfo) Size() int64        { return ti.size }
func (ti testInfo) Mode() fs.FileMode  { return 0644 }
func (ti testInfo) ModTime() time.Time { return ti.modTime }
func (ti testInfo) IsDir() bool        { return ti.dir }
func (ti testInfo) Sys() any           { return nil }

func TestSortInfos(t *testing.T) {
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	infos := []fs.FileInfo{
		testInfo{"z.go", 5, t0, false},
		testInfo{"sub", 0, t0.Add(time.Hour), true},
		testInfo{"a.md", 50, t0.Add(2 * time.Hour), false},
	}
	names := func() []string {
		var ns []string
		for _, i := range infos {
			ns = append(ns, i.Name())
		}
		return ns
	}
	Sort{By: ModTime, Down: true}.SortInfos(infos)
	assert.Equal(t, []string{"a.md", "sub", "z.go"}, names())
	Sort{By: Size}.SortInfos(infos)
	assert.Equal(t, []string{"sub", "z.go", "a.md"}, names())
	Sort{By: Kind}.SortInfos(infos) // Code: Go, Doc: Markdown, Folder
	assert.Equal(t, []string{"z.go", "a.md", "sub"}, names())
	Sort{}.SortInfos(infos)
	assert.Equal(t, []string{"a.md", "sub", "z.go"}, names())
}

// TestStore tests that the sort orders that are set are saved in the
// shared file, and that another process (another store) reads them again
// when they change, also when many of them set sort orders at once.
func TestStore(t *testing.T) {
	DataDir = t.TempDir()
	t.Cleanup(func() { DataDir = "" })
	theStore = store{}
	dir := t.TempDir()
	_, ok := Saved(dir)
	assert.False(t, ok)
	require.NoError(t, Set(dir, Sort{By: ModTime, Down: true}))
	s, ok := Saved(dir)
	assert.True(t, ok)
	assert.Equal(t, Sort{By: ModTime, Down: true}, s)
	assert.Equal(t, s, For(dir))
	s, ok = Saved(dir + "/sub/..") // (the same directory)
	assert.True(t, ok)
	link := filepath.Join(t.TempDir(), "link")
	require.NoError(t, os.Symlink(dir, link))
	_, ok = Saved(link)
	assert.True(t, ok, "by a symbolic link")

	// another process reads it, and its changes are read here
	var other store
	s, ok = other.get(cleanDir(dir))
	assert.True(t, ok)
	assert.Equal(t, Sort{By: ModTime, Down: true}, s)
	time.Sleep(10 * time.Millisecond) // (for a newer modification time)
	require.NoError(t, other.set(cleanDir(dir), Sort{By: Size}))
	s, _ = Saved(dir)
	assert.Equal(t, Sort{By: Size}, s, "read again after another process changed it")

	// many processes at once each keep the changes of the others
	var wg sync.WaitGroup
	for i := range 8 {
		wg.Go(func() {
			var st store
			for j := range 5 {
				assert.NoError(t, st.set(fmt.Sprintf("/p%d/d%d", i, j), Sort{By: Kind}))
			}
		})
	}
	wg.Wait()
	var last store
	last.reload()
	assert.Len(t, last.file.Data, 8*5+1)
	_, err := os.Stat(filepath.Join(DataDir, FileName+".lck"))
	assert.True(t, os.IsNotExist(err), "unlocked")
}

// TestStorePrune tests that the sort orders set least recently are dropped
// beyond MaxDirs.
func TestStorePrune(t *testing.T) {
	DataDir = t.TempDir()
	defer func() { DataDir, MaxDirs = "", 5000 }()
	MaxDirs = 2
	var st store
	require.NoError(t, st.set("/old", Sort{}))
	st.file.Data["/old"] = entry{Time: 1}
	st.file.Data["/mid"] = entry{Time: 2}
	require.NoError(t, st.file.Update(nil)) // (as another process)
	require.NoError(t, st.set("/new", Sort{}))
	assert.Len(t, st.file.Data, 2)
	assert.NotContains(t, st.file.Data, "/old")
}

func TestBplist(t *testing.T) {
	b, err := os.ReadFile("testdata/list.bplist")
	require.NoError(t, err)
	v, err := decodeBplist(b)
	require.NoError(t, err)
	m := v.(map[string]any)
	assert.Equal(t, "dateModified", m["sortColumn"])
	assert.Equal(t, 16.5, m["iconSize"])
	assert.Equal(t, "Ünïcödé ✓", m["title"])
	assert.Equal(t, int64(1234567890123), m["big"])
	assert.Equal(t, int64(-5), m["negative"])
	assert.Equal(t, []byte{1, 2, 3}, m["data"])
	assert.Equal(t, []any{}, m["empty"])
	assert.Len(t, m["fifteen"], 16, "a count after the marker")
	cols := m["columns"].([]any)
	assert.Equal(t, true, cols[1].(map[string]any)["ascending"])

	// not valid, or cut short anywhere
	_, err = decodeBplist([]byte("not a plist at all, not at all, not at all"))
	assert.Error(t, err)
	for i := range len(b) {
		decodeBplist(b[:i]) // (no panic)
	}
}

// dsRecord is a record of a .DS_Store file for [encodeDSStore].
type dsRecord struct {
	name, code, typ string
	value           any
}

// encodeDSStore returns a .DS_Store file with the given records, which
// must be sorted by name, in a B-tree of a leaf, or of an internal node
// with leaves of the records before and after the middle one.
func encodeDSStore(recs []dsRecord, internal bool) []byte {
	be := binary.BigEndian
	rec := func(r dsRecord) []byte {
		u := utf16.Encode([]rune(r.name))
		b := be.AppendUint32(nil, uint32(len(u)))
		for _, c := range u {
			b = be.AppendUint16(b, c)
		}
		b = append(b, r.code...)
		b = append(b, r.typ...)
		switch v := r.value.(type) {
		case []byte:
			b = be.AppendUint32(b, uint32(len(v)))
			b = append(b, v...)
		case string:
			if r.typ == "type" {
				return append(b, v...)
			}
			u := utf16.Encode([]rune(v))
			b = be.AppendUint32(b, uint32(len(u)))
			for _, c := range u {
				b = be.AppendUint16(b, c)
			}
		case bool:
			if v {
				return append(b, 1)
			}
			return append(b, 0)
		case int64:
			return be.AppendUint32(b, uint32(v))
		}
		return b
	}
	leaf := func(rs []dsRecord) []byte {
		b := be.AppendUint32(nil, 0)
		b = be.AppendUint32(b, uint32(len(rs)))
		for _, r := range rs {
			b = append(b, rec(r)...)
		}
		return b
	}
	// blocks: 0 the root, 1 the DSDB header, 2 the top node, 3, 4 leaves
	var nodes [][]byte
	if internal {
		mid := len(recs) / 2
		top := be.AppendUint32(nil, 4) // (the last child)
		top = be.AppendUint32(top, 1)
		top = be.AppendUint32(top, 3)
		top = append(top, rec(recs[mid])...)
		nodes = [][]byte{top, leaf(recs[:mid]), leaf(recs[mid+1:])}
	} else {
		nodes = [][]byte{leaf(recs)}
	}
	hdr := be.AppendUint32(nil, 2)
	hdr = append(hdr, make([]byte, 16)...)
	blocks := append([][]byte{nil, hdr}, nodes...)
	// each block in 4096 bytes at 4096*(i+1), after the 4 bytes of the start
	const size = 4096
	for i, b := range blocks {
		blocks[i] = append(b, make([]byte, size-len(b))...)
	}
	root := be.AppendUint32(nil, uint32(len(blocks)))
	root = be.AppendUint32(root, 0)
	for i := range blocks {
		root = be.AppendUint32(root, uint32(size*(i+1))|12) // (2^12)
	}
	root = append(root, make([]byte, 4*(256-len(blocks)))...)
	root = be.AppendUint32(root, 1)
	root = append(root, 4)
	root = append(root, "DSDB"...)
	root = be.AppendUint32(root, 1)
	blocks[0] = append(root, make([]byte, size-len(root))...)

	out := be.AppendUint32(nil, 1)
	out = append(out, "Bud1"...)
	out = be.AppendUint32(out, size) // (the root block)
	out = be.AppendUint32(out, size)
	out = be.AppendUint32(out, size)
	out = append(out, make([]byte, size+4-len(out))...)
	for _, b := range blocks {
		out = append(out, b...)
	}
	return out
}

func TestDSStore(t *testing.T) {
	list, err := os.ReadFile("testdata/list.bplist")
	require.NoError(t, err)
	recs := []dsRecord{
		{".", "bwsp", "blob", []byte{9}},
		{".", "lsvC", "blob", list},
		{".", "vstl", "type", "Nlsv"},
		{"a file", "Iloc", "blob", []byte{1, 2}},
		{"b ✓", "cmmt", "ustr", "a comment"},
		{"c", "dilc", "bool", true},
		{"d", "ICVO", "long", int64(7)},
	}
	for _, internal := range []bool{false, true} {
		b := encodeDSStore(recs, internal)
		got, err := dsStoreRecords(b, ".")
		require.NoError(t, err)
		assert.Equal(t, map[string]any{"bwsp": []byte{9}, "lsvC": list, "vstl": "Nlsv"}, got)
		got, err = dsStoreRecords(b, "b ✓")
		require.NoError(t, err)
		assert.Equal(t, map[string]any{"cmmt": "a comment"}, got)
		got, _ = dsStoreRecords(b, "d")
		assert.Equal(t, map[string]any{"ICVO": int64(7)}, got)
		for i := 0; i < len(b); i += 97 {
			dsStoreRecords(b[:i], ".") // (no panic)
		}
	}
	_, err = dsStoreRecords([]byte("not a .DS_Store file at all, at all, at all"), ".")
	assert.Error(t, err)
}

// TestFinder tests the sort order of a directory from the view settings of
// the Finder in its .DS_Store file, and else from the standard ones.
func TestFinder(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	list, err := os.ReadFile("testdata/list.bplist")
	require.NoError(t, err)
	icon, err := os.ReadFile("testdata/icon.bplist")
	require.NoError(t, err)
	write := func(dir string, recs ...dsRecord) {
		require.NoError(t, os.WriteFile(filepath.Join(dir, ".DS_Store"), encodeDSStore(recs, false), 0644))
	}
	listDir, iconDir, plainDir := t.TempDir(), t.TempDir(), t.TempDir()
	write(listDir, dsRecord{".", "icvp", "blob", icon}, dsRecord{".", "lsvC", "blob", list}, dsRecord{".", "vstl", "type", "Nlsv"})
	write(iconDir, dsRecord{".", "icvp", "blob", icon}, dsRecord{".", "lsvC", "blob", list}, dsRecord{".", "vstl", "type", "icnv"})

	s, ok := finderDirSort(listDir)
	assert.True(t, ok)
	assert.Equal(t, Sort{By: ModTime}, s, "the list view, by its sort column, ascending as given")
	s, ok = finderDirSort(iconDir)
	assert.True(t, ok)
	assert.Equal(t, Sort{By: Size, Down: true}, s, "the icon view, arranged by size")
	_, ok = finderDirSort(plainDir)
	assert.False(t, ok)

	// the standard view settings, for a directory without its own
	assert.False(t, finderStandardSort().ok)
	prefs := filepath.Join(home, finderPrefs)
	require.NoError(t, os.MkdirAll(filepath.Dir(prefs), 0755))
	// (the icon view, arranged by size, made by plutil)
	b, err := os.ReadFile("testdata/prefs.bplist")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(prefs, b, 0644))
	ns := finderStandardSort()
	assert.True(t, ns.ok)
	assert.Equal(t, Sort{By: Size, Down: true}, ns.s)

	// the column view, and a view that is not sorted, are sorted as lists
	lm := map[string]any{"sortColumn": "kind", "columns": map[string]any{"kind": map[string]any{"ascending": false}}}
	s, ok = viewSort("clmv", lm, nil, nil)
	assert.True(t, ok)
	assert.Equal(t, Sort{By: Kind, Down: true}, s)
	s, ok = viewSort("icnv", lm, map[string]any{"arrangeBy": "none"}, nil)
	assert.True(t, ok)
	assert.Equal(t, Sort{By: Kind, Down: true}, s)
	_, ok = viewSort("Nlsv", map[string]any{"sortColumn": "label"}, nil, nil)
	assert.False(t, ok, "a column that is not a field")
}

func TestLinux(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	dir := t.TempDir()
	write := func(fn, s string) {
		require.NoError(t, os.MkdirAll(filepath.Dir(fn), 0755))
		require.NoError(t, os.WriteFile(fn, []byte(s), 0644))
	}

	// Dolphin, in the .directory file of the directory
	write(filepath.Join(dir, ".directory"), "[Desktop Entry]\nIcon=folder\n[Dolphin]\nSortRole=modificationtime\nSortOrder=1\n")
	s, ok := dolphinSort(dir)
	assert.True(t, ok)
	assert.Equal(t, Sort{By: ModTime, Down: true}, s)
	write(filepath.Join(dir, ".directory"), "[Dolphin]\nViewMode=1\nSortOrder=1\n")
	s, ok = dolphinSort(dir)
	assert.True(t, ok)
	assert.Equal(t, Sort{Down: true}, s, "by name without a role")
	write(filepath.Join(dir, ".directory"), "[Dolphin]\nViewMode=1\n")
	_, ok = dolphinSort(dir)
	assert.False(t, ok)

	// in the view properties of Dolphin, for a directory it can not write
	ro := t.TempDir()
	write(filepath.Join(home, dolphinViews, "local", ro, ".directory"), "[Dolphin]\nSortRole=size\n")
	s, ok = dolphinSort(ro)
	assert.True(t, ok)
	assert.Equal(t, Sort{By: Size}, s)

	// the same for all directories
	_, ok = dolphinDefault()
	assert.False(t, ok)
	write(filepath.Join(home, ".config", "dolphinrc"), "[General]\nGlobalViewProps=true\n")
	write(filepath.Join(home, dolphinViews, "global", ".directory"), "[Dolphin]\nSortRole=type\n")
	_, ok = dolphinSort(ro)
	assert.False(t, ok, "not for each directory")
	s, ok = dolphinDefault()
	assert.True(t, ok)
	assert.Equal(t, Sort{By: Kind}, s)

	// Nautilus, from the metadata of the directory, and its settings
	defer func(f func(string, ...string) []byte) { runCommand = f }(runCommand)
	outputs := map[string]string{
		"gio":                          "standard::name: x\nmetadata::nautilus-list-view-sort-column: date_modified\nmetadata::nautilus-list-view-sort-reversed: true\n",
		"gsettings default-sort-order": "'size'\n",
		"gsettings default-sort-in-reverse-order": "false\n",
	}
	runCommand = func(name string, args ...string) []byte {
		if name == "gsettings" {
			name += " " + args[len(args)-1]
		}
		if out, ok := outputs[name]; ok {
			return []byte(out)
		}
		return nil
	}
	s, ok = nautilusSort(dir)
	assert.True(t, ok)
	assert.Equal(t, Sort{By: ModTime, Down: true}, s)
	outputs["gio"] = "metadata::nautilus-icon-view-sort-by: type\n"
	s, ok = nautilusSort(dir)
	assert.True(t, ok)
	assert.Equal(t, Sort{By: Kind}, s)
	s, ok = nautilusDefault()
	assert.True(t, ok)
	assert.Equal(t, Sort{By: Size}, s)
}
