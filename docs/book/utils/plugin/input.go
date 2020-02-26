/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package plugin

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// Context is a (partial) mdBook execution context.
type Context struct {
	Root   string `json:"root"`
	Config Config `json:"config"`
}

// Config is a (partial) mdBook config
type Config struct {
	Book BookConfig `json:"book"`
}

// BookConfig is a (partial) mdBook [book] stanza
type BookConfig struct {
	Src string `json:"src"`
}

// Book is an mdBook book.
type Book struct {
	Sections      []BookItem `json:"sections"`
	NonExhaustive *struct{}  `json:"__non_exhaustive"`
}

// BookSection is an mdBook section.
type BookSection struct {
	Items []BookItem `json:"items"`
}

// BookItem is an mdBook item.
// It wraps an underlying struct to provide proper marshalling and unmarshalling
// according to what serde produces/expects.
type BookItem bookItem

// UnmarshalJSON implements encoding/json.Unmarshaler
func (b *BookItem) UnmarshalJSON(input []byte) error {
	// match how serde serializes rust enums.
	if input[0] == '"' {
		// actually a an empty variant
		var variant string
		if err := json.Unmarshal(input, &variant); err != nil {
			return err
		}
		switch variant {
		case "Separator":
			b.Separator = true
		default:
			return fmt.Errorf("unknown book item variant %s", variant)
		}
		return nil
	}

	item := bookItem(*b)
	if err := json.Unmarshal(input, &item); err != nil {
		return err
	}
	*b = BookItem(item)
	return nil
}

// MarshalJSON implements encoding/json.Marshaler
func (b BookItem) MarshalJSON() ([]byte, error) {
	if b.Separator {
		return json.Marshal("Separator")
	}

	return json.Marshal(bookItem(b))
}

// bookItem is the underlying mdBook item without custom serialization.
type bookItem struct {
	Chapter   *BookChapter `json:"Chapter"`
	Separator bool         `json:"-"`
}

// BookChapter is an mdBook chapter.
type BookChapter struct {
	Name        string        `json:"name"`
	Content     string        `json:"content"`
	Number      SectionNumber `json:"number"`
	SubItems    []BookItem    `json:"sub_items"`
	Path        string        `json:"path"`
	ParentNames []string      `json:"parent_names"`
}

// SectionNumber is an mdBook section number (e.g. `1.2` is `{1,2}`).
type SectionNumber []uint32

// Input is the tuple that's presented to mdBook plugins.
// It's deserialized from a slice `[context, book]`, matching
// a Rust tuple.
type Input struct {
	Context Context
	Book    Book
}

// UnmarshalJSON implements encoding/json.Unmarshaler
func (p *Input) UnmarshalJSON(input []byte) error {
	// deserialize from the JSON equivalent to the Rust tuple
	// `(context, book)`
	inputBuffer := bytes.NewBuffer(input)
	dec := json.NewDecoder(inputBuffer)

	tok, err := dec.Token()
	if err != nil {
		return err
	}
	if delim, isDelim := tok.(json.Delim); !isDelim || delim != '[' {
		return fmt.Errorf("expected [, got %s", tok)
	}
	if err := dec.Decode(&p.Context); err != nil {
		return err
	}
	if err := dec.Decode(&p.Book); err != nil {
		return err
	}
	tok, err = dec.Token()
	if err != nil {
		return err
	}
	if delim, isDelim := tok.(json.Delim); !isDelim || delim != ']' {
		return fmt.Errorf("expected ], got %s", tok)
	}

	return nil
}

// ChapterVisitor visits each BookChapter in a book, getting an actual
// pointer to the chapter that it can modify.
type ChapterVisitor func(*BookChapter) error

// EachItem calls the given visitor for each chapter in the given item,
// passing a pointer to the actual chapter that the visitor can modify.
func EachItem(parentItem *BookItem, visitor ChapterVisitor) error {
	if parentItem.Chapter == nil {
		return nil
	}

	if err := visitor(parentItem.Chapter); err != nil {
		return err
	}

	// pass a pointer to the structure, not the iteration variable
	for i := range parentItem.Chapter.SubItems {
		if err := EachItem(&parentItem.Chapter.SubItems[i], visitor); err != nil {
			return err
		}
	}

	return nil
}

// EachItemInBook functions identically to EachItem, except that it visits
// all chapters in the book.
func EachItemInBook(book *Book, visitor ChapterVisitor) error {
	// pass a pointer to the structure, not the iteration variable
	for i := range book.Sections {
		if err := EachItem(&book.Sections[i], visitor); err != nil {
			return err
		}
	}
	return nil
}
