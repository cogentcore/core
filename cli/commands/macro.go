// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package commands

import (
	"html"
	"image"
	"reflect"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/tree"
)

// Macro is a version of a subcommand with values for its arguments, some
// of which are prompted for when it is run, such as git commit with
// flags that are always used, prompting for the message. The macros of a
// command are at the top of the menu of its subcommands (see [Cmd.Macros]).
type Macro struct {

	// Name is the name of the macro, which is after the command and the
	// subcommand in its label: "Git commit wip" (see [Macro.Label]). It can
	// be empty, for the default version of the subcommand.
	Name string `json:",omitempty"`

	// SubCmd is the name of the subcommand: "commit", or [MainName].
	SubCmd string

	// Values are the values of the arguments, which are used when the
	// macro runs, other than those prompted for, which they are the
	// defaults of.
	Values *SubCmdArgs `json:",omitempty"`

	// Confirm is whether the command line of the macro is shown for
	// confirmation before it runs, even if nothing is prompted for, as for
	// commands that cannot be undone.
	Confirm bool `json:",omitempty"`

	// Prompts are the arguments that are prompted for when the macro runs,
	// by key: the names of flags ("--message"), and [argsField] for the
	// other arguments.
	Prompts []string `json:",omitempty"`
}

// Cmd returns the command line of the subcommand of the macro of the given
// command, such as "git commit" for git.
func (mc *Macro) Cmd(root string) string {
	if mc.SubCmd == MainName || mc.SubCmd == "" {
		return root
	}
	return root + " " + mc.SubCmd
}

// Label returns the text shown for the macro of the given command: its
// command line and its name, starting with an uppercase letter, such as
// "Git commit wip".
func (mc *Macro) Label(root string) string {
	s := strings.TrimSpace(mc.Cmd(root) + " " + mc.Name)
	r, n := utf8.DecodeRuneInString(s)
	return string(unicode.ToUpper(r)) + s[n:]
}

// IsPrompt returns whether the argument of the given key (see [Macro.Prompts])
// is prompted for when the macro runs.
func (mc *Macro) IsPrompt(key string) bool {
	return slices.Contains(mc.Prompts, key)
}

// same returns whether the given macro is the same as this one: of the
// same subcommand, with the same name.
func (mc *Macro) same(o *Macro) bool {
	return o != nil && mc.SubCmd == o.SubCmd && mc.Name == o.Name
}

// promptFlag is whether an argument of a [SubCmd.MacroStruct] is prompted
// for when the macro runs, which is shown as a checkbox.
type promptFlag bool

// Widget returns a checkbox for the flag.
func (pf *promptFlag) Widget() core.Value {
	return core.NewSwitch().SetType(core.SwitchCheckbox)
}

const (
	// macroNameField and macroConfirmField are the names of the fields of
	// the name of the macro, and of whether it is confirmed, in a
	// [SubCmd.MacroStruct].
	macroNameField    = "Name"
	macroConfirmField = "Confirm"

	// macroPromptField and macroValueField are the names of the fields of
	// the struct of each argument in a [SubCmd.MacroStruct].
	macroPromptField = "Prompt"
	macroValueField  = "Value"
)

// MacroStruct returns a pointer to a new struct for the form of a macro of
// the subcommand: a field for the name of the macro, and then a field for
// each argument of the [SubCmd.FormStruct], with a checkbox of whether it
// is prompted for when the macro runs, followed by its value.
func (cm *SubCmd) MacroStruct() any {
	fields := []reflect.StructField{{
		Name: macroNameField, Type: reflect.TypeFor[string](),
		Tag: `immediate:"+" doc:"The name of the macro, after the command in its label, which can be empty for the default version of the command"`,
	}, {
		Name: macroConfirmField, Type: reflect.TypeFor[bool](),
		Tag: `doc:"Whether the command line is shown for confirmation before it runs, even if nothing is prompted for, as for commands that cannot be undone"`,
	}}
	for i, name := range cm.fieldNames() {
		arg := reflect.StructOf([]reflect.StructField{
			{Name: macroPromptField, Type: reflect.TypeFor[promptFlag](),
				Tag: `label:"" doc:"if checked, this will be prompted for when the macro runs"`},
			{Name: macroValueField, Type: cm.argType(i), Tag: `label:"" immediate:"+"`},
		})
		fields = append(fields, cm.argField(i, name, arg, ` display:"inline"`))
	}
	return reflect.New(reflect.StructOf(fields)).Interface()
}

// MacroOf returns the macro of the given values of the form of a macro of
// the subcommand (see [SubCmd.MacroStruct]).
func (cm *SubCmd) MacroOf(values any) *Macro {
	v := reflect.ValueOf(values).Elem()
	mc := &Macro{SubCmd: cm.Name, Values: cm.ArgsOf(values)}
	mc.Name = strings.TrimSpace(v.FieldByName(macroNameField).String())
	mc.Confirm = v.FieldByName(macroConfirmField).Bool()
	for i, name := range cm.fieldNames() {
		if v.FieldByName(name).FieldByName(macroPromptField).Bool() {
			mc.Prompts = append(mc.Prompts, cm.argKey(i))
		}
	}
	return mc
}

// SetMacroValues sets the given values of the form of a macro of the
// subcommand (see [SubCmd.MacroStruct]) to those of the given macro.
func (cm *SubCmd) SetMacroValues(mc *Macro, values any) {
	v := reflect.ValueOf(values).Elem()
	v.FieldByName(macroNameField).SetString(mc.Name)
	v.FieldByName(macroConfirmField).SetBool(mc.Confirm)
	mc.Values.SetValues(cm, values)
	for i, name := range cm.fieldNames() {
		v.FieldByName(name).FieldByName(macroPromptField).SetBool(mc.IsPrompt(cm.argKey(i)))
	}
}

// promptKeys returns the set of the keys of the arguments that are
// prompted for when the macro runs.
func (mc *Macro) promptKeys() map[string]bool {
	keys := map[string]bool{}
	for _, k := range mc.Prompts {
		keys[k] = true
	}
	return keys
}

// withPrompted returns the values of the macro, with those of the given
// keys from the given values of the form of its prompts.
func (mc *Macro) withPrompted(cm *SubCmd, prompted *SubCmdArgs, keys map[string]bool) *SubCmdArgs {
	sa := &SubCmdArgs{Flags: map[string]string{}}
	if mc.Values != nil {
		sa.Args = mc.Values.Args
		for k, v := range mc.Values.Flags {
			sa.Flags[k] = v
		}
	}
	for k := range keys {
		if k == argsField {
			sa.Args = prompted.Args
			continue
		}
		delete(sa.Flags, k)
		if v, ok := prompted.Flags[k]; ok {
			sa.Flags[k] = v
		}
	}
	return sa
}

// SaveMacro saves the given macro of the given command, such as git, in
// place of the given old one, if it is not nil, and otherwise as a new one
// (in place of one of the same subcommand and name), updating the command
// bars.
func (cs *Commands) SaveMacro(root string, mc, old *Macro) error {
	err := cs.update(func() {
		cm := cs.Parsed[root]
		if cm == nil {
			cm = &Cmd{Cmd: root}
			cs.Parsed[root] = cm
		}
		i := slices.IndexFunc(cm.Macros, func(o *Macro) bool { return o.same(old) })
		if i < 0 {
			i = slices.IndexFunc(cm.Macros, func(o *Macro) bool { return o.same(mc) })
		}
		if i >= 0 {
			cm.Macros[i] = mc
		} else {
			cm.Macros = append(cm.Macros, mc)
		}
	})
	cs.updateViews(nil)
	return err
}

// DeleteMacro deletes the given macro of the given command, such as git,
// updating the command bars.
func (cs *Commands) DeleteMacro(root string, mc *Macro) error {
	err := cs.update(func() {
		if cm := cs.Parsed[root]; cm != nil {
			cm.Macros = slices.DeleteFunc(cm.Macros, func(o *Macro) bool { return o.same(mc) })
		}
	})
	cs.updateViews(nil)
	return err
}

// macroSubCmd returns the parsed subcommand of the given macro of the
// given command, parsing it in the background if needed, and calling the
// given function with it, with the lock of the given context widget.
func (cs *Commands) macroSubCmd(ctx core.Widget, root string, mc *Macro, fun func(sc *SubCmd)) {
	ctx = cs.context(ctx)
	cmd := mc.Cmd(root)
	if sc := cs.cachedSub(cmd); sc != nil {
		fun(sc)
		return
	}
	core.MessageSnackbar(ctx, "Reading the help of "+cmd+"...")
	go func() {
		sc, err := cs.SubCmd(cmd, false)
		ctx.AsWidget().AsyncLock()
		defer ctx.AsWidget().AsyncUnlock()
		if err != nil {
			core.ErrorSnackbar(ctx, err)
			return
		}
		fun(sc)
	}()
}

// RunMacro runs the given macro of the given command, such as git, with
// the given function, after prompting for the arguments it prompts for,
// if any, in a dialog in the context of the given widget, or else after
// confirming it, if it is to be confirmed (see [Macro.Confirm]).
func (cs *Commands) RunMacro(ctx core.Widget, root string, mc *Macro, run RunFunc) {
	ctx = cs.context(ctx)
	cs.macroSubCmd(ctx, root, mc, func(sc *SubCmd) {
		if len(mc.Prompts) == 0 {
			cmdline := sc.ArgsCommandLine(mc.withPrompted(sc, nil, nil))
			if !mc.Confirm {
				run(cmdline)
				return
			}
			d := core.NewBody(mc.Label(root))
			core.NewText(d).SetType(core.TextSupporting).SetText("Run this command?")
			commandLineText(d, func() string { return cmdline })
			d.AddBottomBar(func(bar *core.Frame) {
				d.AddCancel(bar)
				d.AddOK(bar).SetText("Run").SetIcon(icons.PlayArrow).OnClick(func(e events.Event) {
					run(cmdline)
				})
			})
			d.RunDialog(ctx)
			return
		}
		runDialog(cs.NewPromptDialog(ctx, root, sc, mc, run), ctx)
	})
}

// NewPromptDialog returns the body of the dialog that prompts for the
// arguments of the given macro of the given subcommand of the given
// command, which runs its command line with the given function.
func (cs *Commands) NewPromptDialog(ctx core.Widget, root string, sc *SubCmd, mc *Macro, run RunFunc) *core.Body {
	ctx = cs.context(ctx)
	d := core.NewBody(mc.Label(root))
	keys := mc.promptKeys()
	values := sc.formStruct(keys)
	mc.Values.SetValues(sc, values)
	cmdline := func() string {
		return sc.ArgsCommandLine(mc.withPrompted(sc, sc.ArgsOf(values), keys))
	}
	line := commandLineText(d, cmdline)
	fm := newForm(d, values)
	fm.OnChange(func(e events.Event) { line.Update() })
	fm.OnInput(func(e events.Event) { line.Update() })
	d.AddTopBar(func(bar *core.Frame) {
		core.NewToolbar(bar).Maker(func(p *tree.Plan) {
			tree.Add(p, func(w *core.Button) {
				w.SetText("Edit macro").SetIcon(icons.Edit)
				w.OnClick(func(e events.Event) {
					d.Close()
					runDialog(cs.NewMacroDialog(ctx, root, sc, mc, mc, run), ctx)
				})
			})
		})
	})
	d.AddBottomBar(func(bar *core.Frame) {
		d.AddCancel(bar)
		d.AddOK(bar).SetText("Run").SetIcon(icons.PlayArrow).OnClick(func(e events.Event) {
			run(cmdline())
		})
	})
	return d
}

// NewMacroDialog returns the body of the dialog of the form of the given
// macro of the given subcommand of the given command, such as a new macro
// with the values of its form, which saves it in place of the given old
// macro, if it is not nil, and otherwise as a new one. Its macros are run
// with the given function.
func (cs *Commands) NewMacroDialog(ctx core.Widget, root string, sc *SubCmd, mc, old *Macro, run RunFunc) *core.Body {
	ctx = cs.context(ctx)
	title := "New macro"
	if old != nil {
		title = "Edit macro"
	}
	d := core.NewBody(title + ": " + sc.Cmd)
	core.NewText(d).SetType(core.TextSupporting).SetText("The values are used when the macro runs: those that are checked are prompted for, with these values as their defaults.")
	values := sc.MacroStruct()
	sc.SetMacroValues(mc, values)
	line := commandLineText(d, func() string {
		m := sc.MacroOf(values)
		return m.Label(root) + ": " + sc.ArgsCommandLine(m.Values)
	})
	fm := newForm(d, values)
	fm.OnChange(func(e events.Event) { line.Update() })
	fm.OnInput(func(e events.Event) { line.Update() })
	if old != nil {
		d.AddTopBar(func(bar *core.Frame) {
			core.NewToolbar(bar).Maker(func(p *tree.Plan) {
				tree.Add(p, func(w *core.Button) {
					w.SetText("Delete macro").SetIcon(icons.Delete)
					w.OnClick(func(e events.Event) {
						errors.Log(cs.DeleteMacro(root, old))
						d.Close()
					})
				})
			})
		})
	}
	d.AddBottomBar(func(bar *core.Frame) {
		d.AddCancel(bar)
		d.AddOK(bar).SetText("Save").SetIcon(icons.Save).OnClick(func(e events.Event) {
			nm := sc.MacroOf(values)
			if err := cs.SaveMacro(root, nm, old); err != nil {
				core.ErrorSnackbar(ctx, err)
				return
			}
			core.MessageSnackbar(ctx, "Saved the macro "+nm.Label(root))
		})
	})
	return d
}

// commandLineText adds a text of the command line given by the given
// function to the given body, which is updated with it.
func commandLineText(d *core.Body, cmdline func() string) *core.Text {
	line := core.NewText(d)
	line.Styler(func(s *styles.Style) {
		s.Font.Family = rich.Monospace
		s.Grow.Set(1, 0)
	})
	line.Updater(func() { line.SetText(html.EscapeString(cmdline())) })
	return line
}

// macrosMenu adds the buttons of the macros of the given command, such as
// git, to the given menu, which run them with the given function, followed
// by a separator, if there are any.
func (cs *Commands) macrosMenu(m *core.Scene, cm *Cmd, run RunFunc) {
	for _, mc := range cm.Macros {
		b := core.NewButton(m).SetText(mc.Label(cm.Cmd)).SetIcon(icons.PlayArrow)
		tip := "Run the macro"
		if len(mc.Prompts) > 0 {
			tip += ", prompting for " + strings.Join(mc.Prompts, ", ")
		}
		b.SetTooltip(html.EscapeString(tip))
		b.OnClick(func(e events.Event) { cs.RunMacro(b, cm.Cmd, mc, run) })
	}
	if len(cm.Macros) > 0 {
		core.NewSeparator(m)
	}
}

// macrosButton adds a button with a menu of the macros of the given
// subcommand of the given command to the given plan, which opens their
// editors, if there are any.
func (cs *Commands) macrosButton(p *tree.Plan, d *core.Body, ctx core.Widget, root string, sc *SubCmd, run RunFunc) {
	cm := cs.cached(root)
	if cm == nil {
		return
	}
	var mcs []*Macro
	for _, mc := range cm.Macros {
		if mc.SubCmd == sc.Name {
			mcs = append(mcs, mc)
		}
	}
	if len(mcs) == 0 {
		return
	}
	tree.Add(p, func(w *core.Button) {
		w.SetText("Macros").SetIcon(icons.Edit).SetTooltip("Edit the macros of " + sc.Cmd)
		w.SetMenu(func(m *core.Scene, _ image.Point) {
			for _, mc := range mcs {
				core.NewButton(m).SetText(mc.Label(root)).OnClick(func(e events.Event) {
					d.Close()
					runDialog(cs.NewMacroDialog(ctx, root, sc, mc, mc, run), ctx)
				})
			}
		})
	})
}
