// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//! Button widget implementation.

use crate::base::Result;
use crate::colors::Color;
use crate::core::{Renderer, Widget, WidgetBase};
use crate::events::{Event, EventType, MouseButton, Point};
use crate::tree::Node;

/// A button widget that can be clicked.
#[derive(Debug)]
pub struct Button {
    base: WidgetBase,
    text: String,
    icon: Option<String>,
    button_type: ButtonType,
}

/// Types of buttons.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ButtonType {
    /// A filled button with a contrasting background color.
    Filled,
    /// A filled button with a lighter background color.
    Tonal,
    /// An elevated button with a shadow.
    Elevated,
    /// An outlined button.
    Outlined,
    /// A text-only button with no border or background.
    Text,
}

impl Button {
    /// Creates a new button with the given text.
    pub fn new(text: impl Into<String>) -> Self {
        Self {
            base: WidgetBase::new("button"),
            text: text.into(),
            icon: None,
            button_type: ButtonType::Filled,
        }
    }

    /// Sets the button type.
    pub fn set_type(&mut self, button_type: ButtonType) {
        self.button_type = button_type;
    }

    /// Sets the button text.
    pub fn set_text(&mut self, text: impl Into<String>) {
        self.text = text.into();
    }

    /// Returns the button text.
    pub fn text(&self) -> &str {
        &self.text
    }

    /// Sets the icon.
    pub fn set_icon(&mut self, icon: impl Into<Option<String>>) {
        self.icon = icon.into();
    }

    /// Returns the icon.
    pub fn icon(&self) -> Option<&str> {
        self.icon.as_deref()
    }
}

impl Widget for Button {
    fn widget_base(&self) -> &WidgetBase {
        &self.base
    }

    fn widget_base_mut(&mut self) -> &mut WidgetBase {
        &mut self.base
    }

    fn style(&mut self) {
        // Apply button-specific styling based on type
        let mut style = self.base.style().clone();
        match self.button_type {
            ButtonType::Filled => {
                style.background = Some(Color::from_hex("#4285F4").unwrap_or(Color::BLUE));
                style.foreground = Some(Color::WHITE);
                style.padding = crate::styles::Padding::uniform(12.0);
            }
            ButtonType::Tonal => {
                style.background = Some(Color::from_hex("#E8F0FE").unwrap_or(Color::BLUE));
                style.foreground = Some(Color::from_hex("#1967D2").unwrap_or(Color::BLUE));
                style.padding = crate::styles::Padding::uniform(12.0);
            }
            ButtonType::Elevated => {
                style.background = Some(Color::WHITE);
                style.foreground = Some(Color::from_hex("#1967D2").unwrap_or(Color::BLUE));
                style.padding = crate::styles::Padding::uniform(12.0);
            }
            ButtonType::Outlined => {
                style.background = None;
                style.border_color = Some(Color::from_hex("#4285F4").unwrap_or(Color::BLUE));
                style.border_width = 1.0;
                style.foreground = Some(Color::from_hex("#4285F4").unwrap_or(Color::BLUE));
                style.padding = crate::styles::Padding::uniform(12.0);
            }
            ButtonType::Text => {
                style.background = None;
                style.foreground = Some(Color::from_hex("#4285F4").unwrap_or(Color::BLUE));
                style.padding = crate::styles::Padding::uniform(8.0);
            }
        }
        self.base.set_style(style);
    }

    fn size_up(&mut self) {
        // Calculate size based on text content
        // For now, use a simple heuristic
        let text_width = self.text.len() as u32 * 8 + 24; // Rough estimate
        let height = 40;
        self.base.set_actual_size(crate::events::Size::new(text_width, height));
    }

    fn render(&mut self, renderer: &mut dyn Renderer) -> Result<()> {
        let style = self.base.style();
        let bbox = self.base.bbox();

        // Draw background
        if let Some(bg) = style.background {
            renderer.draw_rect(bbox, bg)?;
        }

        // Draw border
        if style.border_width > 0.0 {
            if let Some(border_color) = style.border_color {
                renderer.draw_rect(bbox, border_color)?;
            }
        }

        // Draw text
        if let Some(fg) = style.foreground {
            let text_pos = Point::new(
                bbox.x + bbox.width as i32 / 2 - (self.text.len() as i32 * 4),
                bbox.y + bbox.height as i32 / 2 + 4,
            );
            renderer.draw_text(&self.text, text_pos, fg)?;
        }

        Ok(())
    }

    fn handle_event(&mut self, event: &Event) -> bool {
        match event.event_type {
            EventType::Click => {
                if let crate::events::EventData::Mouse { button: MouseButton::Left } = event.data {
                    self.base.call_on_click(event);
                    return true;
                }
            }
            _ => {}
        }
        false
    }
}

impl Node for Button {
    fn as_any(&self) -> &dyn std::any::Any {
        self
    }

    fn as_any_mut(&mut self) -> &mut dyn std::any::Any {
        self
    }
}
