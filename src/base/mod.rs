// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//! Base utility modules providing foundational functionality.

pub mod errors;
pub mod stack;

pub use errors::{Error, Result};
pub use stack::Stack;
