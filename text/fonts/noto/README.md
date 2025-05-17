# NotoSans reduced character set fonts

These are standard NotoSans fonts (https://github.com/notofonts/latin-greek-cyrillic), but only using the subset of runes present in the standard Roboto fonts, which reduces their size to be comparable to the Roboto files. The fontbrowser tool was used to load in the `Roboto-Regular.ttf` file, and the `Save Unicodes` function was called to save all the unicodes (runes) in hex format to unicodes.md file.

Then, we ran the fonttools `pyftsubset` command, per these instructions (`brew install fonttools` to install): https://fonttools.readthedocs.io/en/latest/subset/ to get the subset files, which are also optimized for size in other ways as well:

```bash
pyftsubset --unicodes-file=unicodes.md NotoSans-Regular.ttf
```

After verifying that the .subset files looked good, they were renamed to replace the starting files.

The original files were around 600 Kb each, and the subset files are just over 100 Kb each, whereas the Roboto files are somewhat larger at ~160 Kb each.

