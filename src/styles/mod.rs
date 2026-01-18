// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//! Styling system for widgets.

use crate::colors::Color;
use crate::events::Size;

/// A style configuration for a widget.
#[derive(Debug, Clone)]
pub struct Style {
    /// The background color.
    pub background: Option<Color>,
    /// The foreground (text) color.
    pub foreground: Option<Color>,
    /// The border color.
    pub border_color: Option<Color>,
    /// The border width in pixels.
    pub border_width: f32,
    /// The padding around the content.
    pub padding: Padding,
    /// The margin around the widget.
    pub margin: Margin,
    /// The minimum size.
    pub min_size: Size,
    /// The maximum size.
    pub max_size: Size,
    /// The preferred size.
    pub preferred_size: Option<Size>,
    /// Whether the widget should grow to fill available space.
    pub grow_x: f32,
    pub grow_y: f32,
    /// The opacity (0.0 to 1.0).
    pub opacity: f32,
    /// Whether the widget is visible.
    pub visible: bool,
    /// Whether the widget is enabled.
    pub enabled: bool,
}

impl Default for Style {
    fn default() -> Self {
        Self {
            background: None,
            foreground: None,
            border_color: None,
            border_width: 0.0,
            padding: Padding::default(),
            margin: Margin::default(),
            min_size: Size::zero(),
            max_size: Size::new(u32::MAX, u32::MAX),
            preferred_size: None,
            grow_x: 0.0,
            grow_y: 0.0,
            opacity: 1.0,
            visible: true,
            enabled: true,
        }
    }
}

/// Padding around content.
#[derive(Debug, Clone, Copy, PartialEq)]
pub struct Padding {
    pub top: f32,
    pub right: f32,
    pub bottom: f32,
    pub left: f32,
}

impl Default for Padding {
    fn default() -> Self {
        Self {
            top: 0.0,
            right: 0.0,
            bottom: 0.0,
            left: 0.0,
        }
    }
}

impl Padding {
    /// Creates uniform padding.
    pub fn uniform(value: f32) -> Self {
        Self {
            top: value,
            right: value,
            bottom: value,
            left: value,
        }
    }

    /// Creates padding with different horizontal and vertical values.
    pub fn symmetric(horizontal: f32, vertical: f32) -> Self {
        Self {
            top: vertical,
            right: horizontal,
            bottom: vertical,
            left: horizontal,
        }
    }
}

/// Margin around a widget.
#[derive(Debug, Clone, Copy, PartialEq)]
pub struct Margin {
    pub top: f32,
    pub right: f32,
    pub bottom: f32,
    pub left: f32,
}

impl Default for Margin {
    fn default() -> Self {
        Self {
            top: 0.0,
            right: 0.0,
            bottom: 0.0,
            left: 0.0,
        }
    }
}

impl Margin {
    /// Creates uniform margin.
    pub fn uniform(value: f32) -> Self {
        Self {
            top: value,
            right: value,
            bottom: value,
            left: value,
        }
    }

    /// Creates margin with different horizontal and vertical values.
    pub fn symmetric(horizontal: f32, vertical: f32) -> Self {
        Self {
            top: vertical,
            right: horizontal,
            bottom: vertical,
            left: horizontal,
        }
    }
}
