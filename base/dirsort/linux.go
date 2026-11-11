// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dirsort

import (
	"bufio"
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// linuxCache caches the sort orders of directories on Linux.
var linuxCache cache[nativeSort]

// linuxSort returns the sort order of the given directory in the file
// managers of Linux, and whether it has one: that of Dolphin (see
// [dolphinSort]) or Nautilus (see [nautilusSort]) for the directory, or
// else the default of the one of the desktop (KDE for Dolphin, else
// Nautilus).
func linuxSort(dir string) (Sort, bool) {
	ns := linuxCache.get(dir, func() nativeSort {
		if s, ok := dolphinSort(dir); ok {
			return nativeSort{s, true}
		}
		if s, ok := nautilusSort(dir); ok {
			return nativeSort{s, true}
		}
		// (keyed by "", which is not a clean directory)
		return linuxCache.get("", func() nativeSort {
			kde := strings.Contains(strings.ToUpper(os.Getenv("XDG_CURRENT_DESKTOP")), "KDE")
			defaults := []func() (Sort, bool){nautilusDefault, dolphinDefault}
			if kde {
				defaults[0], defaults[1] = defaults[1], defaults[0]
			}
			for _, def := range defaults {
				if s, ok := def(); ok {
					return nativeSort{s, true}
				}
			}
			return nativeSort{}
		})
	})
	return ns.s, ns.ok
}

// dolphinViews is the directory of the view properties of Dolphin,
// relative to the home directory: its global/.directory has those of all
// directories, when they are shared (GlobalViewProps in dolphinrc), and
// local/ those of the directories in which it can not write its .directory
// file, under their absolute paths.
var dolphinViews = ".local/share/dolphin/view_properties"

// dolphinSort returns the sort order of the given directory in Dolphin, in
// the [Dolphin] group of its .directory file, or else of the one for it in
// the view properties of Dolphin, and whether it has one. Dolphin sorts
// the directories that have no SortRole by name.
func dolphinSort(dir string) (Sort, bool) {
	if dolphinGlobal() {
		return Sort{}, false // (see dolphinDefault)
	}
	if s, ok := dolphinFile(filepath.Join(dir, ".directory")); ok {
		return s, true
	}
	return dolphinFile(filepath.Join(homeDir(), dolphinViews, "local", dir, ".directory"))
}

// dolphinDefault returns the sort order of all directories in Dolphin,
// when they share their view properties, and whether there is one.
func dolphinDefault() (Sort, bool) {
	if !dolphinGlobal() {
		return Sort{}, false
	}
	return dolphinFile(filepath.Join(homeDir(), dolphinViews, "global", ".directory"))
}

// dolphinGlobal returns whether Dolphin uses the same view properties for
// all directories (GlobalViewProps in the [General] group of dolphinrc).
func dolphinGlobal() bool {
	cfg := os.Getenv("XDG_CONFIG_HOME")
	if cfg == "" {
		cfg = filepath.Join(homeDir(), ".config")
	}
	b, err := os.ReadFile(filepath.Join(cfg, "dolphinrc"))
	if err != nil {
		return false
	}
	return strings.EqualFold(iniValues(b, "General")["GlobalViewProps"], "true")
}

// dolphinFile returns the sort order of the [Dolphin] group of the given
// .directory file, and whether it has one: by its SortRole (text being the
// name), down if its SortOrder is 1 (Qt::DescendingOrder).
func dolphinFile(fn string) (Sort, bool) {
	b, err := os.ReadFile(fn)
	if err != nil {
		return Sort{}, false
	}
	vals := iniValues(b, "Dolphin")
	role, order := vals["SortRole"], vals["SortOrder"]
	if role == "" && order == "" {
		return Sort{}, false
	}
	var s Sort
	if role != "" {
		by, ok := byField(role)
		if !ok {
			return Sort{}, false
		}
		s.By = by
	}
	s.Down = order == "1"
	return s, true
}

// iniValues returns the values of the given group of the given INI file.
func iniValues(b []byte, group string) map[string]string {
	vals := map[string]string{}
	in := false
	sc := bufio.NewScanner(bytes.NewReader(b))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, "[") {
			in = line == "["+group+"]"
			continue
		}
		if k, v, ok := strings.Cut(line, "="); in && ok {
			vals[strings.TrimSpace(k)] = strings.TrimSpace(v)
		}
	}
	return vals
}

// runCommand runs the given command, which has a second to run, and
// returns its output, or nil if it fails.
var runCommand = func(name string, args ...string) []byte {
	if _, err := exec.LookPath(name); err != nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, name, args...).Output()
	if err != nil {
		return nil
	}
	return out
}

// nautilusSort returns the sort order of the given directory in Nautilus,
// from the metadata of the directory in GVfs (as gio info shows it), and
// whether it has one: of its list view, or else of its icon (grid) view.
func nautilusSort(dir string) (Sort, bool) {
	out := runCommand("gio", "info", "-a", "metadata::*", dir)
	if out == nil {
		return Sort{}, false
	}
	meta := map[string]string{}
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		k, v, ok := strings.Cut(strings.TrimSpace(sc.Text()), ": ")
		if name, isMeta := strings.CutPrefix(k, "metadata::nautilus-"); ok && isMeta {
			meta[name] = v
		}
	}
	for _, view := range []struct{ by, reversed string }{
		{"list-view-sort-column", "list-view-sort-reversed"},
		{"icon-view-sort-by", "icon-view-sort-reversed"},
	} {
		if by, ok := byField(meta[view.by]); ok {
			return Sort{By: by, Down: meta[view.reversed] == "true"}, true
		}
	}
	return Sort{}, false
}

// nautilusDefault returns the default sort order of Nautilus, in its
// settings (default-sort-order and default-sort-in-reverse-order), and
// whether it has one.
func nautilusDefault() (Sort, bool) {
	const schema = "org.gnome.nautilus.preferences"
	order := strings.Trim(strings.TrimSpace(string(runCommand("gsettings", "get", schema, "default-sort-order"))), "'")
	by, ok := byField(order)
	if !ok {
		return Sort{}, false
	}
	rev := strings.TrimSpace(string(runCommand("gsettings", "get", schema, "default-sort-in-reverse-order")))
	return Sort{By: by, Down: rev == "true"}, true
}
