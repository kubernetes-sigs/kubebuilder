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

package config

import (
	"fmt"
	"sort"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/golang/glog"
	"github.com/kubernetes-sigs/kubebuilder/cmd/internal/codegen/parse"
	"github.com/kubernetes-sigs/kubebuilder/cmd/kubebuilder/util"
	"github.com/kubernetes-sigs/kubebuilder/cmd/kubebuilder/version"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	extensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/gengo/args"
)

// CodeGenerator generates code for Kubernetes resources and controllers
type CodeGenerator struct{}

var kblabels = map[string]string{
	"kubebuilder.k8s.io": version.GetVersion().KubeBuilderVersion,
}

func addLabels(m map[string]string) map[string]string {
	for k, v := range kblabels {
		m[k] = v
	}
	m["api"] = name
	return m
}

// Execute parses packages and executes the code generators against the resource and controller packages
func (g CodeGenerator) Execute() error {
	arguments := args.Default()
	b, err := arguments.NewBuilder()
	if err != nil {
		return fmt.Errorf("Failed making a parser: %v", err)
	}
	for _, d := range []string{"./pkg/apis", "./pkg/controller", "./pkg/inject"} {
		if err := b.AddDirRecursive(d); err != nil {
			return fmt.Errorf("Failed making a parser: %v", err)
		}
	}
	c, err := parse.NewContext(b)
	if err != nil {
		return fmt.Errorf("Failed making a context: %v", err)
	}

	p := parse.NewAPIs(c, arguments)
	if crds {
		util.WriteString(output, strings.Join(getCrds(p), "---\n"))
		return nil
	}

	result := append([]string{},
		getNamespace(p),
		getClusterRole(p),
		getClusterRoleBinding(p),
	)
	result = append(result, getCrds(p)...)
	if controllerType == "deployment" {
		result = append(result, getDeployment(p))
	} else {
		result = append(result, getStatefuleSetService(p))
		result = append(result, getStatefuleSet(p))
	}

	util.WriteString(output, strings.Join(result, "---\n"))
	return nil
}

func getClusterRole(p *parse.APIs) string {
	rules := []rbacv1.PolicyRule{}
	for _, rule := range p.Rules {
		rules = append(rules, rule)
	}
	for _, g := range p.APIs.Groups {
		rule := rbacv1.PolicyRule{
			APIGroups: []string{g.Group + "." + g.Domain},
			Resources: []string{"*"},
			Verbs:     []string{"*"},
		}
		rules = append(rules, rule)
	}
	role := rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterRole",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   name + "-role",
			Labels: addLabels(map[string]string{}),
		},
		Rules: rules,
	}
	s, err := yaml.Marshal(role)
	if err != nil {
		glog.Fatalf("Error: %v", err)
	}
	return string(s)
}

func getClusterRoleBinding(p *parse.APIs) string {
	rolebinding := &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-rolebinding", name),
			Namespace: fmt.Sprintf("%s-system", name),
			Labels:    addLabels(map[string]string{}),
		},
		Subjects: []rbacv1.Subject{
			{
				Name:      "default",
				Namespace: fmt.Sprintf("%v-system", name),
				Kind:      "ServiceAccount",
			},
		},
		RoleRef: rbacv1.RoleRef{
			Name:     fmt.Sprintf("%v-role", name),
			Kind:     "ClusterRole",
			APIGroup: "rbac.authorization.k8s.io",
		},
	}

	s, err := yaml.Marshal(rolebinding)
	if err != nil {
		glog.Fatalf("Error: %v", err)
	}
	return string(s)
}

func getDeployment(p *parse.APIs) string {
	var replicas int32 = 1
	labels := addLabels(map[string]string{
		"control-plane": "controller-manager",
	})
	dep := appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%v-controller-manager", name),
			Namespace: fmt.Sprintf("%v-system", name),
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: getPodTemplate(labels),
		},
	}

	s, err := yaml.Marshal(dep)
	if err != nil {
		glog.Fatalf("Error: %v", err)
	}
	return string(s)
}

func getStatefuleSet(p *parse.APIs) string {
	var replicas int32 = 1
	labels := addLabels(map[string]string{
		"control-plane": "controller-manager",
	})
	statefulset := appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "StatefulSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%v-controller-manager", name),
			Namespace: fmt.Sprintf("%v-system", name),
			Labels:    labels,
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: fmt.Sprintf("%v-controller-manager-service", name),
			Replicas:    &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: getPodTemplate(labels),
		},
	}

	s, err := yaml.Marshal(statefulset)
	if err != nil {
		glog.Fatalf("Error: %v", err)
	}
	return string(s)

}

func getStatefuleSetService(p *parse.APIs) string {
	labels := addLabels(map[string]string{
		"control-plane": "controller-manager",
	})
	statefulsetservice := corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%v-controller-manager-service", name),
			Namespace: fmt.Sprintf("%v-system", name),
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Selector:  labels,
			ClusterIP: "None",
		},
	}

	s, err := yaml.Marshal(statefulsetservice)
	if err != nil {
		glog.Fatalf("Error: %v", err)
	}
	return string(s)

}

func getPodTemplate(labels map[string]string) corev1.PodTemplateSpec {
	var terminationPeriod int64 = 10
	return corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: labels,
		},
		Spec: corev1.PodSpec{
			TerminationGracePeriodSeconds: &terminationPeriod,
			Containers: []corev1.Container{
				{
					Name:    "controller-manager",
					Image:   controllerImage,
					Command: []string{"/root/controller-manager"},
					Args:    []string{"--install-crds=false"},
					Resources: corev1.ResourceRequirements{
						Requests: map[corev1.ResourceName]resource.Quantity{
							"cpu":    resource.MustParse("100m"),
							"memory": resource.MustParse("20Mi"),
						},
						Limits: map[corev1.ResourceName]resource.Quantity{
							"cpu":    resource.MustParse("100m"),
							"memory": resource.MustParse("30Mi"),
						},
					},
				},
			},
		},
	}
}

func getCrds(p *parse.APIs) []string {
	crds := []extensionsv1beta1.CustomResourceDefinition{}
	for _, g := range p.APIs.Groups {
		for _, v := range g.Versions {
			for _, r := range v.Resources {
				crd := r.CRD
				if len(crdNamespace) > 0 {
					crd.Namespace = crdNamespace
				}
				crd.Labels = addLabels(map[string]string{})
				crds = append(crds, crd)
			}
		}
	}

	sort.Slice(crds, func(i, j int) bool {
		iGroup := crds[i].Spec.Group
		jGroup := crds[j].Spec.Group

		if iGroup != jGroup {
			return iGroup < jGroup
		}

		iKind := crds[i].Spec.Names.Kind
		jKind := crds[j].Spec.Names.Kind

		return iKind < jKind
	})

	result := []string{}
	for i := range crds {
		s, err := yaml.Marshal(crds[i])
		if err != nil {
			glog.Fatalf("Error: %v", err)
		}
		result = append(result, string(s))
	}

	return result
}

func getNamespace(p *parse.APIs) string {
	ns := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   fmt.Sprintf("%v-system", name),
			Labels: addLabels(map[string]string{}),
		},
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Namespace",
		},
	}
	s, err := yaml.Marshal(ns)
	if err != nil {
		glog.Fatalf("Error: %v", err)
	}
	return string(s)
}
