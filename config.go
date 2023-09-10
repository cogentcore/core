// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config

// Config determines the generator options
type Config struct {
	TypeReg  bool
	FuncReg  bool
	VarReg   bool
	ConstReg bool

	// if true, generate instances of each type
	Instances bool

	// if true, generate a global type variable of the form: TypeTypeName
	TypeVar bool
}
