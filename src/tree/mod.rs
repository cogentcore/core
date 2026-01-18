// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//! Tree structure for widget hierarchy.

use std::any::{Any, TypeId};
use std::sync::{Arc, Weak};

/// A node in the widget tree.
pub trait Node: Any + Send + Sync {
    /// Returns the node as a type-erased reference.
    fn as_any(&self) -> &dyn Any;

    /// Returns the node as a mutable type-erased reference.
    fn as_any_mut(&mut self) -> &mut dyn Any;

    /// Returns the type ID of this node.
    fn type_id(&self) -> TypeId {
        TypeId::of::<Self>()
    }
}

/// Base implementation for tree nodes.
pub struct NodeBase {
    /// The parent node, if any.
    parent: Option<Weak<dyn Node>>,
    /// Child nodes.
    children: Vec<Arc<dyn Node + 'static>>,
    /// The name of this node.
    name: String,
}

impl std::fmt::Debug for NodeBase {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("NodeBase")
            .field("name", &self.name)
            .field("num_children", &self.children.len())
            .field("has_parent", &self.parent.is_some())
            .finish()
    }
}

impl NodeBase {
    /// Creates a new node base with the given name.
    pub fn new(name: impl Into<String>) -> Self {
        Self {
            parent: None,
            children: Vec::new(),
            name: name.into(),
        }
    }

    /// Returns the name of this node.
    pub fn name(&self) -> &str {
        &self.name
    }

    /// Sets the name of this node.
    pub fn set_name(&mut self, name: impl Into<String>) {
        self.name = name.into();
    }

    /// Returns the parent node, if any.
    pub fn parent(&self) -> Option<Arc<dyn Node>> {
        self.parent.as_ref()?.upgrade()
    }

    /// Returns the number of children.
    pub fn num_children(&self) -> usize {
        self.children.len()
    }

    /// Returns a reference to the child at the given index.
    pub fn child(&self, index: usize) -> Option<&Arc<dyn Node>> {
        self.children.get(index)
    }

    /// Returns an iterator over the children.
    pub fn children(&self) -> impl Iterator<Item = &Arc<dyn Node>> {
        self.children.iter()
    }
}

impl Node for NodeBase {
    fn as_any(&self) -> &dyn Any {
        self
    }

    fn as_any_mut(&mut self) -> &mut dyn Any {
        self
    }
}
