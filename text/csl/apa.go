// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package csl

import (
	"unicode"

	"cogentcore.org/core/text/rich"
)

// definitive reference:
// https://apastyle.apa.org/style-grammar-guidelines/references/examples

// CiteAPA generates a APA-style citation, as Last[ & Last|et al.] Year
// with a , before Year in Parenthetical style, and Parens around the Year
// in Narrative style.
func CiteAPA(cs CiteStyles, it *Item) string {
	c := ""
	if len(it.Author) > 0 {
		c = NamesCiteEtAl(it.Author)
	} else {
		c = NamesCiteEtAl(it.Editor)
	}
	switch cs {
	case Parenthetical:
		c += ", " + it.Issued.Year()
	case Narrative:
		c += " (" + it.Issued.Year() + ")"
	}
	return c
}

// RefAPA generates an APA-style reference entry from the given item,
// with rich.Text formatting of italics around the source, volume,
// and spans for each separate chunk.
// Use Join method to get full raw text.
func RefAPA(it *Item) rich.Text {
	switch it.Type {
	case Book, Collection:
		return RefAPABook(it)
	case Chapter, PaperConference:
		return RefAPAChapter(it)
	case Thesis:
		return RefAPAThesis(it)
	case Article, ArticleJournal, ArticleMagazine, ArticleNewspaper:
		return RefAPAArticle(it)
	default:
		return RefAPAMisc(it)
	}
}

// RefsAPA generates a list of APA-style reference entries
// and correspondingly ordered items for given keylist.
// APA uses alpha sort order.
func RefsAPA(kl *KeyList) ([]rich.Text, []*Item) {
	refs := make([]rich.Text, kl.Len())
	items := make([]*Item, kl.Len())
	ks := kl.AlphaKeys()
	for i, k := range ks {
		it := kl.At(k)
		refs[i] = RefAPA(it)
		items[i] = it
	}
	return refs, items
}

func RefLinks(it *Item, tx *rich.Text) {
	link := rich.NewStyle().SetLinkStyle()
	if it.URL != "" {
		tx.AddLink(link, it.URL, it.URL)
	}
	if it.DOI != "" {
		url := " http://doi.org/" + it.DOI
		tx.AddLink(link, url, url)
	}
}

// EnsurePeriod returns a string that ends with a . if it doesn't
// already end in some form of punctuation.
func EnsurePeriod(s string) string {
	if !unicode.IsPunct(rune(s[len(s)-1])) {
		s += "."
	}
	return s
}

func RefAPABook(it *Item) rich.Text {
	sty := rich.NewStyle()
	ital := sty.Clone().SetSlant(rich.Italic)
	auths := ""
	if len(it.Author) > 0 {
		auths = NamesLastFirstInitialCommaAmpersand(it.Author)
	} else if len(it.Editor) > 0 {
		auths = NamesLastFirstInitialCommaAmpersand(it.Editor) + " (Ed"
		if len(it.Editor) == 1 {
			auths += ".)"
		} else {
			auths += "s.)"
		}
	}
	tx := rich.NewText(sty, []rune(auths+" "))
	tx.AddSpanString(sty, "("+it.Issued.Year()+"). ")
	if it.Title != "" {
		ttl := it.Title
		end := rune(ttl[len(ttl)-1])
		if it.Edition != "" {
			if unicode.IsPunct(end) {
				ttl = ttl[:len(ttl)-1]
			} else {
				end = '.'
			}
			tx.AddSpanString(ital, ttl)
			tx.AddSpanString(sty, " ("+it.Edition+" ed)"+string(end)+" ")
		} else {
			tx.AddSpanString(ital, EnsurePeriod(ttl)+" ")
		}
	}
	if it.Publisher != "" {
		tx.AddSpanString(sty, EnsurePeriod(it.Publisher)+" ")
	}
	RefLinks(it, &tx)
	return tx
}

func RefAPAChapter(it *Item) rich.Text {
	sty := rich.NewStyle()
	ital := sty.Clone().SetSlant(rich.Italic)
	tx := rich.NewText(sty, []rune(NamesLastFirstInitialCommaAmpersand(it.Author)+" "))
	tx.AddSpanString(sty, "("+it.Issued.Year()+"). ")
	contStyle := ital
	if it.Title != "" {
		if len(it.Editor) == 0 || it.ContainerTitle == "" {
			contStyle = sty
			tx.AddSpanString(ital, EnsurePeriod(it.Title)+" ")
		} else {
			tx.AddSpanString(sty, EnsurePeriod(it.Title)+" ")
		}
	}
	if len(it.Editor) > 0 {
		eds := "In " + NamesFirstInitialLastCommaAmpersand(it.Editor)
		if len(it.Editor) == 1 {
			eds += " (Ed.), "
		} else {
			eds += " (Eds.), "
		}
		tx.AddSpanString(sty, eds)
	} else {
		tx.AddSpanString(sty, "In ")
	}
	if it.ContainerTitle != "" {
		ttl := it.ContainerTitle
		end := rune(ttl[len(ttl)-1])
		pp := ""
		if it.Edition != "" {
			pp = "(" + it.Edition + " ed."
		}
		if it.Page != "" {
			if pp != "" {
				pp += ", "
			} else {
				pp = "("
			}
			pp += "pp. " + it.Page
		}
		if pp != "" {
			pp = " " + pp + ")"
			if unicode.IsPunct(end) {
				ttl = ttl[:len(ttl)-1]
				pp += string(end)
			} else {
				pp += "."
			}
			tx.AddSpanString(contStyle, ttl)
			tx.AddSpanString(sty, pp+" ")
		} else {
			tx.AddSpanString(contStyle, EnsurePeriod(it.ContainerTitle)+" ")
		}
	}
	if it.Publisher != "" {
		tx.AddSpanString(sty, EnsurePeriod(it.Publisher)+" ")
	}
	RefLinks(it, &tx)
	return tx
}

func RefAPAArticle(it *Item) rich.Text {
	sty := rich.NewStyle()
	ital := sty.Clone().SetSlant(rich.Italic)
	tx := rich.NewText(sty, []rune(NamesLastFirstInitialCommaAmpersand(it.Author)+" "))
	tx.AddSpanString(sty, "("+it.Issued.Year()+"). ")
	if it.Title != "" {
		tx.AddSpanString(sty, EnsurePeriod(it.Title)+" ")
	}
	jt := ""
	if it.ContainerTitle != "" {
		jt = it.ContainerTitle + ", "
	}
	if it.Volume != "" {
		jt += it.Volume
	}
	if jt != "" {
		tx.AddSpanString(ital, jt)
	}
	if it.Volume != "" {
		if it.Number != "" {
			tx.AddSpanString(sty, "("+it.Number+"), ")
		} else {
			tx.AddSpanString(sty, ", ")
		}
	}
	if it.Page != "" {
		tx.AddSpanString(sty, it.Page+". ")
	}
	RefLinks(it, &tx)
	return tx
}

func RefAPAThesis(it *Item) rich.Text {
	sty := rich.NewStyle()
	ital := sty.Clone().SetSlant(rich.Italic)
	tx := rich.NewText(sty, []rune(NamesLastFirstInitialCommaAmpersand(it.Author)+" "))
	tx.AddSpanString(sty, "("+it.Issued.Year()+"). ")
	if it.Title != "" {
		tx.AddSpanString(ital, EnsurePeriod(it.Title)+" ")
	}
	tt := "["
	if it.Source == "" {
		tt += "unpublished "
	}
	if it.Genre == "" {
		tt += "thesis"
	} else {
		tt += it.Genre
	}
	if it.Publisher != "" {
		tt += ", " + it.Publisher
	}
	tt += "]. "
	tx.AddSpanString(sty, tt)
	if it.Source != "" {
		tx.AddSpanString(sty, EnsurePeriod(it.Source)+" ")
	}
	RefLinks(it, &tx)
	return tx
}

func RefAPAMisc(it *Item) rich.Text {
	sty := rich.NewStyle()
	ital := sty.Clone().SetSlant(rich.Italic)
	tx := rich.NewText(sty, []rune(NamesLastFirstInitialCommaAmpersand(it.Author)+" "))
	tx.AddSpanString(sty, "("+it.Issued.Year()+"). ")
	if it.Title != "" {
		tx.AddSpanString(ital, EnsurePeriod(it.Title)+" ")
	}
	jt := ""
	if it.ContainerTitle != "" {
		jt = it.ContainerTitle + ", "
	}
	if it.Volume != "" {
		jt += it.Volume
	}
	if jt != "" {
		tx.AddSpanString(sty, jt)
	}
	if it.Volume != "" {
		if it.Number != "" {
			tx.AddSpanString(sty, "("+it.Number+"), ")
		} else {
			tx.AddSpanString(sty, ", ")
		}
	}
	if it.Page != "" {
		tx.AddSpanString(sty, it.Page+". ")
	}
	if it.Genre != "" {
		tx.AddSpanString(sty, EnsurePeriod(it.Genre)+" ")
	} else {
		tx.AddSpanString(sty, it.Type.String()+". ")
	}
	if it.Source != "" {
		tx.AddSpanString(sty, EnsurePeriod(it.Source)+" ")
	}
	if it.Publisher != "" {
		tx.AddSpanString(sty, EnsurePeriod(it.Publisher)+" ")
	}
	RefLinks(it, &tx)
	return tx
}
