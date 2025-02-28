# spell

spell is a spell checking package, originally based on https://github.com/sajari/fuzzy

As of 7/2024, we only use a basic dictionary list of words as the input, and compute all the variants for checking on the fly after loading, to save massively on the file size.

The input dictionary file must be a simple one-word-per-line file, similar to /usr/share/dict/words

# Word lists

It is somewhat difficult to find suitable lists of words, especially in different languages!  Here are some resources:

* https://unix.stackexchange.com/questions/213628/where-do-the-words-in-usr-share-dict-words-come-from

* SCOWL: http://wordlist.aspell.net/ -- source of /usr/share/dict/words

* https://github.com/dwyl/english-words

* https://stackoverflow.com/questions/2213607/how-to-get-english-language-word-database

* https://web.archive.org/web/20120215032551/http://en-gb.pyxidium.co.uk/dictionary

* https://www.cs.hmc.edu/~geoff/ispell-dictionaries.html -- source with lots of different languages.

