// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gtigen

import "text/template"

// Config contains the configuration information
// used by gtigen
type Config struct {

	// [def: .] the source directory to run gtigen on (can be set to multiple through paths like ./...)
	Dir string `def:"." posarg:"0" required:"-" desc:"the source directory to run gtigen on (can be set to multiple through paths like ./...)"`

	// [def: gtigen.go] the output file location relative to the package on which gtigen is being called
	Output string `def:"gtigen.go" desc:"the output file location relative to the package on which gtigen is being called"`

	// whether to add types to gtigen by default
	AddTypes bool `desc:"whether to add types to gtigen by default"`

	// whether to add methods to gtigen by default
	AddMethods bool `desc:"whether to add methods to gtigen by default"`

	// whether to add functions to gtigen by default
	AddFuncs bool `desc:"whether to add functions to gtigen by default"`

	// a map of configs keyed by fully-qualified interface type names; if a type implements the interface, the config will be applied to it
	InterfaceConfigs map[string]*Config `desc:"a map of configs keyed by fully-qualified interface type names; if a type implements the interface, the config will be applied to it"`

	// whether to generate an instance of the type(s)
	Instance bool `desc:"whether to generate an instance of the type(s)"`

	// whether to generate a global type variable of the form 'TypeNameType'
	TypeVar bool `desc:"whether to generate a global type variable of the form 'TypeNameType'"`

	// TODO: should this be called TypeTemplates and should there be a Func/Method Templates?

	// a slice of templates to execute on each type being added; the template data is of the type gtigen.Type
	Templates []*template.Template `desc:"a slice of templates to execute on each type being added; the template data is of the type gtigen.Type"`
}
