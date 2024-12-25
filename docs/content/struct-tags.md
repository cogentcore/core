A cheat sheet of all possible **struct tags** and their meanings.

* `edit:"-"` — prevent users from editing a field in a [[form]] or [[table]]
* `label:"{Label}"` — change the label text for a field in a [[form]] or [[table]]
* `default:"{value}"` — specify a default value for a field in a [[form]], [[cli]], or [[settings]]
* `set:"-"` — do not [[generate]] a [[generate#setter]]
* `save:"-"` — do not save a field in [[settings]]
* `json:"-"`, `xml:"-"`, `toml:"-"` — do not save/load a field with JSON/XML/TOML
* `copier:"-"` — do not copy a field when copying/cloning a widget
* `min:"{number}"`, `max:"{number}"`, `step:"{number}"` — customize the min/max/step of a [[spinner]] or [[slider]]
* `display` — customize the appearance of a field in a [[form]] or [[table]]
    * `display:"-"` — hide a field
    * `display:"add-fields"` — expand subfields in a [[form]]
    * `display:"{inline|no-inline}"` — display a slice/map/struct inline or not inline
    * `display:"{switch-type}"` — customize the type of a [[switch]]
    * `display:"{date|time}"` — only display the date or time of a [[time picker#time input]]
* `table` — override the value of `display` in a [[table]]
    * `table:"-"` — hide a column
    * `table:"+"` — show a column
