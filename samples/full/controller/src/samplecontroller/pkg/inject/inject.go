/*
Copyright 2017 The Kubernetes Authors.

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


package inject

import (
    "github.com/kubernetes-sigs/kubebuilder/pkg/inject/run"
    injectargs "github.com/kubernetes-sigs/kubebuilder/pkg/inject/args"

    "samplecontroller/pkg/inject/args"
)

var (
    // Inject is used to add items to the Injector
    Inject []func(args.InjectArgs) error

    // Injector runs items
    Injector injectargs.Injector
)

// RunAll starts all of the informers and Controllers
func RunAll(rargs run.RunArguments, iargs args.InjectArgs) error {
    // Run functions to initialize injector
    for _, i := range Inject {
        if err := i(iargs); err != nil {
            return err
        }
    }
    Injector.Run(rargs)
    <-rargs.Stop
    return nil
}
