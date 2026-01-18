// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//! A simple hello world example demonstrating basic widget usage.

use cogent_core::core::{Renderer, Scene, Widget, WidgetBase};
use cogent_core::events::{Point, Rect, Size};
use cogent_core::widgets::Button;
use cogent_core::{base::Result, colors::Color};
use softbuffer::Surface;
use std::num::NonZeroU32;
use std::sync::{Arc, Mutex};
use winit::application::ApplicationHandler;
use winit::dpi::LogicalSize;
use winit::event::WindowEvent;
use winit::event_loop::{ActiveEventLoop, EventLoop};
use winit::window::Window;

/// A simple renderer implementation that renders to a pixel buffer.
struct SimpleRenderer {
    width: u32,
    height: u32,
    pixels: Vec<u32>,
}

impl SimpleRenderer {
    fn new(width: u32, height: u32) -> Self {
        Self {
            width,
            height,
            pixels: vec![0; (width * height) as usize],
        }
    }

    fn get_pixels(&self) -> &[u32] {
        &self.pixels
    }

    fn clear(&mut self) {
        self.pixels.fill(0xFFF5F5F5); // Light gray background in ARGB format
    }
}

impl Renderer for SimpleRenderer {
    fn draw_rect(&mut self, rect: Rect, color: Color) -> Result<()> {
        // Convert RGBA to ARGB format (0xAARRGGBB) for softbuffer
        let color_u32 = ((color.a as u32) << 24)
            | ((color.r as u32) << 16)
            | ((color.g as u32) << 8)
            | (color.b as u32);
        
        for y in rect.y..(rect.y + rect.height as i32) {
            for x in rect.x..(rect.x + rect.width as i32) {
                if x >= 0 && x < self.width as i32 && y >= 0 && y < self.height as i32 {
                    let idx = (y as u32 * self.width + x as u32) as usize;
                    self.pixels[idx] = color_u32;
                }
            }
        }
        Ok(())
    }

    fn draw_text(&mut self, _text: &str, _position: Point, _color: Color) -> Result<()> {
        // Simple renderer doesn't implement text rendering yet
        Ok(())
    }

    fn clear(&mut self, color: Color) -> Result<()> {
        // Convert RGBA to ARGB format (0xAARRGGBB) for softbuffer
        let color_u32 = ((color.a as u32) << 24)
            | ((color.r as u32) << 16)
            | ((color.g as u32) << 8)
            | (color.b as u32);
        self.pixels.fill(color_u32);
        Ok(())
    }
}

/// A simple root widget that contains a button.
struct RootWidget {
    base: WidgetBase,
    button: Arc<Mutex<Button>>,
}

impl RootWidget {
    fn new() -> Self {
        let mut button = Button::new("Click Me!");
        button.widget_base_mut().set_on_click(|_event| {
            println!("Button clicked!");
        });
        // Set button position and size
        button.widget_base_mut().set_alloc_size(Size::new(200, 50));
        button.widget_base_mut().set_bbox(Rect::new(300, 275, 200, 50));
        button.style();
        button.size_up();

        Self {
            base: WidgetBase::new("root"),
            button: Arc::new(Mutex::new(button)),
        }
    }
}

impl Widget for RootWidget {
    fn widget_base(&self) -> &WidgetBase {
        &self.base
    }

    fn widget_base_mut(&mut self) -> &mut WidgetBase {
        &mut self.base
    }

    fn render(&mut self, renderer: &mut dyn Renderer) -> Result<()> {
        // Render background
        let bbox = self.base.bbox();
        renderer.draw_rect(bbox, Color::from_hex("#F5F5F5").unwrap_or(Color::WHITE))?;

        // Render button
        let mut button = self.button.lock().unwrap();
        button.render(renderer)?;

        Ok(())
    }
}

impl cogent_core::tree::Node for RootWidget {
    fn as_any(&self) -> &dyn std::any::Any {
        self
    }

    fn as_any_mut(&mut self) -> &mut dyn std::any::Any {
        self
    }
}

struct HelloWorldApp {
    window: Option<Arc<Window>>,
    renderer: Arc<Mutex<SimpleRenderer>>,
    scene: Arc<Mutex<Scene>>,
    surface: Option<Surface<Arc<Window>, Arc<Window>>>,
}

impl ApplicationHandler for HelloWorldApp {
    fn resumed(&mut self, event_loop: &ActiveEventLoop) {
        let window_attributes = Window::default_attributes()
            .with_title("Cogent Core - Hello World")
            .with_inner_size(LogicalSize::new(800, 600));

        let window = Arc::new(event_loop.create_window(window_attributes).unwrap());
        self.window = Some(window.clone());
        
        // Create softbuffer surface - softbuffer 0.4 API
        // Note: This is a simplified version - full integration would need proper context handling
        // For now, we'll render directly without softbuffer to get the window working
        // self.surface = Some(surface);

        // Initialize widget sizes and positions
        let scene = self.scene.lock().unwrap();
        let mut root = scene.root().lock().unwrap();
        root.widget_base_mut().set_alloc_size(Size::new(800, 600));
        root.widget_base_mut().set_bbox(Rect::new(0, 0, 800, 600));
        root.style();
        root.size_up();
        root.size_down(0);
        root.size_final();
        root.position();
        root.apply_scene_pos();
        
        // Request initial redraw
        window.request_redraw();
    }

    fn window_event(&mut self, event_loop: &ActiveEventLoop, _window_id: winit::window::WindowId, event: WindowEvent) {
        match event {
            WindowEvent::CloseRequested => {
                event_loop.exit();
            }
            WindowEvent::RedrawRequested => {
                // Clear and render the scene
                {
                    let mut renderer = self.renderer.lock().unwrap();
                    renderer.clear();
                    drop(renderer);
                }
                
                // Render scene
                {
                    let scene = self.scene.lock().unwrap();
                    let mut root = scene.root().lock().unwrap();
                    let mut renderer = self.renderer.lock().unwrap();
                    if let Err(e) = root.render(&mut *renderer) {
                        eprintln!("Render error: {}", e);
                    }
                }

                // For now, just render to our buffer - window will show but pixels won't be displayed yet
                // Full softbuffer integration requires proper context setup which needs more work
                // This at least opens the window successfully

                // Request another redraw
                if let Some(window) = &self.window {
                    window.request_redraw();
                }
            }
            _ => {}
        }
    }
}

fn main() -> Result<()> {
    println!("Cogent Core Rust - Hello World Example");

    // Create renderer
    let renderer = Arc::new(Mutex::new(SimpleRenderer::new(800, 600)));

    // Create root widget
    let root = Arc::new(Mutex::new(RootWidget::new()));

    // Create scene
    let scene = Arc::new(Mutex::new(Scene::new(root, renderer.clone())));

    // Create event loop and app handler
    let event_loop = EventLoop::new().unwrap();
    let mut hello_app = HelloWorldApp {
        window: None,
        renderer,
        scene,
        surface: None,
    };

    event_loop.set_control_flow(winit::event_loop::ControlFlow::Wait);
    event_loop.run_app(&mut hello_app).unwrap();

    Ok(())
}
