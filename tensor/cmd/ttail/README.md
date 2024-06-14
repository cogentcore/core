# ttail: cli app to display, monitor tabular data files

`ttail` is a `tail`-like command for displaing tabular data in a cli / terminal window, typically .csv / .tsv log / data tabular files.

# Install

This should install into your `$GOPATH/bin` dir:

```bash
$ go install cogentcore.org/core/tensor/cmd/ttail@latest
```

# Run

Just pass files as args, e.g., on the test files in this dir:

```bash
$ ttail RA25*
```

# Keys

This is shown when you press `h` in the app:


| Key(s)  | Function      |
| ------- | ------------------------------------------------------ |
| spc,n   | page down                                                     |
| p       | page up                                                       |
| f       | scroll right-hand panel to the right                          |
| b       | scroll right-hand panel to the left                           |
| w       | widen the left-hand panel of columns                          |
| s       | shrink the left-hand panel of columns                         |
| t       | toggle tail-mode (auto updating as file grows) on/off         |
| a       | jump to top                                                   |
| e       | jump to end                                                   |
| v       | rotate down through the list of files (if not all displayed)  |
| u       | rotate up through the list of files (if not all displayed)    |
| m       | more minimum lines per file -- increase amount shown of each file |
| l       | less minimum lines per file -- decrease amount shown of each file |
| d       | toggle display of file names                                  |
| c       | toggle display of column numbers instead of names             |
| q       | quit                                                          |

# History

The original version was `pdptail`, written in perl, named for the PDP++ software, developed from 2004-2018.  Then there was `etail` written in Go, for the emergent system: https://github.com/emer. 


