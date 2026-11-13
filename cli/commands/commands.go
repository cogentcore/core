// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package commands

import (
	"cmp"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"sync"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/sharedfile"
	"cogentcore.org/core/core"
)

// Entry is a command, or a category of commands, in the [Commands], which
// are edited in a [core.TreeTable] (see [Commands.Edit]). The top-level
// entries are the buttons of the command bar (see [Commands.Bar]): those
// of categories have menus of their entries, and those of commands with
// subcommands, such as git, have menus of their subcommands.
type Entry struct {

	// Name is the name of the command or category, which is shown for it.
	// The default for a command is its last word: "Git" for "git", and
	// "Commit" for "git commit".
	Name string

	// Command is the command, such as "git", or a subcommand, such as
	// "git commit", which is empty for a category.
	Command string `width:"30"`

	// Common are the subcommands of the command that are used the most,
	// such as "status add commit" for git, separated by spaces, which are
	// at the top of the menu of its subcommands, followed by a submenu of
	// the others, sorted by name. All of them are in the menu, by group,
	// if it is empty.
	Common string `width:"30" label:"Common subcommands" doc:"The subcommands used the most, separated by spaces, such as: status add commit, which are at the top of the menu of the command, followed by a submenu of the others"`

	// Children are the entries in the category.
	Children []*Entry `display:"-"`

	// Script is the script of an entry that is a script (see [Script]),
	// rather than a command, which is edited in its own editor.
	Script *Script `json:",omitempty" display:"-"`

	// Closed is whether the category is closed in the editor.
	Closed bool `display:"-" treetable:"closed"`
}

// IsCategory returns whether the entry is a category of commands: it has
// children, or no command or script.
func (en *Entry) IsCategory() bool {
	return len(en.Children) > 0 || (en.Command == "" && en.Script == nil)
}

// Label returns the text shown for the entry: its name, or else that of
// its script, or the last word of its command.
func (en *Entry) Label() string {
	if en.Name != "" {
		return en.Name
	}
	if en.Script != nil && en.Script.Name != "" {
		return en.Script.Name
	}
	return cmp.Or(label(en.Command), "Commands")
}

// version is the version of the format of the file of the [Commands]:
// the parsed commands of other versions are parsed again.
const version = 9

// Commands are the commands of the command bar, organized into categories
// by the user, with the cache of the parsed help of commands, which are
// saved in a JSON file (see [Open]), and the arguments last used for them,
// which are saved in another (see [Args]).
type Commands struct {

	// Version is the version of the format of the file.
	Version int

	// StandardVersion is the version of the [Standard] commands that the
	// entries have (see [addStandard]).
	StandardVersion int

	// Entries are the top-level entries, which are the buttons of the
	// command bar.
	Entries []*Entry

	// Parsed are the parsed commands, by command, such as "git", which are
	// parsed when they are first used (see [Commands.SubCmd]).
	Parsed map[string]*Cmd

	// Args are the arguments last used for the subcommands, which are saved
	// in their own file.
	Args *Args `json:"-"`

	// file is the file the commands are saved in, which other processes
	// (apps) share.
	file sharedfile.File[commandsData]

	// mu protects the commands and the saving of the file.
	mu sync.Mutex

	// views are the command bars and the editor of the commands, which are
	// updated when the commands change (see [Commands.Changed]).
	views struct {
		sync.Mutex
		bars   []*core.Toolbar
		editor *core.TreeTable
	}
}

// File is the file of the [Default] commands, which are shared by the
// apps: commands.json in the Cogent Core data directory, if it is empty.
var File string

// defaultCommands opens the [Default] commands once.
var defaultCommands = sync.OnceValue(func() *Commands {
	if File == "" {
		File = filepath.Join(core.TheApp.CogentCoreDataDir(), "commands.json")
	}
	return Open(File)
})

// Default returns the commands that are shared by the apps, such as Cogent
// Files and Cogent Code, opening them from the [File] the first time.
func Default() *Commands {
	return defaultCommands()
}

// Open returns the commands saved in the given JSON file, or the
// [Standard] ones if it does not exist yet, which are saved in it when
// they change, with the arguments last used for them, saved in the file of
// the same name ending in -args.json (see [OpenArgs]). The files may be
// shared by other processes (apps): they are read again when another
// process changes them. The help of the commands that have not been parsed
// yet is parsed in the background (see [Commands.ParseAll]).
func Open(filename string) *Commands {
	cs := &Commands{file: sharedfile.File[commandsData]{Path: filename}, Parsed: map[string]*Cmd{}}
	cs.mu.Lock()
	cs.syncLocked()
	if cs.Version < 2 { // (the first version had categories of git subcommands)
		cs.Entries = Standard()
		cs.StandardVersion = standardVersion
		cs.addStandardMacrosLocked()
	}
	if cs.Version < 3 { // (the common subcommands were added)
		addCommon(cs.Entries)
	}
	if cs.StandardVersion < standardVersion { // (the standard ones were added)
		cs.Entries = addStandard(cs.Entries)
		if cs.StandardVersion < 2 { // (and then the standard macros)
			cs.addStandardMacrosLocked()
		}
		cs.StandardVersion = standardVersion
	}
	cs.mu.Unlock()
	args := ""
	if filename != "" {
		args = strings.TrimSuffix(filename, ".json") + "-args.json"
	}
	cs.Args = OpenArgs(args)
	go cs.ParseAll()
	return cs
}

// commandsData are the data of the file of the [Commands].
type commandsData struct {
	Version         int
	StandardVersion int
	Entries         []*Entry
	Parsed          map[string]*Cmd

	// merged is whether the data have been merged into the commands, or
	// are theirs (as written): only data read from the file are merged.
	merged bool
}

// data returns the commands as the data of their file.
func (cs *Commands) data() commandsData {
	return commandsData{Version: cs.Version, StandardVersion: cs.StandardVersion, Entries: cs.Entries, Parsed: cs.Parsed}
}

// syncLocked reads the file of the commands if another process has
// changed it, under the mutex, returning whether it did (see
// [Commands.mergeLocked]).
func (cs *Commands) syncLocked() bool {
	if !cs.file.Changed() {
		return false
	}
	errors.Log(cs.file.Reload())
	if cs.file.Data.merged {
		return false
	}
	cs.mergeLocked(&cs.file.Data)
	cs.file.Data.merged = true
	return true
}

// mergeLocked merges the given data of the file into the commands, under
// the mutex: the entries are those of the file, and the parsed commands
// are merged with those parsed here. A file of another version only has
// its macros and scripts taken (made by the user), and one of the first
// version none.
func (cs *Commands) mergeLocked(fc *commandsData) {
	if fc.Version < 2 {
		return
	}
	cs.Version = max(cs.Version, fc.Version)
	cs.StandardVersion = max(cs.StandardVersion, fc.StandardVersion)
	cs.Entries = fc.Entries
	if fc.Version != version {
		for root, fcm := range fc.Parsed {
			if len(fcm.Macros) == 0 && len(fcm.Scripts) == 0 {
				continue
			}
			if cm := cs.Parsed[root]; cm != nil {
				cm.Macros, cm.Scripts = fcm.Macros, fcm.Scripts
			} else {
				cs.Parsed[root] = &Cmd{Cmd: root, Macros: fcm.Macros, Scripts: fcm.Scripts}
			}
		}
		return
	}
	for root, fcm := range fc.Parsed {
		if cm := cs.Parsed[root]; cm != nil && cm != fcm {
			for _, sc := range cm.SubCmds {
				if !sc.IsParsed() {
					continue
				}
				if i := slices.IndexFunc(fcm.SubCmds, func(f *SubCmd) bool { return f.Name == sc.Name }); i < 0 {
					fcm.SubCmds = append(fcm.SubCmds, sc)
				} else if !fcm.SubCmds[i].IsParsed() {
					fcm.SubCmds[i] = sc
				}
			}
		}
		cs.Parsed[root] = fcm
	}
}

// reload reads the file of the commands if another process has changed
// it, updating the command bars and the editor if it did.
func (cs *Commands) reload() {
	cs.mu.Lock()
	changed := cs.syncLocked()
	cs.mu.Unlock()
	if changed {
		cs.updateViews(nil)
	}
}

// update changes the commands with the given function, and saves them,
// with the file locked, after reading the changes of other processes.
func (cs *Commands) update(change func()) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	return cs.file.Update(func(d *commandsData) {
		if !d.merged { // (read again, as another process changed it)
			cs.mergeLocked(d)
		}
		change()
		cs.Version = version
		*d = cs.data()
		d.merged = true
	})
}

// Save saves the commands to their file, with the entries as they are
// here, as edited, and the parsed commands merged with those of other
// processes.
func (cs *Commands) Save() error {
	cs.mu.Lock()
	ens := cs.Entries
	cs.mu.Unlock()
	return cs.update(func() { cs.Entries = ens })
}

// cached returns the cached parse of the given command, such as "git",
// if any, as a copy, which is not changed by parsing.
func (cs *Commands) cached(cmd string) *Cmd {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.syncLocked()
	cm := cs.Parsed[cmd]
	if cm == nil {
		return nil
	}
	c := *cm
	c.SubCmds = slices.Clone(cm.SubCmds)
	return &c
}

// cachedSub returns the cached parse of the given subcommand, such as
// "git commit", or "ls" for the [MainName] one, if it has been parsed.
func (cs *Commands) cachedSub(cmd string) *SubCmd {
	root, name := splitCmd(cmd)
	cm := cs.cached(root)
	if cm == nil {
		return nil
	}
	if sc := cm.SubCmd(cmp.Or(name, MainName)); sc != nil && sc.IsParsed() {
		return sc
	}
	return nil
}

// Cmd returns the parsed command for the given command, such as "git",
// parsing its help and saving it if it has not been parsed yet, or if
// reparse is true, keeping the subcommands that have been parsed.
func (cs *Commands) Cmd(cmd string, reparse bool) (*Cmd, error) {
	if cm := cs.cached(cmd); cm != nil && cm.IsParsed() && !reparse {
		return cm, nil
	}
	cm, err := ParseCmd(cmd)
	if err != nil {
		return nil, err
	}
	err = cs.update(func() {
		if old := cs.Parsed[cmd]; old != nil {
			cm.Macros, cm.Scripts = old.Macros, old.Scripts
			for i, sc := range cm.SubCmds {
				if osc := old.SubCmd(sc.Name); osc != nil && osc.IsParsed() && !sc.IsParsed() {
					cm.SubCmds[i] = osc
				}
			}
		}
		cs.Parsed[cmd] = cm
	})
	return cs.cached(cmd), err
}

// SubCmd returns the parsed subcommand for the given command line, such
// as "git commit", or "ls" for the [MainName] subcommand of a command
// without subcommands, parsing its help and saving it if it has not been
// parsed yet, or if reparse is true. Its documentation defaults to that
// given by its command (or parent subcommand). A subcommand that is not
// one of those of the command, such as "git stash list", is added to them.
func (cs *Commands) SubCmd(cmd string, reparse bool) (*SubCmd, error) {
	cmd = strings.Join(strings.Fields(cmd), " ")
	if sc := cs.cachedSub(cmd); sc != nil && !reparse {
		return sc, nil
	}
	root, name := splitCmd(cmd)
	cm, err := cs.Cmd(root, false)
	if err != nil {
		return nil, err
	}
	if name == "" {
		if cm.HasSubCmds() {
			return nil, fmt.Errorf("commands: %s has subcommands", root)
		}
		name = MainName
	}
	if name == MainName && reparse {
		if _, err := cs.Cmd(root, true); err != nil {
			return nil, err
		}
		return cs.cachedSub(cmd), nil
	}
	pcmd := cmd
	if name == MainName { // (as for "ls Main", the form of a command with macros)
		pcmd = root
	}
	sc, err := parseSub(pcmd)
	if err != nil {
		return nil, err
	}
	sc.Name = name
	// a subcommand whose help is that of the command (as of bun run -h)
	// lists the subcommands of the command, not its own
	if copiedSubCmds(sc, cm) {
		sc.Cmds = nil
	}
	cs.addSharedFlags(root, sc)
	err = cs.update(func() {
		pc := cs.Parsed[root]
		if pc == nil {
			pc = cm
			cs.Parsed[root] = pc
		}
		doc := ""
		for _, other := range pc.SubCmds { // the doc of the command, or of the parent subcommand
			if other.Name == name {
				doc = cmp.Or(doc, other.Doc)
			}
			for _, sub := range other.Cmds {
				if sub.Cmd == cmd {
					doc = cmp.Or(doc, sub.Doc)
				}
			}
		}
		sc.Doc = cmp.Or(sc.Doc, doc)
		if i := slices.IndexFunc(pc.SubCmds, func(o *SubCmd) bool { return o.Name == name }); i >= 0 {
			sc.Group = pc.SubCmds[i].Group // (from the help of the command)
			pc.SubCmds[i] = sc
		} else {
			pc.SubCmds = append(pc.SubCmds, sc)
		}
	})
	return sc, err
}

// parseSub parses the help of a subcommand: [Parse], other than in tests.
var parseSub = Parse

// sharedFlags are the flags that the subcommands of a command share with
// one of its subcommands, which they name in their usage: those of go build,
// which go run, install, test, vet, get, list, clean, and generate have
// ("[build flags]" or "[build/test flags]"), but only describe in their
// help by referring to go build.
var sharedFlags = map[string]struct {
	usage *regexp.Regexp
	from  string
}{
	"go": {regexp.MustCompile(`\[build(/test)? flags`), "build"},
}

// addSharedFlags adds the [sharedFlags] of the given command to the given
// subcommand, if its usage names them, after its own flags, without those
// of the same names, parsing the subcommand that they are from if needed.
func (cs *Commands) addSharedFlags(root string, sc *SubCmd) {
	sf, ok := sharedFlags[root]
	if !ok || sc.Name == sf.from || !sf.usage.MatchString(sc.Usage) {
		return
	}
	from, err := cs.SubCmd(root+" "+sf.from, false)
	if errors.Log(err) != nil {
		return
	}
	sc.Flags = mergeFlags(sc.Flags, from.Flags)
}

// copiedSubCmds returns whether the subcommands of the given subcommand
// are mostly those of the given command: its help is that of the command.
func copiedSubCmds(sc *SubCmd, cm *Cmd) bool {
	if len(sc.Cmds) < 2 {
		return false
	}
	n := 0
	for _, sub := range sc.Cmds {
		if cm.SubCmd(sub.Cmd[strings.LastIndex(sub.Cmd, " ")+1:]) != nil {
			n++
		}
	}
	return 2*n >= len(sc.Cmds)
}

// ParseAll parses the help of all of the commands that have not been parsed
// yet, and of their subcommands, and then of the subcommands of those, such
// as git stash list, saving them, and updating the command bars. It is
// called in the background by [Open], so that the help is cached when the
// commands are used.
func (cs *Commands) ParseAll() {
	var cmds []string
	var add func(ens []*Entry)
	add = func(ens []*Entry) {
		for _, en := range ens {
			add(en.Children)
			if en.Command != "" {
				cmds = append(cmds, en.Command)
			}
		}
	}
	add(cs.Entries)
	parse := func(full string) *SubCmd {
		if sc := cs.cachedSub(full); sc != nil {
			return sc
		}
		sc, err := cs.SubCmd(full, false)
		errors.Log(err)
		return sc
	}
	// the subcommands first, and then the nested ones
	var nested []string
	for _, cmd := range cmds {
		root, name := splitCmd(cmd)
		cm, err := cs.Cmd(root, false)
		if errors.Log(err) != nil {
			continue
		}
		subs := []string{name}
		if name == "" {
			subs = nil
			for _, sc := range cm.SubCmds {
				subs = append(subs, sc.Name)
			}
		}
		for _, sub := range subs {
			full := root
			if sub != MainName {
				full += " " + sub
			}
			if sc := parse(full); sc != nil {
				for _, n := range sc.Cmds {
					nested = append(nested, n.Cmd)
				}
			}
		}
	}
	cs.updateViews(nil)
	for _, full := range nested {
		parse(full)
	}
	cs.updateViews(nil)
}

// standardCommon are the [Entry.Common] subcommands of the [Standard]
// commands. Those that a command does not have are ignored.
var standardCommon = map[string]string{
	"git":       "status add commit diff log show branch switch checkout merge rebase cherry-pick stash fetch pull push clone tag reset restore rm mv remote worktree blame grep clean",
	"brew":      "install upgrade update search info list uninstall outdated cleanup services",
	"port":      "install upgrade selfupdate search info installed uninstall outdated",
	"apt":       "install update upgrade search show list remove autoremove",
	"apt-get":   "install update upgrade dist-upgrade remove autoremove purge",
	"dnf":       "install upgrade search info list remove autoremove",
	"yum":       "install update search info list remove",
	"snap":      "install refresh find info list remove",
	"flatpak":   "install update search info list uninstall",
	"winget":    "install upgrade search show list uninstall",
	"choco":     "install upgrade search info list uninstall",
	"scoop":     "install update search info list uninstall",
	"npm":       "install run test update outdated list uninstall init",
	"pip3":      "install list show freeze uninstall download",
	"cargo":     "build run test check add install update clean doc",
	"systemctl": "status start stop restart reload enable disable list-units daemon-reload",
	"launchctl": "list print load unload start stop kickstart",
	"diskutil":  "list info mount unmount eject",
}

// standardVersion is the version of the [Standard] commands: the standard
// entries are added to those of a file of an earlier version, which does
// not have them (see [addStandard]), and the [StandardMacros] to those of
// a file before version 2.
const standardVersion = 2

// addCommon sets the [Entry.Common] subcommands of the given entries of the
// [Standard] commands that do not have them, for a file of a version from
// before they were added.
func addCommon(ens []*Entry) {
	for _, en := range ens {
		addCommon(en.Children)
		if en.Common == "" {
			en.Common = standardCommon[en.Command]
		}
	}
}

// addStandard adds the [Standard] entries that the given entries do not
// have to them, returning them: the top-level commands and categories, and
// the commands of the categories of the same names.
func addStandard(ens []*Entry) []*Entry {
	has := func(ens []*Entry, en *Entry) *Entry {
		for _, e := range ens {
			if (en.Command != "" && e.Command == en.Command) || (en.Command == "" && e.Command == "" && e.Name == en.Name) {
				return e
			}
		}
		return nil
	}
	for _, en := range Standard() {
		e := has(ens, en)
		if e == nil {
			ens = append(ens, en)
			continue
		}
		for _, c := range en.Children {
			if has(e.Children, c) == nil {
				e.Children = append(e.Children, c)
			}
		}
	}
	return ens
}

// standardCommands are the commands of the [Standard] categories, by
// category, on all platforms, and on the platforms of the given GOOS.
var standardCommands = []struct {
	name, goos string
	cmds       []string
}{
	{"Files", "", []string{"ls", "find", "cp", "mv", "rm", "mkdir", "ln", "chmod", "chown", "du", "df", "tar", "zip", "unzip", "rsync"}},
	{"Text", "", []string{"grep", "sed", "awk", "sort", "uniq", "wc", "head", "tail", "cut", "diff", "jq", "pdflatex", "latex", "latexmk", "bibtex", "biber"}},
	{"Dev", "", []string{"make", "cmake", "pkg-config", "gcc", "g++", "clang", "clang++", "gdb", "lldb", "cargo", "python3", "node"}},
	{"Packages", "darwin", []string{"brew", "port"}},
	{"Packages", "linux", []string{"apt", "apt-get", "apt-cache", "dpkg", "dnf", "yum", "pacman", "zypper", "snap", "flatpak"}},
	{"Packages", "windows", []string{"winget", "choco", "scoop"}},
	{"Packages", "", []string{"pip3", "npm"}},
	{"Network", "", []string{"curl", "wget", "ssh", "scp", "ping"}},
	{"System", "", []string{"ps", "kill", "pkill", "lsof", "uname"}},
	{"System", "darwin", []string{"open", "launchctl", "diskutil"}},
	{"System", "linux", []string{"systemctl", "journalctl"}},
}

// Standard returns the standard commands, which the user starts with: git,
// go, and core, with the menus of their subcommands, and categories of
// commands for files, text, development, packages, the network, and the
// system, those of this platform. Commands that are not installed are left
// out (see [installed]).
func Standard() []*Entry {
	var all []string
	for _, sc := range standardCommands {
		all = append(all, sc.cmds...)
	}
	have := installed(append(all, "git", "go", "core"))
	var ens []*Entry
	for _, cmd := range []string{"git", "go", "core"} {
		if have[cmd] {
			ens = append(ens, &Entry{Command: cmd, Common: standardCommon[cmd]})
		}
	}
	for _, sc := range standardCommands {
		if sc.goos != "" && sc.goos != runtime.GOOS {
			continue
		}
		i := slices.IndexFunc(ens, func(e *Entry) bool { return e.Command == "" && e.Name == sc.name })
		if i < 0 {
			ens = append(ens, &Entry{Name: sc.name, Closed: true})
			i = len(ens) - 1
		}
		for _, cmd := range sc.cmds {
			if have[cmd] {
				ens[i].Children = append(ens[i].Children, &Entry{Command: cmd, Common: standardCommon[cmd]})
			}
		}
	}
	return slices.DeleteFunc(ens, func(e *Entry) bool { return e.Command == "" && len(e.Children) == 0 })
}

// installed returns which of the given commands are installed: found in
// the PATH, or else by the login shell (whose PATH an app started from the
// desktop may not have), in one call for all of those.
func installed(cmds []string) map[string]bool {
	have := map[string]bool{}
	var rest []string
	for _, cmd := range cmds {
		if _, err := exec.LookPath(cmd); err == nil {
			have[cmd] = true
		} else {
			rest = append(rest, cmd)
		}
	}
	sh := os.Getenv("SHELL")
	if len(rest) == 0 || sh == "" || runtime.GOOS == "windows" {
		return have
	}
	quoted := make([]string, len(rest))
	for i, c := range rest {
		quoted[i] = shellQuote(c)
	}
	script := "for c in " + strings.Join(quoted, " ") + "; do command -v \"$c\" >/dev/null 2>&1 && echo \"$c\"; done"
	ctx, cancel := context.WithTimeout(context.Background(), HelpTimeout)
	defer cancel()
	out, _ := exec.CommandContext(ctx, sh, "-l", "-c", script).Output()
	for _, c := range strings.Fields(string(out)) {
		have[c] = true
	}
	return have
}

// StandardMacros are the macros of the standard commands, which the user
// starts with, by command: the high-frequency versions of git, go, and
// core commands (those of Cogent Code), which run in the current directory.
func StandardMacros() map[string][]*Macro {
	args := argsField
	m := func(sub, name, argv string, flags map[string]string, confirm bool, prompts ...string) *Macro {
		mc := &Macro{SubCmd: sub, Name: name, Confirm: confirm, Prompts: prompts}
		if argv != "" || flags != nil {
			mc.Values = &SubCmdArgs{Args: argv, Flags: flags}
		}
		return mc
	}
	on := func(flags ...string) map[string]string {
		fs := map[string]string{}
		for _, f := range flags {
			fs[f] = "true"
		}
		return fs
	}
	return map[string][]*Macro{
		"git": {
			m("status", "", "", nil, false),
			m("log", "", "", nil, false),
			m("diff", "", "", nil, false, args),
			m("add", "", "", nil, false, args),
			m("commit", "", "", on("--all"), false, "--message"),
			m("pull", "", "", nil, false, args),
			m("push", "", "origin HEAD", nil, false, args),
			m("push", "force", "origin HEAD", on("--force"), true, args),
			m("switch", "", "", nil, false, args),
			m("checkout", "", "", nil, false, args),
			m("restore", "", "", nil, false, args),
			m("stash", "", "", nil, false, args),
			m("branch", "", "", nil, false, args),
			m("rebase", "", "", nil, false, args),
			m("rebase", "continue", "--continue", nil, false),
			m("reset", "", "", nil, true, args),
			m("remote", "", "", nil, false, args),
		},
		"go": {
			m("build", "", "", on("-v"), false),
			m("build", "all", "./...", on("-v"), false),
			m("install", "", "", on("-v"), false),
			m("generate", "", "", nil, false),
			m("test", "", "", on("-v"), false, args),
			m("test", "all", "./...", on("-v"), false, args),
			m("vet", "", "", nil, false),
			m("mod tidy", "", "", nil, false),
			m("mod init", "", "", nil, false, args),
			m("get", "", "", nil, false, args),
			m("get", "update", "", on("-u"), false, args),
		},
		"core": {
			m("run", "", "", nil, false, args),
			m("install", "", "", nil, false, args),
			m("generate", "all", "./...", nil, false),
			m("changed", "", "", nil, false),
			m("pull", "", "", nil, false),
			m("release", "", "", nil, false, args),
			m("next-release", "", "", nil, true),
		},
	}
}

// addStandardMacrosLocked adds the [StandardMacros] to the commands, other
// than those that they have (of the same subcommand and name), under the
// mutex.
func (cs *Commands) addStandardMacrosLocked() {
	for root, mcs := range StandardMacros() {
		cm := cs.Parsed[root]
		if cm == nil {
			cm = &Cmd{Cmd: root}
			cs.Parsed[root] = cm
		}
		for _, mc := range mcs {
			if !slices.ContainsFunc(cm.Macros, mc.same) {
				cm.Macros = append(cm.Macros, mc)
			}
		}
	}
}
