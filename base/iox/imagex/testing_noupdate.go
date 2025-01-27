// Copyright 2023 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !update

package imagex

import "os"

var updateTestImages = os.Getenv("CORE_UPDATE_TESTDATA") == "true"
