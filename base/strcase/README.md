# strcase

Package strcase provides functions for manipulating the case of strings (CamelCase, kebab-case, snake_case, Sentence case, etc). It is based on https://github.com/ettle/strcase, which is Copyright (c) 2020 Liyan David Chang under the MIT License. Its principle difference from other strcase packages is that it preserves acronyms in input text for CamelCase. Therefore, you must call `strings.ToLower` on any SCREAMING_INPUT_STRINGS before passing them to `ToCamel`, `ToLowerCamel`, `ToTitle`, and `ToSentence`.

