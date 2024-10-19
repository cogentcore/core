// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package databrowser

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"

	"cogentcore.org/core/base/fsx"
	"cogentcore.org/core/base/logx"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/goal/goalib"
	"cogentcore.org/core/goal/interpreter"
	"cogentcore.org/core/styles"
	"github.com/cogentcore/yaegi/interp"
)

func (br *Browser) InitInterp() {
	br.Interpreter = interpreter.NewInterpreter(interp.Options{})
	br.Interpreter.Config()
	// logx.UserLevel = slog.LevelDebug // for debugging of init loading
}

func (br *Browser) RunScript(snm string) {
	sc, ok := br.Scripts[snm]
	if !ok {
		slog.Error("script not found:", "Script:", snm)
		return
	}
	logx.PrintlnDebug("\n################\nrunning script:\n", sc, "\n")
	_, _, err := br.Interpreter.Eval(sc)
	if err == nil {
		err = br.Interpreter.Goal.TrState.DepthError()
	}
	br.Interpreter.Goal.TrState.ResetDepth()
}

// UpdateScripts updates the Scripts and updates the toolbar.
func (br *Browser) UpdateScripts() { //types:add
	redo := (br.Scripts != nil)
	scr := fsx.Filenames(br.ScriptsDir, ".goal")
	br.Scripts = make(map[string]string)
	for _, s := range scr {
		snm := strings.TrimSuffix(s, ".goal")
		sc, err := os.ReadFile(filepath.Join(br.ScriptsDir, s))
		if err == nil {
			if unicode.IsLower(rune(snm[0])) {
				if !redo {
					fmt.Println("run init script:", snm)
					br.Interpreter.Eval(string(sc))
				}
			} else {
				ssc := string(sc)
				br.Scripts[snm] = ssc
			}
		} else {
			slog.Error(err.Error())
		}
	}
	if br.toolbar != nil {
		br.toolbar.Update()
	}
}

// TrimOrderPrefix trims any optional #- prefix from given string,
// used for ordering items by name.
func TrimOrderPrefix(s string) string {
	i := strings.Index(s, "-")
	if i < 0 {
		return s
	}
	ds := s[:i]
	if _, err := strconv.Atoi(ds); err != nil {
		return s
	}
	return s[i+1:]
}

// PromptOKCancel prompts the user for whether to do something,
// calling the given function if the user clicks OK.
func PromptOKCancel(ctx core.Widget, prompt string, fun func()) {
	d := core.NewBody(prompt)
	d.AddBottomBar(func(bar *core.Frame) {
		d.AddCancel(bar)
		d.AddOK(bar).OnClick(func(e events.Event) {
			if fun != nil {
				fun()
			}
		})
	})
	d.RunDialog(ctx)
}

// PromptString prompts the user for a string value (initial value given),
// calling the given function if the user clicks OK.
func PromptString(ctx core.Widget, str string, prompt string, fun func(s string)) {
	d := core.NewBody(prompt)
	tf := core.NewTextField(d).SetText(str)
	tf.Styler(func(s *styles.Style) {
		s.Min.X.Ch(60)
	})
	d.AddBottomBar(func(bar *core.Frame) {
		d.AddCancel(bar)
		d.AddOK(bar).OnClick(func(e events.Event) {
			if fun != nil {
				fun(tf.Text())
			}
		})
	})
	d.RunDialog(ctx)
}

// PromptStruct prompts the user for the values in given struct (pass a pointer),
// calling the given function if the user clicks OK.
func PromptStruct(ctx core.Widget, str any, prompt string, fun func()) {
	d := core.NewBody(prompt)
	core.NewForm(d).SetStruct(str)
	d.AddBottomBar(func(bar *core.Frame) {
		d.AddCancel(bar)
		d.AddOK(bar).OnClick(func(e events.Event) {
			if fun != nil {
				fun()
			}
		})
	})
	d.RunDialog(ctx)
}

// FirstComment returns the first comment lines from given .goal file,
// which is used to set the tooltip for scripts.
func FirstComment(sc string) string {
	sl := goalib.SplitLines(sc)
	cmt := ""
	for _, l := range sl {
		if !strings.HasPrefix(l, "// ") {
			return cmt
		}
		cmt += strings.TrimSpace(l[3:]) + " "
	}
	return cmt
}
