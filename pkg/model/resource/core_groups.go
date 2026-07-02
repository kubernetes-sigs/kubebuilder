package resource

const k8sIODomainSuffix = "k8s.io"

// coreGroups maps well-known core Kubernetes API group names to their domain suffix.
// An empty string means the group is at the root level (e.g., "apps", "batch").
// A non-empty string is the domain suffix (e.g., "k8s.io").
var coreGroups = map[string]string{
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

// CoreGroupDomain returns the domain suffix for a well-known core Kubernetes API group.
// If the group is a core group, it returns the domain (empty for root-level groups such as
// "apps") and true. Otherwise, it returns an empty string and false.
func CoreGroupDomain(group string) (string, bool) {
	domain, ok := coreGroups[group]
	return domain, ok
}
