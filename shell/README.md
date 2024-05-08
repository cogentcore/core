# Cogent Shell (cosh)

Cogent Shell (cosh) is a shell that combines the best parts of Go and bash to provide an integrated shell experience that allows you to easily run terminal commands while using Go for complicated logic. This allows you to write concise, elegant, and readable shell code that runs quickly on all platforms and transpiles to Go.


# Special shell syntax

* set <var> <value> (space delimited for all exec)
* ; for multiple per line, resolved by parser
* & at end for background job; builtin jobs and kill commands (randy will do)
* standard output redirect: > and >& (and |, |& if needed)
* always stop on err but have a special syntax for ignore error -- how about: `[ ]` around expression?  that is a standard optional kind of thing.  then you could do:

```sh
cd some; [mkdir sub]; cd sub
```



# Syntax: Go vs. Exec

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



