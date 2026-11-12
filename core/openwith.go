// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"image"
	"path/filepath"

	"cogentcore.org/core/base/fileinfo/apps"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
)

// OpenWithMenuItems adds the context menu items for opening the given file,
// which is not a directory, with the apps of the system, as in the
// [FilePicker] and the file tree: Open with, a menu of the apps that can
// open it, with the default one first, and Other..., to choose from all of
// the apps, and Change default app. It adds nothing on platforms that do
// not support that (see [apps.For]).
func OpenWithMenuItems(m *Scene, file string) {
	as, err := apps.For(file)
	if err != nil {
		return // not supported
	}
	NewButton(m).SetText("Open with").SetIcon(icons.Open).
		SetMenu(func(m *Scene, pos image.Point) {
			for _, app := range as {
				text := app.Name
				if app.Default {
					text += " (default)"
				}
				NewButton(m).SetText(text).OnClick(func(e events.Event) {
					ErrorSnackbar(m, apps.OpenWith(file, app))
				})
			}
			if len(as) > 0 {
				NewSeparator(m)
			}
			NewButton(m).SetText(otherApps).
				SetTooltip("Open the file with one of all of the apps").
				OnClick(func(e events.Event) {
					openWithOther(m, file)
				})
		}).SetTooltip("Open the file with one of the apps that can open it")
	NewButton(m).SetText("Change default app").SetIcon(icons.Edit).
		SetTooltip("Change the default app for this type of file").
		OnClick(func(e events.Event) {
			changeDefaultApp(m, file, as)
		})
}

// otherApps is the text of the choices of all of the apps.
const otherApps = "Other..."

// newAppChooser makes a chooser of the given apps, with an Other... item
// at the end, which replaces them with all of the apps, or of all of the
// apps if there are none. It can be typed in to find an app.
func newAppChooser(parent Widget, as []apps.App) *Chooser {
	ch := NewChooser(parent).SetEditable(true)
	setApps := func(as []apps.App, other bool) {
		ch.Items = ch.Items[:0]
		for _, app := range as {
			ch.Items = append(ch.Items, ChooserItem{Value: app, Text: app.Name})
		}
		if other {
			ch.Items = append(ch.Items, ChooserItem{Value: otherApps, Text: otherApps,
				Tooltip: "Choose from all of the apps"})
		}
		if len(ch.Items) > 0 {
			ch.SetCurrentIndex(0) // the default, if any, is first
		}
	}
	all := func() {
		all, err := apps.All()
		ErrorSnackbar(ch, err)
		setApps(all, false)
	}
	if len(as) == 0 {
		all()
	} else {
		setApps(as, true)
	}
	ch.OnChange(func(e events.Event) {
		if ch.CurrentItem.Value == otherApps {
			all()
			ch.Update()
		}
	})
	return ch
}

// chosenApp returns the app chosen in the given chooser of [newAppChooser].
func chosenApp(ch *Chooser) (apps.App, bool) {
	app, ok := ch.CurrentItem.Value.(apps.App)
	return app, ok
}

// openWithOther shows a dialog to choose one of all of the apps
// to open the given file with.
func openWithOther(ctx Widget, file string) {
	d := NewBody("Open with")
	NewText(d).SetType(TextSupporting).
		SetText("The app to open " + filepath.Base(file) + " with: it may not open this type of file")
	ch := newAppChooser(d, nil)
	d.AddBottomBar(func(bar *Frame) {
		d.AddCancel(bar)
		d.AddOK(bar).SetText("Open").OnClick(func(e events.Event) {
			if app, ok := chosenApp(ch); ok {
				ErrorSnackbar(ctx, apps.OpenWith(file, app))
			}
		})
	})
	d.RunDialog(ctx)
}

// changeDefaultApp shows a dialog to choose the default app for the type of
// the given file from the given apps that can open it, or from all of them.
func changeDefaultApp(ctx Widget, file string, as []apps.App) {
	d := NewBody("Default app")
	typ := filepath.Ext(file)
	if typ == "" {
		typ = filepath.Base(file)
	}
	NewText(d).SetType(TextSupporting).
		SetText("The default app for files like " + typ + ", which opens them in all apps")
	ch := newAppChooser(d, as)
	d.AddBottomBar(func(bar *Frame) {
		d.AddCancel(bar)
		d.AddOK(bar).SetText("Set default").OnClick(func(e events.Event) {
			if app, ok := chosenApp(ch); ok {
				ErrorSnackbar(ctx, apps.SetDefault(file, app))
			}
		})
	})
	d.RunDialog(ctx)
}
