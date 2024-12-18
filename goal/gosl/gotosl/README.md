# Implementational details of Go to SL translation process

Overall, there are three main steps:

1. Translate all the `.go` files in the current package, and all the files they `//gosl:import`, into corresponding `.wgsl` files, and put those in `shaders/imports`.  All these files will be pasted into the generated primary kernel files, that go in `shaders`, and are saved to disk for reference. All the key kernel, system, variable info is extracted from the package .go file directives during this phase.

2. Generate the `main` kernel `.wgsl` files, for each kernel function, which: a) declare the global buffer variables; b) include everything from imports; c) define the `main` function entry point. Each resulting file is pre-processed by `naga` to ensure it compiles, and to remove dead code not needed for this particular shader.

3. Generate the `gosl.go` file in the package directory, which contains generated Go code for configuring the gpu compute systems according to the vars.

## Go to SL translation

1. `files.go`: Get a list of all the .go files in the current directory that have a `//gosl:` tag (`ProjectFiles`) and all the `//gosl:import` package files that those files import, recursively.

2. `extract.go`: Extract the `//gosl:start` -> `end` regions from all the package and imported filees.

3. Save all these files as new `.go` files in `shaders/imports`. We manually append a simple go "main" package header with basic gosl imports for each file, which allows the go compiler to process them properly. This is then removed in the next step.

4. `translate.go:` Run `TranslateDir` on shaders/imports using the "golang.org/x/tools/go/packages" `Load` function, which gets `ast` and type information for all that code. Run the resulting `ast.File` for each file through the modified version of the Go stdlib `src/go/printer` code (`printer.go`, `nodes.go`, `gobuild.go`, `comment.go`), which prints out WGSL instead of Go code from the underlying `ast` representation of the Go files. This is what does the actual translation.

5. `sledits.go:` Do various forms of post-processing text replacement cleanup on the generated WGSL files, in `SLEdits` function. 


