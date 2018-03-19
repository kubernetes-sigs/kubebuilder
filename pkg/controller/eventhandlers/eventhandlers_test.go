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

package eventhandlers_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/kubernetes-sigs/kubebuilder/pkg/controller/eventhandlers"
	"github.com/kubernetes-sigs/kubebuilder/pkg/controller/types"

	"fmt"
	//"github.com/kubernetes-sigs/kubebuilder/pkg/controller/predicates"
	"github.com/kubernetes-sigs/kubebuilder/pkg/controller/predicates"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/workqueue"
)

var _ = Describe("Eventhandlers", func() {
	var (
		t   = true
		mae = eventhandlers.MapAndEnqueue{}
		q   = workqueue.NewNamedRateLimitingQueue(workqueue.NewMaxOfRateLimiter(), "world")
	)

	BeforeEach(func() {
		mae = eventhandlers.MapAndEnqueue{
			Map: func(i interface{}) string { return fmt.Sprintf("p-%v", i) },
		}
		q = workqueue.NewNamedRateLimitingQueue(workqueue.NewMaxOfRateLimiter(
			workqueue.NewItemExponentialFailureRateLimiter(0, 0),
		), "world")
	})

	Describe("When mapping and enqueuing an event", func() {
		Context("Where there are no Predicates", func() {
			It("should set the Add function", func() {
				fns := mae.Get(q)
				fns.AddFunc("add")
				Eventually(q.Len).Should(Equal(1))
				Expect(q.Get()).Should(Equal("p-add"))
			})

			It("should set the Delete function", func() {
				fns := mae.Get(q)
				fns.DeleteFunc("delete")
				Eventually(q.Len()).Should(Equal(1))
				Expect(q.Get()).Should(Equal("p-delete"))
			})

			It("should set the Update function", func() {
				fns := mae.Get(q)
				fns.UpdateFunc("old", "update")
				Eventually(q.Len()).Should(Equal(1))
				Expect(q.Get()).Should(Equal("p-update"))
			})
		})

		Context("Where there is one true Predicate", func() {
			It("should set the Add function", func() {
				mae.Predicates = []predicates.Predicate{FakePredicates{create: true}}
				fns := mae.Get(q)
				fns.AddFunc("add")
				Eventually(q.Len()).Should(Equal(1))
				Expect(q.Get()).Should(Equal("p-add"))

				fns.DeleteFunc("delete")
				fns.UpdateFunc("old", "update")
				Consistently(q.Len).Should(Equal(0))
			})

			It("should set the Delete function", func() {
				mae.Predicates = []predicates.Predicate{FakePredicates{delete: true}}
				fns := mae.Get(q)
				fns.DeleteFunc("delete")
				Eventually(q.Len()).Should(Equal(1))
				Expect(q.Get()).Should(Equal("p-delete"))

				fns.AddFunc("add")
				fns.UpdateFunc("old", "add")
				Consistently(q.Len).Should(Equal(0))
			})

			It("should set the Update function", func() {
				mae.Predicates = []predicates.Predicate{FakePredicates{update: true}}
				fns := mae.Get(q)
				fns.UpdateFunc("old", "update")
				Eventually(q.Len()).Should(Equal(1))
				Expect(q.Get()).Should(Equal("p-update"))

				fns.AddFunc("add")
				fns.DeleteFunc("delete")
				Consistently(q.Len).Should(Equal(0))
			})
		})

		Context("Where there are both true and false Predicates", func() {
			Context("Where there is one false Predicate", func() {
				It("should not Add", func() {
					mae.Predicates = []predicates.Predicate{FakePredicates{create: true}, FakePredicates{}}
					fns := mae.Get(q)
					fns.AddFunc("add")
					Consistently(q.Len).Should(Equal(0))
				})

				It("should not Delete", func() {
					mae.Predicates = []predicates.Predicate{FakePredicates{delete: true}, FakePredicates{}}
					fns := mae.Get(q)
					fns.DeleteFunc("delete")
					Consistently(q.Len).Should(Equal(0))
				})

				It("should not Update", func() {
					mae.Predicates = []predicates.Predicate{FakePredicates{update: true}, FakePredicates{}}
					fns := mae.Get(q)
					fns.UpdateFunc("old", "update")
					Consistently(q.Len).Should(Equal(0))
				})

				It("should not Add", func() {
					mae.Predicates = []predicates.Predicate{FakePredicates{}, FakePredicates{create: true}}
					fns := mae.Get(q)
					fns.AddFunc("add")
					Consistently(q.Len).Should(Equal(0))
				})

				It("should not Delete", func() {
					mae.Predicates = []predicates.Predicate{FakePredicates{}, FakePredicates{delete: true}}
					fns := mae.Get(q)
					fns.DeleteFunc("delete")
					Consistently(q.Len).Should(Equal(0))
				})

				It("should not Update", func() {
					mae.Predicates = []predicates.Predicate{FakePredicates{}, FakePredicates{update: true}}
					fns := mae.Get(q)
					fns.UpdateFunc("old", "update")
					Consistently(q.Len).Should(Equal(0))
				})
			})
		})
	})

	Describe("When mapping an object to itself", func() {
		Context("Where the object has key metadata", func() {
			It("should return the reconcile key for itself", func() {
				result := eventhandlers.MapToSelf(&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pod",
						Namespace: "not-default",
						OwnerReferences: []metav1.OwnerReference{
							{
								Name:       "test-replicaset",
								Controller: &t,
								UID:        "uid5",
							},
						},
					},
				})
				Expect(result).Should(Equal("not-default/test-pod"))
			})
		})

		Context("Where the object does not have key metadata", func() {
			It("should return the empty string", func() {
				obj := ""
				result := eventhandlers.MapToSelf(&obj)
				Expect(result).Should(Equal(""))
			})
		})
	})

	Describe("When mapping events for an object to the objects controller", func() {
		var (
			mtc = eventhandlers.MapToController{}
		)
		BeforeEach(func() {
			mtc = eventhandlers.MapToController{}
		})

		Context("Where the object doesn't have metadata", func() {
			It("should return the empty string", func() {
				s := ""
				result := mtc.Map(&s)
				Expect(result).Should(Equal(""))
			})
		})

		Context("Where the path is empty", func() {
			It("should return the empty string", func() {
				result := mtc.Map(&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pod",
						Namespace: "default",
						OwnerReferences: []metav1.OwnerReference{
							{
								Name:       "test-replicaset",
								Controller: &t,
								UID:        "uid5",
							},
						},
					},
				})
				Expect(result).Should(Equal(""))
			})
		})

		Context("Where the controller isn't found", func() {
			It("should return the empty string", func() {
				mtc.Path = eventhandlers.Path{
					func(k types.ReconcileKey) (interface{}, error) {
						return nil, nil
					},
				}
				result := mtc.Map(&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pod",
						Namespace: "default",
						OwnerReferences: []metav1.OwnerReference{
							{
								Name:       "test-replicaset",
								Controller: &t,
								UID:        "uid5",
							},
						},
					},
				})
				Expect(result).Should(Equal(""))
			})
		})

		Context("Where an error is returned when looking up the controller", func() {
			It("should return the empty string", func() {
				mtc.Path = eventhandlers.Path{
					func(k types.ReconcileKey) (interface{}, error) {
						return &appsv1.ReplicaSet{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test-replicaset",
								Namespace: "default",
								UID:       "uid5",
							},
						}, fmt.Errorf("error")
					},
				}
				result := mtc.Map(&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pod",
						Namespace: "default",
						OwnerReferences: []metav1.OwnerReference{
							{
								Name:       "test-replicaset",
								Controller: &t,
								UID:        "uid5",
							},
						},
					},
				})
				Expect(result).Should(Equal(""))
			})
		})

		Context("Where the returned controller doesn't have metadata", func() {
			It("should return the empty string", func() {
				mtc.Path = eventhandlers.Path{
					func(k types.ReconcileKey) (interface{}, error) {
						s := ""
						return &s, nil
					},
				}
				result := mtc.Map(&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pod",
						Namespace: "default",
						OwnerReferences: []metav1.OwnerReference{
							{
								Name:       "test-replicaset",
								Controller: &t,
								UID:        "uid5",
							},
						},
					},
				})
				Expect(result).Should(Equal(""))
			})
		})

		Context("Where the controller UID matches", func() {
			It("should return the controller's namespace/name", func() {
				mtc.Path = eventhandlers.Path{
					func(k types.ReconcileKey) (interface{}, error) {
						return &appsv1.ReplicaSet{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test-replicaset",
								Namespace: "default",
								UID:       "uid5",
							},
						}, nil
					},
				}
				result := mtc.Map(&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pod",
						Namespace: "default",
						OwnerReferences: []metav1.OwnerReference{
							{
								Name:       "test-replicaset",
								Controller: &t,
								UID:        "uid5",
							},
						},
					},
				})
				Expect(result).Should(Equal("default/test-replicaset"))
			})
		})

		Context("Where the controller UID doesn't match", func() {
			It("should not return the controller's namespace/name", func() {
				mtc.Path = eventhandlers.Path{
					func(k types.ReconcileKey) (interface{}, error) {
						defer GinkgoRecover()
						Expect(k).Should(Equal(types.ReconcileKey{Name: "test-replicaset", Namespace: "default"}))
						return &appsv1.ReplicaSet{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test-replicaset",
								Namespace: "default",
								UID:       "uid5",
							},
						}, nil
					},
				}
				result := mtc.Map(&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pod",
						Namespace: "default",
						OwnerReferences: []metav1.OwnerReference{
							{
								Name:       "test-replicaset",
								Controller: &t,
								UID:        "uid3",
							},
						},
					},
				})
				Expect(result).Should(Equal(""))
			})
		})

		Context("Where the controller maps to another controller", func() {
			It("should return the controller's-controller's namespace/name", func() {
				mtc.Path = eventhandlers.Path{
					func(k types.ReconcileKey) (interface{}, error) {
						return &appsv1.ReplicaSet{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test-replicaset",
								Namespace: "default",
								UID:       "uid5",
								OwnerReferences: []metav1.OwnerReference{
									{
										Name:       "test-deployment",
										UID:        "uid7",
										Controller: &t,
									},
								},
							},
						}, nil
					},
					func(k types.ReconcileKey) (interface{}, error) {
						return &appsv1.Deployment{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "test-deployment",
								Namespace: "default",
								UID:       "uid7",
							},
						}, nil
					},
				}
				result := mtc.Map(&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pod",
						Namespace: "default",
						OwnerReferences: []metav1.OwnerReference{
							{
								Name:       "test-replicaset",
								Controller: &t,
								UID:        "uid5",
							},
						},
					},
				})
				Expect(result).Should(Equal("default/test-deployment"))
			})
		})
	})
})

type FakePredicates struct {
	update, delete, create bool
}

func (h FakePredicates) HandleUpdate(old, new interface{}) bool { return h.update }
func (h FakePredicates) HandleDelete(obj interface{}) bool      { return h.delete }
func (h FakePredicates) HandleCreate(obj interface{}) bool      { return h.create }
