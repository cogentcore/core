// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package csl provides support for citation and reference generation
// using the industry-standard Citation Style Language.
package csl

//go:generate core generate

// https://citationstyles.org/developers/ -- implementations of CSL citation code -- none in go..
// https://github.com/citation-style-language/schema/blob/master/schemas/input/csl-data.json
// http://json-schema.org/draft-07/schema#",

// Item represents one item of CSL data.
type Item struct {
	Type                     Types          `json:"type"`
	ID                       string         `json:"id"`
	CitationKey              string         `json:"citation-key,omitempty"`
	Categories               []string       `json:"categories,omitempty"`
	Language                 string         `json:"language,omitempty"`
	JournalAbbreviation      string         `json:"journal-abbreviation,omitempty"`
	ShortTitle               string         `json:"shortTitle,omitempty"`
	Author                   []Name         `json:"author,omitempty"`
	Chair                    []Name         `json:"chair,omitempty"`
	CollectionEditor         []Name         `json:"collection-editor,omitempty"`
	Compiler                 []Name         `json:"compiler,omitempty"`
	Composer                 []Name         `json:"composer,omitempty"`
	ContainerAuthor          []Name         `json:"container-author,omitempty"`
	Contributor              []Name         `json:"contributor,omitempty"`
	Curator                  []Name         `json:"curator,omitempty"`
	Director                 []Name         `json:"director,omitempty"`
	Editor                   []Name         `json:"editor,omitempty"`
	EditorDirector           []Name         `json:"editorial-director,omitempty"`
	ExecutiveProducer        []Name         `json:"executive-producer,omitempty"`
	Guest                    []Name         `json:"guest,omitempty"`
	Host                     []Name         `json:"host,omitempty"`
	Interviewer              []Name         `json:"interviewer,omitempty"`
	Illustrator              []Name         `json:"illustrator,omitempty"`
	Narrator                 []Name         `json:"narrator,omitempty"`
	Organizer                []Name         `json:"organizer,omitempty"`
	OriginalAuthor           []Name         `json:"original-author,omitempty"`
	Performer                []Name         `json:"performer,omitempty"`
	Producer                 []Name         `json:"producer,omitempty"`
	Recipient                []Name         `json:"recipient,omitempty"`
	ReviewedAuthor           []Name         `json:"reviewed-author,omitempty"`
	ScriptWriter             []Name         `json:"script-writer,omitempty"`
	SeriesCreator            []Name         `json:"series-creator,omitempty"`
	Translator               []Name         `json:"translator,omitempty"`
	Accessed                 Date           `json:"accessed,omitempty"`
	AvailableDate            Date           `json:"available-date,omitempty"`
	EventDate                Date           `json:"event-date,omitempty"`
	Issued                   Date           `json:"issued,omitempty"`
	OriginalDate             Date           `json:"original-date,omitempty"`
	Submitted                Date           `json:"submitted,omitempty"`
	Abstract                 string         `json:"abstract,omitempty"`
	Annote                   string         `json:"annote,omitempty"`
	Archive                  string         `json:"archive,omitempty"`
	ArchiveCollection        string         `json:"archive_collection,omitempty"`
	ArchiveLocation          string         `json:"archive_location,omitempty"`
	ArchivePlace             string         `json:"archive-place,omitempty"`
	Authority                string         `json:"authority,omitempty"`
	CallNumber               string         `json:"call-number,omitempty"`
	ChapterNumber            string         `json:"chapter-number,omitempty"`
	CitationNumber           string         `json:"citation-number,omitempty"`
	CitationLabel            string         `json:"citation-label,omitempty"`
	CollectionNumber         string         `json:"collection-number,omitempty"`
	CollectionTitle          string         `json:"collection-title,omitempty"`
	ContainerTitle           string         `json:"container-title,omitempty"`
	ContainerTitleShort      string         `json:"container-title-short,omitempty"`
	Dimensions               string         `json:"dimensions,omitempty"`
	Division                 string         `json:"division,omitempty"`
	DOI                      string         `json:"DOI,omitempty"`
	Edition                  string         `json:"edition,omitempty"`
	Event                    string         `json:"event,omitempty"`
	EventTitle               string         `json:"event-title,omitempty"`
	EventPlace               string         `json:"event-place,omitempty"`
	FirstReferenceNoteNumber string         `json:"first-reference-note-number,omitempty"`
	Genre                    string         `json:"genre,omitempty"`
	ISBN                     string         `json:"ISBN,omitempty"`
	ISSN                     string         `json:"ISSN,omitempty"`
	Issue                    string         `json:"issue,omitempty"`
	Jurisdiction             string         `json:"jurisdiction,omitempty"`
	Keyword                  string         `json:"keyword,omitempty"`
	Locator                  string         `json:"locator,omitempty"`
	Medium                   string         `json:"medium,omitempty"`
	Note                     string         `json:"note,omitempty"`
	Number                   string         `json:"number,omitempty"`
	NumberOfPages            string         `json:"number-of-pages,omitempty"`
	NumberOfVolumes          string         `json:"number-of-volumes,omitempty"`
	OriginalPublisher        string         `json:"original-publisher,omitempty"`
	OriginalPublisherPlace   string         `json:"original-publisher-place,omitempty"`
	OriginalTitle            string         `json:"original-title,omitempty"`
	Page                     string         `json:"page,omitempty"`
	PageFirst                string         `json:"page-first,omitempty"`
	Part                     string         `json:"part,omitempty"`
	PartTitle                string         `json:"part-title,omitempty"`
	PMCID                    string         `json:"PMCID,omitempty"`
	PMID                     string         `json:"PMID,omitempty"`
	Printing                 string         `json:"printing,omitempty"`
	Publisher                string         `json:"publisher,omitempty"`
	PublisherPlace           string         `json:"publisher-place,omitempty"`
	References               string         `json:"references,omitempty"`
	ReviewedGenre            string         `json:"reviewed-genre,omitempty"`
	ReviewedTitle            string         `json:"reviewed-title,omitempty"`
	Scale                    string         `json:"scale,omitempty"`
	Section                  string         `json:"section,omitempty"`
	Source                   string         `json:"source,omitempty"`
	Status                   string         `json:"status,omitempty"`
	Supplement               string         `json:"supplement,omitempty"`
	Title                    string         `json:"title,omitempty"`
	TitleShort               string         `json:"title-short,omitempty"`
	URL                      string         `json:"URL,omitempty"`
	Version                  string         `json:"version,omitempty"`
	Volume                   string         `json:"volume,omitempty"`
	VolumeTitle              string         `json:"volume-title,omitempty"`
	VolumeTitleShort         string         `json:"volume-title-short,omitempty"`
	YearSuffix               string         `json:"year-suffix,omitempty"`
	Custom                   map[string]any `json:"custom,omitempty"`
}
