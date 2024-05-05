// Code generated by "core generate -add-types -add-funcs"; DO NOT EDIT.

package main

import (
	"cogentcore.org/core/types"
)

var _ = types.AddType(&types.Type{Name: "main.Config", IDName: "config", Doc: "Config is the configuration information for the cosh cli.", Directives: []types.Directive{{Tool: "go", Directive: "generate", Args: []string{"core", "generate", "-add-types", "-add-funcs"}}}, Fields: []types.Field{{Name: "File", Doc: "File is the file to run/compile."}}})

var _ = types.AddFunc(&types.Func{Name: "main.Run", Doc: "Run runs the specified cosh file. If no file is specified,\nit runs an interactive shell that allows the user to input cosh.", Directives: []types.Directive{{Tool: "cli", Directive: "cmd", Args: []string{"-root"}}}, Args: []string{"c"}, Returns: []string{"error"}})

var _ = types.AddFunc(&types.Func{Name: "main.Interactive", Doc: "Interactive runs an interactive shell that allows the user to input cosh.", Args: []string{"c"}, Returns: []string{"error"}})
