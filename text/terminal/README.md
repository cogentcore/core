# terminal: a terminal widget

`terminal.Terminal` is a widget that runs a program, typically a shell, in a pseudo-terminal, and shows its output. It can be added to any app, for example as a panel tab, and has an API for the app to use the shell: `Run(cmd)`, `Cd(dir)`, `SendText`, `Paste`, `WorkingDir`, `Commands`, and `OnTitle`, `OnDir`, `OnCommand`, `OnExit` for changes reported by the program.

```go
tm := terminal.NewTerminal(parent) // the user's shell; or SetCommand("goal")
tm.OnDir(func(dir string) { ... }) // with shell integration
...
tm.Cd(filepath.Dir(filename))
```

## Design

There are three parts, each in its own package:

* [vt](vt) has the terminal emulator, `vt.Emulator`, with the terminal state: a grid of character cells for the primary and alternate screens, the cursor, the modes set by the program, and so on, updated from the output of the program, which is parsed with [x/ansi](https://github.com/charmbracelet/x/tree/main/ansi). It has no GUI, and is tested directly with the control sequences. Lines that scroll off the top of the primary screen are taken by the widget for its scrollback.

* [pty](pty) runs the program in a pseudo-terminal: [creack/pty](https://github.com/creack/pty) on Unix, and Windows ConPTY through [x/xpty](https://github.com/charmbracelet/x/tree/main/xpty).

* The `terminal` package has the widget, which is a `Stack` of two `View`s, which are read-only `textcore.Editor`s:
    * The lines view shows the scrollback and the primary screen as `lines.Lines` text, where each line is a logical line of output: the screen lines that the terminal wrapped (soft wraps) are joined, so the text is wrapped to the view width by `lines`, and rewrapped when the view is resized. The terminal size is set from this view, and its last lines are the screen. The text can be selected and copied, and scrolled back, as in an editor.
    * The grid view shows the alternate screen used by full-screen programs (editors, pagers, etc), with one line per screen row. It is on top while the program uses the alternate screen.

Each cell is one character in the `lines.Lines` text, with a zero width space after a wide character, so the columns stay aligned and the cursor position maps directly. The cell styles are converted to `rich.Text` markup (with 16 color palettes for light and dark color schemes, and 256 and 24-bit colors). The terminal cursor is shown as a highlight.

Keys are sent to the program, encoded as xterm does, except for the GUI shortcuts: copy, paste, and clear (Command on macOS, Control+Shift elsewhere, and Control+C with a selection), and Shift with PageUp, PageDown, Home and End to scroll the lines view. Mouse events are reported to full-screen programs that ask for them (Shift selects text instead), and the mouse wheel sends cursor keys to those that do not.

## Shell integration

The shell integration sequences of the [FinalTerm](https://gitlab.freedesktop.org/Per_Bothner/specifications/blob/master/proposals/semantic-prompts.md) (OSC 133) and [VS Code](https://code.visualstudio.com/docs/terminal/shell-integration) (OSC 633) protocols are parsed into `vt.Command`s, with the prompt, input, and output lines of each command, its exit code and working directory, and the current directory (OSC 7) is tracked. The scripts that make shells send these are not installed yet. This is the basis for features such as those of VS Code: decorations for the commands with their exit status, navigation among commands, selecting the output of a command, rerunning commands, and links to files in the output, which have a place in this design:

* The command lines can be mapped to lines in the lines view, for decorations (e.g., `Lines.SetLineColor`) and navigation.
* An editor for command input at the bottom could be added, sending the command when Return is pressed, while the shell is at its prompt (between the B and C marks).
* Links in the output are `rich` links (OSC 8 hyperlinks are already), so file:line links can be added in the markup.
