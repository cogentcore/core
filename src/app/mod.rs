// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//! Application management and lifecycle.

use crate::core::Scene;
use crate::events::{Event, Size};
use std::sync::{Arc, Mutex};

/// The main application instance.
pub struct App {
    scene: Option<Arc<Mutex<Scene>>>,
    size: Size,
}

impl App {
    /// Creates a new application.
    pub fn new() -> Self {
        Self {
            scene: None,
            size: Size::new(800, 600),
        }
    }

    /// Sets the scene for the application.
    pub fn set_scene(&mut self, scene: Arc<Mutex<Scene>>) {
        self.scene = Some(scene);
    }

    /// Returns the current scene.
    pub fn scene(&self) -> Option<&Arc<Mutex<Scene>>> {
        self.scene.as_ref()
    }

    /// Sets the window size.
    pub fn set_size(&mut self, size: Size) {
        self.size = size;
    }

    /// Returns the window size.
    pub fn size(&self) -> Size {
        self.size
    }

    /// Handles an event.
    pub fn handle_event(&self, event: &Event) -> crate::base::Result<bool> {
        if let Some(scene) = &self.scene {
            scene.lock().unwrap().handle_event(event)
        } else {
            Ok(false)
        }
    }

    /// Renders the application.
    pub fn render(&self) -> crate::base::Result<()> {
        if let Some(scene) = &self.scene {
            scene.lock().unwrap().render()
        } else {
            Ok(())
        }
    }
}

impl Default for App {
    fn default() -> Self {
        Self::new()
    }
}

/// The global application instance.
static THE_APP: Mutex<Option<Arc<Mutex<App>>>> = Mutex::new(None);

/// Returns the global application instance.
pub fn the_app() -> Arc<Mutex<App>> {
    let mut app = THE_APP.lock().unwrap();
    if app.is_none() {
        *app = Some(Arc::new(Mutex::new(App::new())));
    }
    app.as_ref().unwrap().clone()
}
