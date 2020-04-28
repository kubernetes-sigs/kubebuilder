/*
Copyright 2020 The Kubernetes Authors.

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

package internal

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// RunCmd prints the provided message and command and then executes it binding stdout and stderr
func RunCmd(msg, cmd string, args ...string) error {
	c := exec.Command(cmd, args...) //nolint:gosec
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	fmt.Println(msg + ":\n$ " + strings.Join(c.Args, " "))
	return c.Run()
}
