# Difflib

> Difflib is a fork of https://github.com/ianbruene/go-difflib, which itself is a fork of https://github.com/pmezard/go-difflib

Difflib is an as yet partial port of python 3's difflib package.

The following publicly visible classes and functions have been ported:

* `SequenceMatcher`
* `Differ`
* `unified_diff()`
* `context_diff()`

## Installation

```sh
$ go get goki.dev/difflib
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
