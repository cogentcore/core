# undo

The `undo` package provides a generic undo / redo functionality based on `[]string` representations of any kind of state representation (typically JSON dump of the 'document' state).  It stores the compact diffs from one state change to the next, with raw copies saved at infrequent intervals to tradeoff cost of computing diffs.

In addition state (which is optional on any given step), a description of the action and arbitrary string-encoded data can be saved with each record.  Thus, for cases where the state doesn't change, you can just save some data about the action sufficient to undo / redo it.

A new record must be saved of the state just *before* an action takes place and the nature of the action taken.

Thus, undoing the action restores the state to that prior state.

Redoing the action means restoring the state *after* the action.

This means that the first Undo action must save the current state before doing the undo.

The Idx is always on the last state saved, which will then be the one that would be undone for an undo action.


