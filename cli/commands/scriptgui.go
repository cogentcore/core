// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package commands

import (
	"errors"
	"html"
	"image"
	"reflect"
	"slices"
	"strconv"
	"strings"

	errs "cogentcore.org/core/base/errors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/text/textcore"
	"cogentcore.org/core/tree"
)

// NewScriptCode is the code of a new script, before it is written.
const NewScriptCode = "#!/bin/sh\n\n"

// fieldNames returns the names of the fields of the form of the arguments
// of the script (see [Script.FormStruct]), which are unique.
func (s *Script) fieldNames() []string {
	var names []string
	used := map[string]bool{}
	for _, a := range s.Args {
		name := fieldName(a.Name)
		for i := 2; used[name]; i++ {
			name = fieldName(a.Name) + strconv.Itoa(i)
		}
		used[name] = true
		names = append(names, name)
	}
	return names
}

// FormStruct returns a pointer to a new struct for the form of the
// arguments of the script: a string, bool (switch), or [core.Filename]
// field for each, labeled with its name, with its documentation.
func (s *Script) FormStruct() any {
	var fields []reflect.StructField
	for i, name := range s.fieldNames() {
		a := s.Args[i]
		typ := reflect.TypeFor[string]()
		tags := ` immediate:"+"`
		switch a.Kind {
		case ArgSwitch:
			typ, tags = reflect.TypeFor[bool](), ""
		case ArgFile:
			typ = reflect.TypeFor[core.Filename]()
		}
		// (the label and doc are shown as markup)
		tag := `label:` + strconv.Quote(html.EscapeString(cmpOr(a.Name, name))) + ` doc:` + strconv.Quote(html.EscapeString(a.Doc)) + tags
		fields = append(fields, reflect.StructField{Name: name, Type: typ, Tag: reflect.StructTag(tag)})
	}
	return reflect.New(reflect.StructOf(fields)).Interface()
}

// SetFormValues sets the given values of the form of the arguments of the
// script (see [Script.FormStruct]) to the given values, by name.
func (s *Script) SetFormValues(form any, values map[string]string) {
	v := reflect.ValueOf(form).Elem()
	for i, name := range s.fieldNames() {
		f := v.FieldByName(name)
		val := values[s.Args[i].Name]
		if f.Kind() == reflect.Bool {
			b, _ := strconv.ParseBool(val)
			f.SetBool(b)
			continue
		}
		f.SetString(val)
	}
}

// FormValues returns the values of the arguments of the given values of
// the form of the arguments of the script (see [Script.FormStruct]), by
// name: "true" or "" for a switch.
func (s *Script) FormValues(form any) map[string]string {
	v := reflect.ValueOf(form).Elem()
	values := map[string]string{}
	for i, name := range s.fieldNames() {
		f := v.FieldByName(name)
		val := ""
		if f.Kind() == reflect.Bool {
			if f.Bool() {
				val = "true"
			}
		} else {
			val = f.String()
		}
		values[s.Args[i].Name] = val
	}
	return values
}

// Script returns the script of the given command, such as git, with the
// given name, or that of an entry, if the command is "", or nil if there
// is none.
func (cs *Commands) Script(root, name string) *Script {
	cs.reload()
	cs.mu.Lock()
	defer cs.mu.Unlock()
	if root != "" {
		if cm := cs.Parsed[root]; cm != nil {
			if i := slices.IndexFunc(cm.Scripts, func(s *Script) bool { return s.Name == name }); i >= 0 {
				return cm.Scripts[i]
			}
		}
		return nil
	}
	if en := findScriptEntry(cs.Entries, nil, name); en != nil {
		return en.Script
	}
	return nil
}

// findScriptEntry returns the entry of the given script among the given
// entries and their children, or else that of a script of the given name,
// or nil if there is none.
func findScriptEntry(ens []*Entry, s *Script, name string) *Entry {
	var byName *Entry
	var find func(ens []*Entry) *Entry
	find = func(ens []*Entry) *Entry {
		for _, en := range ens {
			if en.Script != nil {
				if s != nil && en.Script == s {
					return en
				}
				if byName == nil && en.Script.Name == name {
					byName = en
				}
			}
			if f := find(en.Children); f != nil {
				return f
			}
		}
		return nil
	}
	if en := find(ens); en != nil {
		return en
	}
	return byName
}

// deleteEntry returns the given entries without the given one, which is
// also deleted from their children.
func deleteEntry(ens []*Entry, del *Entry) []*Entry {
	ens = slices.DeleteFunc(ens, func(en *Entry) bool { return en == del })
	for _, en := range ens {
		en.Children = deleteEntry(en.Children, del)
	}
	return ens
}

// SaveScript saves the given script of the given command, such as git, or
// as an entry, if the command is "", in place of the given old one, if it
// is not nil, and otherwise as a new one (in place of one of the same name
// of the command), updating the command bars and the editor. A new script
// of an entry is the last entry.
func (cs *Commands) SaveScript(root string, s, old *Script) error {
	if strings.TrimSpace(s.Name) == "" {
		return errors.New("the script has no name")
	}
	err := cs.update(func() {
		if root == "" {
			if old != nil {
				if en := findScriptEntry(cs.Entries, old, old.Name); en != nil {
					en.Script = s
					en.Name = s.Name
					return
				}
			}
			cs.Entries = append(cs.Entries, &Entry{Name: s.Name, Script: s})
			return
		}
		cm := cs.Parsed[root]
		if cm == nil {
			cm = &Cmd{Cmd: root}
			cs.Parsed[root] = cm
		}
		i := -1
		if old != nil {
			i = slices.IndexFunc(cm.Scripts, func(o *Script) bool { return o == old || o.Name == old.Name })
		}
		if i < 0 {
			i = slices.IndexFunc(cm.Scripts, func(o *Script) bool { return o.Name == s.Name })
		}
		if i >= 0 {
			cm.Scripts[i] = s
		} else {
			cm.Scripts = append(cm.Scripts, s)
		}
	})
	cs.updateViews(nil)
	return err
}

// DeleteScript deletes the given script of the given command, such as git,
// or the entry of it, if the command is "", updating the command bars and
// the editor.
func (cs *Commands) DeleteScript(root string, s *Script) error {
	err := cs.update(func() {
		if root == "" {
			if en := findScriptEntry(cs.Entries, s, s.Name); en != nil {
				cs.Entries = deleteEntry(cs.Entries, en)
			}
			return
		}
		if cm := cs.Parsed[root]; cm != nil {
			cm.Scripts = slices.DeleteFunc(cm.Scripts, func(o *Script) bool { return o == s || o.Name == s.Name })
		}
	})
	cs.updateViews(nil)
	return err
}

// RunScript runs the given script of the given command, such as git, or
// of an entry, if the command is "", with the given function, after
// prompting for its arguments, if it has any, in a dialog in the context
// of the given widget, or else after confirming it, if it is to be
// confirmed (see [Script.Confirm]). The values of its arguments are saved
// in the [Args], as their defaults for the next time.
func (cs *Commands) RunScript(ctx core.Widget, root string, s *Script, run RunFunc) {
	ctx = cs.context(ctx)
	if len(s.Args) > 0 {
		runDialog(cs.NewScriptPromptDialog(ctx, root, s, run), ctx)
		return
	}
	if !s.Confirm {
		cs.runScript(ctx, root, s, nil, run)
		return
	}
	d := core.NewBody(s.Name)
	core.NewText(d).SetType(core.TextSupporting).SetText("Run this script?")
	if s.Doc != "" {
		core.NewText(d).SetType(core.TextSupporting).SetText(html.EscapeString(s.Doc))
	}
	cs.scriptTopBar(d, ctx, root, s)
	d.AddBottomBar(func(bar *core.Frame) {
		d.AddCancel(bar)
		d.AddOK(bar).SetText("Run").SetIcon(icons.PlayArrow).OnClick(func(e events.Event) {
			cs.runScript(ctx, root, s, nil, run)
		})
	})
	d.RunDialog(ctx)
}

// runScript runs the given script of the given command (see
// [Commands.RunScript]) with the given values of its arguments, by name,
// which are saved as their defaults for the next time.
func (cs *Commands) runScript(ctx core.Widget, root string, s *Script, values map[string]string, run RunFunc) {
	if len(s.Args) > 0 {
		errs.Log(cs.Args.SetScriptValues(root, s, values))
	}
	cmdline, err := cs.ScriptCommandLine(root, s, values)
	if err != nil {
		core.ErrorSnackbar(ctx, err, "Error writing the script "+s.Name)
		return
	}
	run(cmdline)
}

// scriptTopBar adds the top bar of the dialogs of running the given script
// of the given command to the given dialog, with a button to edit it.
func (cs *Commands) scriptTopBar(d *core.Body, ctx core.Widget, root string, s *Script) {
	d.AddTopBar(func(bar *core.Frame) {
		core.NewToolbar(bar).Maker(func(p *tree.Plan) {
			tree.Add(p, func(w *core.Button) {
				w.SetText("Edit script").SetIcon(icons.Edit)
				w.OnClick(func(e events.Event) {
					d.Close()
					cs.EditScript(ctx, root, s)
				})
			})
		})
	})
}

// NewScriptPromptDialog returns the body of the dialog that prompts for
// the arguments of the given script of the given command, such as git, or
// of an entry, if the command is "", which runs it with the given function
// (see [Commands.RunScript]).
func (cs *Commands) NewScriptPromptDialog(ctx core.Widget, root string, s *Script, run RunFunc) *core.Body {
	ctx = cs.context(ctx)
	d := core.NewBody(s.Name)
	if s.Doc != "" {
		core.NewText(d).SetType(core.TextSupporting).SetText(html.EscapeString(s.Doc))
	}
	form := s.FormStruct()
	s.SetFormValues(form, cs.Args.ScriptValues(root, s))
	line := commandLineText(d, func() string { return s.CommandLine(s.FormValues(form)) })
	fm := cs.chooserForm(ctx, d, form, cs.scriptChoosers(s))
	fm.OnChange(func(e events.Event) { line.Update() })
	fm.OnInput(func(e events.Event) { line.Update() })
	cs.scriptTopBar(d, ctx, root, s)
	d.AddBottomBar(func(bar *core.Frame) {
		d.AddCancel(bar)
		d.AddOK(bar).SetText("Run").SetIcon(icons.PlayArrow).OnClick(func(e events.Event) {
			cs.runScript(ctx, root, s, s.FormValues(form), run)
		})
	})
	return d
}

// EditScript opens the editor of the given script of the given command,
// such as git, or of an entry, if the command is "" (see
// [Commands.NewScriptDialog]).
func (cs *Commands) EditScript(ctx core.Widget, root string, s *Script) {
	ctx = cs.context(ctx)
	runDialog(cs.NewScriptDialog(ctx, root, s, s), ctx)
}

// NewScript opens the editor of a new script of the given command, such as
// git, or of an entry, if the command is "", with the given code, or
// [NewScriptCode] if it is empty.
func (cs *Commands) NewScript(ctx core.Widget, root, code string) {
	ctx = cs.context(ctx)
	s := &Script{Code: cmpOr(code, NewScriptCode)}
	runDialog(cs.NewScriptDialog(ctx, root, s, nil), ctx)
}

// NewScriptDialog returns the body of the editor of the given script of
// the given command, such as git, or of an entry, if the command is "":
// its name, documentation, and whether it is confirmed, its code, and the
// table of its arguments. It saves it in place of the given old script,
// if it is not nil, and otherwise as a new one (see [Commands.SaveScript]).
func (cs *Commands) NewScriptDialog(ctx core.Widget, root string, s, old *Script) *core.Body {
	ctx = cs.context(ctx)
	title := "New script"
	if old != nil {
		title = "Edit script"
	}
	if root != "" {
		title += ": " + root
	}
	d := core.NewBody(title)
	es := s.clone()
	core.NewText(d).SetType(core.TextSupporting).SetText("The script starts with the #! line of the program that runs it, such as #!/bin/bash, #!/usr/bin/env python3, or #!/usr/bin/env goal. Its arguments are prompted for when it runs, and passed to it as $1, $2, and so on.")
	newForm(d, es)
	core.NewText(d).SetType(core.TextTitleSmall).SetText("Code")
	ed := textcore.NewEditor(d)
	ed.Lines.SetText([]byte(es.Code))
	ed.Lines.SetLanguage(es.Language())
	ed.Styler(func(s *styles.Style) {
		s.Min.Y.Em(12)
		s.Grow.Set(1, 1)
	})
	core.NewText(d).SetType(core.TextTitleSmall).SetText("Arguments")
	core.NewText(d).SetType(core.TextSupporting).SetText("The arguments, $1, $2, and so on, in order, which are prompted for when the script runs: text, a switch (passed as its On value, or empty if it is off), or a file. Text can have a chooser of its values, such as a git branch. Add one with Add argument at the top, or the context menu of the table (right-click).")
	tb := core.NewTable(d).SetSlice(&es.Args)
	d.AddTopBar(func(bar *core.Frame) {
		core.NewToolbar(bar).Maker(func(p *tree.Plan) {
			tree.Add(p, func(w *core.Button) {
				w.SetText("Add argument").SetIcon(icons.Add).SetTooltip("Add an argument at the end")
				w.OnClick(func(e events.Event) {
					es.Args = append(es.Args, ScriptArg{Name: "Arg" + strconv.Itoa(len(es.Args)+1)})
					tb.Update()
				})
			})
			if old != nil {
				tree.Add(p, func(w *core.Button) {
					w.SetText("Delete script").SetIcon(icons.Delete)
					w.OnClick(func(e events.Event) {
						errs.Log(cs.DeleteScript(root, old))
						d.Close()
					})
				})
			}
		})
	})
	d.AddBottomBar(func(bar *core.Frame) {
		d.AddCancel(bar)
		d.AddOK(bar).SetText("Save").SetIcon(icons.Save).OnClick(func(e events.Event) {
			es.Name = strings.TrimSpace(es.Name)
			es.Code = ed.Lines.String()
			if err := cs.SaveScript(root, es, old); err != nil {
				core.ErrorSnackbar(ctx, err, "Error saving the script")
				return
			}
			core.MessageSnackbar(ctx, "Saved the script "+es.Name)
		})
	})
	return d
}

// configScript configures the given button, of a menu or the command
// bar, for the given script.
func configScript(w *core.Button, s *Script) {
	w.SetText(s.Name).SetIcon(icons.Code)
	w.SetTooltip(html.EscapeString(cmpOr(s.Doc, "Run the script")))
}

// scriptContextMenu adds a context menu to the given button, of a menu or
// the command bar, to edit the script given by the given function, of the
// given command, or of an entry, if the command is "", if it is not nil.
func (cs *Commands) scriptContextMenu(w *core.Button, root string, script func() *Script) {
	w.AddContextMenu(func(m *core.Scene, _ image.Point) {
		s := script()
		if s == nil {
			return
		}
		core.NewButton(m).SetText("Edit script").SetIcon(icons.Edit).OnClick(func(e events.Event) {
			cs.EditScript(w, root, s)
		})
	})
}

// scriptsMenu adds the buttons of the scripts of the given command, such
// as git, to the given menu, which run them with the given function,
// followed by a separator, if there are any.
func (cs *Commands) scriptsMenu(m *core.Scene, cm *Cmd, run RunFunc) {
	for _, s := range cm.Scripts {
		b := core.NewButton(m)
		configScript(b, s)
		cs.scriptContextMenu(b, cm.Cmd, func() *Script { return s })
		b.OnClick(func(e events.Event) { cs.RunScript(b, cm.Cmd, s, run) })
	}
	if len(cm.Scripts) > 0 {
		core.NewSeparator(m)
	}
}

// editScriptsMenu adds a button to the given menu of the given command,
// such as git, with a submenu to make a new script of it, and to edit its
// scripts.
func (cs *Commands) editScriptsMenu(m *core.Scene, cm *Cmd) {
	b := core.NewButton(m).SetText("Scripts").SetIcon(icons.Code)
	b.SetTooltip(html.EscapeString("Make and edit the scripts of " + cm.Cmd))
	b.SetMenu(func(sm *core.Scene, _ image.Point) {
		core.NewButton(sm).SetText("New script").SetIcon(icons.Add).
			SetTooltip(html.EscapeString("Make a script of " + cm.Cmd + ", which is in its menu")).
			OnClick(func(e events.Event) { cs.NewScript(b, cm.Cmd, "") })
		for _, s := range cm.Scripts {
			core.NewButton(sm).SetText("Edit " + s.Name).SetIcon(icons.Edit).OnClick(func(e events.Event) {
				cs.EditScript(b, cm.Cmd, s)
			})
		}
	})
}
