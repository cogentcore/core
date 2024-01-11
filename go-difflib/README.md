go-difflib
==========

The previous owner of this project (pmezard) did not have the time to continue
working on it. Additionally I (ianbruene) needed additional ported features.

For these reasons I have taken over maintenance and further development of the
project.

[![GoDoc](https://godoc.org/github.com/ianbruene/go-difflib/difflib?status.svg)](https://godoc.org/github.com/ianbruene/go-difflib/difflib)

Go-difflib is an as yet partial port of python 3's difflib package.

The following publicly visible classes and functions have been ported:

* `SequenceMatcher`
* `Differ`
* `unified_diff()`
* `context_diff()`

## Installation

```bash
$ go get github.com/ianbruene/go-difflib/difflib
```

### UnifiedDiff Quick Start

Diffs are configured with Unified (or ContextDiff) structures, and can
be output to an io.Writer or returned as a string.

```Go
diff := difflib.LineDiffParams{
    A:        difflib.SplitLines("foo\nbar\n"),
    B:        difflib.SplitLines("foo\nbaz\n"),
    FromFile: "Original",
    ToFile:   "Current",
    Context:  3,
}
text, _ := difflib.GetUnifiedDiffString(diff)
fmt.Printf(text)
```

would output:

```
--- Original
+++ Current
@@ -1,3 +1,3 @@
 foo
-bar
+baz
```

### Differ Quick Start

Differ has been implemented primarily for the Compare() function at this time.

```Go
diff := difflib.NewDiffer()
out, err := diff.Compare(
    []string{"foo\n", "bar\n", "baz\n"},
	[]string{"foo\n", "bar1\n", "asdf\n", "baz\n"})
```

would output:

```
  foo
- bar
+ bar1
?    +
+ asdf
  baz
```
