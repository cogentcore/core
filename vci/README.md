# vcs

Package vcs provides a more complete version control system (ex: git) interface, building on [Masterminds/vcs](https://github.com/Masterminds/vcs).

It adds the following methods on top of what is available in `Masterminds/vcs`:

* `Files`: files in the repository, and their `FileStatus`

* `Add`, `Move`, `Delete`, `CommitFile`, `RevertFile`: manipulate files in the repository.

* `Log`, `CommitDesc`, `Blame`: details on prior commits

The total interface is now sufficient for complete management of a VCS.

# Current Status

Only `git` and `svn` are currently supported in `vcs`; other repositories supported in vcs include Mercurial (Hg) and Bazaar (Bzr); contributions from users of those VCS's are welcome.
