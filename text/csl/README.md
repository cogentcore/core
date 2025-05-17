# CSL: Citation Style Language

[CSL](https://citationstyles.org/) is used for managing citations for the major open source citation management systems, including zotero and mendeley, and replaces the outdated bibtex system that is specific to the latex framework.

This package reads and writes JSON formatted data files containing references, which can be exported from zotero or mendeley, in this [format schema](https://github.com/citation-style-language/schema/blob/master/schemas/input/csl-data.json). This is the replacement for the `.bib` file from bibtex, for example.

The result is a set of Go structs that represent the data, with `csl.Item` representing a given reference item.

The main feature of CSL is the ability to define new reference styles in the CSL language, which unfortunately is rather complex and requires significant infrastructure to process, and there is no extant Go native implementation.

Therefore, for the time being (ever), we are just using hand-written generators in Go that create a **Reference** from an Item (i.e., the text that goes in the References section) and a **Citation** (the text that goes in the body of a document that cites the reference). The `Styles` type represents available styles, and there are `Ref` and `Cite` functions that take a style and an item to generate according to the style.

The `KeyList` type provides an ordered list of unique citation items, using the `CitationKey` field, which can be used for collecting items by processing source files for references, and adding only unique entries in usage order. Some citation styles use order, and others are alpha sorted, so this provides either option.

There are two different types of `CiteStyles` that handle `Parenthetical` references (which go in parentheses) (Smith et al., 1985) versus `Narrative` citations: Smith et al. (1985) in APA format.

## APA: American Psychological Association

The default, primary style is [APA](https://apastyle.apa.org/style-grammar-guidelines/references/examples), with the `RefAPA` and `CiteAPA` methods for generating a reference or a citation. References are generated in alpha sorted order.


