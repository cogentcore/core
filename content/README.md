# Cogent Content

Cogent Content is a package for making content-focused apps and websites consisting of Markdown, HTML, and Cogent Core.

## Code blocks

Code blocks marked as ```` ```Go ```` are interactive examples: the page shows their code in an editor and runs it. `core build`, `core run`, `core install`, `core pack` and `core generate` compile them into the app (as `contentcode_gen.go`, see [contentcode](contentcode)), so that they run without an interpreter; edited code runs in the yaegi interpreter if the app includes it (`cogentcore.org/core/yaegicore`). See https://cogentcore.org/core/build#code-blocks-of-content-apps.

## References

Wikilink styles for scientific source references are supported, using the [csl](../text/csl) Citation Style Language format, with one citation per wikilink:

* `[[@CiteKey]]` generates a `Parenthetical` citation, e.g., for APA style it is like: "Smith & Jones, 1989", which should be put in parentheses along with any others, e.g., `([[@SmithJones89]]; [[@West05]])`.

* `[[@^CiteKey]]` generates a `Narrative` citation, where the citation serves as the subject of the sentence. For APA style this is: "Smith & Jones (1989)".


