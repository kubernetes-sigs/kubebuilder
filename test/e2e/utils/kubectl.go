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
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// Kubectl contains context to run kubectl commands
type Kubectl struct {
	*CmdContext
	Namespace      string
	ServiceAccount string
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
	}
	return k.Command(ops...)
}

// Get is a func to run kubectl get commands
func (k *Kubectl) Get(inNamespace bool, cmdOptions ...string) (string, error) {
	ops := append([]string{"get"}, cmdOptions...)
	if inNamespace {
		return k.CommandInNamespace(ops...)
	}
	return k.Command(ops...)
}

// Delete is a func to run kubectl delete commands
func (k *Kubectl) Delete(inNamespace bool, cmdOptions ...string) (string, error) {
	ops := append([]string{"delete"}, cmdOptions...)
	if inNamespace {
		return k.CommandInNamespace(ops...)
	}
	return k.Command(ops...)
}

// Logs is a func to run kubectl logs commands
func (k *Kubectl) Logs(cmdOptions ...string) (string, error) {
	ops := append([]string{"logs"}, cmdOptions...)
	return k.CommandInNamespace(ops...)
}

// Wait is a func to run kubectl wait commands
func (k *Kubectl) Wait(inNamespace bool, cmdOptions ...string) (string, error) {
	ops := append([]string{"wait"}, cmdOptions...)
	if inNamespace {
		return k.CommandInNamespace(ops...)
	}
	return k.Command(ops...)
}

// VersionInfo holds a subset of client/server version information.
type VersionInfo struct {
	Major      string `json:"major"`
	Minor      string `json:"minor"`
	GitVersion string `json:"gitVersion"`

	// Leaving major/minor int fields unexported prevents them from being set
	// while leaving their exported counterparts untouched -> incorrect marshaled format.
	major, minor uint64
}

// GetMajorInt returns the uint64 representation of vi.Major.
func (vi VersionInfo) GetMajorInt() uint64 { return vi.major }

// GetMinorInt returns the uint64 representation of vi.Minor.
func (vi VersionInfo) GetMinorInt() uint64 { return vi.minor }

func (vi *VersionInfo) parseVersionInts() (err error) {
	if vi.Major != "" {
		if vi.major, err = strconv.ParseUint(vi.Major, 10, 64); err != nil {
			return fmt.Errorf("error parsing major version %q: %w", vi.Major, err)
		}
	}
	if vi.Minor != "" {
		if vi.minor, err = strconv.ParseUint(vi.Minor, 10, 64); err != nil {
			return fmt.Errorf("error parsing minor version %q: %w", vi.Minor, err)
		}
	}
	return nil
}

// KubernetesVersion holds a subset of both client and server versions.
type KubernetesVersion struct {
	ClientVersion VersionInfo `json:"clientVersion,omitempty"`
	ServerVersion VersionInfo `json:"serverVersion,omitempty"`
}

func (v *KubernetesVersion) prepare() error {
	if err := v.ClientVersion.parseVersionInts(); err != nil {
		return err
	}
	return v.ServerVersion.parseVersionInts()
}

// Version is a func to run kubectl version command
func (k *Kubectl) Version() (ver KubernetesVersion, err error) {
	out, err := k.Command("version", "-o", "json")
	if err != nil {
		return KubernetesVersion{}, fmt.Errorf("error getting kubernetes version: %w", err)
	}
	if decodeErr := ver.decode(out); decodeErr != nil {
		return KubernetesVersion{}, fmt.Errorf("error parsing kubernetes version: %w", decodeErr)
	}
	return ver, nil
}

func (v *KubernetesVersion) decode(out string) error {
	dec := json.NewDecoder(strings.NewReader(out))
	if err := dec.Decode(&v); err != nil {
		return fmt.Errorf("error decoding kubernetes version: %w", err)
	}
	return v.prepare()
}
