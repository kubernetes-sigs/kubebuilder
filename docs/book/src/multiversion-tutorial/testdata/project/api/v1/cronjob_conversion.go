/*

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
// +kubebuilder:docs-gen:collapse=Apache License

package v1

/*
实现 hub 方法相当容易 -- 我们只需要添加一个叫做 `Hub()` 的空方法来作为一个 [标记](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/conversion?tab=doc#Hub)。我们也可以将这行代码放到 `cronjob_types.go` 文件中。
*/

// Hub 标记这个类型是一个用来转换的 hub。
func (*CronJob) Hub() {}
