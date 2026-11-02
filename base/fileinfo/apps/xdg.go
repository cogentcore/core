// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package apps

import (
	"bufio"
	"errors"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// This file has the freedesktop (XDG) implementation, used on Linux and
// other unix systems: the apps are the desktop entries in the applications
// directories of the XDG data directories, and the default apps are in the
// mimeapps.list files of the XDG config directories, as in
// https://specifications.freedesktop.org/mime-apps-spec.

// desktopEntry is the freedesktop desktop entry of an app.
type desktopEntry struct {

	// ID is the desktop file ID, as in org.gnome.TextEditor.desktop.
	ID string

	// Name is the name of the app.
	Name string

	// Exec is the command line that opens files.
	Exec string

	// MimeTypes are the MIME types of the files that the app opens.
	MimeTypes []string

	// Hidden is whether the entry is deleted.
	Hidden bool
}

// envDirs returns the directories in the given XDG environment variables:
// the one for the user, with the given default under the home directory,
// and the list of the system ones, with the given defaults.
func envDirs(home, homeDefault, list string, listDefault ...string) []string {
	var dirs []string
	if d := os.Getenv(home); d != "" {
		dirs = append(dirs, d)
	} else if hd, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(hd, homeDefault))
	}
	if l := os.Getenv(list); l != "" {
		return append(dirs, filepath.SplitList(l)...)
	}
	return append(dirs, listDefault...)
}

// dataDirs returns the XDG data directories, the user's first.
func dataDirs() []string {
	return envDirs("XDG_DATA_HOME", ".local/share", "XDG_DATA_DIRS", "/usr/local/share", "/usr/share")
}

// configDirs returns the XDG config directories, the user's first.
func configDirs() []string {
	return envDirs("XDG_CONFIG_HOME", ".config", "XDG_CONFIG_DIRS", "/etc/xdg")
}

// readDesktopEntry reads the desktop entry of an app with the given ID
// from the given file, returning nil if it is not one.
func readDesktopEntry(id, file string) *desktopEntry {
	f, err := os.Open(file)
	if err != nil {
		return nil
	}
	defer f.Close()
	de := &desktopEntry{ID: id}
	inEntry, isApp := false, false
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		ln := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(ln, "[") {
			inEntry = ln == "[Desktop Entry]"
			continue
		}
		key, val, ok := strings.Cut(ln, "=")
		if !inEntry || !ok {
			continue
		}
		switch strings.TrimSpace(key) {
		case "Type":
			isApp = strings.TrimSpace(val) == "Application"
		case "Name":
			de.Name = strings.TrimSpace(val)
		case "Exec":
			de.Exec = strings.TrimSpace(val)
		case "MimeType":
			for mt := range strings.SplitSeq(val, ";") {
				if mt = strings.TrimSpace(mt); mt != "" {
					de.MimeTypes = append(de.MimeTypes, mt)
				}
			}
		case "Hidden":
			de.Hidden = strings.TrimSpace(val) == "true"
		}
	}
	if !isApp || sc.Err() != nil {
		return nil
	}
	return de
}

// desktopEntries returns the desktop entries of the apps by ID, from the
// applications directories of the data directories, of which the first
// has precedence. The ID of an entry in a subdirectory has its path,
// with - instead of /.
func desktopEntries() map[string]*desktopEntry {
	des := map[string]*desktopEntry{}
	for _, dd := range dataDirs() {
		ad := filepath.Join(dd, "applications")
		filepath.WalkDir(ad, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".desktop") {
				return nil
			}
			rel, _ := filepath.Rel(ad, path)
			id := strings.ReplaceAll(filepath.ToSlash(rel), "/", "-")
			if _, has := des[id]; has {
				return nil
			}
			if de := readDesktopEntry(id, path); de != nil {
				des[id] = de
			}
			return nil
		})
	}
	return des
}

// mimeApps are the associations of the mimeapps.list files.
type mimeApps struct {

	// defaults are the default apps for each MIME type, in order of preference.
	defaults map[string][]string

	// added and removed are the apps associated with each MIME type,
	// and those not associated with it, in addition to the desktop entries.
	added, removed map[string][]string
}

// mimeAppsFiles returns the mimeapps.list files, in order of precedence.
func mimeAppsFiles() []string {
	var files []string
	for _, cd := range configDirs() {
		files = append(files, filepath.Join(cd, "mimeapps.list"))
	}
	for _, dd := range dataDirs() {
		files = append(files, filepath.Join(dd, "applications", "mimeapps.list"))
	}
	return files
}

// readMimeApps reads the associations of the mimeapps.list files.
func readMimeApps() *mimeApps {
	ma := &mimeApps{defaults: map[string][]string{}, added: map[string][]string{}, removed: map[string][]string{}}
	for _, file := range mimeAppsFiles() {
		f, err := os.Open(file)
		if err != nil {
			continue
		}
		var sect map[string][]string
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			ln := strings.TrimSpace(sc.Text())
			switch ln {
			case "[Default Applications]":
				sect = ma.defaults
				continue
			case "[Added Associations]":
				sect = ma.added
				continue
			case "[Removed Associations]":
				sect = ma.removed
				continue
			}
			if strings.HasPrefix(ln, "[") {
				sect = nil
				continue
			}
			mt, ids, ok := strings.Cut(ln, "=")
			if sect == nil || !ok {
				continue
			}
			for id := range strings.SplitSeq(ids, ";") {
				if id = strings.TrimSpace(id); id != "" {
					sect[mt] = append(sect[mt], id) // earlier files first
				}
			}
		}
		f.Close() // the associations read before an error are used
	}
	return ma
}

// xdgApps returns the apps for the given MIME type, with the default first.
func xdgApps(mime string) []App {
	des := desktopEntries()
	ma := readMimeApps()
	var as []App
	has := func(id string) bool {
		return slices.ContainsFunc(as, func(a App) bool { return a.ID == id })
	}
	add := func(id string, def bool) {
		de := des[id]
		if de == nil || de.Hidden || has(id) || slices.Contains(ma.removed[mime], id) {
			return
		}
		as = append(as, App{Name: de.Name, ID: id, Default: def})
	}
	for _, id := range ma.defaults[mime] {
		if len(as) == 0 {
			add(id, true) // the first one that is installed
		}
	}
	for _, id := range ma.added[mime] {
		add(id, false)
	}
	ids := make([]string, 0, len(des))
	for id := range des {
		ids = append(ids, id)
	}
	slices.SortFunc(ids, func(a, b string) int { return strings.Compare(des[a].Name, des[b].Name) })
	for _, id := range ids {
		if slices.Contains(des[id].MimeTypes, mime) {
			add(id, false)
		}
	}
	return as
}

// execArgs returns the command line of the given Exec value of a desktop
// entry to open the given file, splitting it into arguments as the shell
// does with double quotes and backslashes, and replacing the field codes:
// %f, %F, %u and %U by the file (or its URL), and the others by nothing.
func execArgs(exec, file string) ([]string, error) {
	var args []string
	var cur strings.Builder
	inArg, quoted, hasFile := false, false, false
	rs := []rune(exec)
	for i := 0; i < len(rs); i++ {
		r := rs[i]
		switch {
		case r == '"':
			quoted = !quoted
			inArg = true
		case r == '\\' && i+1 < len(rs):
			i++
			cur.WriteRune(rs[i])
			inArg = true
		case r == '%' && i+1 < len(rs):
			i++
			switch rs[i] {
			case 'f', 'F':
				cur.WriteString(file)
				hasFile, inArg = true, true
			case 'u', 'U':
				cur.WriteString((&url.URL{Scheme: "file", Path: file}).String())
				hasFile, inArg = true, true
			case '%':
				cur.WriteRune('%')
				inArg = true
			}
		case (r == ' ' || r == '\t') && !quoted:
			if inArg {
				args = append(args, cur.String())
				cur.Reset()
				inArg = false
			}
		default:
			cur.WriteRune(r)
			inArg = true
		}
	}
	if inArg {
		args = append(args, cur.String())
	}
	if len(args) == 0 {
		return nil, errors.New("apps: empty Exec command")
	}
	if !hasFile {
		args = append(args, file)
	}
	return args, nil
}

// setXDGDefault makes the app with the given ID the default app for the
// given MIME type in the user's mimeapps.list file, keeping its other
// contents.
func setXDGDefault(mime, id string) error {
	file := filepath.Join(configDirs()[0], "mimeapps.list")
	var lines []string
	if b, err := os.ReadFile(file); err == nil {
		lines = strings.Split(strings.TrimRight(string(b), "\n"), "\n")
	}
	entry := mime + "=" + id + ";"
	sect := slices.Index(lines, "[Default Applications]")
	if sect < 0 {
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, "[Default Applications]", entry)
	} else {
		end := len(lines)
		set := false
		for i := sect + 1; i < len(lines); i++ {
			ln := strings.TrimSpace(lines[i])
			if strings.HasPrefix(ln, "[") {
				end = i
				break
			}
			if strings.HasPrefix(ln, mime+"=") {
				lines[i] = entry
				set = true
				break
			}
		}
		if !set {
			lines = slices.Insert(lines, end, entry)
		}
	}
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return err
	}
	return os.WriteFile(file, []byte(strings.Join(lines, "\n")+"\n"), 0o644)
}
