// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lines

import (
	"fmt"
	"testing"

	"cogentcore.org/core/base/fileinfo"
	_ "cogentcore.org/core/system/driver"
	"cogentcore.org/core/text/highlighting"
	"cogentcore.org/core/text/parse"
	"github.com/stretchr/testify/assert"
)

func TestMarkup(t *testing.T) {
	src := `func (ls *Lines) deleteTextRectImpl(st, ed textpos.Pos) *textpos.Edit {
	tbe := ls.regionRect(st, ed)
	if tbe == nil {
	return nil
	}
`

	lns := &Lines{}
	lns.Defaults()
	lns.width = 40

	fi, err := fileinfo.NewFileInfo("dummy.go")
	assert.Error(t, err)
	var pst parse.FileStates
	pst.SetSrc("dummy.go", "", fi.Known)
	pst.Done()

	lns.Highlighter.Init(fi, &pst)
	lns.Highlighter.SetStyle(highlighting.HighlightingName("emacs"))
	lns.Highlighter.Has = true
	assert.Equal(t, true, lns.Highlighter.UsingParse())

	lns.SetText([]byte(src))
	assert.Equal(t, src+"\n", string(lns.Bytes()))

	mu := `[monospace bold fill-color]: "func "
[monospace]: " ("
[monospace]: "ls "
[monospace fill-color]: " *"
[monospace]: "Lines"
[monospace]: ") "
[monospace]: " deleteTextRectImpl"
[monospace]: "(
"
[monospace]: "st"
[monospace]: ", "
[monospace]: " ed "
[monospace]: " textpos"
[monospace]: "."
[monospace]: "Pos"
[monospace]: ") "
[monospace fill-color]: " *"
[monospace]: "textpos"
[monospace]: "."
[monospace]: "Edit "
[monospace]: " {"
`
	assert.Equal(t, 1, lns.nbreaks[0])
	assert.Equal(t, mu, lns.markup[0].String())
}

func TestLineWrap(t *testing.T) {
	src := `The [rich.Text](http://rich.text.com) type is the standard representation for formatted text, used as the input to the "shaped" package for text layout and rendering. It is encoded purely using "[]rune" slices for each span, with the _style_ information **represented** with special rune values at the start of each span. This is an efficient and GPU-friendly pure-value format that avoids any issues of style struct pointer management etc.
It provides basic font styling properties.
The "n" newline is used to mark the end of a paragraph, and in general text will be automatically wrapped to fit a given size, in the "shaped" package. If the text starting after a newline has a ParagraphStart decoration, then it will be styled according to the "text.Style" paragraph styles (indent and paragraph spacing). The HTML parser sets this as appropriate based on "<br>" vs "<p>" tags.
`

	lns := &Lines{}
	lns.Defaults()
	lns.width = 80

	fi, err := fileinfo.NewFileInfo("dummy.md")
	assert.Error(t, err)
	var pst parse.FileStates
	pst.SetSrc("dummy.md", "", fi.Known)
	pst.Done()

	lns.Highlighter.Init(fi, &pst)
	lns.Highlighter.SetStyle(highlighting.HighlightingName("emacs"))
	lns.Highlighter.Has = true
	assert.Equal(t, true, lns.Highlighter.UsingParse())

	lns.SetText([]byte(src))
	assert.Equal(t, src+"\n", string(lns.Bytes()))

	mu := `[monospace]: "The "
[monospace fill-color]: "[rich.Text]"
[monospace fill-color]: "(http://rich.text.com)"
[monospace]: " type is the standard representation for 
"
[monospace]: "formatted text, used as the input to the "
[monospace fill-color]: ""shaped""
[monospace]: " package for text layout and 
"
[monospace]: "rendering. It is encoded purely using "
[monospace fill-color]: ""[]rune""
[monospace]: " slices for each span, with the
"
[monospace italic]: " _style_"
[monospace]: " information"
[monospace bold]: " **represented**"
[monospace]: " with special rune values at the start of 
"
[monospace]: "each span. This is an efficient and GPU-friendly pure-value format that avoids 
"
[monospace]: "any issues of style struct pointer management etc."
`
	join := `The [rich.Text](http://rich.text.com) type is the standard representation for 
formatted text, used as the input to the "shaped" package for text layout and 
rendering. It is encoded purely using "[]rune" slices for each span, with the
 _style_ information **represented** with special rune values at the start of 
each span. This is an efficient and GPU-friendly pure-value format that avoids 
any issues of style struct pointer management etc.`

	// fmt.Println("\nraw text:\n", string(lns.lines[0]))

	// fmt.Println("\njoin markup:\n", string(nt))
	nt := lns.markup[0].Join()
	assert.Equal(t, join, string(nt))

	// fmt.Println("\nmarkup:\n", lns.markup[0].String())
	assert.Equal(t, 5, lns.nbreaks[0])
	assert.Equal(t, mu, lns.markup[0].String())

	lay := `[1 1:1 1:2 1:3 1:4 1:5 1:6 1:7 1:8 1:9 1:10 1:11 1:12 1:13 1:14 1:15 1:16 1:17 1:18 1:19 1:20 1:21 1:22 1:23 1:24 1:25 1:26 1:27 1:28 1:29 1:30 1:31 1:32 1:33 1:34 1:35 1:36 1:37 1:38 1:39 1:40 1:41 1:42 1:43 1:44 1:45 1:46 1:47 1:48 1:49 1:50 1:51 1:52 1:53 1:54 1:55 1:56 1:57 1:58 1:59 1:60 1:61 1:62 1:63 1:64 1:65 1:66 1:67 1:68 1:69 1:70 1:71 1:72 1:73 1:74 1:75 1:76 1:77 2 2:1 2:2 2:3 2:4 2:5 2:6 2:7 2:8 2:9 2:10 2:11 2:12 2:13 2:14 2:15 2:16 2:17 2:18 2:19 2:20 2:21 2:22 2:23 2:24 2:25 2:26 2:27 2:28 2:29 2:30 2:31 2:32 2:33 2:34 2:35 2:36 2:37 2:38 2:39 2:40 2:41 2:42 2:43 2:44 2:45 2:46 2:47 2:48 2:49 2:50 2:51 2:52 2:53 2:54 2:55 2:56 2:57 2:58 2:59 2:60 2:61 2:62 2:63 2:64 2:65 2:66 2:67 2:68 2:69 2:70 2:71 2:72 2:73 2:74 2:75 2:76 2:77 3 3:1 3:2 3:3 3:4 3:5 3:6 3:7 3:8 3:9 3:10 3:11 3:12 3:13 3:14 3:15 3:16 3:17 3:18 3:19 3:20 3:21 3:22 3:23 3:24 3:25 3:26 3:27 3:28 3:29 3:30 3:31 3:32 3:33 3:34 3:35 3:36 3:37 3:38 3:39 3:40 3:41 3:42 3:43 3:44 3:45 3:46 3:47 3:48 3:49 3:50 3:51 3:52 3:53 3:54 3:55 3:56 3:57 3:58 3:59 3:60 3:61 3:62 3:63 3:64 3:65 3:66 3:67 3:68 3:69 3:70 3:71 3:72 3:73 3:74 3:75 3:76 3:77 4 4:1 4:2 4:3 4:4 4:5 4:6 4:7 4:8 4:9 4:10 4:11 4:12 4:13 4:14 4:15 4:16 4:17 4:18 4:19 4:20 4:21 4:22 4:23 4:24 4:25 4:26 4:27 4:28 4:29 4:30 4:31 4:32 4:33 4:34 4:35 4:36 4:37 4:38 4:39 4:40 4:41 4:42 4:43 4:44 4:45 4:46 4:47 4:48 4:49 4:50 4:51 4:52 4:53 4:54 4:55 4:56 4:57 4:58 4:59 4:60 4:61 4:62 4:63 4:64 4:65 4:66 4:67 4:68 4:69 4:70 4:71 4:72 4:73 4:74 4:75 4:76 5 5:1 5:2 5:3 5:4 5:5 5:6 5:7 5:8 5:9 5:10 5:11 5:12 5:13 5:14 5:15 5:16 5:17 5:18 5:19 5:20 5:21 5:22 5:23 5:24 5:25 5:26 5:27 5:28 5:29 5:30 5:31 5:32 5:33 5:34 5:35 5:36 5:37 5:38 5:39 5:40 5:41 5:42 5:43 5:44 5:45 5:46 5:47 5:48 5:49 5:50 5:51 5:52 5:53 5:54 5:55 5:56 5:57 5:58 5:59 5:60 5:61 5:62 5:63 5:64 5:65 5:66 5:67 5:68 5:69 5:70 5:71 5:72 5:73 5:74 5:75 5:76 5:77 5:78 6 6:1 6:2 6:3 6:4 6:5 6:6 6:7 6:8 6:9 6:10 6:11 6:12 6:13 6:14 6:15 6:16 6:17 6:18 6:19 6:20 6:21 6:22 6:23 6:24 6:25 6:26 6:27 6:28 6:29 6:30 6:31 6:32 6:33 6:34 6:35 6:36 6:37 6:38 6:39 6:40 6:41 6:42 6:43 6:44 6:45 6:46 6:47 6:48 6:49]
`

	// fmt.Println("\nlayout:\n", lns.layout[0])
	assert.Equal(t, lay, fmt.Sprintln(lns.layout[0]))
}
