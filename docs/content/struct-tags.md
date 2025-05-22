A cheat sheet of all possible **struct tags** and their meanings. See the linked pages for further explanations with examples.

* `edit:"-"` — prevent users from editing a field in a [[form]] or [[table]]
* `label:"{Label}"` — change the label text for a field in a [[form]] or [[table]]
* `default:"{value}"` — specify a default value for a field in a [[form]], [[cli]], or [[settings]]
* `set:"-"` — do not [[generate]] a [[generate#setter]]
* `save:"-"` — do not save a field in [[settings]]
* `json:"-"`, `xml:"-"`, `toml:"-"` — do not save/load a field with JSON/XML/TOML
* `copier:"-"` — do not copy a field when copying/cloning a widget
* `min:"{number}"`, `max:"{number}"`, `step:"{number}"` — customize the min/max/step of a [[spinner]] or [[slider]]
* `width:"{number}"`, `height:"{number}"`, `max-width:"{number}"`, `max-height:"{number}"` — specifies the size of the field's value widget in a [[form]] or [[table]], setting the [[styles]] `Min` (no prefix) or `Max` width in `Ch` (chars) or height in `Em` (font height units).
* `grow:"{number}"`, `grow-y:"{number}"` — specifies the [[styles]] `Grow` factor for the field's value widget in the X or Y dimension.
* `new-window:"+"` — causes a struct field in a [[form]] or [[table]] to open in a new popup dialog window by default, instead of requiring a `Shift` key to do so. The default is to open a full window dialog that replaces the current contents.
* `display` — customize the appearance of a field in a [[form]] or [[table]]
    - `display:"-"` — hide a field
    - `display:"add-fields"` — expand subfields in a [[form]]
    - `display:"{inline|no-inline}"` — display a slice/map/struct inline or not inline
    - `display:"{switch-type}"` — customize the type of a [[switch]]
    - `display:"{date|time}"` — only display the date or time of a [[time picker#time input]]
* `table` — override the value of `display` in a [[table]]
    - `table:"-"` — hide a column in a [[table]]
    - `table:"+"` — show a column in a [[table]]
