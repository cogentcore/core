// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package content

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"slices"
	"strings"
	"time"

	errs "cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/strcase"
	"cogentcore.org/core/content/bcontent"
)

// MountFS is a filesystem of content that combines a root filesystem with
// other filesystems mounted in it at URL prefixes, such as the documentation
// of several apps on one website (each also used for the help of its app with
// [SetAppHelp]). Make it with [NewMountFS] and [MountFS.Mount], and use it as
// the source of a [Content] with [Content.SetSource].
//
// The pages of a mounted filesystem are at its prefix: its root page (with
// URL = "" in its front matter, as for the home.md of the help of an app) has
// the URL of the prefix itself (such as "docs/lists"), and its other pages have
// their URLs under it (such as "docs/lists/settings"). The name of the root
// page is its Title, or else the sentence case of the last element of the
// prefix ("Lists"). The other pages keep their names unless a page of the root
// filesystem or of an earlier mount has the same name, in which case the name
// of the mount's root page is added in parentheses ("Settings (Cogent Lists)").
// Relative URLs of images and other resources in a mounted page are relative to
// the page, so that a mounted filesystem works the same as on its own.
type MountFS struct {

	// root is the root filesystem, which may be nil.
	root fs.FS

	// mounts are the mounted filesystems, in the order mounted.
	mounts []fsMount
}

// fsMount is a filesystem mounted in a [MountFS] at a prefix.
type fsMount struct {

	// prefix is the path of the mount, such as "docs/lists".
	prefix string

	// fsys is the mounted filesystem.
	fsys fs.FS
}

// NewMountFS returns a new [MountFS] with the given root filesystem, which may
// be nil for a filesystem that only has mounts.
func NewMountFS(root fs.FS) *MountFS {
	return &MountFS{root: root}
}

// Mount mounts the given filesystem at the given prefix, a slash-separated
// path such as "docs/lists" (see [MountFS]). It panics if the prefix is not a
// valid path of [fs.FS] other than ".", or if it is mounted already.
func (m *MountFS) Mount(prefix string, fsys fs.FS) *MountFS {
	if !fs.ValidPath(prefix) || prefix == "." {
		panic("content.MountFS: invalid mount prefix " + prefix)
	}
	if slices.ContainsFunc(m.mounts, func(mt fsMount) bool { return mt.prefix == prefix }) {
		panic("content.MountFS: prefix mounted already " + prefix)
	}
	m.mounts = append(m.mounts, fsMount{prefix: prefix, fsys: fsys})
	return m
}

// mountOf returns the mount that has the given path in it (the one with the
// longest prefix, for nested mounts), and the path relative to it, or nil.
func (m *MountFS) mountOf(name string) (mt *fsMount, rel string) {
	for i := range m.mounts {
		p := m.mounts[i].prefix
		if name != p && !strings.HasPrefix(name, p+"/") {
			continue
		}
		if mt == nil || len(p) > len(mt.prefix) {
			mt = &m.mounts[i]
		}
	}
	if mt == nil {
		return nil, ""
	}
	if name == mt.prefix {
		return mt, "."
	}
	return mt, name[len(mt.prefix)+1:]
}

// mountDirs returns the names of the next elements of the prefixes of
// the mounts in the given directory, which is not in a mount.
func (m *MountFS) mountDirs(dir string) []string {
	var names []string
	for _, mt := range m.mounts {
		rest := mt.prefix
		if dir != "." {
			if !strings.HasPrefix(mt.prefix, dir+"/") {
				continue
			}
			rest = mt.prefix[len(dir)+1:]
		}
		first, _, _ := strings.Cut(rest, "/")
		if !slices.Contains(names, first) {
			names = append(names, first)
		}
	}
	return names
}

// Open opens the named file, in a mount if it is in one, and otherwise in the
// root filesystem. The directories that contain mounts have the entries of
// those mounts, in addition to any entries of the root filesystem.
func (m *MountFS) Open(name string) (fs.File, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrInvalid}
	}
	if mt, rel := m.mountOf(name); mt != nil {
		f, err := mt.fsys.Open(rel)
		if err != nil || rel != "." {
			return f, err
		}
		return &mountRoot{File: f, name: path.Base(name)}, nil
	}
	if len(m.mountDirs(name)) > 0 {
		entries, err := m.ReadDir(name)
		if err != nil {
			return nil, err
		}
		return &mountDir{name: name, entries: entries}, nil
	}
	if m.root == nil {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}
	return m.root.Open(name)
}

// ReadDir reads the named directory, as with [MountFS.Open].
func (m *MountFS) ReadDir(name string) ([]fs.DirEntry, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: fs.ErrInvalid}
	}
	if mt, rel := m.mountOf(name); mt != nil {
		return fs.ReadDir(mt.fsys, rel)
	}
	dirs := m.mountDirs(name)
	var entries []fs.DirEntry
	if m.root != nil {
		re, err := fs.ReadDir(m.root, name)
		if err != nil && (len(dirs) == 0 || !errors.Is(err, fs.ErrNotExist)) {
			return nil, err
		}
		// The mounts take the place of root entries with the same names.
		entries = slices.DeleteFunc(re, func(e fs.DirEntry) bool {
			return slices.Contains(dirs, e.Name())
		})
	} else if len(dirs) == 0 {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: fs.ErrNotExist}
	}
	for _, d := range dirs {
		entries = append(entries, fs.FileInfoToDirEntry(m.dirStat(path.Join(name, d))))
	}
	slices.SortFunc(entries, func(a, b fs.DirEntry) int {
		return strings.Compare(a.Name(), b.Name())
	})
	return entries, nil
}

// dirStat returns the [fs.FileInfo] of the given directory that is a mount
// or contains mounts, which is not in a mount.
func (m *MountFS) dirStat(name string) fs.FileInfo {
	for _, mt := range m.mounts {
		if mt.prefix != name {
			continue
		}
		if fi, err := fs.Stat(mt.fsys, "."); err == nil {
			return renamedInfo{FileInfo: fi, name: path.Base(name)}
		}
	}
	return dirInfo(path.Base(name))
}

// renamedInfo is an [fs.FileInfo] with another name, for the root
// directory of a mount, which has the name of its prefix.
type renamedInfo struct {
	fs.FileInfo
	name string
}

func (r renamedInfo) Name() string { return r.name }

// mountRoot is the open root directory of a mount, which has the name
// of the last element of its prefix.
type mountRoot struct {
	fs.File
	name string
}

func (r *mountRoot) Stat() (fs.FileInfo, error) {
	fi, err := r.File.Stat()
	if err != nil {
		return nil, err
	}
	return renamedInfo{FileInfo: fi, name: r.name}, nil
}

func (r *mountRoot) ReadDir(n int) ([]fs.DirEntry, error) {
	d, ok := r.File.(fs.ReadDirFile)
	if !ok {
		return nil, &fs.PathError{Op: "readdir", Path: r.name, Err: errors.New("not implemented")}
	}
	return d.ReadDir(n)
}

// dirInfo is the [fs.FileInfo] of a directory of a [MountFS] that
// contains mounts.
type dirInfo string

func (d dirInfo) Name() string       { return string(d) }
func (d dirInfo) Size() int64        { return 0 }
func (d dirInfo) Mode() fs.FileMode  { return fs.ModeDir | 0555 }
func (d dirInfo) ModTime() time.Time { return time.Time{} }
func (d dirInfo) IsDir() bool        { return true }
func (d dirInfo) Sys() any           { return nil }

// mountDir is an open directory of a [MountFS] that contains mounts.
type mountDir struct {
	name    string
	entries []fs.DirEntry
	offset  int
}

func (d *mountDir) Stat() (fs.FileInfo, error) { return dirInfo(path.Base(d.name)), nil }
func (d *mountDir) Close() error               { return nil }

func (d *mountDir) Read([]byte) (int, error) {
	return 0, &fs.PathError{Op: "read", Path: d.name, Err: errors.New("is a directory")}
}

func (d *mountDir) ReadDir(n int) ([]fs.DirEntry, error) {
	rest := d.entries[d.offset:]
	if n <= 0 {
		d.offset = len(d.entries)
		return rest, nil
	}
	if len(rest) == 0 {
		return nil, io.EOF
	}
	n = min(n, len(rest))
	d.offset += n
	return rest[:n], nil
}

// addPage adds the given page to the pages of the content.
func (ct *Content) addPage(pg *bcontent.Page) {
	ct.pages = append(ct.pages, pg)
	ct.pagesByName[strings.ToLower(pg.Name)] = pg
	ct.pagesByURL[pg.URL] = pg
	for _, cat := range pg.Categories {
		ct.pagesByCategory[cat] = append(ct.pagesByCategory[cat], pg)
	}
}

// newMountedPage returns a new page for the file at the given path relative
// to the given mount of a [MountFS] source, with its URL under the prefix of
// the mount (see [MountFS]).
func (ct *Content) newMountedPage(mt *fsMount, rel string) (*bcontent.Page, error) {
	pg, err := bcontent.NewPage(mt.fsys, rel)
	if err != nil {
		return nil, err
	}
	pg.Source = ct.Source
	pg.Filename = path.Join(mt.prefix, rel)
	if pg.URL == "" {
		pg.URL = mt.prefix
		if pg.Title != pg.Name { // a title in the front matter
			pg.Name = pg.Title
		} else {
			pg.Name = strcase.ToSentence(path.Base(mt.prefix))
			pg.Title = pg.Name
		}
	} else {
		pg.URL = mt.prefix + "/" + pg.URL
	}
	ct.pageBases[pg] = path.Dir(pg.Filename) + "/"
	return pg, nil
}

// addMountedPages adds the given pages of the mounts of the given [MountFS]
// source, after the other pages, so that their names do not take the place
// of the names of the other pages: a name that is taken already gets the
// name of the root page of its mount added in parentheses.
func (ct *Content) addMountedPages(mfs *MountFS, pages []*bcontent.Page) {
	mountNames := map[string]string{} // by prefix
	for _, pg := range pages {
		if mt, _ := mfs.mountOf(pg.Filename); pg.URL == mt.prefix {
			mountNames[mt.prefix] = pg.Name
		}
	}
	for _, pg := range pages {
		if _, ok := ct.pagesByURL[pg.URL]; ok {
			errs.Log(fmt.Errorf("content: page %q of a mount has the URL of another page: %q", pg.Filename, pg.URL))
			continue
		}
		if _, ok := ct.pagesByName[strings.ToLower(pg.Name)]; ok {
			mt, _ := mfs.mountOf(pg.Filename)
			if mn := mountNames[mt.prefix]; mn != "" {
				pg.Name += " (" + mn + ")"
			}
		}
		ct.addPage(pg)
	}
}

// pageBase returns the base URL of the given page, against which the relative
// URLs of its resources are resolved: the directory of its file for a page in a
// mount of a [MountFS], and "" otherwise (the root of the source).
func (ct *Content) pageBase(pg *bcontent.Page) string {
	if pg == nil {
		return ""
	}
	return ct.pageBases[pg]
}
