package main

import (
	"fmt"
	"os"
	"io/ioutil"
	"path/filepath"
	"go/scanner"
	"go/token"
	"log"
	"strings"
	"unicode"

	"sigs.k8s.io/kubebuilder/docs/book/utils/plugin"
)

// Literate is a plugin that extracts block comments from Go source and
// interleaves them with the surrounding code as fenced code blocks.
// It should suppot all output formats.
// It's triggered by using the an expression like `{{#literatego ./path/to/source/file.go}}`.
// The marker `+kubebuilder:docs-gen:collapse=<string>` can be used to collapse a description/code
// pair into a details block with the given summary.
type Literate struct {}
func (_ Literate) SupportsOutput(_ string) bool { return true }
func (_ Literate) Process(input *plugin.Input) error {
	return plugin.EachItemInBook(&input.Book, func(chapter *plugin.BookChapter) error {
		if chapter.Content == "" {
			return nil
		}

		// figure out all the trigger expressins
		partsRaw := strings.Split(chapter.Content, "{{#literatego ")
		// the first section won't start with `{{#literatego ` as per how split works
		if len(partsRaw) < 2 {
			return nil
		}

		var res []string
		res = append(res, partsRaw[0])
		for _, part := range partsRaw[1:] {
			endDelim := strings.Index(part, "}}")
			if endDelim < 0 {
				return fmt.Errorf("missing end delimiter in chapter %q", chapter.Name)
			}
			// we need to join the path with the context root and the book's
			// source directory, since we assume paths are relative to the
			// given chapter file, like `{{#include}}`
			relPath := part[:endDelim]
			path := filepath.Join(input.Context.Root, input.Context.Config.Book.Src, filepath.Dir(chapter.Path), relPath)

			// TODO(directxman12): don't escape root?
			contents, err := ioutil.ReadFile(path)
			if err != nil {
				return fmt.Errorf("unable to import %q: %v", path, err)
			}

			newContents, err := extractContents(contents, path)
			if err != nil {
				return fmt.Errorf("unable to process %q: %v", path, err)
			}

			res = append(res, string(newContents))
			res = append(res, part[endDelim+2:])
		}

		chapter.Content = strings.Join(res, "")
		return nil
	})
}

// commentCodePair represents a block of code with some text before it, optionally
// marked as collapsed with the given "collapse summary".
type commentCodePair struct {
	comment string
	code string
	collapse string
}

// collapsePrefix is the marker comment that indicates that the previous commentCodePair
// should be collapsed with the given summary
var collapsePrefix = "+kubebuilder:docs-gen:collapse="

// getCollapse checks if the given token is a collapse marker, and
// extracts the summary if so.
func getCollapse(tok token.Token, lit string) string {
	if tok != token.COMMENT {
		return ""
	}

	if lit[:2] != "//" {
		return ""
	}
	rest := strings.TrimSpace(lit[2:])
	if !strings.HasPrefix(rest, collapsePrefix) {
		return ""
	}

	return rest[len(collapsePrefix):]
}

// isBlockComment checks that the given token is a `/* comment */`-style comment,
// which we consider to be the start of a codeCommentPair
func isBlockComment(tok token.Token, lit string) bool {
	if tok != token.COMMENT {
		return false
	}

	if len(lit) < 3 || lit[0] != '/' || lit[1] != '*' {
		return false
	}

	return true
}

// commentText extracts the text from the given comment, slicing off
// some common amount of whitespace prefix.
func commentText(raw string, lineOffset int) string {
	rawBody := raw[2:len(raw)-2] // chop of the delimiters
	lines := strings.Split(rawBody, "\n")
	if len(lines) == 0 {
		return ""
	}

	for i, line := range lines {
		offset := lineOffset
		if len(line) < offset {
			offset = len(line)
		}
		lines[i] = strings.TrimLeftFunc(line[:offset], unicode.IsSpace) + line[offset:]
	}

	return strings.Join(lines, "\n")
}

// extractPairs extracts all commentCodePairs from the given source code with
// the given path.  A block starts as soon as the last block ends (or at the
// beginning of the file), and ends as soon as a block comment is encountered,
// or if a collapse marker is encountered.
func extractPairs(contents []byte, path string) ([]commentCodePair, error) {
	fileSet := token.NewFileSet()
	file := fileSet.AddFile(path, -1, len(contents))
	scan := scanner.Scanner{}
	var errs []error
	scan.Init(file, []byte(contents), func(pos token.Position, msg string) {
		errs = append(errs, fmt.Errorf("error parsing file %s: %s", pos, msg))
	}, scanner.ScanComments)

	// grab all the different sections
	var pairs []commentCodePair
	var lastPair commentCodePair
	lastCodeBlockStart := 0

	var tok token.Token
	for tok != token.EOF {
		var pos token.Pos
		var lit string
		pos, tok, lit = scan.Scan()
		collapse := getCollapse(tok, lit)
		if collapse != "" {
			lastPair.collapse = collapse
		}
		if collapse == "" && !isBlockComment(tok, lit) {
			continue
		}
		codeEnd := file.Offset(pos)-1
		if codeEnd - lastCodeBlockStart > 0 {
			lastPair.code = string(contents[lastCodeBlockStart:codeEnd])
		}
		pairs = append(pairs, lastPair)
		if collapse == "" {
			line := file.Line(pos)
			lineStart := file.LineStart(line)
			lastPair = commentCodePair{
				comment: commentText(lit, file.Offset(pos) - file.Offset(lineStart)),
			}
		} else {
			lastPair = commentCodePair{}
		}
		lastCodeBlockStart = file.Offset(pos)+len(lit)
	}
	lastPair.code = string(contents[lastCodeBlockStart:])
	pairs = append(pairs, lastPair)

	if len(errs) > 0 {
		return nil, errs[0]
	}
	return pairs, nil
}

// extractContents extracts comment-code pairs from the given named file
// contents, and then renders the result to markdown.
func extractContents(contents []byte, path string) (string, error) {
	pairs, err := extractPairs(contents, path)
	if err != nil {
		return "", err
	}

	out := new(strings.Builder)
	
	for _, pair := range pairs {
		if pair.collapse != "" {
			out.WriteString("<details><summary>")
			out.WriteString(pair.collapse)
			out.WriteString("</summary>")
		}
		if strings.TrimSpace(pair.comment) != "" {
			out.WriteString(pair.comment)
		}

		if strings.TrimSpace(pair.code) != "" {
			out.WriteString("\n```go\n")
			out.WriteString(pair.code)
			out.WriteString("\n```\n")
		}
		if pair.collapse != ""{
			out.WriteString("</details>")
		}
		// TODO(directxman12): nice side-by-side sections
	}

	return out.String(), nil
}

func main() {
	if err := plugin.Run(Literate{}, os.Stdin, os.Stdout, os.Args[1:]...); err != nil {
		log.Fatal(err.Error())
	}
}
