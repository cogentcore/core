# Cogent Shell (cosh)


# Syntax: Go vs. Exec

The critical extension from standard Go syntax is for lines that are processed by the `Exec` functions, used for running arbitrary programs on the user's executable path.  Here are the rules (word = IDENT token):

* Backticks "``" anywhere:  Exec.  Returns a `string`.
* Within Exec, `{}`: Go
* Line starts with `Go` Keyword: Go
* Line is one word: Exec
* Line starts with `.`: Exec
* Line starts with `word word`: Exec
* Line starts with `word {`: Exec
* Otherwise: Go

