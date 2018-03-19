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

package predicates

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Predicates", func() {
	var ()

	BeforeEach(func() {
	})

	Describe("When checking the TrueMixin Predicate", func() {
		It("should return true for Add", func() {
			Expect(TrueMixin{}.HandleCreate("")).Should(BeTrue())
		})
		It("should return true for Update", func() {
			Expect(TrueMixin{}.HandleUpdate("", "")).Should(BeTrue())
		})
		It("should return true for Delete", func() {
			Expect(TrueMixin{}.HandleDelete("")).Should(BeTrue())
		})
	})

	Describe("When checking the FalseMixin Predicate", func() {
		It("should return true for Add", func() {
			Expect(FalseMixin{}.HandleCreate("")).Should(BeFalse())
		})
		It("should return true for Update", func() {
			Expect(FalseMixin{}.HandleUpdate("", "")).Should(BeFalse())
		})
		It("should return true for Delete", func() {
			Expect(FalseMixin{}.HandleDelete("")).Should(BeFalse())
		})
	})

	Describe("When checking a ResourceVersionChangedPredicate", func() {
		Context("Where the old object doesn't have a ResourceVersion", func() {
			It("should return false", func() {
				instance := ResourceVersionChangedPredicate{}
				Expect(instance.HandleDelete("")).Should(BeTrue())
				Expect(instance.HandleCreate("")).Should(BeTrue())
				Expect(instance.HandleUpdate(&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						ResourceVersion: "1",
					},
				}, "")).Should(BeFalse())
			})
		})

		Context("Where the new object doesn't have a ResourceVersion", func() {
			It("should return false", func() {
				instance := ResourceVersionChangedPredicate{}
				Expect(instance.HandleDelete("")).Should(BeTrue())
				Expect(instance.HandleCreate("")).Should(BeTrue())
				Expect(instance.HandleUpdate("", &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						ResourceVersion: "1",
					},
				})).Should(BeFalse())
			})
		})

		Context("Where the ResourceVersion hasn't changed", func() {
			It("should return false", func() {
				instance := ResourceVersionChangedPredicate{}
				Expect(instance.HandleDelete("")).Should(BeTrue())
				Expect(instance.HandleCreate("")).Should(BeTrue())
				Expect(instance.HandleUpdate(&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						ResourceVersion: "1",
					},
				}, &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						ResourceVersion: "1",
					},
				})).Should(BeFalse())
			})
		})

		Context("Where the ResourceVersion has changed", func() {
			It("should return true", func() {
				instance := ResourceVersionChangedPredicate{}
				Expect(instance.HandleDelete("")).Should(BeTrue())
				Expect(instance.HandleCreate("")).Should(BeTrue())
				Expect(instance.HandleUpdate(&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						ResourceVersion: "1",
					},
				}, &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						ResourceVersion: "2",
					},
				})).Should(BeTrue())
			})
		})
	})
})
