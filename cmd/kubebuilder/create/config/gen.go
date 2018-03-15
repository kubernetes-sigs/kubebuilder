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
	"strings"

	"github.com/ghodss/yaml"
	"github.com/golang/glog"
	"github.com/kubernetes-sigs/kubebuilder/cmd/internal/codegen/parse"
	"github.com/kubernetes-sigs/kubebuilder/cmd/kubebuilder/util"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/gengo/args"
)

// CodeGenerator generates code for Kubernetes resources and controllers
type CodeGenerator struct{}

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
	result := append([]string{},
		getNamespace(p),
		getClusterRole(p),
		getClusterRoleBinding(p),
	)
	result = append(result, getCrds(p)...)
	result = append(result, getDeployment(p))

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
			Name: name + "-role",
			Labels: map[string]string{
				"api": name,
			},
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
			Labels: map[string]string{
				"api": name,
			},
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
	labels := map[string]string{
		"api": name,
	}
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
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
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
			},
		},
	}

	s, err := yaml.Marshal(dep)
	if err != nil {
		glog.Fatalf("Error: %v", err)
	}
	return string(s)
}

func getCrds(p *parse.APIs) []string {
	result := []string{}
	for _, g := range p.APIs.Groups {
		for _, v := range g.Versions {
			for _, r := range v.Resources {
				crd := r.CRD
				crd.Labels = map[string]string{
					"api": name,
				}
				s, err := yaml.Marshal(crd)
				if err != nil {
					glog.Fatalf("Error: %v", err)
				}
				result = append(result, string(s))
			}
		}
	}
	return result
}

func getNamespace(p *parse.APIs) string {
	ns := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%v-system", name),
			Labels: map[string]string{
				"api": name,
			},
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
