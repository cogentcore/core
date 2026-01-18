// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//! Generic stack implementation.

/// A generic stack implemented using a Vec.
#[derive(Debug, Clone, Default)]
pub struct Stack<T> {
    items: Vec<T>,
}

impl<T> Stack<T> {
    /// Creates a new empty stack.
    pub fn new() -> Self {
        Self { items: Vec::new() }
    }

    /// Creates a new stack with the given capacity.
    pub fn with_capacity(capacity: usize) -> Self {
        Self {
            items: Vec::with_capacity(capacity),
        }
    }

    /// Pushes one or more items onto the stack.
    pub fn push(&mut self, item: T) {
        self.items.push(item);
    }

    /// Pushes multiple items onto the stack.
    pub fn push_many(&mut self, items: impl IntoIterator<Item = T>) {
        self.items.extend(items);
    }

    /// Pops the top item off the stack.
    /// Returns `None` if the stack is empty.
    pub fn pop(&mut self) -> Option<T> {
        self.items.pop()
    }

    /// Returns a reference to the top item on the stack without removing it.
    /// Returns `None` if the stack is empty.
    pub fn peek(&self) -> Option<&T> {
        self.items.last()
    }

    /// Returns a mutable reference to the top item on the stack without removing it.
    /// Returns `None` if the stack is empty.
    pub fn peek_mut(&mut self) -> Option<&mut T> {
        self.items.last_mut()
    }

    /// Returns the number of items in the stack.
    pub fn len(&self) -> usize {
        self.items.len()
    }

    /// Returns `true` if the stack is empty.
    pub fn is_empty(&self) -> bool {
        self.items.is_empty()
    }

    /// Clears the stack, removing all items.
    pub fn clear(&mut self) {
        self.items.clear();
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_stack_operations() {
        let mut stack = Stack::new();
        assert!(stack.is_empty());

        stack.push(1);
        stack.push(2);
        stack.push(3);

        assert_eq!(stack.len(), 3);
        assert_eq!(stack.peek(), Some(&3));

        assert_eq!(stack.pop(), Some(3));
        assert_eq!(stack.pop(), Some(2));
        assert_eq!(stack.pop(), Some(1));
        assert_eq!(stack.pop(), None);
        assert!(stack.is_empty());
    }
}
