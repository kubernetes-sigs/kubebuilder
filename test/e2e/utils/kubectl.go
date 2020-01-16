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

package utils

import (
	"errors"
	"os/exec"
	"strings"
)

// Kubectl contains context to run kubectl commands
type Kubectl struct {
	*CmdContext
	Namespace string
}

// Command is a general func to run kubectl commands
func (k *Kubectl) Command(cmdOptions ...string) (string, error) {
	cmd := exec.Command("kubectl", cmdOptions...)
	output, err := k.Run(cmd)
	return string(output), err
}

// WithInput is a general func to run kubectl commands with input
func (k *Kubectl) WithInput(stdinInput string) *Kubectl {
	k.Stdin = strings.NewReader(stdinInput)
	return k
}

// CommandInNamespace is a general func to run kubectl commands in the namespace
func (k *Kubectl) CommandInNamespace(cmdOptions ...string) (string, error) {
	if len(k.Namespace) == 0 {
		return "", errors.New("namespace should not be empty")
	}
	return k.Command(append([]string{"-n", k.Namespace}, cmdOptions...)...)
}

// Apply is a general func to run kubectl apply commands
func (k *Kubectl) Apply(inNamespace bool, cmdOptions ...string) (string, error) {
	ops := append([]string{"apply"}, cmdOptions...)
	if inNamespace {
		return k.CommandInNamespace(ops...)
	} else {
		return k.Command(ops...)
	}
}

// Get is a func to run kubectl get commands
func (k *Kubectl) Get(inNamespace bool, cmdOptions ...string) (string, error) {
	ops := append([]string{"get"}, cmdOptions...)
	if inNamespace {
		return k.CommandInNamespace(ops...)
	} else {
		return k.Command(ops...)
	}
}

// Delete is a func to run kubectl delete commands
func (k *Kubectl) Delete(inNamespace bool, cmdOptions ...string) (string, error) {
	ops := append([]string{"delete"}, cmdOptions...)
	if inNamespace {
		return k.CommandInNamespace(ops...)
	} else {
		return k.Command(ops...)
	}
}

// Logs is a func to run kubectl logs commands
func (k *Kubectl) Logs(cmdOptions ...string) (string, error) {
	ops := append([]string{"logs"}, cmdOptions...)
	return k.CommandInNamespace(ops...)
}
