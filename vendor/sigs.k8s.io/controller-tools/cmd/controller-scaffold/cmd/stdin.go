/*
Copyright 2018 The Kubernetes Authors.

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

package cmd

import (
	"bufio"
	"log"
	"os"
	"strings"
)

// yesno reads from stdin looking for one of "y", "yes", "n", "no" and returns
// true for "y" and false for "n"
func yesno() bool {
	reader := bufio.NewReader(os.Stdin)
	for {
		switch readstdin(reader) {
		case "y", "yes":
			return true
		case "n", "no":
			return false
		}
	}
}

// readstdin reads a line from stdin trimming spaces, and returns the value.
// log.Fatal's if there is an error.
func readstdin(reader *bufio.Reader) string {
	text, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal(err)
	}
	return strings.TrimSpace(text)
}
