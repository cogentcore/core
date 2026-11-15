// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmltext

import (
	"testing"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/text/rich"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// spanWith returns the style of the first span whose text is the given text.
func spanWith(t *testing.T, tx rich.Text, text string) *rich.Style {
	t.Helper()
	for i := range tx.NumSpans() {
		s, r := tx.Span(i)
		if string(r) == text {
			return s
		}
	}
	t.Fatalf("no span %q in %s", text, tx.String())
	return nil
}

// TestHTMLTags tests the styles and specials of the tags.
func TestHTMLTags(t *testing.T) {
	src := `<b>b</b> <i>i</i> <u>u</u> <s>s</s> <code>c</code> <q>q</q> <sup>2</sup> <sub>x</sub> <math>m</math> <dfn>d</dfn> <bdo>o</bdo> <err>e</err> <span class="math inline">\(a\)</span> <span class="math display">\[z\]</span> <p>para</p>next`
	tx, err := HTMLToRich([]byte(src), rich.NewStyle(), nil)
	require.NoError(t, err)
	assert.Equal(t, rich.Bold, spanWith(t, tx, "b").Weight)
	assert.Equal(t, rich.Italic, spanWith(t, tx, "i").Slant)
	assert.True(t, spanWith(t, tx, "u").Decoration.HasFlag(rich.Underline))
	assert.True(t, spanWith(t, tx, "s").Decoration.HasFlag(rich.LineThrough))
	assert.Equal(t, rich.Monospace, spanWith(t, tx, "c").Family)
	assert.Equal(t, rich.Quote, spanWith(t, tx, "q").Special)
	sup := spanWith(t, tx, "2")
	assert.Equal(t, rich.Super, sup.Special)
	assert.Equal(t, float32(0.8), sup.Size)
	assert.Equal(t, rich.Sub, spanWith(t, tx, "x").Special)
	assert.Equal(t, rich.MathInline, spanWith(t, tx, "m").Special)
	assert.Equal(t, rich.MathInline, spanWith(t, tx, "a").Special, "the \\( \\) are removed")
	assert.Equal(t, rich.MathDisplay, spanWith(t, tx, "z").Special, "the \\[ \\] are removed")
	assert.True(t, spanWith(t, tx, "para\n").Decoration.HasFlag(rich.ParagraphStart))

	_, err = HTMLToRich([]byte(`<blink>x</blink>`), rich.NewStyle(), nil)
	assert.ErrorContains(t, err, `"blink" tag not recognized`)
	tx, err = HTMLToRich(nil, rich.NewStyle(), nil)
	assert.NoError(t, err)
	assert.Nil(t, tx)
}

// TestHTMLEntitiesStyle tests entities, inline style and other attributes,
// and the CSS properties of classes and tags.
func TestHTMLEntitiesStyle(t *testing.T) {
	tx, err := HTMLToRich([]byte(`a &lt;b&gt; &amp; &eacute; <span style="color:red" font-weight="bold">r</span>`), rich.NewStyle(), nil)
	require.NoError(t, err)
	assert.Equal(t, "a <b> & é r", string(tx.Join()))
	r := spanWith(t, tx, "r")
	assert.Equal(t, colors.AsRGBA(colors.Red), colors.AsRGBA(r.FillColor()))
	assert.Equal(t, rich.Bold, r.Weight, "an attribute that is a property")

	css := map[string]any{
		".big": map[string]any{"font-size": "x-large"},
		"em":   map[string]any{"font-weight": "bold"},
		"bad":  "not a map",
	}
	tx, err = HTMLToRich([]byte(`<span class="big">big</span> <em>em</em>`), rich.NewStyle(), css)
	require.NoError(t, err)
	assert.Equal(t, rich.FontSizes["x-large"], spanWith(t, tx, "big").Size)
	em := spanWith(t, tx, "em")
	assert.Equal(t, rich.Italic, em.Slant)
	assert.Equal(t, rich.Bold, em.Weight, "the css of the tag")

	_, ok := SubProperties("bad", css)
	assert.False(t, ok)
	_, ok = SubProperties("missing", css)
	assert.False(t, ok)
	assert.False(t, FontStyleCSS(rich.NewStyle(), "em", nil))
	assert.False(t, FontStyleCSS(rich.NewStyle(), "p", css))
}

// TestHTMLPre tests the whitespace-preserving decoder.
func TestHTMLPre(t *testing.T) {
	src := "<pre>a  <b>bold</b>\n  <i>it</i> &amp; <a href=\"https://x.org\">ln</a><br>x<q>q</q> <sup>2</sup><sub>3</sub><math>m</math><dfn>d</dfn><bdo>o</bdo> <span style=\"color:red\" class=\"math inline\">e</span> <p>p</p></pre>"
	tx, err := HTMLPreToRich([]byte(src), rich.NewStyle(), map[string]any{"b": map[string]any{"font-style": "italic"}})
	assert.NoError(t, err)
	assert.Contains(t, string(tx.Join()), "a  bold\n  it & ln")
	assert.Equal(t, rich.Bold, spanWith(t, tx, "bold").Weight)
	assert.Equal(t, rich.Italic, spanWith(t, tx, "bold").Slant, "the css of the tag")
	ln := spanWith(t, tx, "ln")
	assert.Equal(t, rich.Link, ln.Special)
	assert.Equal(t, "https://x.org", ln.URL)
	assert.Equal(t, rich.MathInline, spanWith(t, tx, "e").Special)

	_, err = HTMLPreToRich([]byte("<b>x</i>"), rich.NewStyle(), nil)
	assert.ErrorContains(t, err, "doesn't match")
	_, err = HTMLPreToRich([]byte("<blink>x</blink>"), rich.NewStyle(), nil)
	assert.ErrorContains(t, err, "not recognized")
	_, err = HTMLPreToRich([]byte("x</b>"), rich.NewStyle(), nil)
	assert.Error(t, err, "imbalanced")
	tx, err = HTMLPreToRich([]byte("a < b <"), rich.NewStyle(), nil)
	assert.NoError(t, err)
	assert.Equal(t, "a < b <", string(tx.Join()))
	tx, err = HTMLPreToRich(nil, rich.NewStyle(), nil)
	assert.NoError(t, err)
	assert.Nil(t, tx)
}

// TestRichToHTML tests converting rich text to HTML, and back.
func TestRichToHTML(t *testing.T) {
	plain := rich.NewStyle()
	bold := plain.Clone().SetWeight(rich.Bold)
	boldIt := bold.Clone().SetSlant(rich.Italic)
	under := plain.Clone().SetDecoration(rich.Underline)
	var tx rich.Text
	tx.AddSpan(plain, []rune("a "))
	tx.AddSpan(bold, []rune("b "))
	tx.AddSpan(boldIt, []rune("bi"))
	tx.AddSpan(plain, []rune(" "))
	tx.AddSpan(under, []rune("u"))
	tx.AddLink(plain, "https://x.org/?a=1&b=2", "link")
	tx.AddSpan(plain, []rune(" 1 < 2 & 3"))
	h := RichToHTML(tx)
	assert.Equal(t, `a <b>b <i>bi</i></b> <u>u</u><a href="https://x.org/?a=1&amp;b=2">link</a> 1 &lt; 2 &amp; 3`, h)

	back, err := HTMLToRich([]byte(h), rich.NewStyle(), nil)
	require.NoError(t, err)
	assert.Equal(t, string(tx.Join()), string(back.Join()), "the same text")

	var open rich.Text
	open.AddSpan(boldIt.Clone().SetDecoration(rich.Underline), []rune("x"))
	assert.Equal(t, "<b><i><u>x</u></i></b>", RichToHTML(open), "closed at the end, innermost first")
	open.AddSpan(plain.Clone().SetSlant(rich.Italic), []rune("y"))
	assert.Equal(t, "<b><i><u>x</u></i></b><i>y</i>", RichToHTML(open), "the italic is closed with the bold it is in")
}

// TestParseAttrs tests parsing the attributes of start tags.
func TestParseAttrs(t *testing.T) {
	assert.Equal(t, [][2]string{{"style", "color:red; font-size:2em"}, {"class", "math inline"}, {"a", "b=c"}, {"x", "1"}},
		parseAttrs(`style="color:red; font-size:2em" class='math inline' a="b=c" x=1`))
	assert.Nil(t, parseAttrs(""))
}
