// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//! Cogent Core is a free and open source framework for building powerful, fast, elegant
//! 2D and 3D apps that run on macOS, Windows, Linux, iOS, Android, and web with a
//! single Rust codebase, allowing you to Code Once, Run Everywhere (Core).

#![warn(missing_docs)]
#![warn(clippy::all)]

pub mod app;
pub mod base;
pub mod colors;
pub mod core;
pub mod events;
pub mod styles;
pub mod tree;
pub mod widgets;

pub use app::App;
pub use core::{Renderer, Scene, Widget, WidgetBase};
pub use widgets::Button;
