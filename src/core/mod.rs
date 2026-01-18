// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//! Core widget system and scene management.

use crate::base::Result;
use crate::colors::Color;
use crate::events::{Event, Point, Rect, Size};
use crate::styles::Style;
use crate::tree::{Node, NodeBase};
use std::sync::{Arc, Mutex};


/// A widget in the UI hierarchy.
pub trait Widget: Node + Send + Sync {
    /// Returns the widget base for this widget.
    fn widget_base(&self) -> &WidgetBase;

    /// Returns a mutable reference to the widget base.
    fn widget_base_mut(&mut self) -> &mut WidgetBase;

    /// Updates the style properties of the widget.
    fn style(&mut self) {
        // Default implementation does nothing
    }

    /// Calculates the size requirements of the widget (bottom-up).
    fn size_up(&mut self) {
        // Default implementation does nothing
    }

    /// Allocates size to the widget (top-down).
    /// Returns true if the size changed.
    fn size_down(&mut self, _iter: usize) -> bool {
        false
    }

    /// Finalizes the size calculation (bottom-up).
    fn size_final(&mut self) {
        // Default implementation does nothing
    }

    /// Positions the widget within its parent.
    fn position(&mut self) {
        // Default implementation does nothing
    }

    /// Applies scene-based absolute positions.
    fn apply_scene_pos(&mut self) {
        // Default implementation does nothing
    }

    /// Renders the widget.
    fn render(&mut self, _renderer: &mut dyn Renderer) -> Result<()> {
        Ok(())
    }

    /// Handles an event.
    /// Returns true if the event was handled.
    fn handle_event(&mut self, _event: &Event) -> bool {
        false
    }
}

/// Base implementation for widgets.
pub struct WidgetBase {
    node_base: NodeBase,
    style: Style,
    actual_size: Size,
    alloc_size: Size,
    position: Point,
    bbox: Rect,
    tooltip: Option<String>,
    on_click: Option<Box<dyn Fn(&Event) + Send + Sync>>,
}

impl std::fmt::Debug for WidgetBase {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("WidgetBase")
            .field("name", &self.node_base.name())
            .field("style", &self.style)
            .field("actual_size", &self.actual_size)
            .field("alloc_size", &self.alloc_size)
            .field("position", &self.position)
            .field("bbox", &self.bbox)
            .field("tooltip", &self.tooltip)
            .field("on_click", &self.on_click.is_some())
            .finish()
    }
}

impl WidgetBase {
    /// Creates a new widget base.
    pub fn new(name: impl Into<String>) -> Self {
        Self {
            node_base: NodeBase::new(name),
            style: Style::default(),
            actual_size: Size::zero(),
            alloc_size: Size::zero(),
            position: Point::zero(),
            bbox: Rect::new(0, 0, 0, 0),
            tooltip: None,
            on_click: None,
        }
    }

    /// Returns the style.
    pub fn style(&self) -> &Style {
        &self.style
    }

    /// Returns a mutable reference to the style.
    pub fn style_mut(&mut self) -> &mut Style {
        &mut self.style
    }

    /// Sets the style.
    pub fn set_style(&mut self, style: Style) {
        self.style = style;
    }

    /// Returns the actual size.
    pub fn actual_size(&self) -> Size {
        self.actual_size
    }

    /// Sets the actual size.
    pub fn set_actual_size(&mut self, size: Size) {
        self.actual_size = size;
    }

    /// Returns the allocated size.
    pub fn alloc_size(&self) -> Size {
        self.alloc_size
    }

    /// Sets the allocated size.
    pub fn set_alloc_size(&mut self, size: Size) {
        self.alloc_size = size;
    }

    /// Returns the position.
    pub fn position(&self) -> Point {
        self.position
    }

    /// Sets the position.
    pub fn set_position(&mut self, position: Point) {
        self.position = position;
    }

    /// Returns the bounding box.
    pub fn bbox(&self) -> Rect {
        self.bbox
    }

    /// Sets the bounding box.
    pub fn set_bbox(&mut self, bbox: Rect) {
        self.bbox = bbox;
    }

    /// Returns the tooltip text.
    pub fn tooltip(&self) -> Option<&str> {
        self.tooltip.as_deref()
    }

    /// Sets the tooltip text.
    pub fn set_tooltip(&mut self, tooltip: impl Into<Option<String>>) {
        self.tooltip = tooltip.into();
    }

    /// Sets the click handler.
    pub fn set_on_click<F>(&mut self, handler: F)
    where
        F: Fn(&Event) + Send + Sync + 'static,
    {
        self.on_click = Some(Box::new(handler));
    }

    /// Calls the click handler if set.
    pub fn call_on_click(&self, event: &Event) {
        if let Some(handler) = &self.on_click {
            handler(event);
        }
    }

    /// Returns the node base.
    pub fn node_base(&self) -> &NodeBase {
        &self.node_base
    }

    /// Returns a mutable reference to the node base.
    pub fn node_base_mut(&mut self) -> &mut NodeBase {
        &mut self.node_base
    }
}

impl Node for WidgetBase {
    fn as_any(&self) -> &dyn std::any::Any {
        self
    }

    fn as_any_mut(&mut self) -> &mut dyn std::any::Any {
        self
    }
}

/// A renderer for drawing widgets.
pub trait Renderer: Send + Sync {
    /// Draws a rectangle.
    fn draw_rect(&mut self, rect: Rect, color: Color) -> Result<()>;

    /// Draws text at the given position.
    fn draw_text(&mut self, text: &str, position: Point, color: Color) -> Result<()>;

    /// Clears the render target.
    fn clear(&mut self, color: Color) -> Result<()>;
}

/// A scene that contains widgets and manages rendering.
pub struct Scene {
    root: Arc<Mutex<dyn Widget>>,
    renderer: Arc<Mutex<dyn Renderer>>,
}

impl std::fmt::Debug for Scene {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("Scene").finish()
    }
}

impl Scene {
    /// Creates a new scene.
    pub fn new(root: Arc<Mutex<dyn Widget>>, renderer: Arc<Mutex<dyn Renderer>>) -> Self {
        Self { root, renderer }
    }

    /// Returns a reference to the root widget.
    pub fn root(&self) -> &Arc<Mutex<dyn Widget>> {
        &self.root
    }

    /// Handles an event by propagating it through the widget tree.
    pub fn handle_event(&self, event: &Event) -> Result<bool> {
        let mut root = self.root.lock().unwrap();
        Ok(root.handle_event(event))
    }

    /// Renders the scene.
    pub fn render(&self) -> Result<()> {
        let mut renderer = self.renderer.lock().unwrap();
        let mut root = self.root.lock().unwrap();
        root.render(&mut *renderer)?;
        Ok(())
    }
}
