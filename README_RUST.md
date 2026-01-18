# Cogent Core - Rust Implementation

This is a Rust rewrite of the Cogent Core UI framework. The original Go implementation provides a cross-platform framework for building 2D and 3D applications, and this Rust version aims to provide similar functionality with Rust's safety guarantees and performance characteristics.

## Project Structure

```
src/
├── app/          - Application lifecycle and management
├── base/         - Base utilities (errors, stack, etc.)
├── colors/       - Color types and utilities
├── core/         - Core widget system and scene management
├── events/       - Event system for user interactions
├── styles/       - Styling system for widgets
├── tree/         - Tree structure for widget hierarchy
└── widgets/      - Widget implementations (Button, etc.)
```

## Features

- **Type-safe widget system**: Trait-based design with compile-time guarantees
- **Event handling**: Comprehensive event system for user interactions
- **Styling**: Flexible styling system with support for colors, padding, margins, etc.
- **Renderer abstraction**: Pluggable renderer system for different backends
- **Tree-based hierarchy**: Widgets organized in a tree structure

## Example Usage

See `examples/hello_world.rs` for a simple example of creating a button widget.

## Building

```bash
cargo build
```

## Running Examples

```bash
cargo run --example hello_world
```

## Status

This is an initial implementation covering:
- ✅ Base utilities (stack, errors)
- ✅ Core widget system with traits
- ✅ Event system
- ✅ Styling system
- ✅ Button widget
- ✅ Scene and app management
- ✅ Example application

Future work includes:
- More widget types (Text, Frame, List, etc.)
- Layout system
- Graphics rendering backend (wgpu integration)
- Window management
- More comprehensive styling options
- Animation system
- 3D support

## License

BSD-3-Clause (same as the original Go implementation)
