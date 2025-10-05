# Paginate

The `paginate` package takes a set of input Widget trees and returns a corresponding set of page Frame widgets that fit within a specified height, with optional title, headers and footers.

The main purpose is for generating PDF output, via the PDF function, which installs default PDF fonts (Helvetica, Times, Courier) and renders output.

The first step involves extracting a list of leaf-level widgets from surrounding core.Frame elements, that are then processed by the layout function to fit into page-sized chunks. This can be controlled by the properties as described below.

## Properties

Properties can be set on widgets to inform the pagination process. This is done by the `content` package, for example. All properties start with `paginate-`.

* `block` -- marks a Frame as a block that is not to be further extracted from in collecting leaves. Only Frame elements that have direction = Column are 

* `float-top` -- marks a `block` frame to be floated to the top of a page

* `break` -- starts a new page before this element.

* `no-break-after` -- marks an element to not have a page break inserted after it.

