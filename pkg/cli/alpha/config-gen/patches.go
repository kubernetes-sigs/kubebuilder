package configgen

import (
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type crdPatchSelector struct {
	Config *KubebuilderConfigGen
}

func (s *crdPatchSelector) Filter(input []*yaml.RNode) ([]*yaml.RNode, error) {
	kindSelector := framework.Selector{
		Kinds: []string{"CustomResourceDefinition"},
	}
	input, err := kindSelector.Filter(input)
	if err != nil {
		return nil, err
	}
	var keep []*yaml.RNode
	for i := range input {
		m, _ := input[i].GetMeta()
		if _, ok := s.Config.Spec.Webhooks.Conversions[m.Name]; ok {
			keep = append(keep, input[i])
		}
	}
	return keep, nil
}
