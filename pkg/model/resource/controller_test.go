/*
Copyright 2026 The Kubernetes Authors.

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

package resource

import (
	"bytes"
	log "log/slog"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Controller", func() {
	const (
		controller1        = "controller-1"
		controller2        = "controller-2"
		captain            = "captain"
		captainBackup      = "captainbackup"
		captainBackupKebab = "captain-backup"
		crewGroup          = "crew"
		captainKind        = "Captain"
		captainsPlural     = "captains"
		myKind             = "MyKind"
		myKindCtrl         = "mykind"
		myKindKebab        = "my-kind"
		myKindsPlural      = "mykinds"
	)

	DescribeTable("Validate should succeed for valid names",
		func(name string) {
			Expect(Controller{Name: name}.Validate()).To(Succeed())
		},
		Entry("a hyphenated name", "my-controller"),
		Entry("a name with numbers", controller1),
	)

	DescribeTable("Validate should fail for invalid names",
		func(name string) {
			Expect(Controller{Name: name}.Validate()).NotTo(Succeed())
		},
		Entry("an empty name", ""),
		Entry("a name with uppercase characters", "MyController"),
		Entry("a name with an underscore", "my_controller"),
	)

	Context("Controllers", func() {
		DescribeTable("Validate should succeed",
			func(controllers *Controllers) {
				Expect(controllers.Validate()).To(Succeed())
			},
			Entry("for nil controllers", nil),
			Entry("for empty controllers", &Controllers{}),
			Entry("for distinct names", &Controllers{{Name: controller1}, {Name: controller2}}),
		)

		DescribeTable("Validate should fail",
			func(controllers *Controllers) {
				Expect(controllers.Validate()).NotTo(Succeed())
			},
			Entry("for duplicate names", &Controllers{{Name: controller1}, {Name: controller1}}),
			Entry("for an invalid name", &Controllers{{Name: "Controller-1"}}),
		)

		DescribeTable("HasController should report whether a name is present",
			func(name string, expected bool) {
				controllers := &Controllers{{Name: controller1}, {Name: controller2}}
				Expect(controllers.HasController(name)).To(Equal(expected))
			},
			Entry("for the first controller", controller1, true),
			Entry("for the second controller", controller2, true),
			Entry("for an absent controller", "controller-3", false),
		)

		Context("AddController", func() {
			It("should fail on nil controllers", func() {
				var controllers *Controllers
				Expect(controllers.AddController(controller1)).NotTo(Succeed())
			})

			It("should append valid controllers in order", func() {
				controllers := &Controllers{}
				Expect(controllers.AddController(controller1)).To(Succeed())
				Expect(controllers.AddController(controller2)).To(Succeed())
				Expect(controllers.GetControllerNames()).To(Equal([]string{controller1, controller2}))
			})

			It("should reject a duplicate controller", func() {
				controllers := &Controllers{{Name: controller1}}
				Expect(controllers.AddController(controller1)).NotTo(Succeed())
				Expect(*controllers).To(HaveLen(1))
			})

			It("should reject an invalid name", func() {
				controllers := &Controllers{}
				Expect(controllers.AddController("Controller_1")).NotTo(Succeed())
				Expect(*controllers).To(BeEmpty())
			})
		})
	})

	Context("Resource.Validate reconciler names", func() {
		validate := func(kind string, names ...string) error {
			controllers := make(Controllers, 0, len(names))
			for _, name := range names {
				controllers = append(controllers, Controller{Name: name})
			}

			res := Resource{
				GVK:         GVK{Group: crewGroup, Version: "v1", Kind: kind},
				Plural:      captainsPlural,
				Controllers: &controllers,
			}

			return res.validateReconcilerNames()
		}

		It("should accept names differing only by a hyphen", func() {
			// These names generate CaptainbackupReconciler and CaptainBackupReconciler.
			Expect(validate(captainKind, captainBackup, captainBackupKebab)).To(Succeed())
		})

		It("should reject names generating the same reconciler", func() {
			Expect(validate(captainKind, captainBackupKebab, "captain--backup")).NotTo(Succeed())
		})

		It("should reject a name colliding with the kind-based default", func() {
			// Both names generate MyKindReconciler.
			Expect(validate(myKind, myKindCtrl, myKindKebab)).NotTo(Succeed())
		})

		It("should reject collisions with an unmigrated legacy controller", func() {
			res := Resource{
				GVK:    GVK{Group: crewGroup, Version: "v1", Kind: myKind},
				Plural: myKindsPlural, Controller: true, Controllers: &Controllers{{Name: myKindKebab}},
			}
			Expect(res.Validate()).To(MatchError(ContainSubstring("both generate MyKindReconciler")))
		})

		It("should surface the collision through Resource.Validate", func() {
			res := Resource{
				GVK:         GVK{Group: crewGroup, Version: "v1", Kind: myKind},
				Plural:      myKindsPlural,
				Controllers: &Controllers{{Name: myKindCtrl}, {Name: myKindKebab}},
			}
			Expect(res.Validate()).NotTo(Succeed())
		})
	})

	Context("Resource.Migrate", func() {
		DescribeTable("should leave a failed migration unchanged",
			func(res Resource, message string) {
				before := res.Copy()
				Expect(res.Migrate()).To(MatchError(ContainSubstring(message)))
				Expect(res).To(Equal(before))
			},
			Entry("a missing kind", Resource{Controller: true}, "kind cannot be empty"),
			Entry("an invalid default name", Resource{GVK: GVK{Kind: "Invalid_Kind"}, Controller: true},
				`kind "Invalid_Kind"`),
			Entry("a default reconciler collision", Resource{
				GVK: GVK{Kind: myKind}, Controller: true,
				Controllers: &Controllers{{Name: myKindKebab}},
			}, "both generate MyKindReconciler"),
			Entry("invalid existing controllers", Resource{
				GVK: GVK{Kind: captainKind}, Controller: true,
				Controllers: &Controllers{{Name: "Invalid"}},
			}, "invalid controller name"),
		)

		It("should warn when Normalize cannot migrate a legacy resource", func() {
			var output bytes.Buffer
			previous := log.Default()
			log.SetDefault(log.New(log.NewTextHandler(&output, nil)))
			DeferCleanup(func() { log.SetDefault(previous) })
			res := Resource{Controller: true}
			res.Normalize()
			Expect(output.String()).To(ContainSubstring("level=WARN"))
			Expect(output.String()).To(ContainSubstring("kind cannot be empty"))
			Expect(res.Controller).To(BeTrue())
			Expect(res.Controllers).To(BeNil())
		})

		It("should not duplicate an already recorded default controller", func() {
			res := Resource{
				GVK: GVK{Kind: captainKind}, Controller: true,
				Controllers: &Controllers{{Name: captain}},
			}
			Expect(res.Migrate()).To(Succeed())
			Expect(res.Controller).To(BeFalse())
			Expect(res.GetControllerNames()).To(Equal([]string{captain}))
		})

		It("should record a legacy controller under its default name", func() {
			res := Resource{GVK: GVK{Kind: captainKind}, Controller: true}

			Expect(res.Migrate()).To(Succeed())
			Expect(res.Controller).To(BeFalse())
			Expect(res.GetControllerNames()).To(Equal([]string{captain}))
		})

		It("should keep the default alongside names already recorded", func() {
			res := Resource{
				GVK:         GVK{Kind: captainKind},
				Controller:  true,
				Controllers: &Controllers{{Name: captainBackupKebab}},
			}

			Expect(res.Migrate()).To(Succeed())
			Expect(res.Controller).To(BeFalse())
			Expect(res.GetControllerNames()).To(Equal([]string{captainBackupKebab, captain}))
		})

		It("should be a no-op for a resource with no controller", func() {
			res := Resource{GVK: GVK{Kind: captainKind}}

			Expect(res.Migrate()).To(Succeed())
			Expect(res.HasController()).To(BeFalse())
			Expect(res.Controllers).To(BeNil())
		})

		It("should be idempotent", func() {
			res := Resource{GVK: GVK{Kind: captainKind}, Controller: true}

			Expect(res.Migrate()).To(Succeed())
			Expect(res.Migrate()).To(Succeed())
			Expect(res.GetControllerNames()).To(Equal([]string{captain}))
		})
	})

	Context("Resource.Update", func() {
		var gvk GVK

		BeforeEach(func() {
			gvk = GVK{Group: crewGroup, Version: "v1", Kind: captainKind}
		})

		DescribeTable("should reject collisions introduced by merging controllers",
			func(legacy bool) {
				res := Resource{
					GVK:    GVK{Group: crewGroup, Version: "v1", Kind: myKind},
					Plural: myKindsPlural, Controllers: &Controllers{{Name: myKindKebab}},
				}
				other := Resource{GVK: res.GVK, Plural: res.Plural, Controller: legacy}
				if !legacy {
					other.Controllers = &Controllers{{Name: myKindCtrl}}
				}
				before := res.Copy()
				Expect(res.Update(other)).To(MatchError(ContainSubstring("both generate MyKindReconciler")))
				Expect(res).To(Equal(before))
			},
			Entry("with an incoming legacy controller", true),
			Entry("with an incoming named controller", false),
		)

		It("should adopt the controllers of the other resource", func() {
			res := Resource{GVK: gvk, Plural: captainsPlural}
			other := Resource{
				GVK:         gvk,
				Plural:      captainsPlural,
				Controllers: &Controllers{{Name: captainBackupKebab}},
			}

			Expect(res.Update(other)).To(Succeed())
			Expect(res.GetControllerNames()).To(Equal([]string{captainBackupKebab}))
		})

		It("should merge controllers from both resources", func() {
			res := Resource{GVK: gvk, Plural: captainsPlural, Controllers: &Controllers{{Name: captain}}}
			other := Resource{
				GVK:         gvk,
				Plural:      captainsPlural,
				Controllers: &Controllers{{Name: captainBackupKebab}},
			}

			Expect(res.Update(other)).To(Succeed())
			Expect(res.GetControllerNames()).To(Equal([]string{captain, captainBackupKebab}))
		})

		It("should keep its controllers when the other resource has none", func() {
			res := Resource{GVK: gvk, Plural: captainsPlural, Controllers: &Controllers{{Name: captain}}}
			other := Resource{GVK: gvk, Plural: captainsPlural}

			Expect(res.Update(other)).To(Succeed())
			Expect(res.GetControllerNames()).To(Equal([]string{captain}))
		})

		It("should migrate a legacy controller into the controllers list", func() {
			res := Resource{GVK: gvk, Plural: captainsPlural, Controller: true}
			other := Resource{
				GVK:         gvk,
				Plural:      captainsPlural,
				Controllers: &Controllers{{Name: captainBackupKebab}},
			}

			Expect(res.Update(other)).To(Succeed())
			Expect(res.Controller).To(BeFalse())
			Expect(res.GetControllerNames()).To(Equal([]string{captain, captainBackupKebab}))
		})

		It("should migrate a legacy controller coming from the other resource", func() {
			res := Resource{GVK: gvk, Plural: captainsPlural}
			other := Resource{GVK: gvk, Plural: captainsPlural, Controller: true}

			Expect(res.Update(other)).To(Succeed())
			Expect(res.Controller).To(BeFalse())
			Expect(res.GetControllerNames()).To(Equal([]string{captain}))
		})
	})

	DescribeTable("Resource.GetControllerNames",
		func(res Resource, expected []string) {
			Expect(res.GetControllerNames()).To(Equal(expected))
		},
		Entry("returns the names from the controllers list",
			Resource{
				GVK:         GVK{Kind: myKind},
				Controllers: &Controllers{{Name: controller1}, {Name: controller2}},
			},
			[]string{controller1, controller2}),
		Entry("returns the kind-based default for a legacy controller",
			Resource{GVK: GVK{Kind: myKind}, Controller: true},
			[]string{myKindCtrl}),
		Entry("returns nil when there is no controller",
			Resource{GVK: GVK{Kind: myKind}},
			nil),
		Entry("reports the legacy controller alongside the list",
			Resource{
				GVK:         GVK{Kind: myKind},
				Controller:  true,
				Controllers: &Controllers{{Name: "custom-controller"}},
			},
			[]string{"custom-controller", myKindCtrl}),
		Entry("does not repeat a legacy controller already in the list",
			Resource{
				GVK:         GVK{Kind: myKind},
				Controller:  true,
				Controllers: &Controllers{{Name: myKindCtrl}},
			},
			[]string{myKindCtrl}),
	)

	It("should report the same controllers that Migrate records", func() {
		for _, res := range []Resource{
			{GVK: GVK{Kind: captainKind}, Controller: true},
			{GVK: GVK{Kind: captainKind}, Controllers: &Controllers{{Name: captainBackupKebab}}},
			{GVK: GVK{Kind: captainKind}, Controller: true, Controllers: &Controllers{{Name: captainBackupKebab}}},
			{GVK: GVK{Kind: captainKind}},
		} {
			before := res.GetControllerNames()
			Expect(res.Migrate()).To(Succeed())
			Expect(res.GetControllerNames()).To(ConsistOf(before))
		}
	})

	DescribeTable("NormalizeReconcilerName",
		func(controllerName, kind, expected string) {
			Expect(NormalizeReconcilerName(controllerName, kind)).To(Equal(expected))
		},
		Entry("falls back to the kind when no name is given", "", captainKind, "CaptainReconciler"),
		Entry("falls back to the kind when the name matches it", captain, captainKind, "CaptainReconciler"),
		Entry("converts a hyphenated name to PascalCase",
			captainBackupKebab, captainKind, "CaptainBackupReconciler"),
	)

	DescribeTable("GetControllerName",
		func(controllerName, kind, group string, multiGroup bool, expected string) {
			Expect(GetControllerName(controllerName, kind, group, multiGroup)).To(Equal(expected))
		},
		Entry("uses the lowercase kind when no name is given", "", captainKind, crewGroup, false, captain),
		Entry("uses the given name", captainBackupKebab, captainKind, crewGroup, false, captainBackupKebab),
		Entry("prefixes the group in multigroup projects", captain, captainKind, crewGroup, true, "crew-captain"),
		Entry("skips the prefix when there is no group", captain, captainKind, "", true, captain),
	)

	DescribeTable("NormalizeFileName",
		func(controllerName, expected string) {
			Expect(NormalizeFileName(controllerName)).To(Equal(expected))
		},
		Entry("leaves a plain name untouched", captain, captain),
		Entry("replaces hyphens with underscores", captainBackupKebab, "captain_backup"),
	)

	DescribeTable("DefaultControllerName",
		func(kind, expected string) {
			Expect(DefaultControllerName(kind)).To(Equal(expected))
		},
		Entry("lowercases a single-word kind", captainKind, captain),
		Entry("lowercases a multi-word kind", myKind, myKindCtrl),
	)
})
