# GoPi = Interactive Parser in GoKi / GoGi Framework

GoPi is part of the GoKi Go language (golang) full strength tree structure system (ki = 木 = tree in Japanese)

The `pi` package supports a simple and robust form of lexing and parsing based on top-down recursive descent, and allows users to create parsers using the GoGi graphical interface system.  It is used for syntax highlighting, completion, and more advanced language-structure specific functionality in GoGi and in the [Gide](https://github.com/goki/gide) IDE / editor.

We call it `Pi` (or `GoPi`) because Ip is not as easy to pronounce, and also because it makes parsing as easy as pi!  You can think of it as a French acronym, which are typically the reverse of English ones -- "parseur interactif".  Also, it matches GoKi and GoGi. 

[![Go Report Card](https://goreportcard.com/badge/github.com/goki/pi)](https://goreportcard.com/report/github.com/goki/pi)
[![GoDoc](https://godoc.org/github.com/goki/pi?status.svg)](https://godoc.org/github.com/goki/pi)

See the [Wiki](https://github.com/goki/pi/wiki) for more detailed docs, discussion, etc.

Pi uses a robust, top-down __Recursive Descent (RD)__ parsing technique (see [WikiPedia](https://en.wikipedia.org/wiki/Recursive_descent_parser)), which is the approach used by most hand-coded parsers, which are by far the most widely used in practice (e.g., for **gcc**, **clang**, and **Go**) for [various reasons](http://blog.reverberate.org/2013/09/ll-and-lr-in-context-why-parsing-tools.html) -- see this [stack overflow](https://stackoverflow.com/questions/6319086/are-gcc-and-clang-parsers-really-handwritten) thread too.  As far as we can tell (e.g., from this list on [WikiPedia](https://en.wikipedia.org/wiki/Comparison_of_parser_generators) ) there are not many recursive-descent *parser generators*, and none that use the same robust, simple techniques that we employ in GoPi.

It seems that most parsing is dominated by a very strong *sequentiality assumption* -- that you must parse everything in a strictly sequential, incremental, left-to-right, one-token-at-a-time manner.  If you step outside of that box (or break with the [herd](https://en.wikipedia.org/wiki/GNU_Bison) if you will), by loading the entire source in to RAM and processing the entire thing as a whole structured entity (which is entirely trivial these days -- even the biggest source code is typically tiny relative to RAM capacity), then much simpler, more robust solutions are possible.  In other words, instead of using a "1D" solution, we actually use a **3D** solution to parsing (line, char, and nesting depth).  Specifically, we just carve the whole source in to statement-level chunks and then proceed to look for distinctive lexical tokens *anywhere* in the statement to determine what kind of statement it is, and then proceed recursively to carve that up into its respective parts, using the same approach.  There is never any backtracking or shift-reduce conflicts or any of those annoying issues that plague other approaches -- the grammar you write is very directly the grammar of the language, and doesn't require a lot of random tweaks and special cases to get it to work.

The latest version 4 of [ANTLR](https://en.wikipedia.org/wiki/ANTLR) with its `ALL(*)` algorithm sounds like it offers similar abilities to robustly handle intuitive grammars, and is likely more generalizable to a wider range of languages, and is probably faster overall than GoPi, *but* GoPi is likely much simpler and more transparent in terms of how it actually works (disclaimer: I have no idea whatsoever how ANTLR V4 actually works!  And that's kind of the point..).  Anyone should be able to understand how GoPi works, and tweak it as needed, etc.  And it operates directly in AST-order, creating the corresponding AST on the fly as it parses, so you can interactively understand what it is doing as it goes along, making it relatively easy to create your grammar (although this process is, in truth, always a bit complicated and never as easy as one might hope).  And GoPi is fast enough for most uses, taking just a few hundred msec for even relatively large and complex source code.

GoPi does three distinct passes through the source file, each creating a solid foundation upon which the next step operates, producing significantly faster, simpler, and more error-tolerant (robust) results overall.

* **Lexer** -- takes the raw text and turns it into lexical `Tokens` that categorize a sequence of characters as things like a `Name` (identifier -- a string of letters and numbers without quotes) or a `Literal` of some type, for example a `LitStr` string that has some kind of quotes around it, or a `LitNum` which is a number.  There is a nice category (Cat) and subcategory (SubCat) level of organization to these tokens (see `token/token.go`).  Comments are absorbed in this step as well, and stored in a separate lex output so you can always access them, without having to deal with them in the parsing process.  The key advantage for subsequent steps is that any ambiguity about e.g., syntactic elements showing up in comments or strings is completely eliminated right at the start.  Furthermore, the tokenized version of the file is much, much more compact and contains only the essential information for parsing.

* **StepTwo** -- this is a critical second pass through the lexical tokens, performing two important things:

    + **Nesting Depth** -- all programming languages use some form of parens `( )` brackets `[ ]` and braces `{ }` to group elements, and parsing must be sensitive to these.  Instead of dealing with these issues locally at every step, we do a single pass through the entire tokenized version of the source and compute the depth of every token.  Then, the token matching in parsing only needs to compare relative depth values, without having to constantly re-compute that.  As an extra bonus, you can use this depth information in syntax highlighting (as we do in [Gide](https://github.com/goki/gide)).
	
    + **EOS Detection** -- This step detects *end of statement* tokens, which provide an essential first-pass rough-cut chunking of the source into *statements*.  In C / C++ / Go these are the *semicolons* `;` (in Go, semicolons are mostly automatically computed from tokens that appear at the end of lines -- Pi supports this as well).  In Python, this is the end of line itself, unless it is not at the same nesting depth as at the start of the line.
	
* **Parsing** -- finally we parse the tokenized source using rules that match patterns of tokens.  Instead of using bottom-up local token-sequence driven parsing, as is done in tools like `yacc` and `bison`, we use the much more robust recursive descent technique, which starts in our case with those rough-cut statement chunks produced in StepTwo, and proceeds by recognizing the type of each statement, and then further carving each one into its appropriate sub-parts, again in a top-down, progressively-narrowing fashion, until everything has been categorized to the lowest level.  At each step, nodes in an __Abstract Syntax Tree (AST)__ are created, representing this same top-down broad-to-narrow parsing of the source.  Thus, the highest nodes are statement-level nodes, each of which then contain the organized elements of each statement.  These nodes are all in the natural *functional* hierarchical ordering, *not* in the raw left-to-right order of the source, and directly correspond to the way that the parsing proceeds.  Thus, building the AST at the same time as parsing is very natural in the top-down RD framework, unlike traditional bottom-up approaches, and is a major reason that hand-coded parsers use this technique.

## RD Parsing Advantages and Issues

Our top-down approach is generally much more robust: instead of depending on precise matches at every step along the way, it starts with the "big picture" by looking for the *Key Token* that unambiguously indicates what kind of thing we're looking at in a given region, and it then proceeds to process that region as such.  If some parts of that region happen to be malformed, it can issue appropriate errors, *but that will not spill over into processing of the next region.*  Thus, errors are automatically "sandboxed" in these regions, and do not accumulate.   By contrast, in bottom-up parsers, you need to add extensive error-matching rules at every step to achieve this kind of robustness, and that is often a tricky trial-and-error process and is not inherently robust.

### Solving the Associativity problem with RD parsing: Put it in Reverse!

One major problem with RD parsing is that it gets the [associativity](https://en.wikipedia.org/wiki/Operator_associativity) of mathematical operators [backwards](https://eli.thegreenplace.net/2009/03/14/some-problems-of-recursive-descent-parsers/).  To solve this problem, we simply run those rules in reverse: they scan their region from right to left instead of left to right.  This is much simpler than other approaches and works perfectly -- and is again something that you wouldn't even consider from the standard sequential mindset.  You just have to add a '-' minus sign at the start of the `Rule` to set the rule to run in reverse -- this must be set for all binary mathematical operators (e.g., `BinaryExpr` in the standard grammar).  

Also, for RD parsing, to deal properly with the [order of operations](https://en.wikipedia.org/wiki/Order_of_operations), you have to order the rules in the *reverse* order of precedence.  Thus, it matches the *lowest* priority items first, and those become the "outer branch" of the AST, which then proceeds to fill in so that the highest-priority items are in the "leaves" of the tree, which are what gets processed first.

# Lexing and Parsing Rules

## Principle of Preemptive Specificity

A common principle across lexing and parsing rules is the *principle of preemptive specificity* -- all of the different rule options are arranged in order, and the *first to match* preempts consideration of any of the remaining rules.  This is how a `switch` rule works in Go or C.  This is a particularly simple way of dealing with many potential rules and conflicts therefrom.  The overall strategy as a user is to *put the most specific items first* so that they will get considered, and then the general "default" cases are down at the bottom.  This is hopefully very intuitive and easy to use.

In the Lexer, this is particularly important for the `State` elements: when you enter a different context that continues across multiple chars or lines, you push that context onto the State Stack, and then it is critical that all the rules matching those different states are at the top of the list, so they preempt any non-state-specific alternatives. 

## Lexing Rules

Lexing is all about the ordering -- follow the above principle of preemptive specificity.

Another common situation is resolving the ambiguity about multi-character constructs, such as `:=` vs just `:` or `=`.  In this case, you have an outer rule that matches `:`, and then within that, child rules that do a one-step lookahead for a `=` at `Off=1` -- if that matches, then it is `:=` -- you need to put that rule first, and then the default is just the plain `:`.

## Parsing Rules

* order matters!

### Generative Expression Subdomains

There are certain subdomains that have very open-ended combinatorial "generative" expressive power.  These are particular challenges for any parser, and there are a few critical issues and tips for the Pi parser.

#### Arithmetic with Binary and Unary Operators

You can create arbitrarily long expressions by stringing together sequences of binary and unary mathematical / logical operators.  From the top-down parser's perspective, here are the key points:

1. Each operator must be uniquely recognizable from the soup of tokens, and this critically includes distinguishing unary from binary: e.g., correctly recognizing the binary and unary - signs here: `a - b * -c`  

2. The operators must be organized in *reverse* order of priority, so that the lowest priority operations are factored out first, creating the highest-level, broadest splits of the overall expression (in the Ast tree), and then progressively finer, tighter, inner steps are parsed out.  Thus, for example in this expression:

```Go
if a + b * 2 / 7 - 42 > c * d + e / 72
```

The broadest, first split is into the two sides of the `>` operator, and then each of those sides is progressively organized first into an addition operator, then the `*` and `/`.  

3. The binary operators provide the recursive generativity for the expression.  E.g., Addition is specified as:

```Go
AddExpr: Expr '+' Expr
```

so it just finds the + token and then descends recursively to unpack each of those `Expr` chunks on either side, until there are no more tokens left there.

One particularly vexing situation arises if you have the possibility of mixing multiplication with de-pointering, both of which are indicated by the `*` symbol.  In Go, this is particularly challenging because of the frequent use of type literals, including those with pointer types, in general expressions.  We had to add the ability to specify an "exclusion" expression that detects this alternate use and excludes it for the general multiplication case (because multiplication is broader than de-pointer and thus comes first in the matching hierarchy).

#### Path-wise Operators

Another generative domain are the path-wise operators, principally the "selector" operator `.` and the slice operator `'[' SliceExpr ']'`, which can be combined with method calls and other kinds of primary expressions in a very open-ended way, e.g.,:

```Go
ps.Errs[len(ps.Errs)-1].Error()[0][1].String()
```

In the top-down parser, it is essential to create this open-ended scenario by including pre-and-post expressions surrounding the `Slice` and `Selector` operators, which then act like the Expr groups surrounding the AddExpr operator to support recursive chaining.  For Selector, the two Expr's are required, but for Slice, they are optional - that works fine:

```Go
Slice: ?PrimaryExpr '[' SliceExpr ']' ?PrimaryExpr
```

Without those optional exprs on either side, the top-down parser would just stop after getting either side of that expression.

As with the arithmetic case, order matters and in the same inverse way, where you want to match the broader items first.  Also, those cases that are more specific and would otherwise be absorbed by a more general expression (e.g., 


