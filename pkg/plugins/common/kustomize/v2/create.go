/*
Copyright 2022 The Kubernetes Authors.

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

package v2

import (
	"strconv"

	"github.com/spf13/pflag"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
)

type createSubcommand struct {
	config   config.Config
	resource *resource.Resource

	flagSet *pflag.FlagSet

	// force indicates whether to scaffold files even if they exist.
	force bool
}

func (p *createSubcommand) BindFlags(fs *pflag.FlagSet) { p.flagSet = fs }

func (p *createSubcommand) InjectConfig(c config.Config) error {
	p.config = c
	return nil
}

func (p *createSubcommand) InjectResource(res *resource.Resource) error {
	p.resource = res
	return nil
}

func (p *createSubcommand) configure() (err error) {
	if forceFlag := p.flagSet.Lookup("force"); forceFlag != nil {
		if p.force, err = strconv.ParseBool(forceFlag.Value.String()); err != nil {
			return err
		}
	}
	return nil
}
