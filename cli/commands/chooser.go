// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package commands

import (
	"context"
	"html"
	"image"
	"os"
	"os/exec"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"time"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/tree"
)

// Chooser is a type of the values of arguments, such as a git branch,
// whose values are chosen from the lines of the output of a shell command,
// such as git branch --format='%(refname:short)', run in the directory
// that the commands run in (see [Commands.Dir]) when they are chosen. The
// arguments of the subcommands of a command, and of scripts, can have a
// chooser (see [Cmd.Choosers] and [ScriptArg.Chooser]): their text fields
// have a button with a menu of its values, which can still be typed.
type Chooser struct {

	// Name is the name of the chooser, such as "Git branch".
	Name string

	// Command is the shell command whose lines of output are the values,
	// other than empty ones, without spaces around them. It can have
	// pipes, as in git status --short | cut -c4-.
	Command string `width:"50"`

	// Empty is what an empty value means, if anything, which is shown in
	// an empty field, such as "the current directory (.)".
	Empty string `json:",omitempty" width:"30"`

	// Doc is the documentation of the chooser.
	Doc string `json:",omitempty" width:"40"`
}

// ChooserTimeout is the longest time that the command of a [Chooser] can
// take to output its values.
var ChooserTimeout = 5 * time.Second

// Values returns the values of the chooser: the lines of the output of its
// command, run in the given directory, other than empty ones, without
// spaces around them, and without duplicates.
func (ch *Chooser) Values(dir string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), ChooserTimeout)
	defer cancel()
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/C", ch.Command)
	} else {
		cmd = exec.CommandContext(ctx, "/bin/sh", "-c", ch.Command)
	}
	cmd.Dir = dir
	out, err := cmd.Output()
	var vals []string
	for line := range strings.Lines(string(out)) {
		line = strings.TrimSpace(line)
		if line != "" && !slices.Contains(vals, line) {
			vals = append(vals, line)
		}
	}
	if err != nil && len(vals) == 0 {
		if ee, ok := err.(*exec.ExitError); ok && len(ee.Stderr) > 0 {
			return nil, errors.New(strings.TrimSpace(string(ee.Stderr)))
		}
		return nil, err
	}
	return vals, nil
}

// ChooserName is the name of a [Chooser] of the [Commands], or empty for
// none, which is chosen from their names in a form.
type ChooserName string

// Widget returns a chooser of the names of the choosers of the [Default]
// commands, which can also be typed, as for a new one.
func (cn *ChooserName) Widget() core.Value {
	ch := core.NewChooser().SetEditable(true)
	ch.Maker(func(p *tree.Plan) {}) // (keeps the chooser itself)
	items := []core.ChooserItem{{Value: ChooserName(""), Text: "None", Tooltip: "Text, typed"}}
	for _, c := range Default().choosers() {
		items = append(items, core.ChooserItem{Value: ChooserName(c.Name), Text: c.Name, Tooltip: html.EscapeString(cmpOr(c.Doc, c.Command))})
	}
	return ch.SetItems(items...)
}

// StandardChoosers are the choosers of the standard commands, which the
// user starts with (see [StandardChooserArgs]).
func StandardChoosers() []*Chooser {
	return []*Chooser{
		{Name: "Git branch", Command: "git branch --format='%(refname:short)'", Doc: "The local branches of the git repository"},
		{Name: "Git remote branch", Command: "git branch --remotes --format='%(refname:short)'", Doc: "The branches of the remotes of the git repository, such as origin/main"},
		{Name: "Git remote", Command: "git remote", Empty: "the remote of the current branch", Doc: "The remotes of the git repository, such as origin"},
		{Name: "Git tag", Command: "git tag --sort=-creatordate", Doc: "The tags of the git repository, newest first"},
		{Name: "Git changed file", Command: "git ls-files --modified --others --exclude-standard", Doc: "The files of the git repository that are changed or new (not ignored)"},
		{Name: "Go package", Command: goPackages, Empty: "the package in the current directory (.)", Doc: "The Go packages in the current directory (.), all of them (./...), and each one below it"},
	}
}

// goPackages is the command of the standard Go package chooser: the
// current directory, all of the packages below it, and the directory of
// each one, relative to it.
const goPackages = `echo .; echo ./...; go list -f '{{.Dir}}' ./... | sed -e "s|^$(pwd -P)/|./|" -e "s|^$PWD/|./|" -e "s|^$(pwd -P)\$|.|" -e "s|^$PWD\$|.|"`

// oldStandardChoosers are the commands of earlier versions of the
// [StandardChoosers], by name, which are updated to the current ones.
var oldStandardChoosers = map[string][]string{
	"Go package": {"go list ./..."},
}

// StandardChooserArgs are the choosers of the arguments of the subcommands
// of the standard commands, by command, and by the key of the argument
// (see [Cmd.Choosers]).
func StandardChooserArgs() map[string]map[string]string {
	return map[string]map[string]string{
		"git": {
			"checkout " + argsField:     "Git branch",
			"switch " + argsField:       "Git branch",
			"merge " + argsField:        "Git branch",
			"rebase " + argsField:       "Git branch",
			"branch " + argsField:       "Git branch",
			"branch --set-upstream-to":  "Git remote branch",
			"add " + argsField:          "Git changed file",
			"restore " + argsField:      "Git changed file",
			"push " + argsField:         "Git remote",
			"pull " + argsField:         "Git remote",
			"fetch " + argsField:        "Git remote",
			"cherry-pick " + argsField:  "Git branch",
			"log " + argsField:          "Git branch",
			"diff " + argsField:         "Git changed file",
			"tag --delete":              "Git tag",
			"show " + argsField:         "Git tag",
			"worktree add " + argsField: "Git branch",
		},
		"go": {
			"build " + argsField:   "Go package",
			"test " + argsField:    "Go package",
			"install " + argsField: "Go package",
			"vet " + argsField:     "Go package",
		},
	}
}

// choosers returns the choosers of the commands.
func (cs *Commands) choosers() []*Chooser {
	cs.reload()
	cs.mu.Lock()
	defer cs.mu.Unlock()
	return slices.Clone(cs.Choosers)
}

// Chooser returns the chooser of the given name, or nil if there is none.
func (cs *Commands) Chooser(name string) *Chooser {
	if name == "" {
		return nil
	}
	for _, c := range cs.choosers() {
		if c.Name == name {
			return c
		}
	}
	return nil
}

// chooserKey returns the key of the argument of the given key (see
// [SubCmd.argKey]) of the given subcommand in [Cmd.Choosers]: "checkout
// Arguments", "branch --set-upstream-to".
func chooserKey(sc *SubCmd, argKey string) string {
	return sc.Name + " " + argKey
}

// ArgChooser returns the name of the chooser of the argument of the given
// key (see [SubCmd.argKey]) of the given subcommand, or "" if it has none.
func (cs *Commands) ArgChooser(sc *SubCmd, argKey string) string {
	root, _ := splitCmd(sc.Cmd)
	cs.mu.Lock()
	defer cs.mu.Unlock()
	if cm := cs.Parsed[root]; cm != nil {
		return cm.Choosers[chooserKey(sc, argKey)]
	}
	return ""
}

// SetArgChooser sets the chooser of the argument of the given key (see
// [SubCmd.argKey]) of the given subcommand to that of the given name, or
// to none if it is "", and saves it.
func (cs *Commands) SetArgChooser(sc *SubCmd, argKey, name string) error {
	root, _ := splitCmd(sc.Cmd)
	return cs.update(func() {
		cm := cs.Parsed[root]
		if cm == nil {
			cm = &Cmd{Cmd: root}
			cs.Parsed[root] = cm
		}
		if cm.Choosers == nil {
			cm.Choosers = map[string]string{}
		}
		if name == "" {
			delete(cm.Choosers, chooserKey(sc, argKey))
		} else {
			cm.Choosers[chooserKey(sc, argKey)] = name
		}
	})
}

// addStandardChoosersLocked adds the [StandardChoosers] that the commands
// do not have (by name), and the [StandardChooserArgs] of the arguments
// that do not have a chooser, under the mutex.
func (cs *Commands) addStandardChoosersLocked() {
	for _, c := range StandardChoosers() {
		if !slices.ContainsFunc(cs.Choosers, func(o *Chooser) bool { return o.Name == c.Name }) {
			cs.Choosers = append(cs.Choosers, c)
		}
	}
	cs.updateStandardChoosersLocked()
	for root, args := range StandardChooserArgs() {
		cm := cs.Parsed[root]
		if cm == nil {
			cm = &Cmd{Cmd: root}
			cs.Parsed[root] = cm
		}
		if cm.Choosers == nil {
			cm.Choosers = map[string]string{}
		}
		for k, name := range args {
			if _, has := cm.Choosers[k]; !has && name != "" {
				cm.Choosers[k] = name
			}
		}
	}
}

// updateStandardChoosersLocked updates the [StandardChoosers] of earlier
// versions (see [oldStandardChoosers]), which the user has not changed, to
// the current ones, under the mutex.
func (cs *Commands) updateStandardChoosersLocked() {
	for _, c := range StandardChoosers() {
		for i, o := range cs.Choosers {
			if o.Name == c.Name && slices.Contains(oldStandardChoosers[c.Name], o.Command) {
				cs.Choosers[i] = c
			}
		}
	}
}

// dir returns the directory that the commands run in from the given
// context widget (see [Commands.Dir]).
func (cs *Commands) dir(ctx core.Widget) string {
	if cs.Dir != nil && ctx != nil {
		if d := cs.Dir(ctx); d != "" {
			return d
		}
	}
	d, _ := os.Getwd()
	return d
}

// chooserFunc returns the chooser of the field of the given name of a
// form, or nil if it has none.
type chooserFunc func(field string) *Chooser

// subCmdChoosers returns the [chooserFunc] of the forms of the given
// subcommand (see [SubCmd.FormStruct]).
func (cs *Commands) subCmdChoosers(sc *SubCmd) chooserFunc {
	names := sc.fieldNames()
	return func(field string) *Chooser {
		if i := slices.Index(names, field); i >= 0 {
			return cs.Chooser(cs.ArgChooser(sc, sc.argKey(i)))
		}
		return nil
	}
}

// scriptChoosers returns the [chooserFunc] of the form of the arguments of
// the given script (see [Script.FormStruct]).
func (cs *Commands) scriptChoosers(s *Script) chooserFunc {
	names := s.fieldNames()
	return func(field string) *Chooser {
		if i := slices.Index(names, field); i >= 0 && s.Args[i].Kind == ArgString {
			return cs.Chooser(string(s.Args[i].Chooser))
		}
		return nil
	}
}

// formFieldName returns the name of the field of the given value widget of
// a [core.Form], whose name is value-<field>-<type>, or "" if it is not one.
func formFieldName(w core.Widget) string {
	name, ok := strings.CutPrefix(w.AsTree().Name, "value-")
	if !ok {
		return ""
	}
	if i := strings.LastIndex(name, "-"); i >= 0 {
		return name[:i]
	}
	return ""
}

// addChooserButton adds a trailing button to the given text field of the
// field of the given name of a form, with a menu of the values of its
// chooser, given by the given function, if it has one, run in the
// directory of the given context widget, which is updated with the field,
// as the chooser of the field can change.
func (cs *Commands) addChooserButton(ctx core.Widget, tf *core.TextField, field string, chooser chooserFunc) {
	tf.Updater(func() {
		ch := chooser(field)
		if ch == nil {
			if tf.TrailingIcon == icons.ArrowDropDown {
				tf.TrailingIcon, tf.TrailingIconOnClick, tf.Placeholder = icons.None, nil, ""
			}
			return
		}
		tf.SetPlaceholder(ch.Empty)
		tf.SetTrailingIcon(icons.ArrowDropDown, func(e events.Event) {
			cs.chooserMenu(ctx, tf, ch, tf.ContextMenuPos(e))
		})
		tf.SetTooltip(html.EscapeString(ch.Name + ": " + cmpOr(ch.Doc, ch.Command)))
	})
}

// chooserMenu shows a menu of the values of the given chooser, run in the
// directory of the given context widget, at the given position, which sets
// the given text field to the chosen one.
func (cs *Commands) chooserMenu(ctx core.Widget, tf *core.TextField, ch *Chooser, pos image.Point) {
	vals, err := ch.Values(cs.dir(ctx))
	if err != nil {
		core.ErrorSnackbar(tf, err, "Error getting the values of "+ch.Name)
		return
	}
	core.NewMenu(func(m *core.Scene, _ image.Point) { chooserItems(m, tf, ch, vals) }, tf, pos).Run()
}

// chooserItems adds the buttons of the given values of the given chooser
// to the given menu, which set the given text field to them.
func chooserItems(m *core.Scene, tf *core.TextField, ch *Chooser, vals []string) {
	if len(vals) == 0 {
		core.NewText(m).SetText("No values of " + html.EscapeString(ch.Name))
	}
	for _, v := range vals {
		b := core.NewButton(m).SetText(v)
		if v == tf.Text() {
			b.SetIcon(icons.Check)
		}
		b.OnClick(func(e events.Event) {
			tf.SetText(v)
			tf.SendChange(e)
		})
	}
}

// choosersMenu adds the buttons to set the choosers of the arguments of
// the given subcommand, other than switches, to the given menu, which
// update the given form, followed by one to edit the choosers.
func (cs *Commands) choosersMenu(m *core.Scene, ctx core.Widget, sc *SubCmd, fm *core.Form) {
	chs := cs.choosers()
	for i := range sc.fieldNames() {
		if sc.argType(i).Kind() != reflect.String {
			continue
		}
		key := sc.argKey(i)
		label := argsField
		if i > 0 {
			label = sc.Flags[i-1].Label()
		}
		cur := cs.ArgChooser(sc, key)
		b := core.NewButton(m).SetText(html.EscapeString(label))
		if cur != "" {
			b.SetText(html.EscapeString(label + ": " + cur))
		}
		b.SetMenu(func(sm *core.Scene, _ image.Point) {
			add := func(text, chooser, tip string) {
				w := core.NewButton(sm).SetText(text)
				w.SetTooltip(html.EscapeString(tip))
				if chooser == cur {
					w.SetIcon(icons.Check)
				}
				w.OnClick(func(e events.Event) {
					if err := cs.SetArgChooser(sc, key, chooser); err != nil {
						core.ErrorSnackbar(ctx, err)
						return
					}
					fm.Update()
				})
			}
			add("None", "", "Text, typed")
			for _, c := range chs {
				add(c.Name, c.Name, cmpOr(c.Doc, c.Command))
			}
		})
	}
	core.NewSeparator(m)
	core.NewButton(m).SetText("Edit choosers").SetIcon(icons.Edit).
		SetTooltip("Edit the choosers, whose values are the lines of the output of a shell command").
		OnClick(func(e events.Event) { cs.EditChoosers(ctx) })
}

// ChooserArg is the chooser of an argument of a subcommand of a command,
// as edited in the editor of the choosers (see [Cmd.Choosers]).
type ChooserArg struct {

	// Command is the command: "git".
	Command string

	// SubCmd is the subcommand, its words after the command: "checkout",
	// "stash push", or empty for the command itself.
	SubCmd string `label:"Subcommand"`

	// Arg is the argument: Arguments for the other arguments, such as
	// files and branches, or the name of a flag with its dashes:
	// "--set-upstream-to".
	Arg string `label:"Argument"`

	// Chooser is the chooser of the values of the argument.
	Chooser ChooserName
}

// ChooserArgs returns the choosers of the arguments of the subcommands of
// the commands, sorted.
func (cs *Commands) ChooserArgs() []*ChooserArg {
	cs.reload()
	cs.mu.Lock()
	defer cs.mu.Unlock()
	var cas []*ChooserArg
	for root, cm := range cs.Parsed {
		for key, name := range cm.Choosers {
			i := strings.LastIndex(key, " ")
			if i < 0 {
				continue
			}
			sub := key[:i]
			if sub == MainName {
				sub = ""
			}
			cas = append(cas, &ChooserArg{Command: root, SubCmd: sub, Arg: key[i+1:], Chooser: ChooserName(name)})
		}
	}
	slices.SortFunc(cas, func(a, b *ChooserArg) int {
		return strings.Compare(a.Command+" "+a.SubCmd+" "+a.Arg, b.Command+" "+b.SubCmd+" "+b.Arg)
	})
	return cas
}

// SetChoosers sets the choosers of the commands, and of the arguments of
// their subcommands (in place of all of those they have), to the given
// ones, and saves them. Those without a name or command, and arguments
// without a command, argument, or chooser, are left out.
func (cs *Commands) SetChoosers(chs []*Chooser, args []*ChooserArg) error {
	chs = slices.DeleteFunc(slices.Clone(chs), func(c *Chooser) bool {
		c.Name, c.Command = strings.TrimSpace(c.Name), strings.TrimSpace(c.Command)
		return c.Name == "" || c.Command == ""
	})
	return cs.update(func() {
		cs.Choosers = chs
		for _, cm := range cs.Parsed {
			cm.Choosers = nil
		}
		for _, ca := range args {
			root := strings.TrimSpace(ca.Command)
			sub := strings.Join(strings.Fields(ca.SubCmd), " ")
			arg := strings.TrimSpace(ca.Arg)
			if root == "" || arg == "" || ca.Chooser == "" {
				continue
			}
			if sub == "" {
				sub = MainName
			}
			cm := cs.Parsed[root]
			if cm == nil {
				cm = &Cmd{Cmd: root}
				cs.Parsed[root] = cm
			}
			if cm.Choosers == nil {
				cm.Choosers = map[string]string{}
			}
			cm.Choosers[sub+" "+arg] = string(ca.Chooser)
		}
	})
}

// EditChoosers opens the editor of the choosers of the commands, in the
// context of the given widget (see [Commands.NewChoosersDialog]).
func (cs *Commands) EditChoosers(ctx core.Widget) {
	ctx = cs.context(ctx)
	runDialog(cs.NewChoosersDialog(ctx), ctx)
}

// NewChoosersDialog returns the body of the editor of the choosers of the
// commands: a table of their names, commands, and documentation, and a
// table of the choosers of the arguments of the subcommands, which saves
// them.
func (cs *Commands) NewChoosersDialog(ctx core.Widget) *core.Body {
	ctx = cs.context(ctx)
	d := core.NewBody("Choosers")
	core.NewText(d).SetType(core.TextSupporting).SetText("The values of an argument with a chooser, such as a git branch, are chosen from the lines of the output of its shell command, run in the directory of the commands, in a menu of the button of its text field. Add rows with the buttons at the top, or the context menu of a table (right-click).")
	var chs []*Chooser
	for _, c := range cs.choosers() {
		nc := *c
		chs = append(chs, &nc)
	}
	args := cs.ChooserArgs()
	core.NewText(d).SetType(core.TextTitleSmall).SetText("Choosers")
	tb := core.NewTable(d).SetSlice(&chs)
	core.NewText(d).SetType(core.TextTitleSmall).SetText("Arguments")
	core.NewText(d).SetType(core.TextSupporting).SetText("The choosers of the arguments of commands: the argument is Arguments for the other arguments, such as files and branches, or the name of a flag, such as --set-upstream-to. They can also be set with Choosers at the top of the form of a command, and those of scripts in their arguments.")
	at := core.NewTable(d).SetSlice(&args)
	d.AddTopBar(func(bar *core.Frame) {
		core.NewToolbar(bar).Maker(func(p *tree.Plan) {
			tree.Add(p, func(w *core.Button) {
				w.SetText("Add chooser").SetIcon(icons.Add).SetTooltip("Add a chooser at the end")
				w.OnClick(func(e events.Event) {
					chs = append(chs, &Chooser{Name: "Chooser"})
					tb.Update()
				})
			})
			tree.Add(p, func(w *core.Button) {
				w.SetText("Add argument").SetIcon(icons.Add).SetTooltip("Add the chooser of an argument of a command at the end")
				w.OnClick(func(e events.Event) {
					args = append(args, &ChooserArg{Arg: argsField})
					at.Update()
				})
			})
			tree.Add(p, func(w *core.Button) {
				w.SetText("Try").SetIcon(icons.PlayArrow)
				w.SetTooltip("Show the values of the selected chooser, in the directory of the commands")
				w.OnClick(func(e events.Event) {
					i := tb.SelectedIndex
					if i < 0 || i >= len(chs) {
						core.MessageSnackbar(w, "Select a chooser to try")
						return
					}
					vals, err := chs[i].Values(cs.dir(ctx))
					if err != nil {
						core.ErrorSnackbar(w, err)
						return
					}
					core.MessageDialog(w, html.EscapeString(strings.Join(vals, "\n")), chs[i].Name)
				})
			})
		})
	})
	d.AddBottomBar(func(bar *core.Frame) {
		d.AddCancel(bar)
		d.AddOK(bar).SetText("Save").SetIcon(icons.Save).OnClick(func(e events.Event) {
			if err := cs.SetChoosers(chs, args); err != nil {
				core.ErrorSnackbar(ctx, err)
				return
			}
			core.MessageSnackbar(ctx, "Saved the choosers")
		})
	})
	return d
}
