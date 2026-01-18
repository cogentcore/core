// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//! Error types for Cogent Core.


/// A result type alias for Cogent Core operations.
pub type Result<T> = std::result::Result<T, Error>;

/// The main error type for Cogent Core.
#[derive(Debug, thiserror::Error)]
pub enum Error {
    /// An I/O error occurred.
    #[error("I/O error: {0}")]
    Io(#[from] std::io::Error),

    /// A serialization/deserialization error occurred.
    #[error("Serialization error: {0}")]
    Serialization(String),

    /// A rendering error occurred.
    #[error("Rendering error: {0}")]
    Rendering(String),

    /// A widget error occurred.
    #[error("Widget error: {0}")]
    Widget(String),

    /// A style error occurred.
    #[error("Style error: {0}")]
    Style(String),

    /// A generic error with a message.
    #[error("{0}")]
    Message(String),
}

impl Error {
    /// Creates a new error with a message.
    pub fn new(msg: impl Into<String>) -> Self {
        Self::Message(msg.into())
    }
}

