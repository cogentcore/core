// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package keymap

// StandardMaps is the original compiled-in set of standard keymaps that have
// the lastest key functions bound to standard key chords.
var StandardMaps = Maps{
	{"MacStandard", "Standard Mac KeyMap", Map{
		"UpArrow":             MoveUp,
		"Meta+UpArrow":        MoveUp,
		"Control+P":           MoveUp,
		"Control+Meta+P":      MoveUp,
		"DownArrow":           MoveDown,
		"Meta+DownArrow":      MoveDown,
		"Control+N":           MoveDown,
		"Control+Meta+N":      MoveDown,
		"RightArrow":          MoveRight,
		"Meta+RightArrow":     End,
		"Control+F":           MoveRight,
		"Control+Meta+F":      MoveRight,
		"LeftArrow":           MoveLeft,
		"Meta+LeftArrow":      Home,
		"Control+B":           MoveLeft,
		"Control+Meta+B":      MoveLeft,
		"PageUp":              PageUp,
		"Control+UpArrow":     PageUp,
		"Control+U":           PageUp,
		"PageDown":            PageDown,
		"Control+DownArrow":   PageDown,
		"Alt+√":               PageDown,
		"Alt+V":               PageDown,
		"Meta+Home":           DocHome,
		"Control+Home":        DocHome,
		"Alt+Home":            DocHome,
		"Meta+End":            DocEnd,
		"Control+End":         DocEnd,
		"Alt+End":             DocEnd,
		"Meta+H":              DocHome,
		"Meta+L":              DocEnd,
		"Control+RightArrow":  WordRight,
		"Control+LeftArrow":   WordLeft,
		"Alt+RightArrow":      WordRight,
		"Alt+LeftArrow":       WordLeft,
		"Home":                Home,
		"Control+A":           Home,
		"End":                 End,
		"Control+E":           End,
		"Tab":                 FocusNext,
		"Shift+Tab":           FocusPrev,
		"ReturnEnter":         Enter,
		"KeypadEnter":         Enter,
		"Meta+A":              SelectAll,
		"Control+G":           CancelSelect,
		"Control+Spacebar":    SelectMode,
		"Control+ReturnEnter": Accept,
		"Meta+ReturnEnter":    Accept,
		"Escape":              Abort,
		"Backspace":           Backspace,
		"Control+Backspace":   BackspaceWord,
		"Alt+Backspace":       BackspaceWord,
		"Meta+Backspace":      BackspaceWord,
		"Delete":              Delete,
		"Control+Delete":      DeleteWord,
		"Alt+Delete":          DeleteWord,
		"Control+D":           Delete,
		"Control+K":           Kill,
		"Alt+∑":               Copy,
		"Alt+C":               Copy,
		"Meta+C":              Copy,
		"Control+W":           Cut,
		"Meta+X":              Cut,
		"Control+Y":           Paste,
		"Control+V":           Paste,
		"Meta+V":              Paste,
		"Meta+Shift+V":        PasteHist,
		"Alt+D":               Duplicate,
		"Control+T":           Transpose,
		"Alt+T":               TransposeWord,
		"Control+Z":           Undo,
		"Meta+Z":              Undo,
		"Control+Shift+Z":     Redo,
		"Meta+Shift+Z":        Redo,
		"Control+I":           Insert,
		"Control+O":           InsertAfter,
		"Meta+Shift+=":        ZoomIn,
		"Meta+=":              ZoomIn,
		"Meta+-":              ZoomOut,
		"Control+=":           ZoomIn,
		"Control+Shift++":     ZoomIn,
		"Meta+Shift+-":        ZoomOut,
		"Control+-":           ZoomOut,
		"Control+Shift+_":     ZoomOut,
		"F5":                  Refresh,
		"Control+L":           Recenter,
		"Control+.":           Complete,
		"Control+,":           Lookup,
		"Control+S":           Search,
		"Meta+F":              Find,
		"Meta+R":              Replace,
		"Control+J":           Jump,
		"Control+[":           HistPrev,
		"Control+]":           HistNext,
		"Meta+[":              HistPrev,
		"Meta+]":              HistNext,
		"F10":                 Menu,
		"Control+M":           Menu,
		"Meta+`":              WinFocusNext,
		"Meta+W":              WinClose,
		"Control+Alt+G":       WinSnapshot,
		"Control+Shift+G":     WinSnapshot,
		"Meta+N":              New,
		"Meta+Shift+N":        NewAlt1,
		"Meta+Alt+N":          NewAlt2,
		"Meta+O":              Open,
		"Meta+Shift+O":        OpenAlt1,
		"Meta+Alt+O":          OpenAlt2,
		"Meta+S":              Save,
		"Meta+Shift+S":        SaveAs,
		"Meta+Alt+S":          SaveAlt,
		"Meta+Shift+W":        CloseAlt1,
		"Meta+Alt+W":          CloseAlt2,
		"Control+C":           MultiA,
		"Control+X":           MultiB,
	}},
	{"MacEmacs", "Mac with emacs-style navigation -- emacs wins in conflicts", Map{
		"UpArrow":             MoveUp,
		"Meta+UpArrow":        MoveUp,
		"Control+P":           MoveUp,
		"Control+Meta+P":      MoveUp,
		"DownArrow":           MoveDown,
		"Meta+DownArrow":      MoveDown,
		"Control+N":           MoveDown,
		"Control+Meta+N":      MoveDown,
		"RightArrow":          MoveRight,
		"Meta+RightArrow":     End,
		"Control+F":           MoveRight,
		"Control+Meta+F":      MoveRight,
		"LeftArrow":           MoveLeft,
		"Meta+LeftArrow":      Home,
		"Control+B":           MoveLeft,
		"Control+Meta+B":      MoveLeft,
		"PageUp":              PageUp,
		"Control+UpArrow":     PageUp,
		"Control+U":           PageUp,
		"PageDown":            PageDown,
		"Control+DownArrow":   PageDown,
		"Alt+√":               PageDown,
		"Alt+V":               PageDown,
		"Control+V":           PageDown,
		"Control+RightArrow":  WordRight,
		"Control+LeftArrow":   WordLeft,
		"Alt+RightArrow":      WordRight,
		"Alt+LeftArrow":       WordLeft,
		"Home":                Home,
		"Control+A":           Home,
		"End":                 End,
		"Control+E":           End,
		"Meta+Home":           DocHome,
		"Alt+Home":            DocHome,
		"Control+Home":        DocHome,
		"Meta+H":              DocHome,
		"Control+H":           DocHome,
		"Control+Alt+A":       DocHome,
		"Meta+End":            DocEnd,
		"Control+End":         DocEnd,
		"Alt+End":             DocEnd,
		"Meta+L":              DocEnd,
		"Control+Alt+E":       DocEnd,
		"Alt+Ƒ":               WordRight,
		"Alt+F":               WordRight,
		"Alt+∫":               WordLeft,
		"Alt+B":               WordLeft,
		"Tab":                 FocusNext,
		"Shift+Tab":           FocusPrev,
		"ReturnEnter":         Enter,
		"KeypadEnter":         Enter,
		"Meta+A":              SelectAll,
		"Control+G":           CancelSelect,
		"Control+Spacebar":    SelectMode,
		"Control+ReturnEnter": Accept,
		"Meta+ReturnEnter":    Accept,
		"Escape":              Abort,
		"Backspace":           Backspace,
		"Control+Backspace":   BackspaceWord,
		"Alt+Backspace":       BackspaceWord,
		"Meta+Backspace":      BackspaceWord,
		"Delete":              Delete,
		"Control+Delete":      DeleteWord,
		"Alt+Delete":          DeleteWord,
		"Control+D":           Delete,
		"Control+K":           Kill,
		"Alt+∑":               Copy,
		"Alt+C":               Copy,
		"Meta+C":              Copy,
		"Control+W":           Cut,
		"Meta+X":              Cut,
		"Control+Y":           Paste,
		"Meta+V":              Paste,
		"Meta+Shift+V":        PasteHist,
		"Control+Shift+Y":     PasteHist,
		"Alt+∂":               Duplicate,
		"Alt+D":               Duplicate,
		"Control+T":           Transpose,
		"Alt+T":               TransposeWord,
		"Control+Z":           Undo,
		"Meta+Z":              Undo,
		"Control+/":           Undo,
		"Control+Shift+Z":     Redo,
		"Meta+Shift+Z":        Redo,
		"Control+I":           Insert,
		"Control+O":           InsertAfter,
		"Meta+Shift+=":        ZoomIn,
		"Meta+=":              ZoomIn,
		"Meta+-":              ZoomOut,
		"Control+=":           ZoomIn,
		"Control+Shift++":     ZoomIn,
		"Meta+Shift+-":        ZoomOut,
		"Control+-":           ZoomOut,
		"Control+Shift+_":     ZoomOut,
		"F5":                  Refresh,
		"Control+L":           Recenter,
		"Control+.":           Complete,
		"Control+,":           Lookup,
		"Control+S":           Search,
		"Meta+F":              Find,
		"Meta+R":              Replace,
		"Control+R":           Replace,
		"Control+J":           Jump,
		"Control+[":           HistPrev,
		"Control+]":           HistNext,
		"Meta+[":              HistPrev,
		"Meta+]":              HistNext,
		"F10":                 Menu,
		"Control+M":           Menu,
		"Meta+`":              WinFocusNext,
		"Meta+W":              WinClose,
		"Control+Alt+G":       WinSnapshot,
		"Control+Shift+G":     WinSnapshot,
		"Meta+N":              New,
		"Meta+Shift+N":        NewAlt1,
		"Meta+Alt+N":          NewAlt2,
		"Meta+O":              Open,
		"Meta+Shift+O":        OpenAlt1,
		"Meta+Alt+O":          OpenAlt2,
		"Meta+S":              Save,
		"Meta+Shift+S":        SaveAs,
		"Meta+Alt+S":          SaveAlt,
		"Meta+Shift+W":        CloseAlt1,
		"Meta+Alt+W":          CloseAlt2,
		"Control+C":           MultiA,
		"Control+X":           MultiB,
	}},
	{"LinuxEmacs", "Linux with emacs-style navigation -- emacs wins in conflicts", Map{
		"UpArrow":             MoveUp,
		"Alt+UpArrow":         MoveUp,
		"Control+P":           MoveUp,
		"Control+Alt+P":       MoveUp,
		"DownArrow":           MoveDown,
		"Alt+DownArrow":       MoveDown,
		"Control+N":           MoveDown,
		"Control+Alt+N":       MoveDown,
		"RightArrow":          MoveRight,
		"Alt+RightArrow":      End,
		"Control+F":           MoveRight,
		"Control+Alt+F":       MoveRight,
		"LeftArrow":           MoveLeft,
		"Alt+LeftArrow":       Home,
		"Control+B":           MoveLeft,
		"Control+Alt+B":       MoveLeft,
		"PageUp":              PageUp,
		"Control+UpArrow":     PageUp,
		"Control+U":           PageUp,
		"Control+Alt+U":       PageUp,
		"PageDown":            PageDown,
		"Control+DownArrow":   PageDown,
		"Control+V":           PageDown,
		"Control+Alt+V":       PageDown,
		"Alt+Home":            DocHome,
		"Control+Home":        DocHome,
		"Alt+H":               DocHome,
		"Control+Alt+A":       DocHome,
		"Alt+End":             DocEnd,
		"Control+End":         DocEnd,
		"Alt+L":               DocEnd,
		"Control+Alt+E":       DocEnd,
		"Control+RightArrow":  WordRight,
		"Control+LeftArrow":   WordLeft,
		"Home":                Home,
		"Control+A":           Home,
		"End":                 End,
		"Control+E":           End,
		"Tab":                 FocusNext,
		"Shift+Tab":           FocusPrev,
		"ReturnEnter":         Enter,
		"KeypadEnter":         Enter,
		"Alt+A":               SelectAll,
		"Control+G":           CancelSelect,
		"Control+Spacebar":    SelectMode,
		"Control+ReturnEnter": Accept,
		"Escape":              Abort,
		"Backspace":           Backspace,
		"Control+Backspace":   BackspaceWord,
		"Alt+Backspace":       BackspaceWord,
		"Delete":              Delete,
		"Control+D":           Delete,
		"Control+Delete":      DeleteWord,
		"Alt+Delete":          DeleteWord,
		"Control+K":           Kill,
		"Alt+W":               Copy,
		"Alt+C":               Copy,
		"Control+W":           Cut,
		"Alt+X":               Cut,
		"Control+Y":           Paste,
		"Alt+V":               Paste,
		"Alt+Shift+V":         PasteHist,
		"Control+Shift+Y":     PasteHist,
		"Alt+D":               Duplicate,
		"Control+T":           Transpose,
		"Alt+T":               TransposeWord,
		"Control+Z":           Undo,
		"Control+/":           Undo,
		"Control+Shift+Z":     Redo,
		"Control+I":           Insert,
		"Control+O":           InsertAfter,
		"Control+=":           ZoomIn,
		"Control+Shift++":     ZoomIn,
		"Control+-":           ZoomOut,
		"Control+Shift+_":     ZoomOut,
		"F5":                  Refresh,
		"Control+L":           Recenter,
		"Control+.":           Complete,
		"Control+,":           Lookup,
		"Control+S":           Search,
		"Alt+F":               Find,
		"Control+R":           Replace,
		"Control+J":           Jump,
		"Control+[":           HistPrev,
		"Control+]":           HistNext,
		"F10":                 Menu,
		"Control+M":           Menu,
		"Alt+F6":              WinFocusNext,
		"Control+Shift+W":     WinClose,
		"Control+Alt+G":       WinSnapshot,
		"Control+Shift+G":     WinSnapshot,
		"Alt+N":               New, // ctrl keys conflict..
		"Alt+Shift+N":         NewAlt1,
		"Alt+O":               Open,
		"Alt+Shift+O":         OpenAlt1,
		"Control+Alt+O":       OpenAlt2,
		"Alt+S":               Save,
		"Alt+Shift+S":         SaveAs,
		"Control+Alt+S":       SaveAlt,
		"Alt+Shift+W":         CloseAlt1,
		"Control+Alt+W":       CloseAlt2,
		"Control+C":           MultiA,
		"Control+X":           MultiB,
	}},
	{"LinuxStandard", "Standard Linux KeyMap", Map{
		"UpArrow":             MoveUp,
		"DownArrow":           MoveDown,
		"RightArrow":          MoveRight,
		"LeftArrow":           MoveLeft,
		"PageUp":              PageUp,
		"Control+UpArrow":     PageUp,
		"PageDown":            PageDown,
		"Control+DownArrow":   PageDown,
		"Home":                Home,
		"Alt+LeftArrow":       Home,
		"End":                 End,
		"Alt+Home":            DocHome,
		"Control+Home":        DocHome,
		"Alt+End":             DocEnd,
		"Control+End":         DocEnd,
		"Control+RightArrow":  WordRight,
		"Control+LeftArrow":   WordLeft,
		"Alt+RightArrow":      End,
		"Tab":                 FocusNext,
		"Shift+Tab":           FocusPrev,
		"ReturnEnter":         Enter,
		"KeypadEnter":         Enter,
		"Control+A":           SelectAll,
		"Control+Shift+A":     CancelSelect,
		"Control+G":           CancelSelect,
		"Control+Spacebar":    SelectMode, // change input method / keyboard
		"Control+ReturnEnter": Accept,
		"Escape":              Abort,
		"Backspace":           Backspace,
		"Control+Backspace":   BackspaceWord,
		"Alt+Backspace":       BackspaceWord,
		"Delete":              Delete,
		"Control+Delete":      DeleteWord,
		"Alt+Delete":          DeleteWord,
		"Control+K":           Kill,
		"Control+C":           Copy,
		"Control+X":           Cut,
		"Control+V":           Paste,
		"Control+Shift+V":     PasteHist,
		"Alt+D":               Duplicate,
		"Control+T":           Transpose,
		"Alt+T":               TransposeWord,
		"Control+Z":           Undo,
		"Control+Y":           Redo,
		"Control+Shift+Z":     Redo,
		"Control+Alt+I":       Insert,
		"Control+Alt+O":       InsertAfter,
		"Control+=":           ZoomIn,
		"Control+Shift++":     ZoomIn,
		"Control+-":           ZoomOut,
		"Control+Shift+_":     ZoomOut,
		"F5":                  Refresh,
		"Control+L":           Recenter,
		"Control+.":           Complete,
		"Control+,":           Lookup,
		"Alt+S":               Search,
		"Control+F":           Find,
		"Control+H":           Replace,
		"Control+R":           Replace,
		"Control+J":           Jump,
		"Control+[":           HistPrev,
		"Control+]":           HistNext,
		"Control+N":           New,
		"F10":                 Menu,
		"Control+M":           Menu,
		"Alt+F6":              WinFocusNext,
		"Control+W":           WinClose,
		"Control+Alt+G":       WinSnapshot,
		"Control+Shift+G":     WinSnapshot,
		"Control+Shift+N":     NewAlt1,
		"Control+Alt+N":       NewAlt2,
		"Control+O":           Open,
		"Control+Shift+O":     OpenAlt1,
		"Alt+Shift+O":         OpenAlt2,
		"Control+S":           Save,
		"Control+Shift+S":     SaveAs,
		"Control+Alt+S":       SaveAlt,
		"Control+Shift+W":     CloseAlt1,
		"Control+Alt+W":       CloseAlt2,
		"Control+B":           MultiA,
		"Control+E":           MultiB,
	}},
	{"WindowsStandard", "Standard Windows KeyMap", Map{
		"UpArrow":             MoveUp,
		"DownArrow":           MoveDown,
		"RightArrow":          MoveRight,
		"LeftArrow":           MoveLeft,
		"PageUp":              PageUp,
		"Control+UpArrow":     PageUp,
		"PageDown":            PageDown,
		"Control+DownArrow":   PageDown,
		"Home":                Home,
		"Alt+LeftArrow":       Home,
		"End":                 End,
		"Alt+RightArrow":      End,
		"Control+Home":        DocHome,
		"Alt+Home":            DocHome,
		"Control+End":         DocEnd,
		"Alt+End":             DocEnd,
		"Control+RightArrow":  WordRight,
		"Control+LeftArrow":   WordLeft,
		"Tab":                 FocusNext,
		"Shift+Tab":           FocusPrev,
		"ReturnEnter":         Enter,
		"KeypadEnter":         Enter,
		"Control+A":           SelectAll,
		"Control+Shift+A":     CancelSelect,
		"Control+G":           CancelSelect,
		"Control+Spacebar":    SelectMode, // change input method / keyboard
		"Control+ReturnEnter": Accept,
		"Escape":              Abort,
		"Backspace":           Backspace,
		"Control+Backspace":   BackspaceWord,
		"Alt+Backspace":       BackspaceWord,
		"Delete":              Delete,
		"Control+Delete":      DeleteWord,
		"Alt+Delete":          DeleteWord,
		"Control+K":           Kill,
		"Control+C":           Copy,
		"Control+X":           Cut,
		"Control+V":           Paste,
		"Control+Shift+V":     PasteHist,
		"Alt+D":               Duplicate,
		"Control+T":           Transpose,
		"Alt+T":               TransposeWord,
		"Control+Z":           Undo,
		"Control+Y":           Redo,
		"Control+Shift+Z":     Redo,
		"Control+Alt+I":       Insert,
		"Control+Alt+O":       InsertAfter,
		"Control+=":           ZoomIn,
		"Control+Shift++":     ZoomIn,
		"Control+-":           ZoomOut,
		"Control+Shift+_":     ZoomOut,
		"F5":                  Refresh,
		"Control+L":           Recenter,
		"Control+.":           Complete,
		"Control+,":           Lookup,
		"Alt+S":               Search,
		"Control+F":           Find,
		"Control+H":           Replace,
		"Control+R":           Replace,
		"Control+J":           Jump,
		"Control+[":           HistPrev,
		"Control+]":           HistNext,
		"F10":                 Menu,
		"Control+M":           Menu,
		"Alt+F6":              WinFocusNext,
		"Control+W":           WinClose,
		"Control+Alt+G":       WinSnapshot,
		"Control+Shift+G":     WinSnapshot,
		"Control+N":           New,
		"Control+Shift+N":     NewAlt1,
		"Control+Alt+N":       NewAlt2,
		"Control+O":           Open,
		"Control+Shift+O":     OpenAlt1,
		"Alt+Shift+O":         OpenAlt2,
		"Control+S":           Save,
		"Control+Shift+S":     SaveAs,
		"Control+Alt+S":       SaveAlt,
		"Control+Shift+W":     CloseAlt1,
		"Control+Alt+W":       CloseAlt2,
		"Control+B":           MultiA,
		"Control+E":           MultiB,
	}},
	{"ChromeStd", "Standard chrome-browser and linux-under-chrome bindings", Map{
		"UpArrow":             MoveUp,
		"DownArrow":           MoveDown,
		"RightArrow":          MoveRight,
		"LeftArrow":           MoveLeft,
		"PageUp":              PageUp,
		"Control+UpArrow":     PageUp,
		"PageDown":            PageDown,
		"Control+DownArrow":   PageDown,
		"Home":                Home,
		"Alt+LeftArrow":       Home,
		"End":                 End,
		"Alt+Home":            DocHome,
		"Control+Home":        DocHome,
		"Alt+End":             DocEnd,
		"Control+End":         DocEnd,
		"Control+RightArrow":  WordRight,
		"Control+LeftArrow":   WordLeft,
		"Alt+RightArrow":      End,
		"Tab":                 FocusNext,
		"Shift+Tab":           FocusPrev,
		"ReturnEnter":         Enter,
		"KeypadEnter":         Enter,
		"Control+A":           SelectAll,
		"Control+Shift+A":     CancelSelect,
		"Control+G":           CancelSelect,
		"Control+Spacebar":    SelectMode, // change input method / keyboard
		"Control+ReturnEnter": Accept,
		"Escape":              Abort,
		"Backspace":           Backspace,
		"Control+Backspace":   BackspaceWord,
		"Alt+Backspace":       BackspaceWord,
		"Delete":              Delete,
		"Control+Delete":      DeleteWord,
		"Alt+Delete":          DeleteWord,
		"Control+K":           Kill,
		"Control+C":           Copy,
		"Control+X":           Cut,
		"Control+V":           Paste,
		"Control+Shift+V":     PasteHist,
		"Alt+D":               Duplicate,
		"Control+T":           Transpose,
		"Alt+T":               TransposeWord,
		"Control+Z":           Undo,
		"Control+Y":           Redo,
		"Control+Shift+Z":     Redo,
		"Control+Alt+I":       Insert,
		"Control+Alt+O":       InsertAfter,
		"Control+=":           ZoomIn,
		"Control+Shift++":     ZoomIn,
		"Control+-":           ZoomOut,
		"Control+Shift+_":     ZoomOut,
		"F5":                  Refresh,
		"Control+L":           Recenter,
		"Control+.":           Complete,
		"Control+,":           Lookup,
		"Alt+S":               Search,
		"Control+F":           Find,
		"Control+H":           Replace,
		"Control+R":           Replace,
		"Control+J":           Jump,
		"Control+[":           HistPrev,
		"Control+]":           HistNext,
		"F10":                 Menu,
		"Control+M":           Menu,
		"Alt+F6":              WinFocusNext,
		"Control+W":           WinClose,
		"Control+Alt+G":       WinSnapshot,
		"Control+Shift+G":     WinSnapshot,
		"Control+N":           New,
		"Control+Shift+N":     NewAlt1,
		"Control+Alt+N":       NewAlt2,
		"Control+O":           Open,
		"Control+Shift+O":     OpenAlt1,
		"Alt+Shift+O":         OpenAlt2,
		"Control+S":           Save,
		"Control+Shift+S":     SaveAs,
		"Control+Alt+S":       SaveAlt,
		"Control+Shift+W":     CloseAlt1,
		"Control+Alt+W":       CloseAlt2,
		"Control+B":           MultiA,
		"Control+E":           MultiB,
	}},
}
