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
	"fmt"
	"strings"
)

// EachCommand looks for mustache-like declarations of the form `{{#cmd args}}`
// in each book chapter, and calls the callback for each one, substituting it
// for the result.
func EachCommand(book *Book, cmd string, callback func(chapter *BookChapter, args string) (string, error)) error {
	cmdStart := fmt.Sprintf("{{#%s ", cmd)
	return EachItemInBook(book, func(chapter *BookChapter) error {
		if chapter.Content == "" {
			return nil
		}

		// figure out all the trigger expressions
		partsRaw := strings.Split(chapter.Content, cmdStart)
		// the first section won't start with `{{#<cmd> ` as per how split works
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
			newContents, err := callback(chapter, part[:endDelim])
			if err != nil {
				return err
			}
			res = append(res, string(newContents))
			res = append(res, part[endDelim+2:])
		}

		chapter.Content = strings.Join(res, "")
		return nil
	})
}
