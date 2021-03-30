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
	"crypto/rand"
	"math/big"
	"strings"

	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
)

const (
	// KubebuilderBinName define the name of the kubebuilder binary to be used in the tests
	KubebuilderBinName = "kubebuilder"
)

// RandomSuffix returns a 4-letter string.
func RandomSuffix() (string, error) {
	source := []rune("abcdefghijklmnopqrstuvwxyz")
	res := make([]rune, 4)
	for i := range res {
		bi := new(big.Int)
		r, err := rand.Int(rand.Reader, bi.SetInt64(int64(len(source))))
		if err != nil {
			return "", err
		}
		res[i] = source[r.Int64()]
	}
	return string(res), nil
}

// GetNonEmptyLines converts given command output string into individual objects
// according to line breakers, and ignores the empty elements in it.
func GetNonEmptyLines(output string) []string {
	var res []string
	elements := strings.Split(output, "\n")
	for _, element := range elements {
		if element != "" {
			res = append(res, element)
		}
	}

	return res
}

// ImplementWebhooks will mock an webhook data
func ImplementWebhooks(fs machinery.Filesystem, filename string) error {
	return machinery.Replace(
		fs,
		filename,

		"import (",
		`import (
	"errors"`,

		"// TODO(user): fill in your defaulting logic.",
		`if r.Spec.Count == 0 {
		r.Spec.Count = 5
	}`,

		"// TODO(user): fill in your validation logic upon object creation.",
		`if r.Spec.Count < 0 {
		return errors.New(".spec.count must >= 0")
	}`,

		"// TODO(user): fill in your validation logic upon object update.",
		`if r.Spec.Count < 0 {
		return errors.New(".spec.count must >= 0")
	}`,
	)
}
