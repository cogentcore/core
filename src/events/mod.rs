// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//! Event system for handling user interactions and system events.

use std::fmt;

/// A point in 2D space.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub struct Point {
    pub x: i32,
    pub y: i32,
}

impl Point {
    /// Creates a new point.
    pub fn new(x: i32, y: i32) -> Self {
        Self { x, y }
    }

    /// Returns the zero point.
    pub fn zero() -> Self {
        Self { x: 0, y: 0 }
    }
}

/// A size in 2D space.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub struct Size {
    pub width: u32,
    pub height: u32,
}

impl Size {
    /// Creates a new size.
    pub fn new(width: u32, height: u32) -> Self {
        Self { width, height }
    }

    /// Returns the zero size.
    pub fn zero() -> Self {
        Self {
            width: 0,
            height: 0,
        }
    }
}

/// A rectangle in 2D space.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Rect {
    pub x: i32,
    pub y: i32,
    pub width: u32,
    pub height: u32,
}

impl Rect {
    /// Creates a new rectangle.
    pub fn new(x: i32, y: i32, width: u32, height: u32) -> Self {
        Self {
            x,
            y,
            width,
            height,
        }
    }

    /// Creates a rectangle from a point and size.
    pub fn from_point_size(point: Point, size: Size) -> Self {
        Self {
            x: point.x,
            y: point.y,
            width: size.width,
            height: size.height,
        }
    }

    /// Returns the top-left point of the rectangle.
    pub fn point(&self) -> Point {
        Point::new(self.x, self.y)
    }

    /// Returns the size of the rectangle.
    pub fn size(&self) -> Size {
        Size::new(self.width, self.height)
    }

    /// Returns true if the point is inside the rectangle.
    pub fn contains(&self, point: Point) -> bool {
        point.x >= self.x
            && point.x < self.x + self.width as i32
            && point.y >= self.y
            && point.y < self.y + self.height as i32
    }
}

/// Types of events that can occur.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub enum EventType {
    /// A mouse button was pressed.
    MousePress,
    /// A mouse button was released.
    MouseRelease,
    /// The mouse moved.
    MouseMove,
    /// A mouse button was clicked.
    Click,
    /// A key was pressed.
    KeyPress,
    /// A key was released.
    KeyRelease,
    /// A character was typed.
    Char,
    /// The widget gained focus.
    Focus,
    /// The widget lost focus.
    Blur,
    /// The widget was scrolled.
    Scroll,
    /// A custom event.
    Custom(&'static str),
}

/// An event that occurred in the system.
#[derive(Debug, Clone)]
pub struct Event {
    /// The type of event.
    pub event_type: EventType,
    /// The position of the event (for mouse/keyboard events).
    pub position: Point,
    /// Additional event data.
    pub data: EventData,
}

/// Additional data associated with an event.
#[derive(Debug, Clone)]
pub enum EventData {
    /// No additional data.
    None,
    /// Mouse button information.
    Mouse { button: MouseButton },
    /// Keyboard key information.
    Key { key: Key, modifiers: Modifiers },
    /// Character input.
    Char { ch: char },
    /// Scroll delta.
    Scroll { delta_x: f32, delta_y: f32 },
    /// Custom data.
    Custom(String),
}

/// Mouse button types.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub enum MouseButton {
    Left,
    Right,
    Middle,
    Other(u8),
}

/// Keyboard key types.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub enum Key {
    /// A printable character key.
    Char(char),
    /// Escape key.
    Escape,
    /// Enter/Return key.
    Enter,
    /// Tab key.
    Tab,
    /// Backspace key.
    Backspace,
    /// Delete key.
    Delete,
    /// Arrow keys.
    ArrowUp,
    ArrowDown,
    ArrowLeft,
    ArrowRight,
    /// Function keys.
    F1,
    F2,
    F3,
    F4,
    F5,
    F6,
    F7,
    F8,
    F9,
    F10,
    F11,
    F12,
    /// Other key.
    Other(u32),
}

/// Keyboard modifiers.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Default)]
pub struct Modifiers {
    pub shift: bool,
    pub ctrl: bool,
    pub alt: bool,
    pub meta: bool,
}

impl Event {
    /// Creates a new event.
    pub fn new(event_type: EventType, position: Point, data: EventData) -> Self {
        Self {
            event_type,
            position,
            data,
        }
    }

    /// Creates a click event.
    pub fn click(position: Point, button: MouseButton) -> Self {
        Self {
            event_type: EventType::Click,
            position,
            data: EventData::Mouse { button },
        }
    }

    /// Creates a mouse press event.
    pub fn mouse_press(position: Point, button: MouseButton) -> Self {
        Self {
            event_type: EventType::MousePress,
            position,
            data: EventData::Mouse { button },
        }
    }

    /// Creates a key press event.
    pub fn key_press(key: Key, modifiers: Modifiers) -> Self {
        Self {
            event_type: EventType::KeyPress,
            position: Point::zero(),
            data: EventData::Key { key, modifiers },
        }
    }
}

impl fmt::Display for EventType {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            EventType::MousePress => write!(f, "MousePress"),
            EventType::MouseRelease => write!(f, "MouseRelease"),
            EventType::MouseMove => write!(f, "MouseMove"),
            EventType::Click => write!(f, "Click"),
            EventType::KeyPress => write!(f, "KeyPress"),
            EventType::KeyRelease => write!(f, "KeyRelease"),
            EventType::Char => write!(f, "Char"),
            EventType::Focus => write!(f, "Focus"),
            EventType::Blur => write!(f, "Blur"),
            EventType::Scroll => write!(f, "Scroll"),
            EventType::Custom(s) => write!(f, "Custom({})", s),
        }
    }
}
