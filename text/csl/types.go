// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package csl

// Types are CSL content types.
type Types int32 //enums:enum -transform kebab

const (
	Article Types = iota
	ArticleJournal
	ArticleMagazine
	ArticleNewspaper
	Bill
	Book
	Broadcast
	Chapter
	Classic
	Collection
	Dataset
	Document
	Entry
	EntryDictionary
	EntryEncyclopedia
	Event
	Figure
	Graphic
	Hearing
	Interview
	LegalCase
	Legislation
	Manuscript
	Map
	MotionPicture
	MusicalScore
	Pamphlet
	PaperConference
	Patent
	Performance
	Periodical
	PersonalCommunication
	Post
	PostWeblog
	Regulation
	Report
	Review
	ReviewBook
	Software
	Song
	Speech
	Standard
	Thesis
	Treaty
	Webpage
)
