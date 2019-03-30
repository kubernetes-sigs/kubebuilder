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

package main

import (
	"os"
	"sync"
	"io/ioutil"
	"io"
	"fmt"
	"path/filepath"
	"go/scanner"
	"go/token"
	"strings"
	"flag"
)

func errExit(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, msg, args...)
	os.Exit(1)
}

type link struct {
	file string
	section string
}

func makeLink(currDir, currFile, args string) *link {
	argParts := strings.SplitN(args, ":", 2)
	actPath := argParts[0]
	if argParts[0] == "." {
		actPath = filepath.Join(currDir, currFile)
	} else if actPath[0] != '/' {
		actPath = filepath.Join(currDir, argParts[0])
	}
	res := &link{
		file: actPath,
	}
	if len(argParts) > 1 {
		res.section = argParts[1]
	}
	return res
}

type docBlock struct {
	doc string
	code string
	codeFormat string

	linkTo *link
	call *link
	name string
	terminal bool
}

type parsedFile struct {
	name string
	blocks []docBlock
}

type fileTraverser struct {
	fileSet *token.FileSet
	files map[string]*parsedFile
	filesMu sync.Mutex

	done sync.WaitGroup
	errChan chan error

	rootPath string
}

func (t *fileTraverser) TraverseFile(relPath string) {
	defer t.done.Done()

	t.filesMu.Lock()
	_, present := t.files[relPath]
	if present {
		// skip already parsed files
		t.filesMu.Unlock()
		return
	}

	parsed := &parsedFile{name: relPath}
	t.files[relPath] = parsed
	t.filesMu.Unlock()

	absishPath := filepath.Join(t.rootPath, relPath)
	contents, err := ioutil.ReadFile(absishPath)
	if err != nil {
		t.errChan <- err
		return
	}

	pairs := t.parsePairs(relPath, contents)

	for _, pair := range pairs {
		if pair == (commentCodePair{}) {
			// skip empty pairs
			continue
		}
		lines := strings.Split(pair.comment, "\n")

		currDir, currFile := filepath.Split(relPath)
		outLines := make([]string, 0, len(lines))
		block := docBlock{
			code: pair.code,
			codeFormat: "go",
		}
		for _, line := range lines {
			trimmedLine := strings.TrimSpace(line)
			if len(trimmedLine) == 0 || trimmedLine[0] != '+' {
				outLines = append(outLines, line)
				continue
			}
			cmdParts := strings.SplitN(trimmedLine[1:], " ", 2)
			switch cmdParts[0] {
			case "goto":
				block.linkTo = makeLink(currDir, currFile, cmdParts[1])
			case "call":
				block.call = makeLink(currDir, currFile, cmdParts[1])
			case "return":
				block.name = cmdParts[1]
			default:
				t.errChan <- fmt.Errorf("unrecognized directive %q (%q) in file %s", cmdParts[0], line, relPath)
			}
		}
		block.doc = strings.Join(outLines, "\n")
		parsed.blocks = append(parsed.blocks, block)
	}

	for _, block := range parsed.blocks {
		if block.linkTo != nil {
			t.done.Add(1)
			go t.TraverseFile(block.linkTo.file)
		}
		if block.call != nil {
			t.done.Add(1)
			go t.TraverseFile(block.call.file)
		}
	}
}

func (t *fileTraverser) parsePairs(relPath string, contents []byte) []commentCodePair {
	file := t.fileSet.AddFile(relPath, -1, len(contents))
	scan := scanner.Scanner{}
	scan.Init(file, contents, func(pos token.Position, msg string) {
		t.errChan <- fmt.Errorf("error parsing file %s @ %s: %s", relPath, pos, msg)
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
		if !isDocComment(tok, lit) {
			continue
		}
		codeEnd := file.Offset(pos)-1
		if codeEnd - lastCodeBlockStart > 0 {
			lastPair.code = string(contents[lastCodeBlockStart:codeEnd])
		}
		pairs = append(pairs, lastPair)
		lastPair = commentCodePair{
			comment: lit[2:len(lit)-2],  // chop off delimitters
		}
		lastCodeBlockStart = file.Offset(pos)+len(lit)
	}
	lastPair.code = string(contents[lastCodeBlockStart:])
	pairs = append(pairs, lastPair)

	return pairs
}

func isDocComment(tok token.Token, lit string) bool {
	if tok != token.COMMENT {
		return false
	}

	if len(lit) < 3 || lit[0] != '/' || lit[1] != '*' {
		return false
	}

	return true
}

type commentCodePair struct {
	comment string
	code string
}

func (t *fileTraverser) Wait() bool {
	hadErrors := false
	go func() {
		t.done.Wait()
		close(t.errChan)
	}()

	for nextErr := range t.errChan {
		hadErrors = true
		fmt.Fprintf(os.Stderr, "error traversing files: %v\n", nextErr)
	}

	return hadErrors
}

func (t *fileTraverser) Init() {
	t.done.Add(1)
}

type fileMap map[string]*parsedFile

func (t *fileTraverser) Files() fileMap {
	return t.files
}

type page struct {
	blocks []docBlock
}

func makePages(rootFile string, files fileMap) []page {
	var pages []page
	var currentPage page
	currentLink := &link{file: rootFile}
	for {
		thisFile := files[currentLink.file]
		thisLink := *currentLink
		skipTillSection := (thisLink.section != "")
		currentLink = nil
		for _, block := range thisFile.blocks {
			if block.name == thisLink.section {
				skipTillSection = false
			}
			if skipTillSection {
				continue
			}
			currentPage.blocks = append(currentPage.blocks, block)
			if block.terminal {
				break
			}
			if block.linkTo == nil {
				continue
			}
			pages = append(pages, currentPage)
			currentPage = page{}
			currentLink = block.linkTo
			break
		}
		if currentLink == nil {
			break // we got to the end
		}
	}
	if len(currentPage.blocks) > 0 {
		pages = append(pages, currentPage)
	}
	return pages
}

var (
	outputDir = flag.String("o", "", "path to output pages into")
)

func makeFragmentName(raw string) string {
	return strings.Map(func(in rune) rune {
		// alpha or digit, just to be safe
		if in >= 48 && in <= 57 {
			return in
		}
		if in >= 65 && in <= 90 {
			return in
		}
		if in >= 97 && in <= 122 {
			return in
		}
		return '-'
	}, raw)
}

func main() {
	flag.Parse()
	if flag.NArg() != 1 {
		errExit("USAGE: docgen [-o DIR] rootfile.go\n")
	}

	rootFile := flag.Arg(0)
	rootPath := filepath.Dir(rootFile)
	traverser := &fileTraverser{
		fileSet: token.NewFileSet(),
		files: make(map[string]*parsedFile),
		rootPath: rootPath,
		errChan: make(chan error, 100),
	}
	traverser.Init()
	traverser.TraverseFile(filepath.Base(rootFile))
	if traverser.Wait() {
		errExit("unable to successfully traverse files")
	}

	files := traverser.Files()
	pages := makePages(filepath.Base(rootFile), files)

	for i, page := range pages {
		pageName := fmt.Sprintf("page-%v.md", i) /* TODO: better names */
		func() {
			var out io.Writer
			if *outputDir != "" {
				pagePath := filepath.Join(*outputDir, pageName)
				outFile, err := os.Create(pagePath)
				if err != nil {
					errExit("unable to write page %q: %v", pagePath, err)
				}
				defer outFile.Close()
				out = outFile
			} else {
				out = os.Stdout
				fmt.Fprintf(out, "%s\n---\n\n", pageName)
				defer func() { fmt.Println("---") }()
			}
			for _, block := range page.blocks {
				hasCode := strings.TrimSpace(block.code) != ""
				if hasCode {
					fmt.Fprint(out, "{% method %}\n")
				}
				fmt.Fprintf(out, "%s\n", block.doc)
				if hasCode {
					fmt.Fprintf(out, "```%s\n%s\n```\n", block.codeFormat, block.code)
				}

				if block.call != nil {
					normalName := makeFragmentName(block.call.section)
					fmt.Fprintf(out, "<a name=\"return-from-%s\"></a>\n[%s](#jump-to-%s)\n", normalName, block.call.section, normalName)
				}
				if block.name != "" {
					normalName := makeFragmentName(block.name)
					fmt.Fprintf(out, "<a name=\"jump-to-%s\"></a>\n[return](#return-from-%s)\n", normalName, block.name, normalName)
				}

				if hasCode {
					fmt.Fprint(out, "{%% endmethod %%}\n\n")
				}
			}

			if i+1 < len(pages) {
				fmt.Fprintf(out, "\n[Next](./page-%v.md)\n", i+1)
			}
		}()
	}
}
