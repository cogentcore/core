+++
Categories = ["Resources"]
+++

The **[[doc:keymap]]** package maps keyboard input [[events]] into a [[doc:keymap.Functions]] enum, which has a standard list of app functions that are typically activated by keyboard events. [[Settings]] allow the user to select from a set of different standard keyboard mappings, which are documented in the tables on this page.

As a programmer, it is best to use these standard functions in your app wherever possible, so that the standard keyboard mappings will automatically apply, including any custom mappings the user may have set up. See [[events#key function]] and [[button#events|button events]] for usage examples.

## Global shortcuts

| Function  | Key         |
| --------- | ----------- |
| Settings  | `Control+,` |
| Inspector | `Control+Shift+I` |
| Snapshot (grab) | `Control+Shift+G` |

On Mac, use the Command key ⌘ instead of Control, except for Snapshot.

Snapshot saves a PNG image and an SVG vector representation of the current window.

## Standard mappings

### By function

| Function                         | `MacStandard` | `MacEmacs` | `LinuxEmacs` | `LinuxStandard` | `WindowsStandard` | `ChromeStd` |
| ---------------------------- | ------------- | ---------- | ------------ | --------------- | ----------------- | ----------- |
| MoveUp | UpArrow, Control+P, Meta+UpArrow | UpArrow, Control+P, Meta+UpArrow | UpArrow, Control+P, Alt+UpArrow, Control+Alt+P | UpArrow | UpArrow | UpArrow |
| MoveDown | DownArrow, Control+N, Meta+DownArrow | DownArrow, Control+N, Meta+DownArrow | DownArrow, Control+N, Alt+DownArrow, Control+Alt+N | DownArrow | DownArrow | DownArrow |
| MoveRight | RightArrow, Control+F | RightArrow, Control+F | RightArrow, Control+F, Control+Alt+F | RightArrow | RightArrow | RightArrow |
| MoveLeft | LeftArrow, Control+B | LeftArrow, Control+B | LeftArrow, Control+B, Control+Alt+B | LeftArrow | LeftArrow | LeftArrow |
| PageUp | PageUp, Control+U, Control+UpArrow | PageUp, Control+U, Control+UpArrow | PageUp, Control+U, Control+UpArrow, Control+Alt+U | PageUp, Control+UpArrow | PageUp, Control+UpArrow | PageUp, Control+UpArrow |
| PageDown | PageDown, Control+DownArrow, Alt+V | PageDown, Control+V, Control+DownArrow, Alt+V | PageDown, Control+V, Control+DownArrow, Control+Alt+V | PageDown, Control+DownArrow | PageDown, Control+DownArrow | PageDown, Control+DownArrow |
| Home | Home, Control+A, Meta+LeftArrow | Home, Control+A, Meta+LeftArrow | Home, Control+A, Alt+LeftArrow | Home, Alt+LeftArrow | Home, Alt+LeftArrow | Home, Alt+LeftArrow |
| End | End, Control+E, Meta+RightArrow | End, Control+E, Meta+RightArrow | End, Control+E, Alt+RightArrow | End, Alt+RightArrow | End, Alt+RightArrow | End, Alt+RightArrow |
| DocHome | Control+Home, Meta+H, Meta+Home, Alt+Home | Control+H, Control+Home, Meta+H, Meta+Home, Alt+Home, Control+Alt+A | Control+Home, Alt+H, Alt+Home, Control+Alt+A | Control+Home, Alt+Home | Control+Home, Alt+Home | Control+Home, Alt+Home |
| DocEnd | Control+End, Meta+L, Meta+End, Alt+End | Control+End, Meta+L, Meta+End, Alt+End, Control+Alt+E | Control+End, Alt+L, Alt+End, Control+Alt+E | Control+End, Alt+End | Control+End, Alt+End | Control+End, Alt+End |
| WordRight | Control+RightArrow, Alt+RightArrow | Control+RightArrow, Alt+F, Alt+RightArrow | Control+RightArrow | Control+RightArrow | Control+RightArrow | Control+RightArrow |
| WordLeft | Control+LeftArrow, Alt+LeftArrow | Control+LeftArrow, Alt+B, Alt+LeftArrow | Control+LeftArrow | Control+LeftArrow | Control+LeftArrow | Control+LeftArrow |
| FocusNext | Tab | Tab | Tab | Tab | Tab | Tab |
| FocusPrev | Shift+Tab | Shift+Tab | Shift+Tab | Shift+Tab | Shift+Tab | Shift+Tab |
| Enter | ReturnEnter, KeypadEnter | ReturnEnter, KeypadEnter | ReturnEnter, KeypadEnter | ReturnEnter, KeypadEnter | ReturnEnter, KeypadEnter | ReturnEnter, KeypadEnter |
| Accept | Control+ReturnEnter, Meta+ReturnEnter | Control+ReturnEnter, Meta+ReturnEnter | Control+ReturnEnter | Control+ReturnEnter | Control+ReturnEnter | Control+ReturnEnter |
| CancelSelect | Control+G | Control+G | Control+G | Control+G, Control+Shift+A | Control+G, Control+Shift+A | Control+G, Control+Shift+A |
| SelectMode | Control+Spacebar | Control+Spacebar | Control+Spacebar | Control+Spacebar | Control+Spacebar | Control+Spacebar |
| SelectAll | Meta+A | Meta+A | Alt+A | Control+A | Control+A | Control+A |
| Abort | Escape | Escape | Escape | Escape | Escape | Escape |
| Copy | Meta+C, Alt+C | Meta+C, Alt+C | Alt+C, Alt+W | Control+C | Control+C | Control+C |
| Cut | Control+W, Meta+X | Control+W, Meta+X | Control+W, Alt+X | Control+X | Control+X | Control+X |
| Paste | Control+V, Control+Y, Meta+V | Control+Y, Meta+V | Control+Y, Alt+V | Control+V | Control+V | Control+V |
| PasteHist | Meta+Shift+V | Control+Shift+Y, Meta+Shift+V | Control+Shift+Y, Alt+Shift+V | Control+Shift+V | Control+Shift+V | Control+Shift+V |
| Backspace | Backspace | Backspace | Backspace | Backspace | Backspace | Backspace |
| BackspaceWord | Control+Backspace, Meta+Backspace, Alt+Backspace | Control+Backspace, Meta+Backspace, Alt+Backspace | Control+Backspace, Alt+Backspace | Control+Backspace, Alt+Backspace | Control+Backspace, Alt+Backspace | Control+Backspace, Alt+Backspace |
| Delete | Delete, Control+D | Delete, Control+D | Delete, Control+D | Delete | Delete | Delete |
| DeleteWord | Control+Delete, Alt+Delete | Control+Delete, Alt+Delete | Control+Delete, Alt+Delete | Control+Delete, Alt+Delete | Control+Delete, Alt+Delete | Control+Delete, Alt+Delete |
| Kill | Control+K | Control+K | Control+K | Control+K | Control+K | Control+K |
| Duplicate | Alt+D | Alt+D | Alt+D | Alt+D | Alt+D | Alt+D |
| Transpose | Control+T | Control+T | Control+T | Control+T | Control+T | Control+T |
| TransposeWord | Alt+T | Alt+T | Alt+T | Alt+T | Alt+T | Alt+T |
| Undo | Control+Z, Meta+Z | Control+Z, Meta+Z | Control+Z | Control+Z | Control+Z | Control+Z |
| Redo | Control+Shift+Z, Meta+Shift+Z | Control+Shift+Z, Meta+Shift+Z | Control+Shift+Z | Control+Y, Control+Shift+Z | Control+Y, Control+Shift+Z | Control+Y, Control+Shift+Z |
| Insert | Control+I | Control+I | Control+I | Control+Alt+I | Control+Alt+I | Control+Alt+I |
| InsertAfter | Control+O | Control+O | Control+O | Control+Alt+O | Control+Alt+O | Control+Alt+O |
| ZoomOut |               |            |              |                 |                   |             |
| ZoomIn |               |            |              |                 |                   |             |
| Refresh | F5 | F5 | F5 | F5 | F5 | F5 |
| Recenter | Control+L | Control+L | Control+L | Control+L | Control+L | Control+L |
| Complete |               |            |              |                 |                   |             |
| Lookup |               |            |              |                 |                   |             |
| Search | Control+S | Control+S | Control+S | Alt+S | Alt+S | Alt+S |
| Find | Meta+F | Meta+F | Alt+F | Control+F | Control+F | Control+F |
| Replace | Meta+R | Control+R, Meta+R | Control+R | Control+H, Control+R | Control+H, Control+R | Control+H, Control+R |
| Jump | Control+J | Control+J | Control+J | Control+J | Control+J | Control+J |
| HistPrev |               |            |              |                 |                   |             |
| HistNext |               |            |              |                 |                   |             |
| Menu | F10, Control+M | F10, Control+M | F10, Control+M | F10, Control+M | F10, Control+M | F10, Control+M |
| WinFocusNext |               |            | Alt+F6 | Alt+F6 | Alt+F6 | Alt+F6 |
| WinClose | Meta+W | Meta+W | Control+Shift+W | Control+W | Control+W | Control+W |
| WinSnapshot | Control+Shift+G, Control+Alt+G | Control+Shift+G, Control+Alt+G | Control+Shift+G, Control+Alt+G | Control+Shift+G, Control+Alt+G | Control+Shift+G, Control+Alt+G | Control+Shift+G, Control+Alt+G |
| New | Meta+N | Meta+N | Alt+N | Control+N | Control+N | Control+N |
| NewAlt1 | Meta+Shift+N | Meta+Shift+N | Alt+Shift+N | Control+Shift+N | Control+Shift+N | Control+Shift+N |
| NewAlt2 | Meta+Alt+N | Meta+Alt+N |              | Control+Alt+N | Control+Alt+N | Control+Alt+N |
| Open | Meta+O | Meta+O | Alt+O | Control+O | Control+O | Control+O |
| OpenAlt1 | Meta+Shift+O | Meta+Shift+O | Alt+Shift+O | Control+Shift+O | Control+Shift+O | Control+Shift+O |
| OpenAlt2 | Meta+Alt+O | Meta+Alt+O | Control+Alt+O | Alt+Shift+O | Alt+Shift+O | Alt+Shift+O |
| Save | Meta+S | Meta+S | Alt+S | Control+S | Control+S | Control+S |
| SaveAs | Meta+Shift+S | Meta+Shift+S | Alt+Shift+S | Control+Shift+S | Control+Shift+S | Control+Shift+S |
| SaveAlt | Meta+Alt+S | Meta+Alt+S | Control+Alt+S | Control+Alt+S | Control+Alt+S | Control+Alt+S |
| CloseAlt1 | Meta+Shift+W | Meta+Shift+W | Alt+Shift+W | Control+Shift+W | Control+Shift+W | Control+Shift+W |
| CloseAlt2 | Meta+Alt+W | Meta+Alt+W | Control+Alt+W | Control+Alt+W | Control+Alt+W | Control+Alt+W |
| MultiA | Control+C | Control+C | Control+C | Control+B | Control+B | Control+B |
| MultiB | Control+X | Control+X | Control+X | Control+E | Control+E | Control+E |

### No Modifiers

| Key                          | `MacStandard` | `MacEmacs` | `LinuxEmacs` | `LinuxStandard` | `WindowsStandard` | `ChromeStd` |
| ---------------------------- | ------------- | ---------- | ------------ | --------------- | ----------------- | ----------- |
| ReturnEnter | Enter | Enter | Enter | Enter | Enter | Enter |
| Escape | Abort | Abort | Abort | Abort | Abort | Abort |
| Backspace | Backspace | Backspace | Backspace | Backspace | Backspace | Backspace |
| Tab | FocusNext | FocusNext | FocusNext | FocusNext | FocusNext | FocusNext |
| F5 | Refresh | Refresh | Refresh | Refresh | Refresh | Refresh |
| F10 | Menu | Menu | Menu | Menu | Menu | Menu |
| Home | Home | Home | Home | Home | Home | Home |
| PageUp | PageUp | PageUp | PageUp | PageUp | PageUp | PageUp |
| Delete | Delete | Delete | Delete | Delete | Delete | Delete |
| End | End | End | End | End | End | End |
| PageDown | PageDown | PageDown | PageDown | PageDown | PageDown | PageDown |
| RightArrow | MoveRight | MoveRight | MoveRight | MoveRight | MoveRight | MoveRight |
| LeftArrow | MoveLeft | MoveLeft | MoveLeft | MoveLeft | MoveLeft | MoveLeft |
| DownArrow | MoveDown | MoveDown | MoveDown | MoveDown | MoveDown | MoveDown |
| UpArrow | MoveUp | MoveUp | MoveUp | MoveUp | MoveUp | MoveUp |
| KeypadEnter | Enter | Enter | Enter | Enter | Enter | Enter |


### Shift

| Key                          | `MacStandard` | `MacEmacs` | `LinuxEmacs` | `LinuxStandard` | `WindowsStandard` | `ChromeStd` |
| ---------------------------- | ------------- | ---------- | ------------ | --------------- | ----------------- | ----------- |
| Shift+Tab | FocusPrev | FocusPrev | FocusPrev | FocusPrev | FocusPrev | FocusPrev |


### Control

| Key                          | `MacStandard` | `MacEmacs` | `LinuxEmacs` | `LinuxStandard` | `WindowsStandard` | `ChromeStd` |
| ---------------------------- | ------------- | ---------- | ------------ | --------------- | ----------------- | ----------- |
| Control+A | Home | Home | Home | SelectAll | SelectAll | SelectAll |
| Control+B | MoveLeft | MoveLeft | MoveLeft | MultiA | MultiA | MultiA |
| Control+C | MultiA | MultiA | MultiA | Copy | Copy | Copy |
| Control+D | Delete | Delete | Delete |                 |                   |             |
| Control+E | End | End | End | MultiB | MultiB | MultiB |
| Control+F | MoveRight | MoveRight | MoveRight | Find | Find | Find |
| Control+G | CancelSelect | CancelSelect | CancelSelect | CancelSelect | CancelSelect | CancelSelect |
| Control+H |               | DocHome |              | Replace | Replace | Replace |
| Control+I | Insert | Insert | Insert |                 |                   |             |
| Control+J | Jump | Jump | Jump | Jump | Jump | Jump |
| Control+K | Kill | Kill | Kill | Kill | Kill | Kill |
| Control+L | Recenter | Recenter | Recenter | Recenter | Recenter | Recenter |
| Control+M | Menu | Menu | Menu | Menu | Menu | Menu |
| Control+N | MoveDown | MoveDown | MoveDown | New | New | New |
| Control+O | InsertAfter | InsertAfter | InsertAfter | Open | Open | Open |
| Control+P | MoveUp | MoveUp | MoveUp |                 |                   |             |
| Control+R |               | Replace | Replace | Replace | Replace | Replace |
| Control+S | Search | Search | Search | Save | Save | Save |
| Control+T | Transpose | Transpose | Transpose | Transpose | Transpose | Transpose |
| Control+U | PageUp | PageUp | PageUp |                 |                   |             |
| Control+V | Paste | PageDown | PageDown | Paste | Paste | Paste |
| Control+W | Cut | Cut | Cut | WinClose | WinClose | WinClose |
| Control+X | MultiB | MultiB | MultiB | Cut | Cut | Cut |
| Control+Y | Paste | Paste | Paste | Redo | Redo | Redo |
| Control+Z | Undo | Undo | Undo | Undo | Undo | Undo |
| Control+ReturnEnter | Accept | Accept | Accept | Accept | Accept | Accept |
| Control+Backspace | BackspaceWord | BackspaceWord | BackspaceWord | BackspaceWord | BackspaceWord | BackspaceWord |
| Control+Spacebar | SelectMode | SelectMode | SelectMode | SelectMode | SelectMode | SelectMode |
| Control+Home | DocHome | DocHome | DocHome | DocHome | DocHome | DocHome |
| Control+Delete | DeleteWord | DeleteWord | DeleteWord | DeleteWord | DeleteWord | DeleteWord |
| Control+End | DocEnd | DocEnd | DocEnd | DocEnd | DocEnd | DocEnd |
| Control+RightArrow | WordRight | WordRight | WordRight | WordRight | WordRight | WordRight |
| Control+LeftArrow | WordLeft | WordLeft | WordLeft | WordLeft | WordLeft | WordLeft |
| Control+DownArrow | PageDown | PageDown | PageDown | PageDown | PageDown | PageDown |
| Control+UpArrow | PageUp | PageUp | PageUp | PageUp | PageUp | PageUp |


### Control+Shift

| Key                          | `MacStandard` | `MacEmacs` | `LinuxEmacs` | `LinuxStandard` | `WindowsStandard` | `ChromeStd` |
| ---------------------------- | ------------- | ---------- | ------------ | --------------- | ----------------- | ----------- |
| Control+Shift+A |               |            |              | CancelSelect | CancelSelect | CancelSelect |
| Control+Shift+G | WinSnapshot | WinSnapshot | WinSnapshot | WinSnapshot | WinSnapshot | WinSnapshot |
| Control+Shift+N |               |            |              | NewAlt1 | NewAlt1 | NewAlt1 |
| Control+Shift+O |               |            |              | OpenAlt1 | OpenAlt1 | OpenAlt1 |
| Control+Shift+S |               |            |              | SaveAs | SaveAs | SaveAs |
| Control+Shift+V |               |            |              | PasteHist | PasteHist | PasteHist |
| Control+Shift+W |               |            | WinClose | CloseAlt1 | CloseAlt1 | CloseAlt1 |
| Control+Shift+Y |               | PasteHist | PasteHist |                 |                   |             |
| Control+Shift+Z | Redo | Redo | Redo | Redo | Redo | Redo |


### Meta

| Key                          | `MacStandard` | `MacEmacs` | `LinuxEmacs` | `LinuxStandard` | `WindowsStandard` | `ChromeStd` |
| ---------------------------- | ------------- | ---------- | ------------ | --------------- | ----------------- | ----------- |
| Meta+A | SelectAll | SelectAll |              |                 |                   |             |
| Meta+C | Copy | Copy |              |                 |                   |             |
| Meta+F | Find | Find |              |                 |                   |             |
| Meta+H | DocHome | DocHome |              |                 |                   |             |
| Meta+L | DocEnd | DocEnd |              |                 |                   |             |
| Meta+N | New | New |              |                 |                   |             |
| Meta+O | Open | Open |              |                 |                   |             |
| Meta+R | Replace | Replace |              |                 |                   |             |
| Meta+S | Save | Save |              |                 |                   |             |
| Meta+V | Paste | Paste |              |                 |                   |             |
| Meta+W | WinClose | WinClose |              |                 |                   |             |
| Meta+X | Cut | Cut |              |                 |                   |             |
| Meta+Z | Undo | Undo |              |                 |                   |             |
| Meta+ReturnEnter | Accept | Accept |              |                 |                   |             |
| Meta+Backspace | BackspaceWord | BackspaceWord |              |                 |                   |             |
| Meta+Home | DocHome | DocHome |              |                 |                   |             |
| Meta+End | DocEnd | DocEnd |              |                 |                   |             |
| Meta+RightArrow | End | End |              |                 |                   |             |
| Meta+LeftArrow | Home | Home |              |                 |                   |             |
| Meta+DownArrow | MoveDown | MoveDown |              |                 |                   |             |
| Meta+UpArrow | MoveUp | MoveUp |              |                 |                   |             |


### Meta+Shift

| Key                          | `MacStandard` | `MacEmacs` | `LinuxEmacs` | `LinuxStandard` | `WindowsStandard` | `ChromeStd` |
| ---------------------------- | ------------- | ---------- | ------------ | --------------- | ----------------- | ----------- |
| Meta+Shift+N | NewAlt1 | NewAlt1 |              |                 |                   |             |
| Meta+Shift+O | OpenAlt1 | OpenAlt1 |              |                 |                   |             |
| Meta+Shift+S | SaveAs | SaveAs |              |                 |                   |             |
| Meta+Shift+V | PasteHist | PasteHist |              |                 |                   |             |
| Meta+Shift+W | CloseAlt1 | CloseAlt1 |              |                 |                   |             |
| Meta+Shift+Z | Redo | Redo |              |                 |                   |             |


### Alt

| Key                          | `MacStandard` | `MacEmacs` | `LinuxEmacs` | `LinuxStandard` | `WindowsStandard` | `ChromeStd` |
| ---------------------------- | ------------- | ---------- | ------------ | --------------- | ----------------- | ----------- |
| Alt+A |               |            | SelectAll |                 |                   |             |
| Alt+B |               | WordLeft |              |                 |                   |             |
| Alt+C | Copy | Copy | Copy |                 |                   |             |
| Alt+D | Duplicate | Duplicate | Duplicate | Duplicate | Duplicate | Duplicate |
| Alt+F |               | WordRight | Find |                 |                   |             |
| Alt+H |               |            | DocHome |                 |                   |             |
| Alt+L |               |            | DocEnd |                 |                   |             |
| Alt+N |               |            | New |                 |                   |             |
| Alt+O |               |            | Open |                 |                   |             |
| Alt+S |               |            | Save | Search | Search | Search |
| Alt+T | TransposeWord | TransposeWord | TransposeWord | TransposeWord | TransposeWord | TransposeWord |
| Alt+V | PageDown | PageDown | Paste |                 |                   |             |
| Alt+W |               |            | Copy |                 |                   |             |
| Alt+X |               |            | Cut |                 |                   |             |
| Alt+Backspace | BackspaceWord | BackspaceWord | BackspaceWord | BackspaceWord | BackspaceWord | BackspaceWord |
| Alt+F6 |               |            | WinFocusNext | WinFocusNext | WinFocusNext | WinFocusNext |
| Alt+Home | DocHome | DocHome | DocHome | DocHome | DocHome | DocHome |
| Alt+Delete | DeleteWord | DeleteWord | DeleteWord | DeleteWord | DeleteWord | DeleteWord |
| Alt+End | DocEnd | DocEnd | DocEnd | DocEnd | DocEnd | DocEnd |
| Alt+RightArrow | WordRight | WordRight | End | End | End | End |
| Alt+LeftArrow | WordLeft | WordLeft | Home | Home | Home | Home |
| Alt+DownArrow |               |            | MoveDown |                 |                   |             |
| Alt+UpArrow |               |            | MoveUp |                 |                   |             |


### Alt+Shift

| Key                          | `MacStandard` | `MacEmacs` | `LinuxEmacs` | `LinuxStandard` | `WindowsStandard` | `ChromeStd` |
| ---------------------------- | ------------- | ---------- | ------------ | --------------- | ----------------- | ----------- |
| Alt+Shift+N |               |            | NewAlt1 |                 |                   |             |
| Alt+Shift+O |               |            | OpenAlt1 | OpenAlt2 | OpenAlt2 | OpenAlt2 |
| Alt+Shift+S |               |            | SaveAs |                 |                   |             |
| Alt+Shift+V |               |            | PasteHist |                 |                   |             |
| Alt+Shift+W |               |            | CloseAlt1 |                 |                   |             |


### Control+Alt

| Key                          | `MacStandard` | `MacEmacs` | `LinuxEmacs` | `LinuxStandard` | `WindowsStandard` | `ChromeStd` |
| ---------------------------- | ------------- | ---------- | ------------ | --------------- | ----------------- | ----------- |
| Control+Alt+A |               | DocHome | DocHome |                 |                   |             |
| Control+Alt+B |               |            | MoveLeft |                 |                   |             |
| Control+Alt+E |               | DocEnd | DocEnd |                 |                   |             |
| Control+Alt+F |               |            | MoveRight |                 |                   |             |
| Control+Alt+G | WinSnapshot | WinSnapshot | WinSnapshot | WinSnapshot | WinSnapshot | WinSnapshot |
| Control+Alt+I |               |            |              | Insert | Insert | Insert |
| Control+Alt+N |               |            | MoveDown | NewAlt2 | NewAlt2 | NewAlt2 |
| Control+Alt+O |               |            | OpenAlt2 | InsertAfter | InsertAfter | InsertAfter |
| Control+Alt+P |               |            | MoveUp |                 |                   |             |
| Control+Alt+S |               |            | SaveAlt | SaveAlt | SaveAlt | SaveAlt |
| Control+Alt+U |               |            | PageUp |                 |                   |             |
| Control+Alt+V |               |            | PageDown |                 |                   |             |
| Control+Alt+W |               |            | CloseAlt2 | CloseAlt2 | CloseAlt2 | CloseAlt2 |


### Meta+Alt

| Key                          | `MacStandard` | `MacEmacs` | `LinuxEmacs` | `LinuxStandard` | `WindowsStandard` | `ChromeStd` |
| ---------------------------- | ------------- | ---------- | ------------ | --------------- | ----------------- | ----------- |
| Meta+Alt+N | NewAlt2 | NewAlt2 |              |                 |                   |             |
| Meta+Alt+O | OpenAlt2 | OpenAlt2 |              |                 |                   |             |
| Meta+Alt+S | SaveAlt | SaveAlt |              |                 |                   |             |
| Meta+Alt+W | CloseAlt2 | CloseAlt2 |              |                 |                   |             |


