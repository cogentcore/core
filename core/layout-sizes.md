# Min and preferred sizes in layout (exploration notes)

Notes for the `layout-sizes` branch (made from `html` on 2026-11-04), to
explore whether core layout should track a minimum and a preferred size
for each widget systematically. Not for the `html` PR.

## Status (2026-11-05)

The first step of the design below is implemented (core/layoutintrinsic.go):

- `GeomSize.minWidth` and `prefWidth` are computed in `SizeUp` for every
  widget: text has the width of its longest word and its width on one line;
  frames combine those of their children per layout (row: sum, column: max,
  wrapped row: max of mins and sum of prefs, grid: per column, with spans).
  A frame that scrolls in width has no minimum of its own.
- `Text` starts at its preferred (one-line) width, and is re-wrapped in
  `SizeDown` at its allocated width with unlimited height (the shaper
  truncated text to twice the lines that fit the height), and again
  whenever that width changes, after the first iterations too.
- Every widget whose minimum is less than its preferred width shrinks in
  rows and grids (`Shrink` 0 now means 1), down to its `minWidth` (the
  separate `minContentWidth` walk is gone). A column that grows up to a
  `Max` (a CSS width) starts shrinking from it, as for the flex basis.
- Children of a column are at most as wide as the column: frames even below
  their minimum (as blocks in CSS: their content overflows), other widgets
  (text) not below it (the shaper would otherwise break words). Wrapped
  rows also limit their items, not below their minimum. `limitsWidth` is
  gone.
- A frame scrolls in width in the first iteration only if its content is
  wider than it at its min-content width.
- A scene sized to its content (windows, dialogs) limits the preferred
  width of text to 70ch (`contentSizingMeasure`), as the old aspect ratio
  estimate did in its way; otherwise a long message is a screen-wide line.
- Fixes from the demo (narrow Inputs tab): a frame is never allocated less
  than its own `Min` (as a CSS min-width; its content then overflows up to
  the nearest scroller, instead of every nested scrolling frame scrolling);
  a child that overflows its allocation (a frame narrowed below its
  minimum, as the non-wrapping segmented buttons) counts at its actual
  width in the SizeFinal cells, so the frame that it is in scrolls to it;
  widgets with parts (buttons, switches, choosers) are not narrowed
  (min = pref = actual), as their parts are laid out in their actual size.
- When the columns of a row or grid had to shrink to fit (`shrinkWidths`
  reports `shrunk`), their widths are final: they do not also grow from
  stale cell widths in later iterations (unequal "100%" columns). Children
  of a column start from their preferred width within it in every
  iteration (was only for `Shrink` > 0).
- Focused tests in core/layout_test.go (TestLayoutNestedScrollMin,
  OverflowScrolls, PartsNotNarrowed, TextWrapHeight, ShrinkToMin,
  ShrinkMaxBasis, GridGrowShrinkable, ContentSizingMeasure, TextScrollbar,
  OverflowBeyondMax, GrowWithinRow; htmlcore TestSVGMaxWidth),
  each checked to fail with its fix reverted, except: the late rewrap of
  text is only caught by the w3c snapshot of htmlcore, and the widening to
  the preferred width in columns by none (kept as the general rule).
- Visible overflow beyond a Max (a row of cards in a page with a
  max-width): `GeomSize.extent` and `layoutState.overflow`
  (`Frame.sizeOverflow`) carry the content that overflows a frame beyond its
  Max up to the frame that scrolls, whose `ScrollValues` (and first-pass
  `overflowWidth`) include it, so it scrolls over all of it.
- When the columns of a row fit at the widths from `shrinkWidths` (widened
  to their preferred widths), `growWidths` grows the growing ones into the
  width that is left, passing on what a column cannot take beyond its Max,
  instead of growing from stale cell widths (the Rust header overflowed).
- htmlcore: an img of an svg without width and height is scaled down to an
  absolute max-width / max-height, keeping its aspect (the Rust icons were
  512px wide with a max-width of 6rem).
- Removed: `Text.FitWidth`, `Text.availableWidth`, the aspect ratio estimate
  in `Text.SizeUp`. `GrowWrap` stays only for text that is not aligned at
  the start (centered text is aligned within all of the available width).
- Fixed on the way: `shrinkWidths` returned nil (no shrinking at all) when
  the columns reached their minimum in the second pass; a column with extra
  width that grows kept only its preferred width from `shrinkWidths`.

Results: 13 core images changed, all reviewed as the same or better (short
texts on one line, no spurious horizontal scrollbars in lists and trees,
mixed-vertical splits one line per sentence, dialogs sized to content);
htmlcore fidelity vs Chrome 285 -> 291 of 370 boxes (23-email-complex
15/15), no case worse; snapshots the same or better (w3c, rust columns).

Not done yet (candidates for the next steps): `childSpace`/`growExtra`,
toolbar overflow (`preferred`), `nosz` for wrapped rows, the 40ch `Max` of
text fields, the `Shrink` settings in htmlcore that are now the default,
and `GrowWrap` for centered text (could become alignment within the
allocation at SizeFinal).

## The question

Core layout has one size per widget from `SizeUp` (`Actual`), which is
both the minimum (the hard requirement) and the starting size. An early
version of the layout had both min and preferred sizes, but that seemed
unnecessary and made everything much more complicated, and shrinking was
left out for the same reason.

Since then, the `html` branch has added several separate mechanisms that
each approximate one of the two sizes in their own way:

- **Wrapping text** (`GrowWrap`, `WhiteSpace` with word wrap): `Text.SizeUp`
  starts from `shaped.WrapSizeEstimate`, a balanced aspect ratio, which is
  neither its minimum nor its preferred width. `SizeDown` then re-wraps it
  to the allocation, often with another iteration.
- **`Text.FitWidth`** (used by htmlcore): starts the text as wide as it is
  on one line, up to `Text.availableWidth`, as for the shrink-to-fit
  widths of CSS. This is the preferred (max-content) width, but it is
  then also the minimum.
- **`Shrink` and min-content widths**: `minContentWidth` (core/layout.go)
  walks the subtree of a widget on demand to compute its min-content width
  (the longest word of text, with the cached `Text.minWord`), and
  `Frame.shrinkWidths` narrows columns from their `SizeUp` widths down to
  at most those. This is the minimum, computed separately from `SizeUp`.
- **Frames that do not grow** (`Frame.childSpace`, `layoutState.growExtra`):
  children that grow only get the actual size of the frame, while wrapping
  text still gets the whole allocation to wrap in.
- **`GrowWrap`** (styles/style.go, set by `Text`): wrapping text "grows"
  into the available width in the first `SizeDown` iteration, so that it
  can wrap at that width, but not afterward, so that it can still be
  aligned (elements that `Grow` absorb the extra space and cannot be).
  It is special-cased in several places of core/layout.go (the cells of
  `sizeDownGrowCells`, `childSpace`, the shrinking of columns, and
  `growToAlloc`). It stands in for a preferred width: "as wide as the
  space allows, up to the width of the text on one line".
- **Text fields** (core/textfield.go): `Min.X` 20ch, `Max.X` 40ch, and
  `Grow.X` 1, with `GrowWrap` off ("doesn't work with Grow"). The 40ch
  `Max` keeps text fields from growing across a whole form or window,
  which is a preferred width expressed as a cap on growth. These limits
  may be a legacy of earlier problems of the layout logic (a todo of the
  user is to revisit them).
- In general, the many issues over time with text growing too much or too
  little: wrapped text that is too narrow (the aspect ratio estimate),
  text that does not wrap when it should (`FitWidth`), and text that
  grows past the frame that it is in (fixed by `childSpace`).
- Smaller ones: toolbars counting the items that grow at their size before
  overflowing, `TreeTable.FitWidth` and the table column widths, the
  percentage widths of htmlcore tables, the wrapped rows that shrink
  (`nosz` in `Frame.laySetContentFitOverflow`).

## The experiment that prompted this

`FitWidth` as the default for all `Text` (one line in `Text.Init`):
only core changes (content, htmlcore, textcore and filetree are the same,
as htmlcore already sets it), 25 images and one assertion:

- Better: short texts in windows sized to their content stay on one line
  (`text/type/*`: "Hello, world!" at display size went from a 157x304
  wrapped box to 359x92; also `text/rem`, `text/link`, `bind/text`), and
  `splits/mixed-vertical` is wide (one line per sentence) instead of a
  123px wrapped column.
- Worse: in `layout/parent-relative-splits`, "20% Split Frame" no longer
  wraps in its narrow split: it is clipped, and the split scrolls
  horizontally instead. The one-line width became the minimum, so a frame
  that can scroll scrolls instead of letting the text wrap (a browser
  wraps it).
- `TestShrinkRow` fails: two texts of different lengths both get 152 (it
  expects the longer one to be wider), probably for the same reason.

The fix is the distinction that core lacks: the minimum should stay the
min-content width (the longest word), and the one-line width should only
be the preferred size.

## A design to try: intrinsic widths only

The earlier complexity came from tracking min and preferred sizes in both
dimensions everywhere. CSS gets most of the benefit with less:

- **Widths are intrinsic, heights follow.** Each widget computes two
  widths in `SizeUp`: its _min-content_ width (below which its content
  overflows) and its _max-content_ (preferred) width (its width without
  any wrapping). Heights are not computed this way: they come from laying
  out the content at the width that the widget is finally given, as now.
- **Combining is simple and per layout type**: a row sums both of its
  children's widths (plus gaps); a column takes the maximum of each; a
  wrapped row takes the maximum of the min-contents and the sum of the
  max-contents; a grid does the same per column, with spans spread over
  their columns (as now for `ColSpan`).
- **Allocation in `SizeDown`** then starts every widget at its preferred
  width, clamped to the allocation, and shrinks toward the minimum only
  when there is too little space (that is `Shrink`, which would no longer
  need a separate walk), and grows by `Grow` when there is extra. A widget
  is only wider than its allocation (overflowing, or scrolling) when the
  allocation is less than its minimum.
- `Min` and `Max` styles clamp both widths; `Actual` stays the final size.

What this could replace or simplify:

- the aspect ratio estimate of wrapping text, and `FitWidth` (text would
  always start at its preferred width and wrap when narrowed);
- `GrowWrap`: wrapping text would not need to grow to wrap well, as its
  preferred width is the width of its text on one line, so it takes the
  space that it needs (up to the allocation) without `Grow`, and can be
  aligned;
- the 40ch `Max` of text fields: a text field would have a preferred width
  (for example 20ch, or that of its content) and grow from there only as
  its `Grow` says, so a cap on growth would no longer be needed to keep it
  from filling a form;
- `minContentWidth` and the `minWord` cache (computed once in `SizeUp`);
- most `SizeDown` re-wrap iterations, as widths are right from the start;
- the special cases above (`childSpace`/`growExtra`, toolbar overflow,
  shrink-to-fit widths in htmlcore, `nosz`).

Costs and risks:

- two more numbers per widget, and the combining rules in each layout
  (flex row, column, wrap, grid, stacked, floats, out-of-flow);
- widgets with custom sizing (`SizeUp` overrides) need both widths;
- many test images would change, as the starting widths of wrapping text
  change everywhere (the 25 above are a lower bound: htmlcore already
  behaves this way).

Whether it is simpler overall is the open question: it adds one concept,
and could remove several ad hoc ones. A first step that tests it cheaply:
compute min-content and max-content widths in `SizeUp` for `Text` and
frames only, use them for `Text` instead of the aspect ratio estimate,
and see how many of the special cases can then be removed.
