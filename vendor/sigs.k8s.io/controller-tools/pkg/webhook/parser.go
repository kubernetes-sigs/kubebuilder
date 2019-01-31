/*
Copyright 2018 The Kubernetes Authors.

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

package webhook

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-tools/pkg/internal/general"
)

const webhookAnnotationPrefix = "kubebuilder:webhook"

var (
	webhookTags = sets.NewString([]string{"groups", "versions", "resources", "verbs", "type", "name", "path", "failure-policy"}...)
	serverTags  = sets.NewString([]string{"port", "cert-dir", "service", "selector", "secret", "host", "mutating-webhook-config-name", "validating-webhook-config-name"}...)
)

// parseAnnotation parses webhook annotations
func (o *Options) parseAnnotation(commentText string) error {
	webhookKVMap, serverKVMap := map[string]string{}, map[string]string{}
	for _, comment := range strings.Split(commentText, "\n") {
		comment := strings.TrimSpace(comment)
		anno := general.GetAnnotation(comment, webhookAnnotationPrefix)
		if len(anno) == 0 {
			continue
		}
		for _, elem := range strings.Split(anno, ",") {
			key, value, err := general.ParseKV(elem)
			if err != nil {
				log.Fatalf("// +kubebuilder:webhook: tags must be key value pairs. Example "+
					"keys [groups=<group1;group2>,resources=<resource1;resource2>,verbs=<verb1;verb2>] "+
					"Got string: [%s]", anno)
			}
			switch {
			case webhookTags.Has(key):
				webhookKVMap[key] = value
			case serverTags.Has(key):
				serverKVMap[key] = value
			}
		}
	}

	if err := o.parseWebhookAnnotation(webhookKVMap); err != nil {
		return err
	}
	return o.parseServerAnnotation(serverKVMap)
}

// parseWebhookAnnotation parses webhook annotations in the same comment group
// nolint: gocyclo
func (o *Options) parseWebhookAnnotation(kvMap map[string]string) error {
	if len(kvMap) == 0 {
		return nil
	}
	rule := admissionregistrationv1beta1.RuleWithOperations{}
	w := &admissionWebhook{}
	for key, value := range kvMap {
		switch key {
		case "groups":
			values := strings.Split(value, ";")
			normalized := []string{}
			for _, v := range values {
				if v == "core" {
					normalized = append(normalized, "")
				} else {
					normalized = append(normalized, v)
				}
			}
			rule.APIGroups = values

		case "versions":
			values := strings.Split(value, ";")
			rule.APIVersions = values

		case "resources":
			values := strings.Split(value, ";")
			rule.Resources = values

		case "verbs":
			values := strings.Split(value, ";")
			var ops []admissionregistrationv1beta1.OperationType
			for _, v := range values {
				switch strings.ToLower(v) {
				case strings.ToLower(string(admissionregistrationv1beta1.Create)):
					ops = append(ops, admissionregistrationv1beta1.Create)
				case strings.ToLower(string(admissionregistrationv1beta1.Update)):
					ops = append(ops, admissionregistrationv1beta1.Update)
				case strings.ToLower(string(admissionregistrationv1beta1.Delete)):
					ops = append(ops, admissionregistrationv1beta1.Delete)
				case strings.ToLower(string(admissionregistrationv1beta1.Connect)):
					ops = append(ops, admissionregistrationv1beta1.Connect)
				case strings.ToLower(string(admissionregistrationv1beta1.OperationAll)):
					ops = append(ops, admissionregistrationv1beta1.OperationAll)
				default:
					return fmt.Errorf("unknown operation: %v", v)
				}
			}
			rule.Operations = ops

		case "type":
			switch strings.ToLower(value) {
			case "mutating":
				w.t = webhookTypeMutating
			case "validating":
				w.t = webhookTypeValidating
			default:
				return fmt.Errorf("unknown webhook type: %v", value)
			}

		case "name":
			w.name = value

		case "path":
			w.path = value

		case "failure-policy":
			switch strings.ToLower(value) {
			case strings.ToLower(string(admissionregistrationv1beta1.Ignore)):
				fp := admissionregistrationv1beta1.Ignore
				w.failurePolicy = &fp
			case strings.ToLower(string(admissionregistrationv1beta1.Fail)):
				fp := admissionregistrationv1beta1.Fail
				w.failurePolicy = &fp
			default:
				return fmt.Errorf("unknown webhook failure policy: %v", value)
			}
		}
	}
	w.rules = []admissionregistrationv1beta1.RuleWithOperations{rule}
	if o.registry == nil {
		o.registry = map[string]Webhook{}
	}
	o.registry[w.path] = w
	return nil
}

// parseWebhookAnnotation parses webhook server annotations in the same comment group
// nolint: gocyclo
func (o *Options) parseServerAnnotation(kvMap map[string]string) error {
	if len(kvMap) == 0 {
		return nil
	}
	for key, value := range kvMap {
		switch key {
		case "port":
			port, err := strconv.Atoi(value)
			if err != nil {
				return err
			}
			o.port = int32(port)
		case "cert-dir":
			o.certDir = value
		case "service":
			// format: <service=namespace:name>
			split := strings.Split(value, ":")
			if len(split) != 2 || len(split[0]) == 0 || len(split[1]) == 0 {
				return fmt.Errorf("invalid service format: expect <namespace:name>, but got %q", value)
			}
			if o.service == nil {
				o.service = &service{}
			}
			o.service.namespace = split[0]
			o.service.name = split[1]
		case "selector":
			// selector of the service. Format: <selector=label1:value1;label2:value2>
			split := strings.Split(value, ";")
			if len(split) == 0 {
				return fmt.Errorf("invalid selector format: expect <label1:value1;label2:value2>, but got %q", value)
			}
			if o.service == nil {
				o.service = &service{}
			}
			for _, v := range split {
				l := strings.Split(v, ":")
				if len(l) != 2 || len(l[0]) == 0 || len(l[1]) == 0 {
					return fmt.Errorf("invalid selector format: expect <label1:value1;label2:value2>, but got %q", value)
				}
				if o.service.selectors == nil {
					o.service.selectors = map[string]string{}
				}
				o.service.selectors[l[0]] = l[1]
			}
		case "host":
			if len(value) == 0 {
				return errors.New("host should not be empty if specified")
			}
			o.host = &value

		case "mutating-webhook-config-name":
			if len(value) == 0 {
				return errors.New("mutating-webhook-config-name should not be empty if specified")
			}
			o.mutatingWebhookConfigName = value

		case "validating-webhook-config-name":
			if len(value) == 0 {
				return errors.New("validating-webhook-config-name should not be empty if specified")
			}
			o.validatingWebhookConfigName = value

		case "secret":
			// format: <secret=namespace:name>
			split := strings.Split(value, ":")
			if len(split) != 2 || len(split[0]) == 0 || len(split[1]) == 0 {
				return fmt.Errorf("invalid secret format: expect <namespace:name>, but got %q", value)
			}
			if o.secret == nil {
				o.secret = &types.NamespacedName{}
			}
			o.secret.Namespace = split[0]
			o.secret.Name = split[1]
		}
	}
	return nil
}
