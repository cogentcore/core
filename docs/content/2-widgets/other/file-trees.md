# File trees

Cogent Core provides powerful file trees that allow users to view directories as nested trees.

You can make a file tree and open it at any filepath:

```Go
filetree.NewTree(parent).OpenPath(".")
```
