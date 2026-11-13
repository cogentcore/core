// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package commands

import (
	"cmp"
	"fmt"
	"html"
	"image"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/strcase"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/tree"
)

// RunFunc runs the given command line, such as in a terminal.
type RunFunc func(cmdline string)

// Changed saves the commands, and updates the command bars and the editor
// of the commands, other than the given editor, in which the change was
// made (or none if it is nil).
func (cs *Commands) Changed(editor *core.TreeTable) {
	errors.Log(cs.Save())
	cs.updateViews(editor)
}

// updateViews updates the command bars and the editor of the commands,
// other than the given editor (or none if it is nil).
func (cs *Commands) updateViews(editor *core.TreeTable) {
	cs.views.Lock()
	cs.views.bars = slices.DeleteFunc(cs.views.bars, func(tb *core.Toolbar) bool { return tb.This == nil })
	var widgets []core.Widget
	for _, tb := range cs.views.bars {
		widgets = append(widgets, tb)
	}
	if tt := cs.views.editor; tt != nil && tt.This != nil && tt != editor {
		widgets = append(widgets, tt)
	}
	cs.views.Unlock()
	for _, w := range widgets {
		// (not with the lock of this window, which another window may be
		// waiting on)
		go func() {
			wb := w.AsWidget()
			wb.AsyncLock()
			defer wb.AsyncUnlock()
			if wb.This != nil {
				wb.Update()
			}
		}()
	}
}

// Bar adds the command bar to the given parent: a button for each
// top-level entry, which shows the form of its command (see
// [Commands.RunDialog]) to run with the given function, or the menu of
// the entries of a category, or of the subcommands of a command, and which
// can be dragged to another place in the bar, followed by a button to edit
// the commands.
func (cs *Commands) Bar(parent core.Widget, run RunFunc) *core.Toolbar {
	tb := core.NewToolbar(parent)
	tb.SetName("commands")
	tb.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 0)
	})
	tb.MoveFunc = func(from, to int) {
		ens := cs.Entries
		if from < 0 || from >= len(ens) || to < 0 || to >= len(ens) {
			return
		}
		en := ens[from]
		ens = slices.Delete(ens, from, from+1)
		cs.Entries = slices.Insert(ens, to, en)
		cs.Changed(nil)
	}
	// the changes of other apps are shown when the bar is shown, or the
	// mouse enters it (the menus are made from the file as it is)
	tb.OnShow(func(e events.Event) { go cs.reload() })
	tb.On(events.MouseEnter, func(e events.Event) { go cs.reload() })
	tb.Maker(func(p *tree.Plan) {
		for _, en := range cs.Entries {
			tree.AddAt(p, fmt.Sprintf("entry-%p", en), func(w *core.Button) {
				w.SetType(core.ButtonAction)
				w.OnClick(func(e events.Event) {
					if w.Menu == nil && !en.IsCategory() {
						cs.RunDialog(w, en.Command, run)
					}
				})
				w.Updater(func() { cs.configEntry(w, en, run) })
			})
		}
		tree.AddAt(p, "edit-commands", func(w *core.Button) {
			w.SetType(core.ButtonAction).SetIcon(icons.Edit).SetTooltip("Edit the commands and their categories")
			w.OnClick(func(e events.Event) { cs.Edit(w) })
		})
	})
	cs.views.Lock()
	cs.views.bars = append(cs.views.bars, tb)
	cs.views.Unlock()
	return tb
}

// Menu adds the commands to the given menu, as in the command bar (see
// [Commands.Bar]), running their command lines with the given function,
// followed by a button to edit the commands, as for the Command menu of an
// app.
func (cs *Commands) Menu(m *core.Scene, run RunFunc) {
	cs.reload()
	cs.mu.Lock()
	ens := cs.Entries
	cs.mu.Unlock()
	cs.menu(m, ens, run)
	core.NewSeparator(m)
	core.NewButton(m).SetText("Edit commands").SetIcon(icons.Edit).SetTooltip("Edit the commands and their categories").
		OnClick(func(e events.Event) { cs.Edit(m) })
}

// configEntry configures the given button, of the command bar or of a
// menu, for the given entry: a category has the menu of its entries, and
// a command that of its subcommands (see [Commands.configCommand]).
func (cs *Commands) configEntry(w *core.Button, en *Entry, run RunFunc) {
	w.SetText(en.Label())
	if en.IsCategory() {
		w.SetIcon(icons.Folder).SetTooltip(en.Label())
		w.SetMenu(func(m *core.Scene, _ image.Point) { cs.menu(m, en.Children, run) })
		return
	}
	cs.configCommand(w, en.Command, run)
}

// configCommand configures the given button for the given command or
// subcommand: a command with subcommands, such as git, has the menu of
// them, as does a subcommand with subcommands, such as git stash, and
// otherwise clicking it shows the form of the command (see
// [Commands.RunDialog], which the caller does on a click without a menu).
// A command is parsed when it is first clicked, if it has not been parsed
// yet, and its menu shown if it has subcommands.
func (cs *Commands) configCommand(w *core.Button, cmd string, run RunFunc) {
	w.SetIcon(icons.Terminal)
	w.Menu = nil
	root, name := splitCmd(cmd)
	tip := cmd
	if cm := cs.cached(root); cm != nil {
		if name == "" && (cm.HasSubCmds() || len(cm.Macros) > 0) {
			w.SetTooltip(tooltip(root, cm.Doc))
			w.SetMenu(func(m *core.Scene, _ image.Point) { cs.subCmdsMenu(m, root, run) })
			return
		}
		if sc := cm.SubCmd(cmp.Or(name, MainName)); sc != nil {
			tip = tooltip(cmd, cmp.Or(sc.Doc, cm.Doc))
			if len(sc.Cmds) > 0 {
				w.SetTooltip(tip)
				w.SetMenu(func(m *core.Scene, _ image.Point) { cs.nestedMenu(m, sc, run) })
				return
			}
		}
	}
	w.SetTooltip(tip)
}

// tooltip returns the tooltip of the given command with the given
// documentation, as markup.
func tooltip(cmd, doc string) string {
	if doc == "" {
		return html.EscapeString(cmd)
	}
	return html.EscapeString(cmd + ": " + doc)
}

// commandButton adds a button for the given command or subcommand to the
// given menu, with the given text (see [Commands.configCommand]).
func (cs *Commands) commandButton(m *core.Scene, text, cmd string, run RunFunc) {
	b := core.NewButton(m).SetText(text)
	cs.configCommand(b, cmd, run)
	b.OnClick(func(e events.Event) {
		if b.Menu == nil {
			cs.RunDialog(b, cmd, run)
		}
	})
}

// menu adds the buttons of the given entries to the given menu: a category
// has a submenu of its entries, and a command with subcommands a submenu
// of them.
func (cs *Commands) menu(m *core.Scene, ens []*Entry, run RunFunc) {
	if len(ens) == 0 {
		core.NewText(m).SetText("(no commands)")
		return
	}
	for _, en := range ens {
		if en.IsCategory() {
			b := core.NewButton(m)
			cs.configEntry(b, en, run)
			continue
		}
		cs.commandButton(m, en.Label(), en.Command, run)
	}
}

// subCmdsMenu adds the buttons of the subcommands of the given command,
// such as git, to the given menu: those of the command, not the nested
// ones (see [Commands.nestedMenu]), after its macros (see [Macro]), or
// the command itself, if it has none. If the command has [Entry.Common]
// subcommands, they are in the menu, followed by a submenu of the others,
// sorted by name. Otherwise, the subcommands of the first group (see
// [SubCmd.Group]) are in the menu, and those of the others in submenus
// after them.
func (cs *Commands) subCmdsMenu(m *core.Scene, root string, run RunFunc) {
	cm := cs.cached(root)
	if cm == nil {
		return
	}
	cs.macrosMenu(m, cm, run)
	if !cm.HasSubCmds() { // the command itself
		// (its form, not the menu of its macros, which this is)
		b := core.NewButton(m).SetText(label(root)).SetIcon(icons.Terminal).SetTooltip(tooltip(root, cm.Doc))
		b.OnClick(func(e events.Event) { cs.RunDialog(b, root+" "+MainName, run) })
		return
	}
	var subs []*SubCmd
	for _, sc := range cm.SubCmds {
		if !strings.Contains(sc.Name, " ") { // not nested
			subs = append(subs, sc)
		}
	}
	add := func(m *core.Scene, scs []*SubCmd) {
		for _, sc := range scs {
			cs.commandButton(m, sc.Label(), sc.Cmd, run)
		}
	}
	submenu := func(text string, scs []*SubCmd) {
		b := core.NewButton(m).SetText(text).SetIcon(icons.Folder)
		b.SetMenu(func(sm *core.Scene, _ image.Point) { add(sm, scs) })
	}
	var top, rest []*SubCmd
	if common := strings.Fields(cs.common(root)); len(common) > 0 {
		rest = slices.Clone(subs)
		for _, name := range common {
			if i := slices.IndexFunc(rest, func(sc *SubCmd) bool { return sc.Name == name }); i >= 0 {
				top = append(top, rest[i])
				rest = slices.Delete(rest, i, i+1)
			}
		}
	}
	if len(top) > 0 { // (otherwise, as without common subcommands)
		subs = rest
		add(m, top)
		if len(subs) > 0 {
			slices.SortFunc(subs, func(a, b *SubCmd) int { return strings.Compare(a.Name, b.Name) })
			core.NewSeparator(m)
			submenu("More", subs)
		}
		return
	}
	var groups []string
	byGroup := map[string][]*SubCmd{}
	for _, sc := range subs {
		if _, ok := byGroup[sc.Group]; !ok {
			groups = append(groups, sc.Group)
		}
		byGroup[sc.Group] = append(byGroup[sc.Group], sc)
	}
	for i, g := range groups {
		if i == 0 {
			add(m, byGroup[g])
			continue
		}
		if i == 1 {
			core.NewSeparator(m)
		}
		submenu(cmp.Or(g, "Other"), byGroup[g])
	}
}

// common returns the [Entry.Common] subcommands of the first entry of the
// given command, such as git, if any.
func (cs *Commands) common(root string) string {
	var find func(ens []*Entry) string
	find = func(ens []*Entry) string {
		for _, en := range ens {
			if en.Command == root && en.Common != "" {
				return en.Common
			}
			if c := find(en.Children); c != "" {
				return c
			}
		}
		return ""
	}
	cs.mu.Lock()
	defer cs.mu.Unlock()
	return find(cs.Entries)
}

// nestedMenu adds the buttons of the given subcommand, such as git stash,
// and of its subcommands, such as git stash list, to the given menu.
func (cs *Commands) nestedMenu(m *core.Scene, sc *SubCmd, run RunFunc) {
	b := core.NewButton(m).SetText(sc.Label()).SetIcon(icons.Terminal).SetTooltip(tooltip(sc.Cmd, sc.Doc))
	b.OnClick(func(e events.Event) { cs.RunDialog(b, sc.Cmd, run) })
	core.NewSeparator(m)
	for _, sub := range sc.Cmds {
		cs.commandButton(m, label(sub.Cmd), sub.Cmd, run)
	}
}

// Edit opens the editor of the commands, a [core.TreeTable] of them in a new
// window, in which they are saved as they are edited.
func (cs *Commands) Edit(ctx core.Widget) {
	if core.RecycleMainWindow(&cs.Entries) {
		return
	}
	b := core.NewBody("Commands")
	b.SetData(&cs.Entries)
	tt := core.NewTreeTable(b)
	tt.SetDragMoves(true)
	tt.SetItems(&cs.Entries)
	tt.OnChange(func(e events.Event) { cs.Changed(tt) })
	cs.views.Lock()
	cs.views.editor = tt
	cs.views.Unlock()
	b.OnClose(func(e events.Event) {
		cs.views.Lock()
		if cs.views.editor == tt {
			cs.views.editor = nil
		}
		cs.views.Unlock()
		go cs.ParseAll() // for new commands
	})
	b.AddTopBar(func(bar *core.Frame) {
		core.NewToolbar(bar).Maker(func(p *tree.Plan) {
			tree.Add(p, func(w *core.Button) {
				w.SetText("Add category").SetIcon(icons.CreateNewFolder).SetTooltip("Add a category of commands at the end")
				w.OnClick(func(e events.Event) {
					cs.Entries = append(cs.Entries, &Entry{Name: "New category"})
					tt.Update()
					cs.Changed(tt)
				})
			})
			tree.Add(p, func(w *core.Button) {
				w.SetText("Add command").SetIcon(icons.Add).SetTooltip("Add a command at the end, such as git log, to be entered")
				w.OnClick(func(e events.Event) {
					cs.Entries = append(cs.Entries, &Entry{Name: "New command"})
					tt.Update()
					cs.Changed(tt)
				})
			})
			tree.Add(p, func(w *core.Button) {
				w.SetText("Reset").SetIcon(icons.Reset).SetTooltip("Replace the commands with the standard ones")
				w.OnClick(func(e events.Event) {
					d := core.NewBody("Reset commands")
					core.NewText(d).SetType(core.TextSupporting).SetText("Replace all of the commands and categories with the standard ones?")
					d.AddBottomBar(func(bar *core.Frame) {
						d.AddCancel(bar)
						d.AddOK(bar).SetText("Reset").OnClick(func(e events.Event) {
							cs.Entries = Standard()
							tt.Update()
							cs.Changed(tt)
						})
					})
					d.RunDialog(w)
				})
			})
		})
	})
	b.RunWindow()
}

// RunDialog shows the form of the flags and arguments of the given
// subcommand, such as "git commit", or command without subcommands, such
// as "ls", once its help is parsed, which runs the resulting command line
// with the given function. For a command with subcommands, such as git, it
// shows the menu of its subcommands, as the menu of the given widget.
func (cs *Commands) RunDialog(ctx core.Widget, cmd string, run RunFunc) {
	ctx = cs.context(ctx)
	if sc := cs.cachedSub(cmd); sc != nil {
		cs.runDialog(ctx, sc, run)
		return
	}
	core.MessageSnackbar(ctx, "Reading the help of "+cmd+"...")
	go func() {
		root, name := splitCmd(cmd)
		cm, err := cs.Cmd(root, false)
		var sc *SubCmd
		if err == nil && !(name == "" && (cm.HasSubCmds() || len(cm.Macros) > 0)) {
			sc, err = cs.SubCmd(cmd, false)
		}
		cs.updateViews(nil)
		ctx.AsWidget().AsyncLock()
		defer ctx.AsWidget().AsyncUnlock()
		switch {
		case err != nil:
			core.ErrorSnackbar(ctx, err)
		case sc == nil: // the menu of the subcommands
			core.NewMenu(func(m *core.Scene, _ image.Point) { cs.subCmdsMenu(m, root, run) }, ctx, ctx.AsWidget().ContextMenuPos(nil)).Run()
		default:
			cs.runDialog(ctx, sc, run)
		}
	}()
}

// runDialog shows the form of the given subcommand (see [Commands.RunDialog]).
func (cs *Commands) runDialog(ctx core.Widget, sc *SubCmd, run RunFunc) {
	runDialog(cs.NewDialog(ctx, sc, run), ctx)
}

// NewDialog returns the body of the dialog of the form of the given
// subcommand (see [Commands.RunDialog]), in the context of the given
// widget, which runs the command line with the given function. The form
// starts with the arguments last used for the subcommand, which are saved
// when it is run (see [Args]).
func (cs *Commands) NewDialog(ctx core.Widget, sc *SubCmd, run RunFunc) *core.Body {
	ctx = cs.context(ctx)
	d := core.NewBody(sc.Cmd)
	if sc.Doc != "" {
		core.NewText(d).SetType(core.TextSupporting).SetText(html.EscapeString(sc.Doc))
	}
	if sc.Usage != "" {
		core.NewText(d).SetType(core.TextSupporting).SetText(html.EscapeString(sc.Usage)).Styler(func(s *styles.Style) {
			s.Font.Family = rich.Monospace
		})
	}
	values := sc.FormStruct()
	if cs.Args != nil {
		cs.Args.SetValues(sc, values)
	}
	line := core.NewText(d)
	line.Styler(func(s *styles.Style) {
		s.Font.Family = rich.Monospace
		s.Grow.Set(1, 0)
	})
	line.Updater(func() { line.SetText(html.EscapeString(sc.CommandLine(values))) })
	fm := newForm(d, values)
	fm.OnChange(func(e events.Event) { line.Update() })
	fm.OnInput(func(e events.Event) { line.Update() })
	d.AddTopBar(func(bar *core.Frame) {
		core.NewToolbar(bar).Maker(func(p *tree.Plan) {
			if len(sc.Cmds) > 0 {
				tree.Add(p, func(w *core.Button) {
					w.SetText("Subcommands").SetIcon(icons.Menu)
					w.SetMenu(func(m *core.Scene, _ image.Point) {
						for _, sub := range sc.Cmds {
							core.NewButton(m).SetText(sub.Cmd).SetTooltip(html.EscapeString(sub.Doc)).OnClick(func(e events.Event) {
								d.Close()
								cs.RunDialog(ctx, sub.Cmd, run)
							})
						}
					})
				})
			}
			root, _ := splitCmd(sc.Cmd)
			cs.macrosButton(p, d, ctx, root, sc, run)
			tree.Add(p, func(w *core.Button) {
				w.SetText("Clear").SetIcon(icons.ClearAll).SetTooltip("Clear the flags and arguments")
				w.OnClick(func(e events.Event) {
					reflect.ValueOf(values).Elem().SetZero()
					fm.Update()
					line.Update()
				})
			})
			tree.Add(p, func(w *core.Button) {
				w.SetText("Reread help").SetIcon(icons.Refresh).SetTooltip(html.EscapeString("Read the help of the command again, as for a new version (from " + sc.Source + ")"))
				w.OnClick(func(e events.Event) {
					go func() {
						nsc, err := cs.SubCmd(sc.Cmd, true)
						ctx.AsWidget().AsyncLock()
						defer ctx.AsWidget().AsyncUnlock()
						if err != nil {
							core.ErrorSnackbar(ctx, err)
							return
						}
						d.Close()
						cs.runDialog(ctx, nsc, run)
					}()
				})
			})
		})
	})
	d.AddBottomBar(func(bar *core.Frame) {
		d.AddCancel(bar)
		core.NewButton(bar).SetType(core.ButtonOutlined).SetText("New macro").SetIcon(icons.Add).
			SetTooltip("Make a macro of the command with these values, at the top of the menu of the command, which can prompt for some of them").
			OnClick(func(e events.Event) {
				root, _ := splitCmd(sc.Cmd)
				mc := &Macro{SubCmd: sc.Name, Values: sc.ArgsOf(values)}
				d.Close()
				runDialog(cs.NewMacroDialog(ctx, root, sc, mc, nil, run), ctx)
			})
		d.AddOK(bar).SetText("Run").SetIcon(icons.PlayArrow).OnClick(func(e events.Event) {
			if cs.Args != nil {
				errors.Log(cs.Args.Set(sc, values))
			}
			run(sc.CommandLine(values))
		})
	})
	return d
}

// argsField is the name of the field of the arguments in the
// [SubCmd.FormStruct].
const argsField = "Arguments"

// fieldNames returns the names of the fields of the forms of the
// subcommand (see [SubCmd.FormStruct]): [argsField] for the other
// arguments, and then one for each flag, which are unique.
func (cm *SubCmd) fieldNames() []string {
	names := []string{argsField}
	used := map[string]bool{argsField: true, macroNameField: true, macroConfirmField: true}
	for _, fl := range cm.Flags {
		name := fieldName(fl.Name)
		for i := 2; used[name]; i++ {
			name = fieldName(fl.Name) + strconv.Itoa(i)
		}
		used[name] = true
		names = append(names, name)
	}
	return names
}

// argKey returns the key of the argument of the given index in the
// [SubCmd.fieldNames]: [argsField] for the other arguments, and the name
// of the flag for the others, as in [SubCmdArgs.Flags] and
// [Macro.Prompts].
func (cm *SubCmd) argKey(i int) string {
	if i == 0 {
		return argsField
	}
	return cm.Flags[i-1].Name
}

// argField returns the struct field of the form of the argument of the
// given index in the [SubCmd.fieldNames], with the given type, and with
// the given extra tags.
func (cm *SubCmd) argField(i int, name string, typ reflect.Type, tags string) reflect.StructField {
	if i == 0 {
		return reflect.StructField{Name: name, Type: typ,
			Tag: reflect.StructTag(`doc:"The other arguments of the command, such as files and paths, as typed in a shell"` + tags)}
	}
	fl := cm.Flags[i-1]
	// (the label and doc are shown as markup)
	tag := `label:` + strconv.Quote(html.EscapeString(fl.Label())) + ` doc:` + strconv.Quote(html.EscapeString(fl.Doc)) + tags
	return reflect.StructField{Name: name, Type: typ, Tag: reflect.StructTag(tag)}
}

// argType returns the type of the value of the argument of the given index
// in the [SubCmd.fieldNames]: bool for a flag without a value, and
// otherwise string.
func (cm *SubCmd) argType(i int) reflect.Type {
	if i > 0 && cm.Flags[i-1].IsBool() {
		return reflect.TypeFor[bool]()
	}
	return reflect.TypeFor[string]()
}

// FormStruct returns a pointer to a new struct for a form of the flags of
// the command: a field for the other arguments, and a bool or string field
// for each flag, labeled with its names, with its documentation.
func (cm *SubCmd) FormStruct() any {
	return cm.formStruct(nil)
}

// formStruct returns a pointer to a new struct for a form of the flags of
// the command (see [SubCmd.FormStruct]), with only the arguments whose keys
// (see [SubCmd.argKey]) are in the given set, if it is not nil.
func (cm *SubCmd) formStruct(keys map[string]bool) any {
	var fields []reflect.StructField
	for i, name := range cm.fieldNames() {
		if keys != nil && !keys[cm.argKey(i)] {
			continue
		}
		fields = append(fields, cm.argField(i, name, cm.argType(i), ` immediate:"+"`))
	}
	return reflect.New(reflect.StructOf(fields)).Interface()
}

// fieldName returns an exported Go field name for the given flag name.
func fieldName(flag string) string {
	name := strcase.ToCamel(strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		return '-'
	}, strings.TrimLeft(flag, "-")))
	if name == "" || !unicode.IsUpper(rune(name[0])) {
		name = "F" + name
	}
	return name
}

// valueField returns the field of the value of the argument of the given
// field name in the given values of a form of a subcommand, which is
// invalid if there is none: the field itself, or the value field of the
// field of a [SubCmd.MacroStruct].
func valueField(values any, name string) reflect.Value {
	f := reflect.ValueOf(values).Elem().FieldByName(name)
	if f.Kind() == reflect.Struct {
		f = f.FieldByName(macroValueField)
	}
	return f
}

// ArgsOf returns the arguments of the given values of a form of the
// subcommand (see [SubCmd.FormStruct] and [SubCmd.MacroStruct]): only those
// that are used.
func (cm *SubCmd) ArgsOf(values any) *SubCmdArgs {
	sa := &SubCmdArgs{}
	for i, name := range cm.fieldNames() {
		f := valueField(values, name)
		val := ""
		switch f.Kind() {
		case reflect.Bool:
			if f.Bool() {
				val = "true"
			}
		case reflect.String:
			val = f.String()
		}
		switch {
		case i == 0:
			sa.Args = val
		case val != "":
			if sa.Flags == nil {
				sa.Flags = map[string]string{}
			}
			sa.Flags[cm.argKey(i)] = val
		}
	}
	return sa
}

// SetValues sets the given values of a form of the given subcommand (see
// [SubCmd.FormStruct] and [SubCmd.MacroStruct]) to the arguments, for
// those it has.
func (sa *SubCmdArgs) SetValues(cm *SubCmd, values any) {
	if sa == nil {
		return
	}
	for i, name := range cm.fieldNames() {
		f := valueField(values, name)
		if !f.IsValid() {
			continue
		}
		val := sa.Args
		if i > 0 {
			var ok bool
			if val, ok = sa.Flags[cm.argKey(i)]; !ok {
				continue
			}
		}
		switch f.Kind() {
		case reflect.Bool:
			b, _ := strconv.ParseBool(val)
			f.SetBool(b)
		case reflect.String:
			f.SetString(val)
		}
	}
}

// CommandLine returns the command line for the given values of a form of
// the command made by [SubCmd.FormStruct]: the command, the flags that are
// on or have values, and the other arguments.
func (cm *SubCmd) CommandLine(values any) string {
	return cm.ArgsCommandLine(cm.ArgsOf(values))
}

// ArgsCommandLine returns the command line for the given arguments: the
// command, the flags that are on or have values, and the other arguments.
func (cm *SubCmd) ArgsCommandLine(sa *SubCmdArgs) string {
	words := []string{cm.Cmd}
	for _, fl := range cm.Flags {
		val, ok := sa.Flags[fl.Name]
		switch {
		case !ok || val == "":
		case fl.IsBool():
			if b, _ := strconv.ParseBool(val); b {
				words = append(words, fl.Arg(""))
			}
		default:
			words = append(words, fl.Arg(val))
		}
	}
	if args := strings.TrimSpace(sa.Args); args != "" {
		words = append(words, args)
	}
	return strings.Join(words, " ")
}

// DialogMinSize is the minimum initial size of the dialogs of the commands,
// in dp: wide enough for the layout of forms on a desktop, with one line
// per field, rather than that of a phone.
var DialogMinSize = math32.Vec2(720, 0)

// runDialog runs the given dialog in a new window, in the context of the
// given widget, with the [DialogMinSize].
func runDialog(d *core.Body, ctx core.Widget) {
	d.NewWindowDialog(ctx).SetMinSize(DialogMinSize).Run()
}

// context returns the context widget to use for the dialogs and snackbars
// of the commands, which lasts beyond the given one: the scene of the
// window of an item of a menu (which closes when it is clicked), or else
// the given widget, or a command bar if it is gone.
func (cs *Commands) context(ctx core.Widget) core.Widget {
	if live(ctx) {
		if st := ctx.AsWidget().Scene.Stage; st.Type == core.MenuStage && st.Main != nil && st.Main.Scene != nil {
			return st.Main.Scene
		}
		return ctx
	}
	cs.views.Lock()
	defer cs.views.Unlock()
	for _, tb := range cs.views.bars {
		if live(tb) {
			return tb
		}
	}
	return ctx
}

// live returns whether the given widget is shown in a stage that is
// running: not deleted, as an item of a closed menu is.
func live(w core.Widget) bool {
	if w == nil {
		return false
	}
	wb := w.AsWidget()
	return wb != nil && wb.This != nil && wb.Scene != nil && wb.Scene.Stage != nil && wb.Scene.Stage.Mains != nil
}

// newForm adds a form of the given values to the given dialog, whose
// fields are wide (see [styleFormFields]), with the first one in focus
// (see [focusForm]).
func newForm(d *core.Body, values any) *core.Form {
	fm := core.NewForm(d)
	styleFormFields(fm)
	fm.SetStruct(values)
	focusForm(d, fm)
	return fm
}

// FieldMinWidth is the minimum width of the text fields of the forms of
// the commands, in ch; they grow with the window (as text fields in forms
// have no maximum width).
var FieldMinWidth float32 = 60

// styleFormFields makes the text fields of the given form, and of its
// inline forms (as of the arguments of a macro), at least [FieldMinWidth]
// wide.
func styleFormFields(fm *core.Form) {
	fm.SetOnChildAdded(func(n tree.Node) {
		switch w := n.(type) {
		case *core.TextField:
			w.Styler(func(s *styles.Style) {
				s.Min.X.Ch(FieldMinWidth)
			})
		case *core.Form:
			styleFormFields(w)
		}
	})
}

// focusForm makes the first field of the given form of the given dialog
// the one in focus when it is shown, rather than the first button of its
// top bar: such as the first argument prompted for by a macro.
func focusForm(d *core.Body, fm *core.Form) {
	fm.Update() // (makes the fields)
	var first core.Widget
	fm.WidgetWalkDown(func(cw core.Widget, cwb *core.WidgetBase) bool {
		if first != nil { // (Break only skips the children of a widget)
			return tree.Break
		}
		switch cw.(type) {
		case *core.TextField, *core.Switch, *core.Spinner, *core.Chooser:
			first = cw
			return tree.Break
		}
		return tree.Continue
	})
	if first != nil {
		d.Scene.Events.SetStartFocus(first)
	}
}
