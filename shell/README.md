# Cogent Shell (cosh)

Cogent Shell (cosh) is a shell that combines the best parts of Go and bash to provide an integrated shell experience that allows you to easily run terminal commands while using Go for complicated logic. This allows you to write concise, elegant, and readable shell code that runs quickly on all platforms and transpiles to Go.

Examples:

```go
for i, f := range `ls` {   // `ls` executes shell and ranges over result
    echo {i} {strings.ToLower(f)}           // {} surrounds go within shell
    git status {f}
}
```

The simple idea is that each line is either Go or shell commands, determined in a fairly intuitive way mostly on the content at the start of the line (formal rules below), and they can be intermixed by wrapping Go within `{ }` and shell from within backticks. 

* Go code is processed and formatted as usual (e.g., white space is irrelevant, etc).
* Shell code is space separated, like normal command-line invocations.

# Special syntax

* Multiple statements can be combined on one line, separated by `;` as in regular Go and shell languages.  Critically, the language determination for the first statement determines the language for the remaining statements; you cannot intermix the two.

# Shell mode

## Environment variables

* `set <var> <value>` (space delimited as in all shell)

## Output redirction

* Standard output redirect: `>` and `>&` (and `|`, `|&` if needed)

## Control flow

* Any error stops the script execution, except for statements wrapped in `[ ]`, indicating an "optional" statement, e.g.:

```sh
cd some; [mkdir sub]; cd sub
```

* `&` at the end of a statement runs in the background (as in bash) -- otherwise it waits until it completes before it continues.

* `jobs`, `fg`, `bg`, and `kill` builtin commands function as in usual bash.

# Go vs. Shell determination

The critical extension from standard Go syntax is for lines that are processed by the `Exec` functions, used for running arbitrary programs on the user's executable path.  Here are the rules (word = IDENT token):

* Backticks "``" anywhere:  Exec.  Returns a `string`.
* Within Exec, `{}`: Go
* Line starts with `Go` Keyword: Go
* Line is one word: Exec
* Line starts with `path`: Exec
* Line starts with `"string"`: Exec
* Line starts with `word word`: Exec
* Line starts with `word {`: Exec
* Otherwise: Go



