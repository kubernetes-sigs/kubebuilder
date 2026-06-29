package coregroups

const k8sIODomainSuffix = "k8s.io"

// Groups maps core Kubernetes API group names to their domain suffix.
// An empty string means the group is at the root (e.g., "apps", "batch").
// A non-empty string is the domain suffix (e.g., "k8s.io").
var Groups = map[string]string{
	"admission":             k8sIODomainSuffix,
	"admissionregistration": k8sIODomainSuffix,
	"apps":                  "",
	"auditregistration":     k8sIODomainSuffix,
	"apiextensions":         k8sIODomainSuffix,
	"authentication":        k8sIODomainSuffix,
	"authorization":         k8sIODomainSuffix,
	"autoscaling":           "",
	"batch":                 "",
	"certificates":          k8sIODomainSuffix,
	"coordination":          k8sIODomainSuffix,
	"core":                  "",
	"events":                k8sIODomainSuffix,
	"extensions":            "",
	"imagepolicy":           k8sIODomainSuffix,
	"networking":            k8sIODomainSuffix,
	"node":                  k8sIODomainSuffix,
	"metrics":               k8sIODomainSuffix,
	"policy":                "",
	"rbac.authorization":    k8sIODomainSuffix,
	"scheduling":            k8sIODomainSuffix,
	"setting":               k8sIODomainSuffix,
	"storage":               k8sIODomainSuffix,
}
